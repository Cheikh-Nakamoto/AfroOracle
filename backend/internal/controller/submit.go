package controller

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

// SubmitReq is the controller's request DTO (not model).
type SubmitReq struct {
	NodeID     string   `json:"node_id"`
	Tier       int      `json:"tier"`
	RepScore   int      `json:"rep_score"`
	Value      float64  `json:"value"`
	PhotoPrice *float64 `json:"photo_price"`
	Channel    string   `json:"channel"`
	Transcript string   `json:"transcript"`
	PhotoB64   string   `json:"photo_b64"`
}

func (c *Controllers) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	principal, credential, ok := c.authenticate(r)
	if !ok {
		writeAuthError(w, http.StatusUnauthorized, "valid bearer credential required")
		return
	}
	var req SubmitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.NodeID != principal {
		writeAuthError(w, http.StatusForbidden, "credential does not match node_id")
		return
	}
	if (req.Tier != 0 && req.Tier != credential.Tier) || (req.RepScore != 0 && req.RepScore != credential.RepScore) {
		writeAuthError(w, http.StatusForbidden, "tier and rep_score are assigned by the server")
		return
	}
	req.Tier = credential.Tier
	req.RepScore = credential.RepScore
	status, resp := c.processSubmit(req)
	view.WriteJSON(w, status, resp)
}

func (c *Controllers) processSubmit(req SubmitReq) (int, any) {
	transcript := req.Transcript
	if strings.TrimSpace(transcript) == "" && req.Value == 0 {
		return http.StatusBadRequest, map[string]string{"error": "transcript or a typed value is required"}
	}
	var typed *float64
	if req.Value != 0 {
		typed = &req.Value
	}
	ext := model.Extract(transcript, typed)
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
		photoBytes, _ := base64.StdEncoding.DecodeString(req.PhotoB64)
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
