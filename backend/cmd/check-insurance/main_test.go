package main

import (
	"math/big"
	"testing"
)

func TestRequireAmoy(t *testing.T) {
	if err := requireAmoy(big.NewInt(80002)); err != nil {
		t.Fatalf("Amoy rejected: %v", err)
	}
	for _, chainID := range []*big.Int{big.NewInt(137), big.NewInt(1337), nil} {
		if err := requireAmoy(chainID); err == nil {
			t.Errorf("chain ID %v accepted", chainID)
		}
	}
}

func TestValidateDate(t *testing.T) {
	if err := validateDate("2026-11-02"); err != nil {
		t.Fatalf("valid ISO date rejected: %v", err)
	}
	for _, date := range []string{"2026-02-30", "02/11/2026", ""} {
		if err := validateDate(date); err == nil {
			t.Errorf("date %q accepted", date)
		}
	}
}

func TestFormatX100(t *testing.T) {
	for input, want := range map[string]string{
		"0":     "0.00",
		"5":     "0.05",
		"1300":  "13.00",
		"18700": "187.00",
	} {
		if got := formatX100(new(big.Int).SetInt64(mustInt64(t, input))); got != want {
			t.Errorf("formatX100(%s) = %s, want %s", input, got, want)
		}
	}
}

func mustInt64(t *testing.T, value string) int64 {
	t.Helper()
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || !parsed.IsInt64() {
		t.Fatalf("invalid int64 test value %q", value)
	}
	return parsed.Int64()
}
