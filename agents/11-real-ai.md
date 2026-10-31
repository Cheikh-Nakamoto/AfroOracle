# Agent 11 — wire real Meta models

Read `AGENTS.md`, `PIPELINE.md`, `spec.md`, this file, and only the files listed below.

## Goal

Connect the already-tested Meta client to the backend while preserving the deterministic offline demo path.

## Files owned

- `backend/internal/model/ai_real.go` (+ tests)
- `backend/internal/meta/meta.go` (+ tests)
- `backend/internal/controller/controller.go`, `submit.go`, `whatsapp.go`
- `backend/main.go`
- `state/PROJECT_STATE.md`

## Requirements

- Select mock mode when `META_API_KEY` is empty and real mode otherwise; read env only at startup and log `ai=mock|real`.
- Inject an AI interface into controllers. Real transcription, text extraction, and photo price/object inspection use the Meta client.
- A provider error returns a visible gateway error; never silently fall back to mock data.
- In real mode, do not trust client-supplied `photo_price`; derive it from the uploaded image. Keep duplicate photo score 12 and server-owned score bands (nil slate 40, within 2% 94, within 10% 70, otherwise 20).
- Accept optional `audio_b64` and `audio_format` on `/api/submit`; existing transcript-based `scripts/demo.sh` remains unchanged.
- Use an HTTP timeout. Do not make live Meta API calls in tests or development verification.

## Done criteria

- `cd backend && go test ./... && go vet ./...`
- `./scripts/demo.sh` still yields median 187, confidence 100, and excludes duplicate-photo agent D.
- Fake Meta clients cover text extraction, image evidence, and provider errors offline.
