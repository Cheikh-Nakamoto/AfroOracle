package model

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
)

// The simulated EVM here is go-ethereum's in-process node: real signed
// transactions, real receipts, real contract bytecode compiled from contracts/*.sol.

type chainEnv struct {
	be      *simulated.Backend
	cl      simulated.Client
	key     *ecdsa.PrivateKey
	agri    common.Address
	stopMin func()
}

func newChainEnv(t *testing.T) *chainEnv {
	t.Helper()
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	be := simulated.NewBackend(types.GenesisAlloc{addr: {Balance: new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))}})
	cl := be.Client()
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() { // auto-mine so WaitMined returns
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-time.After(5 * time.Millisecond):
				be.Commit()
			}
		}
	}()
	t.Cleanup(func() {
		close(done)
		<-stopped // miner must exit before the backend closes
		be.Close()
	})
	e := &chainEnv{be: be, cl: cl, key: key}
	e.agri = e.deploy(t, "AgriculturalData")
	return e
}

func (e *chainEnv) deploy(t *testing.T, name string, args ...any) common.Address {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name + ".bin")
	if err != nil {
		t.Fatal(err)
	}
	data := common.FromHex(strings.TrimSpace(string(raw)))
	if len(args) > 0 {
		abiData, _ := os.ReadFile("../../../contracts/abi.json")
		var all map[string]json.RawMessage
		_ = json.Unmarshal(abiData, &all)
		a, err := abi.JSON(strings.NewReader(string(all[name])))
		if err != nil {
			t.Fatal(err)
		}
		packed, err := a.Pack("", args...)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, packed...)
	}
	opts := bind.NewKeyedTransactor(e.key, big.NewInt(1337))
	c := bind.NewBoundContract(common.Address{}, abi.ABI{}, e.cl, e.cl, e.cl)
	tx, err := c.RawCreationTransact(opts, data)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	addr, err := bind.WaitDeployed(ctx, e.cl, tx.Hash())
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func validResult() ConsensusResult {
	return ConsensusResult{Median: 187, Confidence: 100, Validated: true}
}

func TestSimulatedWhenUnconfigured(t *testing.T) {
	t.Setenv("POLYGON_RPC", "")
	t.Setenv("PRIVATE_KEY", "")
	UseChain(nil)
	report := map[string]any{"median": 187.0}
	tx, chain, err := ChainPublish(validResult(), report)
	if err != nil || chain != "simulated" || tx != ReportHash(report) {
		t.Fatalf("got %q %q %v", tx, chain, err)
	}
	if _, c := Mode(); c != "simulated" {
		t.Fatalf("mode chain = %q", c)
	}
}

func TestPartialConfigStaysSimulated(t *testing.T) {
	t.Setenv("POLYGON_RPC", "http://127.0.0.1:1")
	t.Setenv("PRIVATE_KEY", "")
	t.Setenv("CONTRACT_ADDRESS", "")
	UseChain(nil)
	_, chain, err := ChainPublish(validResult(), map[string]any{})
	if err != nil || chain != "simulated" {
		t.Fatalf("chain=%q err=%v", chain, err)
	}
}

func TestRealPublishAndInsuranceRead(t *testing.T) {
	e := newChainEnv(t)
	p, err := NewPublisher(e.cl, e.key, e.agri, "Kaolack", "millet")
	if err != nil {
		t.Fatal(err)
	}
	UseChain(p)
	t.Cleanup(func() { UseChain(nil) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, chain, err := p.Publish(ctx, validResult(), map[string]any{"median": 187.0})
	if err != nil {
		t.Fatal(err)
	}
	if chain != "evm-1337" {
		t.Fatalf("unknown chain id must not be labelled polygon-amoy, got %q", chain)
	}
	rcpt, err := e.cl.TransactionReceipt(ctx, common.HexToHash(tx))
	if err != nil || rcpt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx %s has no successful receipt: %v", tx, err)
	}
	if len(rcpt.Logs) != 1 {
		t.Fatalf("expected Published event, got %d logs", len(rcpt.Logs))
	}

	// An insurer contract floor of 200.00: price 187.00 => payout 13.00 (x100 = 1300).
	ins := e.deploy(t, "ParametricInsurance", e.agri, big.NewInt(20000))
	insABI := mustABI(t, "ParametricInsurance")
	out, err := e.cl.CallContract(ctx, callMsg(ins, insABI, "check", "Kaolack", time.Now().UTC().Format("2006-01-02")), nil)
	if err != nil {
		t.Fatal(err)
	}
	vals, _ := insABI.Unpack("check", out)
	if got := vals[0].(*big.Int).Int64(); got != 1300 {
		t.Fatalf("insurance payout = %d, want 1300", got)
	}
}

func TestRevertedTxIsNotLabelledReal(t *testing.T) {
	e := newChainEnv(t)
	other, _ := crypto.GenerateKey() // not the contract's publisher, but funded
	opts := bind.NewKeyedTransactor(e.key, big.NewInt(1337))
	otherAddr := crypto.PubkeyToAddress(other.PublicKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nonce, _ := e.cl.PendingNonceAt(ctx, crypto.PubkeyToAddress(e.key.PublicKey))
	fund := types.MustSignNewTx(e.key, types.LatestSignerForChainID(big.NewInt(1337)), &types.LegacyTx{
		Nonce: nonce, To: &otherAddr, Value: big.NewInt(params.Ether), Gas: 21000, GasPrice: big.NewInt(params.GWei * 10),
	})
	_ = opts
	if err := e.cl.SendTransaction(ctx, fund); err != nil {
		t.Fatal(err)
	}
	if _, err := bind.WaitMined(ctx, e.cl, fund.Hash()); err != nil {
		t.Fatal(err)
	}
	p, _ := NewPublisher(e.cl, other, e.agri, "Kaolack", "millet")
	tx, chain, err := p.Publish(ctx, validResult(), map[string]any{})
	if err == nil {
		t.Fatalf("expected error from onlyPublisher revert, got tx=%q chain=%q", tx, chain)
	}
	if chain == "polygon-amoy" || strings.HasPrefix(chain, "evm-") {
		t.Fatalf("failed publish carries real label %q", chain)
	}
}

func TestChainLabel(t *testing.T) {
	if chainLabel(big.NewInt(80002)) != "polygon-amoy" {
		t.Fatal("amoy label")
	}
	if chainLabel(big.NewInt(31337)) != "evm-31337" {
		t.Fatal("other label")
	}
}

func TestEmbeddedABIMatchesContracts(t *testing.T) {
	raw, _ := os.ReadFile("../../../contracts/abi.json")
	var all map[string]json.RawMessage
	_ = json.Unmarshal(raw, &all)
	var a, b any
	_ = json.Unmarshal(all["AgriculturalData"], &a)
	_ = json.Unmarshal(agriABIJSON, &b)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Fatal("model/AgriculturalData.abi.json drifted from contracts/abi.json")
	}
}

func mustABI(t *testing.T, name string) abi.ABI {
	raw, _ := os.ReadFile("../../../contracts/abi.json")
	var all map[string]json.RawMessage
	_ = json.Unmarshal(raw, &all)
	a, err := abi.JSON(strings.NewReader(string(all[name])))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func callMsg(to common.Address, a abi.ABI, method string, args ...any) ethereum.CallMsg {
	data, _ := a.Pack(method, args...)
	return ethereum.CallMsg{To: &to, Data: data}
}
