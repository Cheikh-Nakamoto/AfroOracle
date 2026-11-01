package controller

import (
	"net/http"
	"time"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

func (c *Controllers) HandleConsensus(w http.ResponseWriter, r *http.Request) {
	c.consensusMu.Lock()
	defer c.consensusMu.Unlock()

	stored := c.Store.PendingBatch()
	subs := make([]model.Submission, 0, len(stored))
	for _, s := range stored {
		subs = append(subs, model.Submission{
			NodeID:     s.NodeID,
			Tier:       s.Tier,
			RepScore:   s.RepScore,
			Value:      s.Value,
			ProofScore: s.ProofScore,
			Channel:    s.Channel,
		})
	}

	// Quorum = three distinct nodes, not three entries.
	sources := map[string]bool{}
	for _, s := range subs {
		if s.Accepted() && s.Weight() > 0 {
			sources[s.NodeID] = true
		}
	}
	if len(sources) < 3 {
		view.WriteJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "collecte rule: minimum 3 independent nodes required",
		})
		return
	}

	result := model.RunConsensus(subs, model.BaseReward)
	report := map[string]any{"median": result.Median, "confidence": result.Confidence, "outcomes": len(result.Outcomes)}
	tx, chain := model.ChainPublish(result, report)

	pub := model.Publication{
		Median:       result.Median,
		Confidence:   result.Confidence,
		Validated:    result.Validated,
		Tx:           tx,
		Chain:        chain,
		ValueX100:    int64(result.Median*100 + 0.5),
		AIReportHash: model.ReportHash(report),
		Timestamp:    time.Now().Unix(),
	}

	c.Store.CommitBatch(pub, result.Outcomes, stored)

	view.WriteJSON(w, http.StatusOK, map[string]any{
		"median":       result.Median,
		"confidence":   result.Confidence,
		"validated":    result.Validated,
		"outcomes":     result.Outcomes,
		"tx":           tx,
		"chain":        chain,
		"value_x100":   pub.ValueX100,
		"aiReportHash": pub.AIReportHash,
	})
}
