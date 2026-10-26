package model

// Domain types for AfriOracle. Structs carry `json` tags matching the HTTP
// shapes (`node_id`, `rep_score`, `proof_score`, `value_x100`, `aiReportHash`,
// …). Submission/Outcome/ProofResult intentionally keep Go field names on the
// wire — frozen by scripts/demo.sh selectors.

// BaseReward is the consensus base reward (Couche 4).
const BaseReward = 5.0

type Submission struct {
	NodeID     string
	Tier       int
	RepScore   int
	Value      float64
	ProofScore *int // nil = automatic node (source hash) -> always accepted
	Channel    string
}

type Outcome struct {
	NodeID     string
	Status     string
	Note       string
	Reward     float64
	ScoreDelta float64
	Deviation  float64
	Weight     float64
}

type ConsensusResult struct {
	Median     float64
	Confidence int
	Validated  bool
	Outcomes   []Outcome
}

type StateSub struct {
	NodeID     string  `json:"node_id"`
	Tier       int     `json:"tier"`
	RepScore   int     `json:"rep_score"`
	Value      float64 `json:"value"`
	ProofScore *int    `json:"proof_score"`
	Channel    string  `json:"channel"`
	Status     string  `json:"status"`
	Reward     float64 `json:"reward"`
	ScoreDelta float64 `json:"score_delta"`
}

type Publication struct {
	Median       float64 `json:"median"`
	Confidence   int     `json:"confidence"`
	Validated    bool    `json:"validated"`
	Tx           string  `json:"tx"`
	Chain        string  `json:"chain"`
	ValueX100    int64   `json:"value_x100"`
	AIReportHash string  `json:"aiReportHash"`
	Timestamp    int64   `json:"timestamp"`
}

type RewardEntry struct {
	NodeID     string  `json:"node_id"`
	Reward     float64 `json:"reward"`
	ScoreDelta float64 `json:"score_delta"`
}
