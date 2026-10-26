# Agent 03 — Backend: Consensus + reputation (Go)

## Goal
Implement the weighted-median consensus, the single reputation grid, the penalty table, a `contextAgent` stub, and the golden unit test.

## Depends on
`spec.md` §Consensus interface.

## Read
- `spec.md` §Consensus interface
- `our_projetc.md` Couches 2–3 (tier weights, reputation grid, penalty table, anti-collusion)

## Build
- `backend/consensus.go` and `backend/consensus_test.go`. Package `main`. Nothing else.

Implement exactly:
```go
var tierWeight = map[int]float64{1: 3.0, 2: 1.0, 3: 0.7}
func reputation(score int) (string, float64, float64)
type Submission struct { NodeID string; Tier int; RepScore int; Value float64; ProofScore *int; Channel string }
func (s Submission) Weight() float64
func (s Submission) Accepted() bool
type Outcome struct { NodeID, Status, Note string; Reward, ScoreDelta, Deviation, Weight float64 }
type ConsensusResult struct { Median float64; Confidence int; Validated bool; Outcomes []Outcome }
func weightedMedian(subs []Submission) float64
func runConsensus(subs []Submission, baseReward float64) ConsensusResult
```

Rules:
- `Weight = tierWeight[tier] * consensusMult`.
- `Accepted` is true when `ProofScore == nil` (automatic node) or `*ProofScore >= 50`.
- Weighted median over accepted, weight>0 nodes.
- Confidence = % of accepted weight within 15% of the median; `Validated = Confidence >= 66`.
- Rewards: tier 3 → flat 0.5; others → `baseReward * rewardMult`.
- Excluded (`*ProofScore < 50`) → reward 0, ScoreDelta −2.
- Add a `contextAgent(outliers []Submission) string` stub: returns a note; do not implement web search (first thing to cut if time runs short).

## Done
- `backend/consensus_test.go` passes `go test .` and reproduces the doc's Kaolack example exactly: median **187**, confidence **100**, rewards A=6.5 B=5 C=5 FAO=0.5 D=0.
- Automatic node (`ProofScore == nil`) accepted at tier weight 0.7.

## Handoff
Append to `state/PROJECT_STATE.md`: function signatures, the golden test result, `contextAgent` status.
