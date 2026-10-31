# Task 15 — make consensus batches independent and idempotent

## Goal

Enforce distinct node sources, consume each pending snapshot once, make retries idempotent, and never publish or reward an unvalidated result.

## Scope

- `backend/internal/model/{model,consensus,store}.go` (+ tests)
- `backend/internal/controller/consensus.go` (+ tests)
- `state/PROJECT_STATE.md`

## Requirements

- Quorum requires at least three distinct accepted, nonempty `node_id` values; only one accepted contribution per node can affect a batch.
- Atomically snapshot and claim pending submissions. Insufficient quorum leaves them pending.
- Accept an optional client `batch_id`, generate one if absent, return it, cache completed results, and never publish/reward the same batch twice.
- For `Validated == false`, consume/cache the result but do not publish or reward it; keep prior validated publications available to `/api/ask`.
- Preserve the golden 187/100 demo and public response field names; additive `batch_id`/`replayed` fields are allowed.

## Done criteria

- Three submissions from the same node return 422.
- Retrying one batch ID and a second consensus request create at most one publication.
- A low-confidence run creates no publication, and `/api/ask` still quotes the previous validated price.
- `cd backend && go test ./... && go vet ./...` and `./scripts/demo.sh` pass.
