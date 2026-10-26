# Agent 02 — Backend: AI adapters (Go)

## Goal
Implement the Meta AI adapters in `backend/ai.go`, package `main`, with a deterministic mock fallback so the whole demo runs with zero API keys.

## Depends on
`spec.md` §AI interface.

## Read
- `spec.md` §AI interface (the Go signatures and mock rules are the contract)
- Meta model docs: Muse Spark 1.3, Muse Voice Transcribe, SAM 3.1

## Build
- `backend/ai.go` only. Package `main`, module `afrioracle`. Nothing else.

Implement exactly:
```go
var mock = os.Getenv("META_API_KEY") == ""

type ExtractResult struct { Produit string; Prix *float64; Unite string; Marche string; NeedsConfirmation bool; Question string }
type ProofResult struct { Score int; SlatePrice *float64; DuplicatePhoto bool; ObjectsFound []string; Model string }

func transcribe(audioHint string) string
func extract(transcript string, typedValue *float64) ExtractResult
func proofScore(photoPrice *float64, submitted float64, photoBytes []byte, seen map[string]bool) ProofResult
func reportHash(payload any) string
```

Mock rules (must hold when `mock` is true):
- `transcribe` returns the hint, or a default French sentence.
- `extract` parses a number from the transcript; `NeedsConfirmation` when no price; question "le kilo ou le sac ?".
- `proofScore`: duplicate photo (sha256 hex already in `seen`) → score 12; `photoPrice == nil` → score 40; else gap ≤2% → 94, ≤10% → 70, else 20. Add the sha256 hex of `photoBytes` to `seen`.
- `reportHash` returns `"0x" + sha256(json)` (stable key order).

## Done
- `mock == true` path runs end-to-end with no keys (agent 04 exercises it).
- Signatures and field names match `spec.md` §AI exactly.
- `proofScore` marks the doc's fraud case (reused photo) at score 12.

## Handoff
Append to `state/PROJECT_STATE.md`: how real Meta API is enabled (which env var, which model each function calls), and the mock-vs-real switch.
