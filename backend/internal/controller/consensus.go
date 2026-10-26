package controller

import (
	"net/http"
	"time"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

func (c *Controllers) HandleConsensus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	stored := c.Store.Submissions()
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

	accepted := 0
	for _, s := range subs {
		if s.Accepted() && s.Weight() > 0 {
			accepted++
		}
	}
	if accepted < 3 {
		view.WriteJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "collecte rule: minimum 3 independent submissions required",
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

	c.Store.AddPublication(pub, result.Outcomes)

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
