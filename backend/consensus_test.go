package main

import (
	"math"
	"testing"
)

func ptr(v int) *int { return &v }

// TestRunConsensusKaolackGolden reproduces the doc's Kaolack example exactly:
// median 187, confidence 100, rewards A=6.5 B=5 C=5 FAO=0.5 D=0.
func TestRunConsensusKaolackGolden(t *testing.T) {
	subs := []Submission{
		{NodeID: "A", Tier: 2, RepScore: 88, Value: 187, ProofScore: ptr(94), Channel: "app"},
		{NodeID: "B", Tier: 2, RepScore: 72, Value: 191, ProofScore: ptr(91), Channel: "whatsapp"},
		{NodeID: "C", Tier: 2, RepScore: 65, Value: 185, ProofScore: ptr(89), Channel: "app"},
		{NodeID: "FAO", Tier: 3, RepScore: 0, Value: 190, ProofScore: nil, Channel: "api"},
		{NodeID: "D", Tier: 2, RepScore: 61, Value: 240, ProofScore: ptr(12), Channel: "app"},
	}

	res := runConsensus(subs, 5.0)

	if res.Median != 187 {
		t.Fatalf("median = %v, want 187", res.Median)
	}
	if res.Confidence != 100 {
		t.Fatalf("confidence = %d, want 100", res.Confidence)
	}
	if !res.Validated {
		t.Fatalf("validated = false, want true")
	}

	wantReward := map[string]float64{"A": 6.5, "B": 5, "C": 5, "FAO": 0.5, "D": 0}
	for _, o := range res.Outcomes {
		want := wantReward[o.NodeID]
		if math.Abs(o.Reward-want) > 1e-9 {
			t.Errorf("reward[%s] = %v, want %v", o.NodeID, o.Reward, want)
		}
		if o.NodeID == "D" {
			if o.Status != "excluded" {
				t.Errorf("D status = %q, want excluded", o.Status)
			}
			if o.ScoreDelta != -2 {
				t.Errorf("D scoreDelta = %v, want -2", o.ScoreDelta)
			}
		}
	}
}

// Automatic node (ProofScore nil) is accepted at tier weight 0.7.
func TestAutomaticNodeWeight(t *testing.T) {
	sub := Submission{NodeID: "FAO", Tier: 3, Value: 190, ProofScore: nil, Channel: "api"}
	if !sub.Accepted() {
		t.Fatalf("automatic node should be accepted")
	}
	if w := sub.Weight(); math.Abs(w-0.7) > 1e-9 {
		t.Fatalf("automatic node weight = %v, want 0.7", w)
	}
}

// Proof score threshold: < 50 excluded, >= 50 accepted.
func TestAcceptedProofThreshold(t *testing.T) {
	low := Submission{NodeID: "low", Tier: 2, ProofScore: ptr(49)}
	high := Submission{NodeID: "high", Tier: 2, ProofScore: ptr(50)}
	if low.Accepted() {
		t.Fatalf("proof score 49 should be rejected")
	}
	if !high.Accepted() {
		t.Fatalf("proof score 50 should be accepted")
	}
}
