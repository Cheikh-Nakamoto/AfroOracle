# Agent 04 — Backend: HTTP server + webhook + chain publish (Go)

## Goal
Implement the `net/http` server that wires AI + consensus + chain into the public API, and own the Go module.

## Depends on
Agents 01 (ABI), 02 (`ai.go`), 03 (`consensus.go`). Code against `spec.md` interfaces, not their files.

## Read
- `spec.md` §HTTP interface, §AI, §Consensus, §Storage
- Meta WhatsApp Cloud API docs (webhook handshake + message shape)

## Build
- `backend/main.go`, `backend/go.mod`, `backend/go.sum`. Module `afrioracle`; external dep `github.com/ethereum/go-ethereum`. Nothing else.
- Persistence (from `spec.md` §Storage): store submissions/publications in PostgreSQL when `DATABASE_URL` is set, else an in-memory store (mock path). Never store media bytes — hash them via `reportHash` and keep only the hash.

Endpoints (from `spec.md`):
```
POST /api/submit      -> transcribe + extract + proofScore + Submission; clarify if needed
POST /api/consensus   -> runConsensus + chainPublish; 422 under 3 sources
POST /webhook/whatsapp-> parse Meta Cloud API, fall through to /api/submit
GET  /webhook/whatsapp-> verification handshake
GET  /api/state       -> {submissions, publications, rewards, fingerprints}
GET  /api/ask?q=...   -> assistant (published data only, never estimates)
GET  /                -> dashboard (Next.js build)
```

`chainPublish(result, report)`: simulated tx (`"0x" + sha256`) when `POLYGON_RPC`/`PRIVATE_KEY` unset; otherwise `go-ethereum` call to `AgriculturalData.publish` (leave a clear TODO if you cannot wire the ABI). Include `aiReportHash` and `value_x100` in every publication. Set CORS headers on `/api/*`.

## Done
- `cd backend && go run .` serves the API and responds to every endpoint with the `spec.md` shapes.
- Submitting the doc's 4 agents + FAO via `/api/submit` then `/api/consensus` yields median 187, confidence 100, fraud (D) excluded.
- `< 3` submissions → 422 with the collecte-rule error.

## Handoff
Append to `state/PROJECT_STATE.md`: the exact `/api/state` JSON shape (agent 05 depends on it), and how to run the server.
