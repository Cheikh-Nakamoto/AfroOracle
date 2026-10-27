# BACKEND_AUDIT — AfriOracle Backend

- Scope: `backend/main.go`, `backend/go.mod`, `backend/internal/{model,controller,view,meta}/*.go` incl. tests; skim of `spec.md`, `scripts/demo.sh`, `docker-compose.yml`, `.env.example`, `audit-to-fix/UI_AUDIT.md` (cross-refs only).
- Date: 2026-10-26. Read-only audit; no code modified. Baseline re-verified: `go vet ./...` OK, `go test ./...` OK (`model`+`meta` only), `gofmt -l` clean.
- Verdict: offline mock demo is sound and golden-pinned, but backend is not production-honest: real Meta client is dead code, chain label fakes `polygon-amoy`, no persistence, and anyone can mint trusted submissions. **P0 ×1, P1 ×7, P2 ×6 (BE-01…BE-14).**

## Executive summary

Demo path works: deterministic mock AI (`internal/model/ai.go:17,39-152`), weighted-median consensus (median 187, confidence 100, fraudster D excluded at proof 12, `internal/model/consensus_test.go:10-48`), simulated publication, assistant quoting only the published value (`internal/controller/ask.go:36-48`). It is frozen by `scripts/demo.sh:29-41` (`META_API_KEY=` forced empty) and must not change. Findings concern off-script reality: missing Task 11 (BE-03), fake chain claim (BE-04), missing Postgres (BE-05), unsigned WhatsApp webhook (BE-06), oracle poisoning (BE-01/BE-02), operability/test gaps (BE-07…BE-14).

Frozen contract honoured by all fixes: demo.sh golden = median 187, confidence 100, D excluded (proof 12), `/api/ask` quotes 187 and never estimates; wire field names unchanged; mock mode (`META_API_KEY` unset) stays deterministic/offline.

## Inventory (evidence)

| Area | Files / lines | State |
| --- | --- | --- |
| Entry/routing | `backend/main.go:12-29` | in-mem store only; bare `ListenAndServe` |
| Controllers | `internal/controller/{controller,submit,consensus,state,ask,whatsapp,root,middleware}.go` | no auth; CORS only |
| Model | `internal/model/{model,ai,consensus,store,chain}.go` | mock AI; golden-tested consensus |
| Meta client | `internal/meta/meta.go:48-295` (`Client:48`, `New:55`, `Chat:120`, `Transcribe:158`, `Segment:205`, `GenerateImage:278`) | complete, tested, **unwired** |
| View/tests/config | `internal/view/view.go`; `consensus_test.go`, `meta_test.go`; `.env.example:1-10`, `docker-compose.yml` | no controller/store/chain/webhook tests; `DATABASE_URL`/`WHATSAPP_TOKEN` unused |

## Findings

### BE-01 — P0 — Unauthenticated, self-attested submissions can poison the oracle
- Evidence: no auth on any route (`main.go:17-21`; `middleware.go` is CORS-only). `submit.go:14-22` trusts caller `tier`, `rep_score`, `photo_price`, `value`; `submit.go:60-62` makes any self-declared `tier:3` proof-exempt; proof compares caller's own `photo_price` vs own `value` (`submit.go:63-66`); `photo_b64` only fingerprinted.
- Impact: one curl mints Expert/tier-1/tier-3 "sources" and moves the published median — the core trust claim.
- Fix: per-node credentials (API key/HMAC) on `/api/submit`+`/api/consensus`; server-side registry supplies tier/rep; tier-3 only for allow-listed IDs. Seed demo credentials for `scripts/demo.sh` (add headers, not payload changes).
- Acceptance: unauthenticated/forged-tier submits → 401/403; demo.sh golden passes; new handler tests cover both.

### BE-02 — P1 — Empty submission fabricates the canonical 187 FCFA price
- Evidence: `submit.go:41-42` sends empty transcript to `model.Transcribe("")`; `ai.go:39-43` returns literal "Le prix du mil à Kaolack est de 187…"; extract parses 187; `submit.go:88-92` stores it, replies `201 {"status":"accepted"}`.
- Impact: malformed/empty request (or text-less webhook event) becomes a genuine-looking observation of the demo price.
- Fix: empty transcript + no typed value + no audio → 400 or 200 clarify; delete default-sentence fallback in real mode; mock keeps it only as explicit demo fixture.
- Acceptance: `POST /api/submit {}` → 4xx, store unchanged; demo.sh still yields 187/100/D-out.

### BE-03 — P1 — Task 11 not wired: real Meta client is dead code
- Evidence: `ai.go:17` `var mock = os.Getenv("META_API_KEY") == ""`, wrappers `ai.go:146-152`; zero `meta.` imports in `internal/model/`/`main.go`; `main.go:13-14` builds only store + single-arg `NewControllers(store)` (`controller.go:14`); `ai_real.go` and `agents/11-real-ai.md` absent.
- Impact: setting `META_API_KEY` changes only a comment — demo still mocks while claiming Muse/SAM (judging/honesty risk).
- Fix/acceptance: see dedicated "Task 11" section below.

### BE-04 — P1 — Chain publication is simulated but labelled `polygon-amoy`
- Evidence: `internal/model/chain.go` returns `"0x"+sha256(...)` in both branches, only switching label to `polygon-amoy` when `POLYGON_RPC`/`PRIVATE_KEY` set; `go.mod` has no `go-ethereum` (contradicts `spec.md:28,129`); `controller/consensus.go:45-56` publishes tx/chain verbatim.
- Impact: UI/ask display fake on-chain proof; insurer "reading the contract" reads nothing.
- Fix: return `chain:"simulated"` unconditionally until the real `go-ethereum` `AgriculturalData.publish` call lands behind the same seam.
- Acceptance: never claim `polygon-amoy` without a real RPC receipt; demo.sh selectors (`tx`, `chain` present) pass.

### BE-05 — P1 — `DATABASE_URL`/Postgres promised, never implemented; state is volatile
- Evidence: `spec.md:41,49` + `.env.example:4` promise PostgreSQL when set; `main.go:13` always `NewInMemoryStore()`; no SQL driver in `go.mod`; `store.go:18-26` in-process only.
- Impact: restart/redeploy wipes submissions, publications, fingerprints (duplicate protection resets); compounds UI-02 in `UI_AUDIT.md`.
- Fix: implement `Store` interface (`store.go:7-16`) with Postgres backend selected in `main.go` when `DATABASE_URL != ""`; in-memory stays explicit dev/demo default.
- Acceptance: submit → restart → `/api/state` unchanged; demo.sh unaffected with var unset.

### BE-06 — P1 — WhatsApp webhook: no signature, fail-open verify, swallowed errors
- Evidence: `whatsapp.go:12-14` compares `hub.verify_token` to env — both empty ⇒ passes; `whatsapp.go:34` ignores decode error; no `X-Hub-Signature-256` check, no `WHATSAPP_APP_SECRET` in `.env.example`, no `wamid` dedup; text-only (`whatsapp.go:43-46`), no media download/reply despite `spec.md:125` and unused `WHATSAPP_TOKEN` (`.env.example:5`).
- Impact: forged "whatsapp" submissions; Meta retries duplicate; real voice notes silently dropped.
- Fix: fail closed if verify token unset; HMAC-verify signature (constant-time); dedup on `wamid`; 200-fast + idempotent processing; log drops.
- Acceptance: unsigned/badly-signed POST → 401; duplicate `wamid` → single submission; fixture-payload webhook tests.

### BE-07 — P1 — Consensus counts entries, not independent sources; reruns republish/re-reward
- Evidence: `controller/consensus.go:30-41` counts entries, no distinct-`node_id` check (spec: "3 independent submissions"); runs never consume/clear submissions (`store.go:54-70` only re-marks, collapsing by node in a map); each POST appends a publication and re-applies rewards.
- Impact: one node ×3 meets quorum alone; double-click/retry double-publishes and double-rewards.
- Fix: count distinct node IDs; snapshot-and-consume per batch; idempotent per batch ID.
- Acceptance: 3 submits from one node → 422; two consecutive consensus calls → one publication; golden test untouched.

### BE-08 — P1 — Unvalidated results are published and quoted as authoritative
- Evidence: `controller/consensus.go:43-58` calls `ChainPublish`/`AddPublication` unconditionally even when `Validated == false` (confidence < 66, `consensus.go:91-95`); `ask.go:44-47` quotes last publication with no validated check.
- Impact: low-confidence median becomes the "published" price for insurance flow and assistant.
- Fix: publish only when `Validated`; else return `validated:false`, store nothing (or flagged-unvalidated, never quoted by `/api/ask`).
- Acceptance: low-confidence test → no new publication, ask keeps previous validated value; golden (confidence 100) unchanged.

### BE-09 — P2 — Input validation conflates absent/invalid, ignores decode errors
- Evidence: `submit.go:45` treats `value:0` as absent; `submit.go:63` discards base64 error (`photoBytes, _ :=`); no range checks (`tier`, `rep_score`, `photo_price`, empty `node_id`); `extract` (`ai.go:60-63`) grabs first number anywhere; unknown tiers get weight 0 (`consensus.go:9,35`) yet outcome "accepted".
- Impact/fix: garbage stored as pending data polluting state/fingerprints. Validate ranges/enums, 400 on bad base64, `*float64` presence semantics, reject unknown tiers.
- Acceptance: table-driven handler tests per bad field; demo payloads remain valid.

### BE-10 — P2 — No body limits, timeouts, or startup error handling; unbounded growth
- Evidence: bare `json.NewDecoder(r.Body)` (`submit.go:31`, `whatsapp.go:34`), no `MaxBytesReader`, unbounded `photo_b64`; `main.go:29` bare `ListenAndServe` (no timeouts, error ignored — busy port exits silently); `store.go:22-25` slices grow forever.
- Impact/fix: slowloris/oversized-photo exhaustion; silent demo startup failure. Use `http.Server` timeouts, 1–2 MB cap, `log.Fatal` serve error, retention/BE-05.
- Acceptance: oversized body → 413; busy port fails loudly; demo.sh-spawned server still starts.

### BE-11 — P2 — HTTP contract inconsistencies: methods, catch-all, error shapes
- Evidence: `HandleState`/`HandleAsk` ignore method; unknown `/api/*` falls to HTML root 200 (`main.go:22`, `root.go`); errors mix `http.Error` text (`submit.go:27,32`) with JSON; `view.WriteJSON` ignores encode errors; contradicts `spec.md:118-127`.
- Impact/fix: client bugs become silent HTML-200 (partly roots UI-01). Method guards everywhere, JSON 404 for unknown `/api/*`, uniform `{"error":…}`.
- Acceptance: contract tests for 404/405/error shape; frontend/demo.sh selectors unchanged.

### BE-12 — P2 — Concurrency: mutex sound, cross-call atomicity is not
- Evidence: store methods lock+copy (`store.go:32-107`, `go test -race` passes — no race found), but proof-then-store is two locked calls (`submit.go:64` then `:78`) and consensus snapshot→publish is non-atomic (`controller/consensus.go:17,58`); no concurrent tests.
- Impact/fix: racing submits/runs interleave logically (compounds BE-07). Add atomic store ops (score+append; snapshot+publish batch), keep interface shape.
- Acceptance: `go test -race` with parallel submit/consensus tests; single-threaded golden unchanged.

### BE-13 — P2 — Test coverage stops at model+meta; HTTP surface untested
- Evidence: only `consensus_test.go` (golden `:10-48`) and `meta_test.go` exist; `controller`, `view`, `main`, `store`, `chain`, `whatsapp` report "no test files".
- Impact/fix: P1 fixes can regress API silently; only manual demo.sh guards wire contract. Add `httptest` suites + in-process demo-flow replica (9 submits → consensus → ask) as Go test.
- Acceptance: `go test ./...` covers controller/store; Go golden asserts 187/100/D-excluded and ask quoting.

### BE-14 — P2 — Config/observability: env at init, no mode reporting, wildcard CORS
- Evidence: `ai.go:17` reads env at package init (untestable); no startup log of ai/chain/store mode; `WHATSAPP_TOKEN` set-but-unused; `middleware.go` `Access-Control-Allow-Origin: *` on all `/api/*` (demo-OK per `spec.md:128`, undifferentiated); no `/healthz` distinct from `/api/state`.
- Impact/fix: operators can't tell mock from real; BE-03/04/05 stay invisible. Read env once in `main.go`, pass explicit config, log mode flags (never secrets), add `/healthz`.
- Acceptance: boot log shows `ai=mock store=memory chain=simulated`; demo.sh startup wait (`/api/state`) works.

## Task 11 — real Meta wiring

**Status: NOT STARTED (client built, nothing wired).** `internal/meta/meta.go` is complete/tested (`Client:48`, `New:55`, `Chat:120`, `Transcribe:158`, `Segment:205`, `GenerateImage:278`; `meta_test.go` passes via `httptest`). Missing: any `meta.` import in `internal/model/`/`main.go`; `internal/model/ai_real.go`; `agents/11-real-ai.md`; injection at `main.go:13-14` / `controller.go:14` (single-arg `NewControllers(store)`); audio field on `SubmitReq` (`submit.go:14-22` — `submit.go:42` hardcodes `model.Transcribe("")`).

**Audio scope flag:** `spec.md:77` pins `transcribe(audioHint string)`; HTTP body (`spec.md:119`) has `transcript?`/`photo_b64?` but **no audio field** — audio upload via `/api/submit` is *out of frozen demo scope*. Real audio enters via WhatsApp media (`spec.md:125`, BE-06) or an owner-approved additive field. Do not silently invent wire fields.

**Scoped fix steps:**
1. Write `agents/11-real-ai.md` first (scope/seams/acceptance), mirroring `agents/10-meta-client.md`.
2. Add `internal/model/ai_real.go`: an `AI` interface (Chat/Transcribe/Segment over `meta.Client`), mock as offline impl; real impl maps Chat JSON → `ExtractResult`, Segment+slate-read → `ProofResult`. Keep server-side rules authoritative (duplicate → 12, nil slate → 40, bands 94/70/20, `ai.go:80-125`) so D stays excluded in real mode too.
3. Inject, don't getenv: thread client `main.go:14` → `NewControllers(store, ai)` (`controller.go:14`); read `META_API_KEY` once in `main` (fixes BE-14 here); keep wrappers (`ai.go:146-152`) delegating to injected impl so call sites barely move.
4. Audio (only if approved): optional `audio_b64`/`audio_format` on `SubmitReq`, size-capped decode, 400 on error, `Transcribe(ctx, bytes, fmt)` when present; empty-everything → clarify/400 (BE-02), never default sentence. Else defer to WhatsApp-media task and state so.
5. Timeouts/errors: `meta.New` uses `http.DefaultClient` (`meta.go:56`) — build with timeout client or 10–15 s ctx; on Meta error return 502/clarify, never silently fall back to mock while claiming real.
6. Do not wire `GenerateImage` (no product use; `AGENTS.md` budget: real API only for final demo run).

**Risks to frozen contract:** endpoint JSON shapes are OpenAI-compatible assumptions (`meta.go:15-19`) — verify live once, behind the interface; nondeterminism/latency must not leak into mock mode (demo.sh forces `META_API_KEY=`, `demo.sh:31`, so golden stays identical if mock paths untouched); cache/dev against `httptest` for cost.

**Task 11 acceptance:** no key → `go test ./...` + demo.sh unchanged; fake `httptest` Meta server → real adapters run (extract, slate score, D excluded); live key → one end-to-end run reproduces 187/100; boot log prints `ai=real|mock`.

## Ordered implementation plan

1. BE-02 empty-submit fabrication (demo-safe, one function). 2. BE-04 `chain:"simulated"` label honesty. 3. BE-01 node auth + server-side tier/rep. 4. BE-07+BE-08 consensus integrity. 5. BE-06 WhatsApp signature/dedup/fail-closed. 6. BE-03 Task 11 wiring incl. BE-14 mode logging. 7. BE-05 Postgres `Store`. 8. BE-09/BE-10 validation/limits/timeouts. 9. BE-11/BE-12/BE-13 contract cleanup, atomic ops, HTTP tests.

## Do not touch / frozen

- `scripts/demo.sh` payloads/selectors/expected values (187, 100, D out/proof 12, ask quotes 187); golden `consensus_test.go:10-48`; wire names (untagged `Submission`/`Outcome` JSON, `model.go:3-6`, all `json:"…"` tags).
- Mock rules in `ai.go` (dup 12, nil 40, 94/70/20) and mock determinism with `META_API_KEY` unset; weights/reputation grids (`consensus.go:9,13-26`); "ask never estimates" (`ask.go:39-41`); frontend files (`UI_AUDIT.md`); `spec.md` text.
