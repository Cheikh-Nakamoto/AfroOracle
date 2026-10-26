# AGENTS.md — AfriOracle

AfriOracle is a decentralized oracle for African market data (Meta Global AI Developer Hackathon prototype). It publishes a weighted-median price on-chain; AI verifies proof, humans sign, consensus decides.

## Read first

- [`PIPELINE.md`](PIPELINE.md) — how agents work here (roles, handoff, context budget).
- [`spec.md`](spec.md) — the demo implementation spec (interfaces + acceptance criteria). **Single source of truth** for what to build.
- [`our_projetc.md`](our_projetc.md) — full protocol v2.0 (business model, tiers, penalties). Consult it for domain rules; `spec.md` wins where they disagree on demo scope.
- [`scripts/run-pipeline.sh`](scripts/run-pipeline.sh) — the supervisor loop that launches the worker agents in order.

## Stack

- Backend: Go (`net/http` stdlib), single external dep `go-ethereum` for Polygon. No web framework.
- Contracts: Solidity 0.8.20, Polygon testnet (Amoy).
- Frontend: Next.js (React + TypeScript).
- AI: Meta models (Muse Spark 1.3, Muse Voice Transcribe, SAM 3.1) with a mock fallback when `META_API_KEY` is absent.

## How to work

1. Read your task file in [`agents/`](agents/) and `spec.md`. Do **not** re-explore the whole repo.
2. Build only the files your task file names. No overlapping files between agents.
3. Run tests before finishing. Each task has concrete done-criteria in its file.
4. Append a handoff block to [`state/PROJECT_STATE.md`](state/PROJECT_STATE.md) when done — `done`, `decisions_made`, `files_changed`, `open_questions`, `next_task`. Never forward full conversation history.

## Cost rules ($150 budget)

- Develop with the mock AI path (`META_API_KEY` unset). Real Meta API only for the final demo run.
- Keep every handoff block under ~1k tokens. Reference files by path; do not paste code into state.
- Each agent reads: `AGENTS.md` + `PIPELINE.md` + `spec.md` + its own task file + the files it touches. Nothing else.
