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

## Issue #42 (backend HTTP test harness + hardening) — done

- **done**: `controller.NewRouter(c, maxBody)` owns routing: per-route method guard (405 + `Allow`), body cap (413), JSON 404 for unknown paths, one error shape `{"error": ...}` (`view.WriteError`). `main.go` uses it with read/write/idle timeouts, exits non-zero if the port is busy, logs `ai=`/`chain=` mode (`model.Mode()`). `router_test.go` covers golden path over HTTP, 422, 413, 405/404, ask-without-data. `scripts/demo.sh` still DEMO OK (187, conf 100, D excluded, same tx).
- **decisions_made**: default body cap 8 MiB (base64 photos); `/webhook/whatsapp` keeps no CORS; method checks moved out of handlers into the router.
- **files_changed**: `backend/main.go`, `backend/internal/controller/{router.go,router_test.go,submit.go,consensus.go,whatsapp.go}`, `backend/internal/view/view.go`, `backend/internal/model/model.go`
- **open_questions**: none.
- **next_task**: GitHub issues #43, #44, #47, #48 (unblocked by #42).

## #43 Reject empty and invalid submissions (BE-02, BE-09)
- done: `POST /api/submit` validates before any store write (400 `{"error"}`): node_id 1-64 chars, tier 1-3, rep_score 0-100, value/photo_price >0 and finite, bad base64, and no value+no transcript. Table-driven HTTP tests in router_test.go.
- decisions_made: `SubmitReq.Value` is `*float64` (absent vs explicit 0; 0 is rejected). Invented-187 sentence now only from `model.Transcribe("")` in mock mode (fixture); real mode returns "" and the controller no longer calls it. Validation lives in `HandleSubmit` only; WhatsApp path unchanged. No audio field exists in the request, so "no audio" = no transcript.
- files_changed: backend/internal/controller/submit.go, router_test.go, backend/internal/model/ai.go
- open_questions: WhatsApp webhook still stores node_id "" / tier 0 submissions (not in #43 scope).
- next_task: next ready-for-agent audit issue. demo.sh: DEMO OK (187, conf 100, D excluded).

## #44 Consensus counts distinct sources, runs once per batch (BE-07, BE-12)
- done: quorum = 3 distinct accepted NodeIDs (else 422). A consensus run snapshots pending submissions, publishes, and consumes exactly that batch; a second call with no new batch gets 422 (one publication, rewards once). Runs are serialised by `Controllers.consensusMu`. Tests in router_test.go (distinct nodes, repeat call, 8 concurrent calls, -race).
- decisions_made: Store `AddPublication` replaced by `PendingBatch()` + `CommitBatch(pub, outcomes, batch)`; StateSub gains non-wire `Seq`/`Consumed`; outcomes applied only to batch rows by Seq. Mutex is held during ChainPublish (simple, correct; fine for demo). demo.sh unchanged.
- files_changed: backend/internal/model/{model.go,store.go}, backend/internal/controller/{controller.go,consensus.go,router_test.go}
- open_questions: duplicate NodeIDs inside a valid batch still each enter the median (only quorum counts distinct). Stale server on :8000 breaks demo.sh; use PORT/BASE override.
- next_task: next ready-for-agent audit issue. demo.sh DEMO OK (187, conf 100, D excluded).

## #45 Publish and quote only validated results (BE-08)
- done: confidence < 66 run returns 200 `validated:false` (empty tx/chain), no ChainPublish, no publication, so `/api/ask` keeps quoting the last validated value or says none. Also closes #44 gap: one value per node (latest in batch) enters the median/outcomes. Tests in router_test.go (low confidence with/without prior publication, duplicate node).
- decisions_made: unvalidated batch is NOT consumed (stays pending, no rewards/penalties applied) so more data can arrive; superseded duplicate rows are consumed with the batch on a validated run.
- files_changed: backend/internal/controller/{consensus.go,router_test.go}
- open_questions: a never-validating pending batch keeps blocking fresh runs from excluding old scattered rows (no expiry/reset); WhatsApp node_id "" still unvalidated.
- next_task: next ready-for-agent audit issue. demo.sh DEMO OK (187, conf 100, D excluded).

## #46 Authenticated nodes, server-side tier/reputation (BE-01)
- done: `POST /api/submit` and `/api/consensus` require `Authorization: Bearer <token>` of a registered node (401 otherwise; `WWW-Authenticate`). Submit also needs payload `node_id` == credential's node (403). Tier/rep always come from the registry; payload `tier`/`rep_score` are accepted on the wire but ignored (no longer validated), so a forged tier 3 cannot skip proof scoring. Tests in router_test.go + model/registry_test.go; demo.sh DEMO OK (187, conf 100, D excluded).
- decisions_made: `model.Registry` (token -> Node{ID,Tier,Rep}), loaded by `model.LoadRegistry()` from env `AFRIORACLE_NODES="id:token:tier:rep,..."`; if unset and mock mode, demo nodes A,B,C,E,F,G,H,D (tier 2) and FAO (tier 3) are seeded with token `dev-token-<ID>`; real mode with no env = empty registry (all 401). Any registered node may run consensus (no separate admin role). `NewControllers(store, registry)`; CORS allows `Authorization`. Tokens are plaintext in env/memory (demo scope).
- files_changed: backend/internal/model/{registry.go,registry_test.go}, backend/internal/controller/{controller.go,middleware.go,router.go,submit.go,consensus.go,router_test.go}, backend/main.go, scripts/demo.sh
- open_questions: WhatsApp webhook still unauthenticated (node "" / tier 0); frontend/README do not yet mention tokens; no rate limiting or token hashing.
- next_task: next ready-for-agent audit issue.

## #47 Real Meta AI path (task 11)
- done: with `META_API_KEY` set, `model` routes transcription (Muse Voice Transcribe), extraction (Muse Spark JSON mode), slate reading (Muse Spark vision) and object detection (SAM 3.1) through `meta.Client`; unset = unchanged deterministic mock. `POST /api/submit` accepts `audio_b64` (+`audio_format`), transcribed when no transcript/value; mock returns the 187 fixture. `GET /api/mode` returns `{"ai":"mock|real","chain":...}`. Offline tests in controller/realai_test.go (httptest fake Meta API) and meta_test.go. demo.sh DEMO OK (187, conf 100, D excluded).
- decisions_made: real mode ignores client `photo_price` (slate price only from the model); model/API failure on slate => score 40 (fail safe, rejected); transcribe/extract failure => 502. `model.Extract` now returns `(ExtractResult, error)`. Added `meta.NewWithBaseURL`, `Message.Images` (vision parts), `model.UseMeta` (test seam).
- files_changed: backend/internal/meta/{meta.go,meta_test.go}, backend/internal/model/ai.go, backend/internal/controller/{submit.go,router.go,root.go,realai_test.go}
- open_questions: SAM `/responses` shape, chat vision `image_url` parts, ASR `{text}` response and `audio/<fmt>` content type are OPENAI-COMPAT guesses: confirm on first live call (flagged in meta.go). Frontend does not yet send audio.
- next_task: live-call smoke test with a real key; frontend audio capture.

## #48 Honest chain publishing via go-ethereum (BE-04)
- done: `model.ChainPublish` now returns `(tx, chain, err)`. Real publish (go-ethereum v1.17.10, `bind/v2`) runs `AgriculturalData.publish` only when `POLYGON_RPC`, `PRIVATE_KEY` and `CONTRACT_ADDRESS` are all set; it waits for the receipt, requires status 1, and labels `polygon-amoy` only for chain id 80002 (other chains `evm-<id>`). Otherwise tx is the deterministic sha256 hash labelled `simulated` (demo hash unchanged). A failed/reverted publish returns 502 and commits nothing. Offline tests (model/chain_test.go) use `ethclient/simulated` with contracts compiled from contracts/*.sol (testdata/*.bin): real receipt, Published event, `ParametricInsurance.check` reads the price (payout 1300), revert is not labelled real, ABI drift check. demo.sh DEMO OK (187, conf 100, D excluded).
- decisions_made: new env `CONTRACT_ADDRESS` (deployed AgriculturalData), optional `ORACLE_ZONE`/`ORACLE_PRODUIT` (default Kaolack/millet); date = UTC today. ABI copied to model/AgriculturalData.abi.json (go:embed cannot leave module), test guards drift. `Mode()` reports chain real only with all three vars. `model.UseChain` is the test seam.
- files_changed: backend/go.mod, go.sum, internal/model/{chain.go,chain_test.go,model.go,AgriculturalData.abi.json,testdata/*.bin}, internal/controller/consensus.go
- open_questions: never run against live Amoy (no creds); testdata .bin compiled with solc 0.8.20 via solcjs, regenerate if contracts change.
- next_task: deploy to Amoy and smoke test.

## #49 Postgres store behind DATABASE_URL (BE-05)
- done: `model.PGStore` (pgx/v5 stdlib) selected in main.go when `DATABASE_URL` is set; unset = in-memory as before. Tables submissions, publications, last_outcomes, photo_fingerprints created idempotently on startup under an advisory lock. Verified against a real postgres:16 container (since removed): contract tests, restart persistence (`/api/state` byte-identical after restart), duplicate photo caught after restart, demo.sh DEMO OK (187, conf 100, D excluded) on Postgres and in-memory.
- decisions_made: Store write methods (`AddSubmission`, `ScorePhoto`, `CommitBatch`) now return errors; controllers answer 500 `storage unavailable`. Fingerprint claimed by `INSERT .. ON CONFLICT DO NOTHING`. `CommitBatch` is one tx (publication + outcomes + consume); a row already consumed aborts it. Publications ordered by timestamp (in-memory now too). Reads log errors and return empty. PG tests gated on `TEST_DATABASE_URL`.
- files_changed: backend/go.mod, go.sum, main.go, internal/model/{pgstore.go,store.go,store_test.go}, internal/controller/{submit.go,consensus.go}
- open_questions: read methods swallow DB errors (log only); no versioned migrations beyond IF NOT EXISTS; docker-compose has no postgres service/DATABASE_URL passthrough.
- next_task: add postgres service to docker-compose if wanted.

## #50 Secure the WhatsApp webhook (BE-06)
- done: GET verify refuses an unset token (constant-time compare). POST requires valid `X-Hub-Signature-256` (HMAC-SHA256 of raw body, `hmac.Equal`); missing/bad signature or unset `WHATSAPP_APP_SECRET` => 401. Bad JSON logged + 400. Sender `from` resolves to a registered node via the registry (`WHATSAPP_NODES="wa_id:nodeID,..."`), tier/rep from registry; unknown sender => 403, nothing stored. Duplicate `wamid` => one submission (200 `duplicate`). Fixture tests in controller/whatsapp_test.go. demo.sh DEMO OK (187, conf 100, D excluded).
- decisions_made: wamid dedupe is in-memory (`Controllers.waSeen`), released on 5xx so Meta retries work; lost on restart. Non-text messages and status updates are ignored with 200. Success returns processSubmit's body (201 accepted).
- files_changed: backend/internal/controller/{whatsapp.go,whatsapp_test.go,controller.go}, backend/internal/model/registry.go, .env.example
- open_questions: persistent wamid dedupe (store); `internal/model` has an intermittent -race failure seen once (passes on rerun), not investigated; waSeen unbounded.
- next_task: persist wamid in Store; investigate model race flake.

## #51 WhatsApp voice and photo end to end (BE-06)
- done: webhook handles `audio` and `image` messages. Media is fetched through the `controller.WAClient` seam (id -> URL -> bytes with `WHATSAPP_TOKEN`, size capped) and fed to `processSubmit` (audio_b64 / photo_b64). The sender gets a WhatsApp reply: accepted price + proof score, or the clarify question. Offline tests with an httptest fake Cloud API in controller/whatsapp_media_test.go. demo.sh DEMO OK (187, conf 100, D excluded).
- decisions_made: a bare photo is held per sender (in memory, latest only) and attached to their next voice/text; a photo with a caption is submitted directly (caption = transcript). Download failure => 502 and wamid released (Meta retries); no client (`WHATSAPP_TOKEN` unset) => 503 for media. Reply failure never undoes the stored submission: logged and returned as `reply_error`. New env: `WHATSAPP_PHONE_NUMBER_ID`, `WHATSAPP_API_BASE`, `WHATSAPP_MAX_MEDIA_BYTES` (default 8 MiB, checked on declared file_size and while streaming). Reply text is French, unlocalised.
- files_changed: backend/internal/controller/{whatsapp.go,whatsapp_client.go,whatsapp_media_test.go,controller.go}, backend/main.go, .env.example
- open_questions: never run against the live Cloud API (graph version v21.0 and payload shapes from docs); held photos and wamid dedupe are lost on restart and unbounded; reply is sent synchronously inside the webhook request.
- next_task: live smoke test with a real WhatsApp number; persist wamid/held photos.
