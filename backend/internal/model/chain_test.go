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
	data := e.creationCode(t, name, args...)
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

func (e *chainEnv) creationCode(t *testing.T, name string, args ...any) []byte {
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
	return data
}

func (e *chainEnv) deploymentReverts(t *testing.T, name, wantRevert string, args ...any) {
	t.Helper()
	opts := bind.NewKeyedTransactor(e.key, big.NewInt(1337))
	c := bind.NewBoundContract(common.Address{}, abi.ABI{}, e.cl, e.cl, e.cl)
	tx, err := c.RawCreationTransact(opts, e.creationCode(t, name, args...))
	if err != nil {
		if !strings.Contains(err.Error(), wantRevert) {
			t.Fatalf("deployment failed for unexpected reason: %v", err)
		}
		return // gas estimation rejects the constructor revert before broadcast
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, e.cl, tx.Hash())
	if err != nil {
		t.Fatalf("wait on invalid deployment tx: %v", err)
	}
	if receipt.Status == types.ReceiptStatusSuccessful {
		t.Fatalf("%s deployed successfully with invalid constructor arguments", name)
	}
}

func validResult() ConsensusResult {
	return ConsensusResult{Median: 187, Confidence: 100, Validated: true}
}

func TestOracleRejectsZeroPriceAndExcessConfidence(t *testing.T) {
	e := newChainEnv(t)
	contractABI := mustABI(t, "AgriculturalData")
	bound := bind.NewBoundContract(e.agri, contractABI, e.cl, e.cl, e.cl)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, tc := range []struct {
		name       string
		valueX100  int64
		confidence int64
		wantRevert string
	}{
		{name: "zero price", valueX100: 0, confidence: 100, wantRevert: "price must be positive"},
		{name: "confidence above 100", valueX100: 18700, confidence: 101, wantRevert: "confidence out of range"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := bind.NewKeyedTransactor(e.key, big.NewInt(1337))
			opts.Context = ctx
			tx, err := bound.Transact(opts, "publish", "Kaolack", "millet", time.Now().UTC().Format("2006-01-02"), big.NewInt(tc.valueX100), big.NewInt(tc.confidence), "test", "consensus", "report")
			if err != nil {
				if !strings.Contains(err.Error(), tc.wantRevert) {
					t.Fatalf("publication failed for unexpected reason: %v", err)
				}
				return
			}
			receipt, err := bind.WaitMined(ctx, e.cl, tx.Hash())
			if err != nil {
				t.Fatalf("wait on invalid publication: %v", err)
			}
			if receipt.Status == types.ReceiptStatusSuccessful {
				t.Fatalf("invalid settlement was published (tx %s)", tx.Hash())
			}
		})
	}
}

func TestInsuranceRejectsInvalidConstructorArgs(t *testing.T) {
	e := newChainEnv(t)
	e.deploymentReverts(t, "ParametricInsurance", "oracle address is zero", common.Address{}, big.NewInt(20000))
	e.deploymentReverts(t, "ParametricInsurance", "threshold must be positive", e.agri, big.NewInt(0))
}

func TestOracleStringKeysAreUnambiguous(t *testing.T) {
	e := newChainEnv(t)
	contractABI := mustABI(t, "AgriculturalData")
	bound := bind.NewBoundContract(e.agri, contractABI, e.cl, e.cl, e.cl)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	publish := func(zone, produit, date string, value int64) {
		opts := bind.NewKeyedTransactor(e.key, big.NewInt(1337))
		opts.Context = ctx
		transactAndMine(t, ctx, e.cl, bound, opts, "publish", zone, produit, date, big.NewInt(value), big.NewInt(90), "test", "consensus", "report")
	}
	publish("A", "BC", "2026-11-02", 11100)
	publish("AB", "C", "2026-11-02", 22200)

	for _, tc := range []struct {
		zone, produit, date string
		want                int64
	}{
		{"A", "BC", "2026-11-02", 11100},
		{"AB", "C", "2026-11-02", 22200},
	} {
		out, err := e.cl.CallContract(ctx, callMsg(e.agri, contractABI, "getPrice", tc.zone, tc.produit, tc.date), nil)
		if err != nil {
			t.Fatal(err)
		}
		values, err := contractABI.Unpack("getPrice", out)
		if err != nil {
			t.Fatal(err)
		}
		if got := values[0].(*big.Int).Int64(); got != tc.want {
			t.Errorf("getPrice(%q,%q,%q) = %d, want %d", tc.zone, tc.produit, tc.date, got, tc.want)
		}
	}

	for _, tc := range []struct {
		zone, date string
		value      int64
	}{
		{"A", "BC", 120},
		{"AB", "C", 340},
	} {
		opts := bind.NewKeyedTransactor(e.key, big.NewInt(1337))
		opts.Context = ctx
		transactAndMine(t, ctx, e.cl, bound, opts, "setRainfall", tc.zone, tc.date, big.NewInt(tc.value))
	}
	for _, tc := range []struct {
		zone, date string
		want       int64
	}{
		{"A", "BC", 120},
		{"AB", "C", 340},
	} {
		out, err := e.cl.CallContract(ctx, callMsg(e.agri, contractABI, "getRainfall", tc.zone, tc.date), nil)
		if err != nil {
			t.Fatal(err)
		}
		values, err := contractABI.Unpack("getRainfall", out)
		if err != nil {
			t.Fatal(err)
		}
		if got := values[0].(*big.Int).Int64(); got != tc.want {
			t.Errorf("getRainfall(%q,%q) = %d, want %d", tc.zone, tc.date, got, tc.want)
		}
	}
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
	ins := e.deploy(t, "ParametricInsurance", e.agri, big.NewInt(20000))
	insABI := mustABI(t, "ParametricInsurance")
	date := time.Now().UTC().Format("2006-01-02")
	if _, err := e.cl.CallContract(context.Background(), callMsg(ins, insABI, "check", "Kaolack", date), nil); err == nil {
		t.Fatal("insurance check should fail when the oracle has no price for the date")
	}
	p, err := newPublisher(e.cl, e.key, e.agri, "Kaolack", "millet", false)
	if err != nil {
		t.Fatal(err)
	}
	UseChain(p)
	t.Cleanup(func() { UseChain(nil) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report := map[string]any{"median": 187.0}
	tx, chain, err := p.Publish(ctx, validResult(), report)
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
	agriABI := mustABI(t, "AgriculturalData")
	recordData, err := e.cl.CallContract(ctx, callMsg(e.agri, agriABI, "getPriceRecord", "Kaolack", "millet", date), nil)
	if err != nil {
		t.Fatal(err)
	}
	record, err := agriABI.Unpack("getPriceRecord", recordData)
	if err != nil {
		t.Fatal(err)
	}
	if record[0].(*big.Int).Int64() != 18700 || record[1].(*big.Int).Sign() == 0 || record[2].(*big.Int).Int64() != 100 ||
		record[3].(string) != "afrioracle" || record[4].(string) != ReportHash(validResult()) ||
		record[5].(string) != ReportHash(report) {
		t.Fatalf("getPriceRecord returned unexpected record: %#v", record)
	}

	// An insurer contract floor of 200.00: price 187.00 => payout 13.00 (x100 = 1300).
	out, err := e.cl.CallContract(ctx, callMsg(ins, insABI, "check", "Kaolack", date), nil)
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
	p, _ := newPublisher(e.cl, other, e.agri, "Kaolack", "millet", false)
	tx, chain, err := p.Publish(ctx, validResult(), map[string]any{})
	if err == nil {
		t.Fatalf("expected error from onlyPublisher revert, got tx=%q chain=%q", tx, chain)
	}
	if chain == "polygon-amoy" || strings.HasPrefix(chain, "evm-") {
		t.Fatalf("failed publish carries real label %q", chain)
	}
}

func TestPublisherRefusesNonAmoyNetwork(t *testing.T) {
	e := newChainEnv(t)
	p, err := NewPublisher(e.cl, e.key, e.agri, "Kaolack", "millet")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if tx, chain, err := p.Publish(ctx, validResult(), map[string]any{}); err == nil || tx != "" || chain != "" {
		t.Fatalf("Publish on chain 1337 = (%q, %q, %v), want refusal without tx", tx, chain, err)
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

func transactAndMine(t *testing.T, ctx context.Context, client bind.DeployBackend, contract *bind.BoundContract, opts *bind.TransactOpts, method string, args ...any) {
	t.Helper()
	tx, err := contract.Transact(opts, method, args...)
	if err != nil {
		t.Fatalf("transact %s: %v", method, err)
	}
	receipt, err := bind.WaitMined(ctx, client, tx.Hash())
	if err != nil {
		t.Fatalf("wait for %s receipt: %v", method, err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("%s tx %s reverted", method, tx.Hash())
	}
}
