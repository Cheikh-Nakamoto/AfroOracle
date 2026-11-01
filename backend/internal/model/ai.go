package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"afrioracle/internal/meta"
)

// mock is true when META_API_KEY is unset: the demo runs fully offline with
// deterministic adapters. Set META_API_KEY to hit the real Meta models
// (Muse Voice Transcribe, Muse Spark 1.3, SAM 3.1).
var mock = os.Getenv("META_API_KEY") == ""

// metaAPI is the slice of the Meta client the model uses; *meta.Client
// satisfies it.
type metaAPI interface {
	Chat(ctx context.Context, req meta.ChatRequest) (meta.ChatResponse, error)
	Transcribe(ctx context.Context, audio []byte, format string) (string, error)
	Segment(ctx context.Context, prompt string, image []byte) ([]meta.Segment, error)
}

var client metaAPI = newEnvClient()

func newEnvClient() metaAPI {
	if k := os.Getenv("META_API_KEY"); k != "" {
		return meta.New(k)
	}
	return nil
}

// UseMeta switches the model to real mode with the given client and returns
// a restore func. Tests use it with a client pointed at an httptest fake.
func UseMeta(c *meta.Client) (restore func()) {
	prevMock, prevClient := mock, client
	mock, client = false, c
	return func() { mock, client = prevMock, prevClient }
}

const aiTimeout = 25 * time.Second

func aiCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), aiTimeout)
}

type ExtractResult struct {
	Produit           string
	Prix              *float64
	Unite             string
	Marche            string
	NeedsConfirmation bool
	Question          string
}

type ProofResult struct {
	Score          int
	SlatePrice     *float64
	DuplicatePhoto bool
	ObjectsFound   []string
	Model          string
}

// transcribe converts an audio hint to text. Mock path: returns the hint, or
// the explicit demo fixture sentence when the hint is empty. Real path (no
// hint): empty string, never an invented price. Real: Muse Voice Transcribe.
func transcribe(audioHint string) string {
	if strings.TrimSpace(audioHint) == "" {
		if !mock {
			return ""
		}
		return "Le prix du mil à Kaolack est de 187 francs CFA le kilo."
	}
	return audioHint
}

// transcribeAudio turns a voice note into text. Mock: the demo fixture
// sentence (offline, deterministic). Real: Muse Voice Transcribe.
func transcribeAudio(audio []byte, format string) (string, error) {
	if mock {
		return transcribe(""), nil
	}
	ctx, cancel := aiCtx()
	defer cancel()
	return client.Transcribe(ctx, audio, format)
}

// extract turns a transcript (plus an optional typed value) into structured data.
// Real path: Muse Spark 1.3 with JSON output. Mock path: parse a number.
func extract(transcript string, typedValue *float64) (ExtractResult, error) {
	res := ExtractResult{
		Produit: "mil",
		Unite:   "kilo",
		Marche:  "Kaolack",
	}

	if typedValue != nil {
		res.Prix = typedValue
		return res, nil
	}
	if !mock {
		return extractReal(transcript)
	}

	re := regexp.MustCompile(`\d+(?:[.,]\d+)?`)
	m := re.FindString(transcript)
	if m != "" {
		v, err := strconv.ParseFloat(strings.Replace(m, ",", ".", 1), 64)
		if err == nil {
			res.Prix = &v
			return res, nil
		}
	}

	res.NeedsConfirmation = true
	res.Question = "le kilo ou le sac ?"
	return res, nil
}

// extractReal asks Muse Spark for a JSON object describing the quote.
func extractReal(transcript string) (ExtractResult, error) {
	ctx, cancel := aiCtx()
	defer cancel()
	resp, err := client.Chat(ctx, meta.ChatRequest{
		JSONObject: true,
		MaxTokens:  300,
		Messages: []meta.Message{
			{Role: "system", Content: `Extract a market price quote. Reply with one JSON object: {"produit":string,"prix":number|null,"unite":string,"marche":string,"needs_confirmation":bool,"question":string}. Use prix null and needs_confirmation true if the price or unit (kilo or sac) is ambiguous.`},
			{Role: "user", Content: transcript},
		},
	})
	if err != nil {
		return ExtractResult{}, err
	}
	var out struct {
		Produit string   `json:"produit"`
		Prix    *float64 `json:"prix"`
		Unite   string   `json:"unite"`
		Marche  string   `json:"marche"`
		Needs   bool     `json:"needs_confirmation"`
		Q       string   `json:"question"`
	}
	if err := json.Unmarshal([]byte(resp.Text), &out); err != nil {
		return ExtractResult{}, fmt.Errorf("extract: bad model JSON: %w", err)
	}
	res := ExtractResult{Produit: out.Produit, Prix: out.Prix, Unite: out.Unite, Marche: out.Marche, NeedsConfirmation: out.Needs, Question: out.Q}
	if res.Prix != nil && !(*res.Prix > 0) {
		res.Prix = nil
	}
	if res.Prix == nil {
		res.NeedsConfirmation = true
		if res.Question == "" {
			res.Question = "le kilo ou le sac ?"
		}
	}
	return res, nil
}

// readSlate asks Muse Spark (vision) for the price written on the slate and
// SAM 3.1 for the objects in the photo. Any API failure yields no reading and
// no objects, which scores as unverified (never accepted).
func readSlate(photo []byte) (price *float64, objects []string) {
	if len(photo) == 0 {
		return nil, nil
	}
	ctx, cancel := aiCtx()
	defer cancel()
	if resp, err := client.Chat(ctx, meta.ChatRequest{
		JSONObject: true,
		MaxTokens:  100,
		Messages: []meta.Message{{
			Role:    "user",
			Content: `Read the price written on the market price slate in this photo. Reply with one JSON object {"price":number|null}; null if unreadable.`,
			Images:  [][]byte{photo},
		}},
	}); err == nil {
		var out struct {
			Price *float64 `json:"price"`
		}
		if json.Unmarshal([]byte(resp.Text), &out) == nil && out.Price != nil && *out.Price > 0 {
			price = out.Price
		}
	}
	if segs, err := client.Segment(ctx, "price slate, sack of millet", photo); err == nil {
		for _, sg := range segs {
			objects = append(objects, sg.Label)
		}
	}
	return price, objects
}

// proofScore scores a photo against the submitted value. Mock rules:
// reused photo (sha256 already seen) -> 12; unreadable slate (photoPrice nil) -> 40;
// otherwise gap <=2% -> 94, <=10% -> 70, else 20.
// Real path: Muse Spark reads the slate, SAM 3.1 detects objects, the server
// fingerprints the photo.
func proofScore(photoPrice *float64, submitted float64, photoBytes []byte, seen map[string]bool) ProofResult {
	fp := sha256Hex(photoBytes)
	if seen[fp] {
		return ProofResult{
			Score:          12,
			SlatePrice:     photoPrice,
			DuplicatePhoto: true,
			ObjectsFound:   nil,
			Model:          modelName(),
		}
	}
	seen[fp] = true

	objects := []string{"ardoise de prix", "sac de mil"}
	if !mock {
		// Real mode: the slate reading comes from the models, never from the
		// client-supplied photo_price.
		photoPrice, objects = readSlate(photoBytes)
	}

	if photoPrice == nil {
		return ProofResult{
			Score:          40,
			SlatePrice:     nil,
			DuplicatePhoto: false,
			ObjectsFound:   nil,
			Model:          modelName(),
		}
	}

	gap := 100.0
	if submitted > 0 {
		gap = math.Abs(*photoPrice-submitted) / submitted * 100
	}

	score := 20
	switch {
	case gap <= 2:
		score = 94
	case gap <= 10:
		score = 70
	}

	return ProofResult{
		Score:          score,
		SlatePrice:     photoPrice,
		DuplicatePhoto: false,
		ObjectsFound:   objects,
		Model:          modelName(),
	}
}

// reportHash returns "0x" + sha256 of the canonical JSON of payload. Map keys are
// sorted by encoding/json, so the hash is stable for a given payload.
func reportHash(payload any) string {
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return "0x" + hex.EncodeToString(sum[:])
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func modelName() string {
	if mock {
		return "mock"
	}
	return "Muse Spark 1.3 + SAM 3.1"
}

// Controller-facing wrappers; implementations stay unexported.
func Transcribe(audioHint string) string { return transcribe(audioHint) }

func Extract(transcript string, typedValue *float64) (ExtractResult, error) {
	return extract(transcript, typedValue)
}

func TranscribeAudio(audio []byte, format string) (string, error) {
	return transcribeAudio(audio, format)
}

func ReportHash(payload any) string { return reportHash(payload) }
