package model

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"afrioracle/internal/meta"
)

// AI is injected at startup so a missing Meta key selects a predictable demo
// adapter, while a configured key uses real model calls with no silent fallback.
type AI interface {
	Transcribe(context.Context, []byte, string) (string, error)
	Extract(context.Context, string, *float64) (ExtractResult, error)
	InspectPhoto(context.Context, []byte, float64) (*float64, []string, error)
	Name() string
}

type MockAI struct{}

func (MockAI) Name() string { return "mock" }
func (MockAI) Transcribe(_ context.Context, audio []byte, _ string) (string, error) {
	if len(audio) == 0 {
		return transcribe(""), nil
	}
	return "", fmt.Errorf("mock transcription accepts a transcript hint, not audio bytes")
}
func (MockAI) Extract(_ context.Context, transcript string, typed *float64) (ExtractResult, error) {
	return extract(transcript, typed), nil
}
func (MockAI) InspectPhoto(_ context.Context, _ []byte, _ float64) (*float64, []string, error) {
	return nil, nil, nil
}

type metaModels interface {
	Chat(context.Context, meta.ChatRequest) (meta.ChatResponse, error)
	ChatWithImage(context.Context, string, []byte) (meta.ChatResponse, error)
	Transcribe(context.Context, []byte, string) (string, error)
	Segment(context.Context, string, []byte) ([]meta.Segment, error)
}

type MetaAI struct{ client metaModels }

func NewMetaAI(client metaModels) *MetaAI { return &MetaAI{client: client} }
func (MetaAI) Name() string               { return "Muse Spark 1.3 + SAM 3.1" }
func (a *MetaAI) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("audio is required for real transcription")
	}
	return a.client.Transcribe(ctx, audio, format)
}
func (a *MetaAI) Extract(ctx context.Context, transcript string, typed *float64) (ExtractResult, error) {
	if typed != nil {
		return extract(transcript, typed), nil
	}
	resp, err := a.client.Chat(ctx, meta.ChatRequest{JSONObject: true, MaxTokens: 256, Messages: []meta.Message{
		{Role: "system", Content: "Extract a field market observation. Return JSON only: produit, prix (number or null), unite, marche. Never invent a price. Use French names when possible."},
		{Role: "user", Content: transcript},
	}})
	if err != nil {
		return ExtractResult{}, err
	}
	var got struct {
		Produit string   `json:"produit"`
		Prix    *float64 `json:"prix"`
		Unite   string   `json:"unite"`
		Marche  string   `json:"marche"`
	}
	if err := json.Unmarshal([]byte(resp.Text), &got); err != nil {
		return ExtractResult{}, fmt.Errorf("parse extraction: %w", err)
	}
	if got.Prix != nil && (*got.Prix <= 0 || math.IsNaN(*got.Prix) || math.IsInf(*got.Prix, 0)) {
		got.Prix = nil
	}
	if got.Produit == "" {
		got.Produit = "mil"
	}
	if got.Unite == "" {
		got.Unite = "kilo"
	}
	if got.Marche == "" {
		got.Marche = "Kaolack"
	}
	out := ExtractResult{Produit: got.Produit, Prix: got.Prix, Unite: got.Unite, Marche: got.Marche}
	if out.Prix == nil {
		out.NeedsConfirmation = true
		out.Question = "Quel est le prix observé au kilo ?"
	}
	return out, nil
}
func (a *MetaAI) InspectPhoto(ctx context.Context, photo []byte, submitted float64) (*float64, []string, error) {
	if len(photo) == 0 {
		return nil, nil, nil
	}
	resp, err := a.client.ChatWithImage(ctx, "Read the price written on the price slate in this market photo. Return JSON only: {\"price\": number or null}. Do not infer missing digits.", photo)
	if err != nil {
		return nil, nil, err
	}
	var got struct {
		Price *float64 `json:"price"`
	}
	if err := json.Unmarshal([]byte(resp.Text), &got); err != nil {
		return nil, nil, fmt.Errorf("parse slate reading: %w", err)
	}
	if got.Price != nil && (*got.Price <= 0 || math.IsNaN(*got.Price) || math.IsInf(*got.Price, 0)) {
		got.Price = nil
	}
	segs, err := a.client.Segment(ctx, "price slate and millet sack", photo)
	if err != nil {
		return nil, nil, err
	}
	objects := make([]string, 0, len(segs))
	for _, seg := range segs {
		label := strings.TrimSpace(seg.Label)
		if label != "" {
			objects = append(objects, label)
		}
	}
	return got.Price, objects, nil
}

// ScoreRealProof applies the protocol's fixed score bands to model evidence.
// Photo reuse is checked by the store before this score can be used.
func ScoreRealProof(photoPrice *float64, submitted float64, objects []string) int {
	if photoPrice == nil {
		return 40
	}
	gap := math.Abs(*photoPrice-submitted) / submitted * 100
	if submitted <= 0 {
		return 20
	}
	if gap <= 2 {
		return 94
	}
	if gap <= 10 {
		return 70
	}
	return 20
}
