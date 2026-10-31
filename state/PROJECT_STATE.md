# PROJECT_STATE.md

Shared memory for the pipeline. Each agent appends one block when it finishes. Read this + `spec.md` + your task file; do not re-explore the repo. Keep blocks under ~1k tokens.

## Phase
Contracts + AI + consensus done (01/02/03). Next: 04 backend server.

## Done
- **00-bootstrap**: `.gitignore`, `.env.example` (6 vars). Toolchain go1.27.3 / node v24.22.1. Commit by user (d1c8233).
- **01-contracts**: `contracts/AgriculturalData.sol` (publish/getPrice/getRainfall, `Record{valueX100,timestamp,confidence,source,consensusHash,aiReportHash}`, only publisher), `contracts/ParametricInsurance.sol` (IAgri + `check(zone,date)`), `contracts/abi.json` (AgriculturalData, ParametricInsurance, IAgri). Compiles with solc 0.8.20 (via npx).
- **02-backend-ai**: `backend/ai.go`. Mock when `META_API_KEY` unset. `transcribe` (hint/default FR), `extract` (parse number; NeedsConfirmation+question when none), `proofScore` (dup→12, nil slate→40, gap ≤2%→94, ≤10%→70, else 20), `reportHash` ("0x"+sha256).
- **03-backend-consensus**: `backend/consensus.go` + `backend/consensus_test.go`. `reputation`, `Submission{Weight,Accepted}`, `weightedMedian`, `runConsensus`, `contextAgent` stub. Golden test passes: median 187, conf 100, rewards A=6.5 B=5 C=5 FAO=0.5 D=0, D scoreDelta −2.

## Next task
[`agents/04-backend-server.md`](../agents/04-backend-server.md)

## Decisions made
- (unchanged) Demo = millet price in Kaolack; Go + Solidity 0.8.20 + Next.js; mock AI in dev; hybrid storage; supervisor loop.
- `backend/go.mod` = `module afrioracle`, `go 1.22` (created as test prerequisite; agent 04 adds `go-ethereum`).

## Open questions
- `.git` kept (user committed on top of Initial commit). Resolved.

## 06-integration — done

- **done**: `scripts/demo.sh` (end-to-end: 8 honest agents A/B/C/E/F/G/H + FAO auto node + fraudster D via `/api/submit`, then `/api/consensus`, then `/api/ask`) and `demo/script.md` (3-min script, six beats shot-by-shot, each tagged [LIVE]/[SCREEN]). Verified fresh: median 187, confidence 100, D excluded (dup photo → 12, reward 0, delta −2), assistant quotes 187 with no estimation. `go test -count=1 .` passes.
- **decisions_made**: extra agents E–H at 184/186/188/189 keep weighted median on 187 (all deviations <2%, conf stays 100); fraud via byte-identical photo reuse; script builds a backend binary (not `go run`) so its EXIT trap kills exactly what it started; assertions tolerate accumulated state on a reused server (no reset endpoint in agent 04's frozen surface); `Outcome` JSON uses Go field names (`NodeID`, `Status`) — jq selectors match that.
- **files_changed**: `scripts/demo.sh`, `demo/script.md`
- **open_questions**: none blocking. Real demo run needs `META_API_KEY` + `POLYGON_RPC`/`PRIVATE_KEY` for a live Amoy tx in beat 4 (dev shows `chain: simulated`).
- **next_task**: [`agents/07-demo-capture.md`](../agents/07-demo-capture.md) — capture/montage from `demo/script.md`. Run: `./scripts/demo.sh` (starts its own backend if none at `$BASE`, default `http://localhost:8000`).

## 07-demo-capture — done

- **done**: `demo/capture-list.md` (6 beats, [LIVE]/[SCREEN] tagged, status per asset), `demo/montage.md` (3-min edit-project: order/timings/voiceover/remaining-shoot list), `demo/assets/` (7 beat stills 1080p SVG: beat1-title, beat2a-live, beat2b-whatsapp, beat3-dashboard, beat4-publication, beat5-assistant, beat6-cost) + `demo/assets/raw/{state,consensus,ask}.json` (real `./scripts/demo.sh` snapshots).
- **decisions_made**: stills are REAL data for screen beats (median 187, conf 100, D=12 excluded, tx `0x6783…cb7d25` simulated); [LIVE] market beats are placeholders to shoot on demo day; beats 2b–5 video + real Amoy run deferred to a live run (`META_API_KEY`, `POLYGON_RPC`, `PRIVATE_KEY`).
- **files_changed**: `demo/capture-list.md`, `demo/montage.md`, `demo/assets/*.svg`, `demo/assets/raw/*.json`
- **open_questions**: none blocking. Final ~3-min mp4 must be rendered and dropped under `demo/` on demo day; path to be recorded here.
- **next_task**: none — pipeline complete (00→07).

## 08-documentation — done

- **done**: `README.md` (replaces `# AfroOracle` stub): pitch, ASCII architecture, component table, quickstart, API surface, consensus summary, golden numbers, demo-day TODO. All commands/endpoints verified against `backend/main.go`, `backend/ai.go`, `backend/consensus.go`, `contracts/*.sol`, `frontend/app/page.tsx`, `frontend/package.json`, `scripts/demo.sh`, `.env.example`.
- **files_changed**: `README.md`
- **open_questions**: none.
- **next_task**: none — pipeline complete (00→08).

## 09-backend-mvc — done

- **done**: refactored flat `package main` backend into Model/Controller/View with zero wire change. `go build`, `go vet`, `go test` pass from `backend/`; `./scripts/demo.sh` prints `DEMO OK: median 187 ... D excluded` with the same tx hash as pre-refactor (`0x6783…cb7d25`).
- **decisions_made**: model keeps unexported impls (`runConsensus`, `transcribe`, `extract`, `proofScore`, `reportHash`, `chainPublish`) + thin exported wrappers (`RunConsensus`, `Transcribe`, `Extract`, `ReportHash`, `ChainPublish`) so `consensus_test.go` (now `package model`) still calls unexported fns; fingerprint map encapsulated in `InMemoryStore` via `ScorePhoto`; wire shapes preserved via identical map literals + verbatim-moved types (Go field names on submission/proof/outcome payloads); `SubmitReq` lives in controller as request DTO; `BaseReward` const lives in model.
- **files_changed**: `backend/main.go` (composition root), `backend/internal/model/{model,consensus,ai,chain,store}.go` + `consensus_test.go`, `backend/internal/controller/{controller,submit,consensus,state,ask,whatsapp,root,middleware}.go`, `backend/internal/view/view.go`; deleted `backend/ai.go`, `backend/consensus.go`, `backend/consensus_test.go`.
- **open_questions**: none.
- **next_task**: none — pipeline complete (00→09).

## 10-meta-client — done

- **done**: `backend/internal/meta/{meta.go,meta_test.go}` — stdlib-only Meta Model API client. `New(apiKey)`, `Chat` (OpenAI-compatible `/v1/chat/completions`, JSON-object mode), `Transcribe` (`/v1/asr/transcribe`), `Segment` (SAM 3.1 via `/v1/responses`), `GenerateImage` (`/v1/images/generations`). Offline request-construction tests. `go build/vet/test` pass.
- **decisions_made**: base `https://api.meta.ai/v1`, `Authorization: Bearer`; `New` takes the key (no env read in-package); no live calls in dev — tests assert URL/header/body.
- **files_changed**: `backend/internal/meta/meta.go`, `backend/internal/meta/meta_test.go`
- **open_questions**: none. Exact SAM/image field shapes adapted from docs; confirm on first live call.
- **next_task**: [`agents/11-real-ai.md`](../agents/11-real-ai.md) — wire `model` AI funcs to `meta` + add audio upload to submit.

## 12-node-auth — done

- **done**: Added a startup-loaded per-node Bearer key registry. `/api/submit` now rejects missing credentials, mismatched node IDs, and forged tier/reputation, then applies registry metadata. `/api/consensus` requires a registered credential; state and ask remain public. Added constant-time hashed-key comparisons, credential JSON config, explicit demo-only seeded keys, CORS Authorization support, handler tests, and demo request headers.
- **decisions_made**: Any registered node may trigger consensus after enough accepted submissions; this follows the issue's per-node credential decision. Public demo keys are only loaded with `AFRIORACLE_DEMO_MODE=1`; production starts with no submit credentials unless `AFRIORACLE_NODE_CREDENTIALS` is configured.
- **files_changed**: `agents/12-node-auth.md`, `.env.example`, `backend/internal/controller/auth.go`, `backend/internal/controller/auth_test.go`, `backend/internal/controller/controller.go`, `backend/internal/controller/submit.go`, `backend/internal/controller/consensus.go`, `backend/internal/controller/middleware.go`, `backend/main.go`, `scripts/demo.sh`.
- **open_questions**: None for the demo. Production deployment must provision unique random node keys in `AFRIORACLE_NODE_CREDENTIALS`.
- **next_task**: BE-02 empty-submission fabrication (reject empty transcript/value/audio and preserve deterministic demo behavior).

## 13-empty-submit — done

- **done**: `/api/submit` returns 400 for an empty or whitespace-only transcript when no typed value is provided, before writing to the store. Removed the mock transcription's implicit 187 FCFA sentence; the demo continues to provide its observation transcript explicitly.
- **decisions_made**: A nonzero typed value remains a valid input without a transcript; empty and zero-valued requests are not treated as observations.
- **files_changed**: `agents/13-empty-submit.md`, `backend/internal/controller/submit.go`, `backend/internal/controller/auth_test.go`, `backend/internal/model/ai.go`, `backend/internal/model/ai_test.go`.
- **open_questions**: None.
- **next_task**: BE-04 chain publication must report `simulated` rather than `polygon-amoy` when no real transaction is sent.

## 14-chain-label — done

- **done**: `ChainPublish` now always labels the deterministic report hash `simulated`, even when Polygon credentials are present. Added a regression test for configured RPC/key values.
- **decisions_made**: A configured endpoint and key do not demonstrate that a contract call or receipt exists; the label remains simulated until the actual go-ethereum publish path is implemented.
- **files_changed**: `agents/14-chain-label.md`, `backend/internal/model/chain.go`, `backend/internal/model/chain_test.go`.
- **open_questions**: Real Amoy publication remains separate work and needs deployed contract configuration plus a receipt-backed transaction.
- **next_task**: BE-07/BE-08 consensus integrity: count independent sources and prevent repeated consensus calls from publishing/rewarding again.

## 15-consensus-integrity — done

- **done**: Consensus quorum now requires three distinct, nonempty node IDs, and repeated accepted submissions from one node contribute only once per batch. The in-memory store atomically snapshots and consumes pending submissions, caches batch outcomes by `batch_id`, and returns cached results for retries. Unvalidated runs consume/cache their batch without publication or rewards; the last validated publication remains available to `/api/ask`.
- **decisions_made**: Empty request bodies remain supported by generating a cryptographically random batch ID; clients can supply an ID to retry safely. Insufficient batches are not consumed. Duplicate accepted entries are retained in the audit outcomes as excluded with zero reward.
- **files_changed**: `agents/15-consensus-integrity.md`, `backend/internal/model/{model,consensus,store}.go`, `backend/internal/controller/consensus.go`, `backend/internal/controller/consensus_batch_test.go`.
- **open_questions**: Batch idempotency survives only for the lifetime of the in-memory store; durable persistence is a separate task.
- **next_task**: BE-06 WhatsApp signature validation and fail-closed webhook handling.
