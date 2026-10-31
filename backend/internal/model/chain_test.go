package model

import "testing"

func TestChainPublishNeverClaimsPolygonWithoutReceipt(t *testing.T) {
	t.Setenv("POLYGON_RPC", "https://rpc.example.invalid")
	t.Setenv("PRIVATE_KEY", "configured-but-not-used")
	report := map[string]any{"median": 187}
	tx, chain := ChainPublish(ConsensusResult{Median: 187}, report)
	if chain != "simulated" {
		t.Fatalf("chain=%q, want simulated until a real receipt exists", chain)
	}
	if tx != ReportHash(report) {
		t.Fatalf("tx=%q, want deterministic report hash %q", tx, ReportHash(report))
	}
}
