# Task 14 — report simulated chain publication honestly

## Goal

Ensure the API never labels a fabricated hash as a Polygon Amoy transaction.

## Scope

- `backend/internal/model/chain.go`
- `backend/internal/model/chain_test.go`
- `state/PROJECT_STATE.md`

## Requirements

- Keep the existing `ChainPublish` interface and deterministic simulated hash.
- Always return `chain: simulated` until the backend actually submits a contract transaction and waits for a successful receipt.
- Do not change the consensus JSON fields or the demo golden values.

## Done criteria

- A test with `POLYGON_RPC` and `PRIVATE_KEY` set still verifies `chain == simulated`.
- `cd backend && go test ./... && go vet ./...`
- `./scripts/demo.sh` still gives median 187, confidence 100, and excludes D.
