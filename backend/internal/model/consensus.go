package model

import (
	"math"
	"sort"
)

// tierWeight is the consensus weight of each tier (Couche 2).
var tierWeight = map[int]float64{1: 3.0, 2: 1.0, 3: 0.7}

// reputation maps a rep score (0-100) to (level, consensus multiplier, reward
// multiplier) using the single grid from the doc (Couche 3).
func reputation(score int) (level string, consensusMult, rewardMult float64) {
	switch {
	case score >= 90:
		return "Expert", 2.0, 1.8
	case score >= 75:
		return "Confirmé", 1.5, 1.3
	case score >= 60:
		return "Actif", 1.0, 1.0
	case score >= 40:
		return "Débutant", 0.5, 0.5
	default:
		return "Non qualifié", 0.0, 0.0
	}
}

// Weight = tierWeight[tier] * consensusMult. Automatic nodes (ProofScore nil)
// have no reputation, so they use a neutral multiplier of 1.0.
func (s Submission) Weight() float64 {
	mult := 1.0
	if s.ProofScore != nil {
		_, mult, _ = reputation(s.RepScore)
	}
	return tierWeight[s.Tier] * mult
}

// Accepted is true for automatic nodes (ProofScore nil) or proof score >= 50.
func (s Submission) Accepted() bool {
	if s.ProofScore == nil {
		return true
	}
	return *s.ProofScore >= 50
}

// weightedMedian returns the weighted median over accepted, weight>0 nodes.
// The caller guarantees at least one accepted submission.
func weightedMedian(subs []Submission) float64 {
	accepted := make([]Submission, 0, len(subs))
	total := 0.0
	for _, s := range subs {
		if s.Accepted() && s.Weight() > 0 {
			accepted = append(accepted, s)
			total += s.Weight()
		}
	}
	sort.Slice(accepted, func(i, j int) bool { return accepted[i].Value < accepted[j].Value })

	half := total / 2
	cum := 0.0
	for _, s := range accepted {
		cum += s.Weight()
		if cum >= half {
			return s.Value
		}
	}
	if len(accepted) > 0 {
		return accepted[len(accepted)-1].Value
	}
	return 0
}

func runConsensus(subs []Submission, baseReward float64) ConsensusResult {
	median := weightedMedian(subs)

	total := 0.0
	within := 0.0
	var outliers []Submission
	for _, s := range subs {
		if !s.Accepted() || s.Weight() <= 0 {
			continue
		}
		total += s.Weight()
		if deviation(s.Value, median) <= 15 {
			within += s.Weight()
		} else {
			outliers = append(outliers, s)
		}
	}

	confidence := 0
	if total > 0 {
		confidence = int(math.Round(within / total * 100))
	}
	validated := confidence >= 66

	note := contextAgent(outliers)

	outcomes := make([]Outcome, 0, len(subs))
	for _, s := range subs {
		o := Outcome{
			NodeID:    s.NodeID,
			Weight:    s.Weight(),
			Deviation: deviation(s.Value, median),
		}
		if !s.Accepted() {
			o.Status = "excluded"
			o.Note = "preuve rejetée (score < 50)"
			o.Reward = 0
			o.ScoreDelta = -2
		} else {
			o.Status = "accepted"
			if s.Tier == 3 {
				o.Reward = 0.5
			} else {
				_, _, rewardMult := reputation(s.RepScore)
				o.Reward = baseReward * rewardMult
			}
			o.ScoreDelta = 0
			if deviation(s.Value, median) > 15 {
				o.Note = note
			}
		}
		outcomes = append(outcomes, o)
	}

	return ConsensusResult{
		Median:     median,
		Confidence: confidence,
		Validated:  validated,
		Outcomes:   outcomes,
	}
}

func deviation(value, median float64) float64 {
	if median == 0 {
		return 0
	}
	return math.Abs(value-median) / median * 100
}

// contextAgent returns a note explaining outliers. Stub only — no web search
// (first thing to cut if time runs short).
func contextAgent(outliers []Submission) string {
	if len(outliers) == 0 {
		return ""
	}
	return "écart > 15 % de la médiane; revue manuelle recommandée"
}

// RunConsensus is the controller-facing entry point; the implementation stays
// unexported so the internal test can call it directly.
func RunConsensus(subs []Submission, baseReward float64) ConsensusResult {
	return runConsensus(subs, baseReward)
}
