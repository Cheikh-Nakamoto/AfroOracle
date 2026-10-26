package controller

import (
	"net/http"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

func (c *Controllers) HandleState(w http.ResponseWriter, r *http.Request) {
	submissions := c.Store.Submissions()
	publications := c.Store.Publications()
	outcomes := c.Store.LastOutcomes()
	fps := c.Store.Fingerprints()
	rewards := make([]model.RewardEntry, 0, len(outcomes))
	for _, o := range outcomes {
		rewards = append(rewards, model.RewardEntry{NodeID: o.NodeID, Reward: o.Reward, ScoreDelta: o.ScoreDelta})
	}
	view.WriteJSON(w, http.StatusOK, map[string]any{
		"submissions":  submissions,
		"publications": publications,
		"rewards":      rewards,
		"fingerprints": fps,
	})
}
