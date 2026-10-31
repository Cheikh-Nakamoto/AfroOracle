package model

import (
	"context"
	"errors"
	"testing"

	"afrioracle/internal/meta"
)

type fakeMeta struct {
	chat       string
	vision     string
	transcript string
	segments   []meta.Segment
	err        error
}

func (f fakeMeta) Chat(context.Context, meta.ChatRequest) (meta.ChatResponse, error) {
	return meta.ChatResponse{Text: f.chat}, f.err
}
func (f fakeMeta) ChatWithImage(context.Context, string, []byte) (meta.ChatResponse, error) {
	return meta.ChatResponse{Text: f.vision}, f.err
}
func (f fakeMeta) Transcribe(context.Context, []byte, string) (string, error) {
	return f.transcript, f.err
}
func (f fakeMeta) Segment(context.Context, string, []byte) ([]meta.Segment, error) {
	return f.segments, f.err
}

func TestMetaAIExtractsAndDoesNotInventMissingPrice(t *testing.T) {
	a := NewMetaAI(fakeMeta{chat: `{"produit":"mil","prix":187,"unite":"kg","marche":"Kaolack"}`})
	got, err := a.Extract(context.Background(), "mil à Kaolack 187", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prix == nil || *got.Prix != 187 || got.NeedsConfirmation {
		t.Fatalf("unexpected extraction: %+v", got)
	}

	a = NewMetaAI(fakeMeta{chat: `{"produit":"mil","prix":null}`})
	got, err = a.Extract(context.Background(), "prix inaudible", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prix != nil || !got.NeedsConfirmation {
		t.Fatalf("expected clarification for absent price: %+v", got)
	}
}

func TestMetaAIInspectsPhotoAndAppliesDemoProofBands(t *testing.T) {
	a := NewMetaAI(fakeMeta{
		vision:   `{"price":187}`,
		segments: []meta.Segment{{Label: "price slate"}, {Label: "millet sack"}},
	})
	price, objects, err := a.InspectPhoto(context.Background(), []byte("photo"), 187)
	if err != nil {
		t.Fatal(err)
	}
	if price == nil || *price != 187 || len(objects) != 2 {
		t.Fatalf("price=%v objects=%v", price, objects)
	}
	if score := ScoreRealProof(price, 187, objects); score != 94 {
		t.Fatalf("score=%d, want 94", score)
	}
	if score := ScoreRealProof(price, 240, objects); score != 20 {
		t.Fatalf("fraud price score=%d, want 20", score)
	}
}

func TestMetaAIFailsClosedOnProviderError(t *testing.T) {
	a := NewMetaAI(fakeMeta{err: errors.New("unavailable")})
	if _, err := a.Extract(context.Background(), "mil 187", nil); err == nil {
		t.Fatal("expected provider error")
	}
}

func TestMissingPhotoIsNotFingerprintDuplicate(t *testing.T) {
	store := NewInMemoryStore()
	first := store.ScorePhoto(nil, 187, nil)
	second := store.ScorePhoto(nil, 187, nil)
	if first.Score != 40 || second.Score != 40 || first.DuplicatePhoto || second.DuplicatePhoto {
		t.Fatalf("missing photo scores should remain unreadable, got first=%+v second=%+v", first, second)
	}
}
