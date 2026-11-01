package controller

import (
	"log"
	"net/http"
	"time"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

func (c *Controllers) HandleConsensus(w http.ResponseWriter, r *http.Request, _ model.Node) {
	c.consensusMu.Lock()
	defer c.consensusMu.Unlock()

	stored := c.Store.PendingBatch()
	// One node contributes one value: the latest submission in the batch.
	// Earlier rows are still consumed with the batch (CommitBatch gets all).
	latest := map[string]int{}
	for i, s := range stored {
		latest[s.NodeID] = i
	}
	subs := make([]model.Submission, 0, len(latest))
	for i, s := range stored {
		if latest[s.NodeID] != i {
			continue
		}
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
	if !result.Validated {
		// Never published, stored or quoted; the batch stays pending.
		view.WriteJSON(w, http.StatusOK, map[string]any{
			"median":     result.Median,
			"confidence": result.Confidence,
			"validated":  false,
			"outcomes":   result.Outcomes,
			"tx":         "",
			"chain":      "",
		})
		return
	}
	report := map[string]any{"median": result.Median, "confidence": result.Confidence, "outcomes": len(result.Outcomes)}
	tx, chain, err := model.ChainPublish(result, report)
	if err != nil {
		log.Printf("chain publish failed: %v", err)
		view.WriteError(w, http.StatusBadGateway, "chain publish failed")
		return
	}

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

	if err := c.Store.CommitBatch(pub, result.Outcomes, stored); err != nil {
		log.Printf("store: commit batch: %v", err)
		view.WriteError(w, http.StatusInternalServerError, "storage unavailable")
		return
	}

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
