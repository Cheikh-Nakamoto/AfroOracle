# PROJECT_STATE.md

Shared memory for the pipeline. Each agent appends one block when it finishes. Read this + `spec.md` + your task file; do not re-explore the repo. Keep blocks under ~1k tokens; reference files by path, never paste code.

## Phase
Bootstrap (scaffold in place, no application code yet).

## Scaffold status (done by setup, not an agent)
- Repo cleaned to `our_projetc.md` + `README.md` + this pipeline.
- Vendored skills in [`skills/`](../skills/), pipeline in [`PIPELINE.md`](../PIPELINE.md), root contract in [`AGENTS.md`](../AGENTS.md), spec in [`spec.md`](../spec.md).
- Agent task files in [`agents/`](../agents/).

## Done
_(none yet — agents append here)_

## In progress
_(none)_

## Next task
[`agents/00-bootstrap.md`](../agents/00-bootstrap.md)

## Decisions made
- Demo scope = millet price in Kaolack, end to end (see `spec.md`).
- Stack = Go (backend) + Solidity 0.8.20 (Amoy) + Next.js/TypeScript (frontend).
- Mock AI during development; real Meta API only for the demo run.
- Demo video = 6-beat montage from live footage + screen captures (see `demo/capture-list.md`).
- Storage = hybrid on-chain/off-chain (see `spec.md` §Storage): settlement on-chain, proof media content-addressed off-chain, submissions/reputation in PostgreSQL (`DATABASE_URL`, in-memory mock when unset).
- Supervisor loop = `scripts/run-pipeline.sh` launches each worker in order (01/02/03 in parallel).

## Open questions
- Is `.git` to be kept or re-initialized? (flagged to user)
