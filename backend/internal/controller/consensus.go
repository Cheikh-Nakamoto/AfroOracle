package controller

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

type consensusRequest struct {
	BatchID string `json:"batch_id"`
}

func (c *Controllers) HandleConsensus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := c.authenticate(r); !ok {
		writeAuthError(w, http.StatusUnauthorized, "valid bearer credential required")
		return
	}

	var req consensusRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
	}
	batchID := strings.TrimSpace(req.BatchID)
	if len(batchID) > 128 {
		http.Error(w, "batch_id too long", http.StatusBadRequest)
		return
	}
	if batchID == "" {
		var err error
		batchID, err = newConsensusBatchID()
		if err != nil {
			view.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create consensus batch"})
			return
		}
	}

	batch, status := c.Store.TakeConsensusBatch(batchID)
	switch status {
	case model.BatchInsufficient:
		view.WriteJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "collecte rule: minimum 3 independent submissions required",
		})
		return
	case model.BatchInProgress:
		view.WriteJSON(w, http.StatusConflict, map[string]string{"error": "consensus batch is already processing"})
		return
	case model.BatchReplay:
		writeConsensusRun(w, *batch.Run, true)
		return
	case model.BatchStarted:
	default:
		view.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid consensus batch state"})
		return
	}

	subs := make([]model.Submission, 0, len(batch.Submissions))
	for _, s := range batch.Submissions {
		subs = append(subs, model.Submission{
			NodeID: s.NodeID, Tier: s.Tier, RepScore: s.RepScore, Value: s.Value,
			ProofScore: s.ProofScore, Channel: s.Channel, ID: s.ID,
			DuplicateSource: s.DuplicateSource,
		})
	}
	result := model.RunConsensus(subs, model.BaseReward)
	report := map[string]any{"median": result.Median, "confidence": result.Confidence, "outcomes": len(result.Outcomes)}
	tx, chain := "", "not-published"
	var publication *model.Publication
	if result.Validated {
		tx, chain = model.ChainPublish(result, report)
		publication = &model.Publication{
			Median: result.Median, Confidence: result.Confidence, Validated: true,
			Tx: tx, Chain: chain, ValueX100: int64(result.Median*100 + 0.5),
			AIReportHash: model.ReportHash(report), Timestamp: time.Now().Unix(),
		}
	} else {
		for i := range result.Outcomes {
			if result.Outcomes[i].Status != "excluded" {
				result.Outcomes[i].Status = "unvalidated"
				result.Outcomes[i].Note = "confiance insuffisante; aucune publication ni récompense"
				result.Outcomes[i].Reward = 0
				result.Outcomes[i].ScoreDelta = 0
			}
		}
	}

	run := model.ConsensusRun{BatchID: batchID, Result: result, Tx: tx, Chain: chain, Publication: publication}
	if !c.Store.CompleteConsensusBatch(batchID, run) {
		view.WriteJSON(w, http.StatusConflict, map[string]string{"error": "consensus batch was already completed"})
		return
	}
	writeConsensusRun(w, run, false)
}

func newConsensusBatchID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func writeConsensusRun(w http.ResponseWriter, run model.ConsensusRun, replay bool) {
	valueX100 := int64(run.Result.Median*100 + 0.5)
	aiReportHash := ""
	if run.Publication != nil {
		valueX100 = run.Publication.ValueX100
		aiReportHash = run.Publication.AIReportHash
	}
	view.WriteJSON(w, http.StatusOK, map[string]any{
		"batch_id":     run.BatchID,
		"replayed":     replay,
		"median":       run.Result.Median,
		"confidence":   run.Result.Confidence,
		"validated":    run.Result.Validated,
		"outcomes":     run.Result.Outcomes,
		"tx":           run.Tx,
		"chain":        run.Chain,
		"value_x100":   valueX100,
		"aiReportHash": aiReportHash,
	})
}
