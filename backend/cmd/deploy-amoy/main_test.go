package main

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestRequireAmoy(t *testing.T) {
	if err := requireAmoy(big.NewInt(80002)); err != nil {
		t.Fatalf("Amoy rejected: %v", err)
	}
	if err := requireAmoy(big.NewInt(137)); err == nil {
		t.Fatal("Polygon mainnet accepted; deployment must stop before sending")
	}
	if err := requireAmoy(nil); err == nil {
		t.Fatal("nil chain ID accepted")
	}
}

func TestInsuranceThreshold(t *testing.T) {
	if got, err := insuranceThreshold(""); err != nil || got.Int64() != 20000 {
		t.Fatalf("default threshold = %v, %v", got, err)
	}
	if got, err := insuranceThreshold("18750"); err != nil || got.Int64() != 18750 {
		t.Fatalf("configured threshold = %v, %v", got, err)
	}
	for _, invalid := range []string{"0", "-1", "not-a-number"} {
		if _, err := insuranceThreshold(invalid); err == nil {
			t.Errorf("threshold %q accepted", invalid)
		}
	}
}

func TestReadBytecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contract.bin")
	if err := os.WriteFile(path, []byte("0x60006000\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readBytecode(path)
	if err != nil || len(got) != 4 || got[0] != 0x60 {
		t.Fatalf("readBytecode = %x, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("not hex"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBytecode(path); err == nil {
		t.Fatal("invalid bytecode accepted")
	}
}
