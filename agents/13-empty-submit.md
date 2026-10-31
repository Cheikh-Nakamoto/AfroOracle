# Task 13 — reject empty observations

## Goal

Ensure a missing transcript and missing typed value cannot become a market observation through a mock default.

## Scope

- `backend/internal/controller/submit.go`
- `backend/internal/controller/auth_test.go`
- `backend/internal/model/ai.go` (+ focused test)
- `backend/internal/model/ai_test.go`
- `state/PROJECT_STATE.md`

## Requirements

- Reject a whitespace-only transcript when no nonzero typed value is supplied.
- Do not synthesize the canonical demo sentence from empty input.
- Preserve transcript and explicitly typed value submissions, including the unchanged `scripts/demo.sh` golden flow.
- Reject before adding any submission to the store.

## Done criteria

- An authenticated request with a valid node ID but no observation returns 400 and leaves store state unchanged; unauthenticated `{}` still receives 401.
- A whitespace-only transcript with no typed value also returns 400.
- `cd backend && go test ./... && go vet ./...` and `./scripts/demo.sh` pass.
