package model

import (
	"context"
	"crypto/ecdsa"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// agriABIJSON is a copy of the AgriculturalData entry of contracts/abi.json
// (go:embed cannot reach outside the module); chain_test.go fails on drift.
//
//go:embed AgriculturalData.abi.json
var agriABIJSON []byte

const amoyChainID = 80002

// ChainClient is the subset of an Ethereum client the publisher needs
// (ethclient.Client and the in-process simulated backend both satisfy it).
type ChainClient interface {
	bind.ContractBackend
	bind.DeployBackend
	ChainID(ctx context.Context) (*big.Int, error)
}

// Publisher sends AgriculturalData.publish transactions and checks the receipt.
type Publisher struct {
	client      ChainClient
	key         *ecdsa.PrivateKey
	contract    *bind.BoundContract
	zone        string
	produit     string
	requireAmoy bool
}

// NewPublisher binds to an already deployed AgriculturalData contract.
func NewPublisher(c ChainClient, key *ecdsa.PrivateKey, contract common.Address, zone, produit string) (*Publisher, error) {
	return newPublisher(c, key, contract, zone, produit, true)
}

// newPublisher can target a local simulated EVM in tests. Production callers
// must use NewPublisher, which only permits Polygon Amoy.
func newPublisher(c ChainClient, key *ecdsa.PrivateKey, contract common.Address, zone, produit string, requireAmoy bool) (*Publisher, error) {
	parsed, err := abi.JSON(strings.NewReader(string(agriABIJSON)))
	if err != nil {
		return nil, err
	}
	return &Publisher{
		client:      c,
		key:         key,
		contract:    bind.NewBoundContract(contract, parsed, c, c, c),
		zone:        zone,
		produit:     produit,
		requireAmoy: requireAmoy,
	}, nil
}

// chainLabel names a chain only when it is known. Amoy is the only chain we
// call "polygon-amoy"; any other (local node, other testnet) is "evm-<id>".
func chainLabel(id *big.Int) string {
	if id.Cmp(big.NewInt(amoyChainID)) == 0 {
		return "polygon-amoy"
	}
	return "evm-" + id.String()
}

// Publish sends the transaction, waits for the receipt and returns the real tx
// hash and the receipt-checked chain label. On any failure (send error, revert,
// timeout) it returns an error and no label.
func (p *Publisher) Publish(ctx context.Context, result ConsensusResult, report any) (tx, chain string, err error) {
	id, err := p.client.ChainID(ctx)
	if err != nil {
		return "", "", fmt.Errorf("chain id: %w", err)
	}
	if p.requireAmoy && id.Cmp(big.NewInt(amoyChainID)) != 0 {
		return "", "", fmt.Errorf("refusing to publish: expected Polygon Amoy chain id %d, connected to %s", amoyChainID, id)
	}
	opts := bind.NewKeyedTransactor(p.key, id)
	opts.Context = ctx
	date := time.Now().UTC().Format("2006-01-02")
	valueX100 := big.NewInt(int64(math.Round(result.Median * 100)))
	sent, err := p.contract.Transact(opts, "publish",
		p.zone, p.produit, date, valueX100, big.NewInt(int64(result.Confidence)),
		"afrioracle", reportHash(result), reportHash(report))
	if err != nil {
		return "", "", fmt.Errorf("send publish: %w", err)
	}
	rcpt, err := bind.WaitMined(ctx, p.client, sent.Hash())
	if err != nil {
		return "", "", fmt.Errorf("wait receipt %s: %w", sent.Hash(), err)
	}
	if rcpt.Status != types.ReceiptStatusSuccessful {
		return "", "", fmt.Errorf("publish tx %s reverted", sent.Hash())
	}
	return sent.Hash().Hex(), chainLabel(id), nil
}

var (
	chainMu   sync.Mutex
	chainPub  *Publisher
	chainOnce *Publisher // test override
)

// UseChain overrides the publisher (tests); nil restores env-driven behaviour.
func UseChain(p *Publisher) {
	chainMu.Lock()
	defer chainMu.Unlock()
	chainOnce = p
	chainPub = nil
}

// chainConfigured reports whether a real chain publish is requested. All of
// POLYGON_RPC, PRIVATE_KEY and CONTRACT_ADDRESS are needed; otherwise we stay
// simulated and say so.
func chainConfigured() bool {
	chainMu.Lock()
	over := chainOnce != nil
	chainMu.Unlock()
	return over || (os.Getenv("POLYGON_RPC") != "" && os.Getenv("PRIVATE_KEY") != "" && os.Getenv("CONTRACT_ADDRESS") != "")
}

func envPublisher() (*Publisher, error) {
	chainMu.Lock()
	defer chainMu.Unlock()
	if chainOnce != nil {
		return chainOnce, nil
	}
	if chainPub != nil {
		return chainPub, nil
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(os.Getenv("PRIVATE_KEY"), "0x"))
	if err != nil {
		return nil, errors.New("invalid PRIVATE_KEY")
	}
	addr := os.Getenv("CONTRACT_ADDRESS")
	if !common.IsHexAddress(addr) {
		return nil, errors.New("invalid CONTRACT_ADDRESS")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cl, err := ethclient.DialContext(ctx, os.Getenv("POLYGON_RPC"))
	if err != nil {
		return nil, fmt.Errorf("dial POLYGON_RPC: %w", err)
	}
	zone, produit := envOr("ORACLE_ZONE", "Kaolack"), envOr("ORACLE_PRODUIT", "millet")
	p, err := NewPublisher(cl, key, common.HexToAddress(addr), zone, produit)
	if err != nil {
		return nil, err
	}
	chainPub = p
	return p, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// chainPublish publishes on-chain when configured; otherwise returns the
// deterministic simulated hash labelled "simulated". It never reports a real
// label unless a mined, successful receipt exists.
func chainPublish(result ConsensusResult, report any) (tx, chain string, err error) {
	if !chainConfigured() {
		return reportHash(report), "simulated", nil
	}
	p, err := envPublisher()
	if err != nil {
		return "", "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return p.Publish(ctx, result, report)
}

// ChainPublish is the controller-facing entry point.
func ChainPublish(result ConsensusResult, report any) (tx, chain string, err error) {
	return chainPublish(result, report)
}
