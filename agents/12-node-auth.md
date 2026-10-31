# Task 12 — authenticate oracle nodes

## Goal

Prevent unauthenticated or self-attested requests from inserting trusted nodes or triggering consensus.

## Scope

- `backend/internal/controller/auth.go` and `auth_test.go`
- `backend/internal/controller/{controller,submit,consensus,middleware}.go`
- `backend/main.go`
- `.env.example`
- `scripts/demo.sh`
- `state/PROJECT_STATE.md`

## Requirements

- Require a registered Bearer API key for `POST /api/submit` and `POST /api/consensus`.
- Bind the credential to `node_id`; return 403 when a caller claims another node or submits a forged tier/reputation.
- Source tier and reputation from the server-side registry. Only explicitly registered tier-3 nodes can submit automatic observations.
- Keep `/api/state` and `/api/ask` public. Add `Authorization` to CORS allowed headers.
- Load production credentials from `AFRIORACLE_NODE_CREDENTIALS`; only enable public deterministic demo keys behind explicit `AFRIORACLE_DEMO_MODE=1`.
- Update `scripts/demo.sh` with Authorization headers while preserving its payloads and golden result.

## Done criteria

- Handler tests prove unauthenticated submits and forged tiers are rejected without mutating state; registered requests use server-owned tier/reputation; consensus rejects missing credentials.
- `cd backend && go test ./... && go vet ./...`
- `./scripts/demo.sh` still reports median 187, confidence 100, and D excluded.
