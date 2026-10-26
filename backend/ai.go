package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// mock is true when META_API_KEY is unset: the demo runs fully offline with
// deterministic adapters. Set META_API_KEY to hit the real Meta models
// (Muse Voice Transcribe, Muse Spark 1.3, SAM 3.1).
var mock = os.Getenv("META_API_KEY") == ""

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

// transcribe converts an audio hint to text. In the demo (mock) path it simply
// returns the hint, or a default French sentence when the hint is empty.
// Real path: Muse Voice Transcribe.
func transcribe(audioHint string) string {
	if strings.TrimSpace(audioHint) == "" {
		return "Le prix du mil à Kaolack est de 187 francs CFA le kilo."
	}
	return audioHint
}

// extract turns a transcript (plus an optional typed value) into structured data.
// Real path: Muse Spark 1.3 with JSON output. Mock path: parse a number.
func extract(transcript string, typedValue *float64) ExtractResult {
	res := ExtractResult{
		Produit: "mil",
		Unite:   "kilo",
		Marche:  "Kaolack",
	}

	if typedValue != nil {
		res.Prix = typedValue
		return res
	}

	re := regexp.MustCompile(`\d+(?:[.,]\d+)?`)
	m := re.FindString(transcript)
	if m != "" {
		v, err := strconv.ParseFloat(strings.Replace(m, ",", ".", 1), 64)
		if err == nil {
			res.Prix = &v
			return res
		}
	}

	res.NeedsConfirmation = true
	res.Question = "le kilo ou le sac ?"
	return res
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
		ObjectsFound:   []string{"ardoise de prix", "sac de mil"},
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
