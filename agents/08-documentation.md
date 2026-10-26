# Agent 08 — Full Documentation

You are worker agent 08 of the AfriOracle pipeline. You have full autonomy to read and write any files you need. Read `AGENTS.md`, `PIPELINE.md`, `spec.md`, and `our_projetc.md` first, then read every source file you document so each claim is verified against the actual code (`backend/*.go`, `backend/go.mod`, `contracts/*.sol`, `contracts/abi.json`, `frontend/app/*.tsx`, `frontend/package.json`, `frontend/tsconfig.json`, `scripts/demo.sh`, `.env.example`).

## Goal — a COMPLETE documentation set, not just a README

Build these files. Own them all; edit anything under `docs/` and `README.md`. Do NOT modify source code, contracts, scripts, or `state/` (those belong to other agents) — but you MAY read them all.

1. `README.md` — keep/refresh: one-line pitch, ASCII architecture, component table, quickstart, demo numbers, demo-day TODO, and a table of contents linking to every file in `docs/`.
2. `docs/architecture.md` — system overview, data flow (field agent → transcribe → extract → proof score → weighted median → on-chain publish → insurance check → assistant), component responsibilities, storage model (on-chain settlement / off-chain proof / PostgreSQL with in-memory fallback), failure modes.
3. `docs/consensus.md` — full consensus spec: weighted median, tier weights, reputation grid, proof-score thresholds, automatic nodes, exclusion rules, collecte rule, confidence formula, worked golden example (median 187, rewards A=6.5 B=5 C=5 FAO=0.5 D=0, D excluded with delta −2).
4. `docs/api.md` — complete API reference: every endpoint (`/api/submit`, `/api/consensus`, `/api/state`, `/api/ask`, `/webhook/whatsapp`, `/`), method, request body fields, response shape, status codes, error cases. Derive field names exactly from the Go structs (`SubmitReq`, `Outcome`, etc.).
5. `docs/contracts.md` — `AgriculturalData.sol` (Record struct, publish/getPrice/setRainfall/getRainfall, only publisher, events) and `ParametricInsurance.sol` (IAgri, check(zone,date), payout formula); ABI locations; deploy notes for Polygon Amoy.
6. `docs/deployment.md` — env vars (from `.env.example`), how to build/run backend, frontend, run `./scripts/demo.sh`, run tests (`go test -count=1 .`), and the live Amoy steps (`META_API_KEY`, `POLYGON_RPC`, `PRIVATE_KEY`).
7. `docs/demo.md` — the 3-min demo walkthrough: six beats, what each shows, the golden numbers, and the remaining day-of shoot list (market b-roll, screen recordings, live tx, final mp4).

## Done criteria

- All 7 files above exist and are non-stub.
- Every endpoint, field name, flag, and command is copied/derived from real source (verify against code, not the older docs).
- `README.md` links to all `docs/` files.

## Handoff

Append one block to `state/PROJECT_STATE.md` under `## 08-documentation — done`: `done` (list the files), `files_changed`, `open_questions`, `next_task` (none). Keep under ~200 tokens. Reference files by path; never paste code.
