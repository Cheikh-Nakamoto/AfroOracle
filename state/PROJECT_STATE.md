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
