package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

// SubmitReq is the controller's request DTO (not model).
type SubmitReq struct {
	NodeID      string   `json:"node_id"`
	Tier        int      `json:"tier"`
	RepScore    int      `json:"rep_score"`
	Value       float64  `json:"value"`
	PhotoPrice  *float64 `json:"photo_price"`
	Channel     string   `json:"channel"`
	Transcript  string   `json:"transcript"`
	PhotoB64    string   `json:"photo_b64"`
	AudioB64    string   `json:"audio_b64"`
	AudioFormat string   `json:"audio_format"`
}

func (c *Controllers) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req SubmitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	status, resp := c.processSubmit(r.Context(), req)
	view.WriteJSON(w, status, resp)
}

func (c *Controllers) processSubmit(ctx context.Context, req SubmitReq) (int, any) {
	transcript := req.Transcript
	if req.AudioB64 != "" {
		audio, err := base64.StdEncoding.DecodeString(req.AudioB64)
		if err != nil {
			return http.StatusBadRequest, map[string]string{"error": "invalid audio_b64"}
		}
		if c.AI.Name() == "mock" {
			return http.StatusUnprocessableEntity, map[string]string{"error": "audio transcription requires META_API_KEY"}
		}
		transcript, err = c.AI.Transcribe(ctx, audio, req.AudioFormat)
		if err != nil {
			return http.StatusBadGateway, map[string]string{"error": "Meta transcription failed"}
		}
	} else if strings.TrimSpace(transcript) == "" && c.AI.Name() == "mock" {
		transcript = model.Transcribe("")
	} else if strings.TrimSpace(transcript) == "" {
		return http.StatusBadRequest, map[string]string{"error": "transcript or audio_b64 is required"}
	}
	var typed *float64
	if req.Value != 0 {
		typed = &req.Value
	}
	ext, err := c.AI.Extract(ctx, transcript, typed)
	if err != nil {
		return http.StatusBadGateway, map[string]string{"error": "Meta extraction failed"}
	}
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
		photoBytes, err := base64.StdEncoding.DecodeString(req.PhotoB64)
		if err != nil {
			return http.StatusBadRequest, map[string]string{"error": "invalid photo_b64"}
		}
		photoPrice := req.PhotoPrice
		objects := []string(nil)
		if c.AI.Name() != "mock" {
			photoPrice = nil // never trust a client-supplied slate reading in real mode
			if len(photoBytes) > 0 {
				photoPrice, objects, err = c.AI.InspectPhoto(ctx, photoBytes, value)
				if err != nil {
					return http.StatusBadGateway, map[string]string{"error": "Meta photo verification failed"}
				}
			}
		}
		pr := c.Store.ScorePhoto(photoPrice, value, photoBytes)
		if c.AI.Name() != "mock" {
			pr.Score = model.ScoreRealProof(photoPrice, value, objects)
			pr.ObjectsFound = objects
			pr.Model = c.AI.Name()
			if pr.DuplicatePhoto {
				pr.Score = 12
			}
		}
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
