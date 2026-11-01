package controller

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"strings"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

// SubmitReq is the controller's request DTO (not model). Tier and RepScore
// are accepted on the wire for compatibility but ignored: the registry
// supplies them.
type SubmitReq struct {
	NodeID     string   `json:"node_id"`
	Tier       int      `json:"tier"`
	RepScore   int      `json:"rep_score"`
	Value      *float64 `json:"value"`
	PhotoPrice *float64 `json:"photo_price"`
	Channel    string   `json:"channel"`
	Transcript string   `json:"transcript"`
	PhotoB64   string   `json:"photo_b64"`
}

func (c *Controllers) HandleSubmit(w http.ResponseWriter, r *http.Request, node model.Node) {
	var req SubmitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		view.WriteError(w, http.StatusBadRequest, "bad json")
		return
	}
	if msg := validateSubmit(req); msg != "" {
		view.WriteError(w, http.StatusBadRequest, msg)
		return
	}
	if strings.TrimSpace(req.NodeID) != node.ID {
		view.WriteError(w, http.StatusForbidden, "node_id does not match credentials")
		return
	}
	// Tier and reputation come from the registry; payload values are ignored.
	req.Tier, req.RepScore = node.Tier, node.Rep
	status, resp := c.processSubmit(req)
	view.WriteJSON(w, status, resp)
}

const (
	maxNodeID = 64
	maxPrice  = 1e9
)

func validPrice(p float64) bool {
	return p > 0 && p <= maxPrice && !math.IsNaN(p)
}

// validateSubmit returns a human-readable reason, or "" when req is well formed.
// Absent value (nil) is distinct from an explicit 0, which is invalid.
func validateSubmit(req SubmitReq) string {
	id := strings.TrimSpace(req.NodeID)
	switch {
	case id == "" || len(id) > maxNodeID:
		return "node_id must be 1-64 characters"
	case req.Value != nil && !validPrice(*req.Value):
		return "value must be a positive number"
	case req.PhotoPrice != nil && !validPrice(*req.PhotoPrice):
		return "photo_price must be a positive number"
	case req.Value == nil && strings.TrimSpace(req.Transcript) == "":
		return "value or transcript required"
	}
	if req.PhotoB64 != "" {
		if _, err := base64.StdEncoding.DecodeString(req.PhotoB64); err != nil {
			return "photo_b64 is not valid base64"
		}
	}
	return ""
}

func (c *Controllers) processSubmit(req SubmitReq) (int, any) {
	ext := model.Extract(req.Transcript, req.Value)
	if ext.NeedsConfirmation {
		return http.StatusOK, map[string]string{"status": "clarify", "question": ext.Question}
	}

	value := 0.0
	if ext.Prix != nil {
		value = *ext.Prix
	}

	var proofScorePtr *int
	proof := model.ProofResult{}
	if req.Tier == 3 {
		proofScorePtr = nil // automatic node (source hash) -> always accepted
	} else {
		photoBytes, _ := base64.StdEncoding.DecodeString(req.PhotoB64) // checked by validateSubmit
		pr := c.Store.ScorePhoto(req.PhotoPrice, value, photoBytes)
		proofScorePtr = &pr.Score
		proof = pr
	}

	sub := model.Submission{
		NodeID:     req.NodeID,
		Tier:       req.Tier,
		RepScore:   req.RepScore,
		Value:      value,
		ProofScore: proofScorePtr,
		Channel:    req.Channel,
	}

	c.Store.AddSubmission(model.StateSub{
		NodeID:     sub.NodeID,
		Tier:       sub.Tier,
		RepScore:   sub.RepScore,
		Value:      sub.Value,
		ProofScore: sub.ProofScore,
		Channel:    sub.Channel,
		Status:     "pending",
	})

	return http.StatusCreated, map[string]any{
		"status":     "accepted",
		"submission": sub,
		"proof":      proof,
	}
}
