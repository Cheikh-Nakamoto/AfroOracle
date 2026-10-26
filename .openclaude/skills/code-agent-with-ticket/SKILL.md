---
name: code-agent-with-ticket
description: Launch the AfriOracle coding pipeline in manual (supervised) mode.
disable-model-invocation: true
---

# code-agent-with-ticket

Launch the AfriOracle coding pipeline in **manual mode**.

## What to do

1. Read `AGENTS.md`, `PIPELINE.md`, and `spec.md` to load the pipeline contract.
2. Dispatch the worker agents in execution order, one phase at a time:

   - `agents/00-bootstrap.md` (serial)
   - `agents/01-contracts.md` + `agents/02-backend-ai.md` + `agents/03-backend-consensus.md` (parallel, disjoint files)
   - `agents/04-backend-server.md` (after 01–03)
   - `agents/05-frontend.md` (after 04)
   - `agents/06-integration.md` (after 05)
   - `agents/07-demo-capture.md` (after 06)

3. **Manual mode = pause between agents.** After each worker finishes, read its handoff block in `state/PROJECT_STATE.md`, summarize it, and wait for the user's go-ahead before dispatching the next agent. Do not auto-run `scripts/run-pipeline.sh` end to end.

## Rules

- Each worker builds only the files its task file names. Enforce disjoint ownership.
- Develop on the mock AI path (`META_API_KEY` unset); real Meta API only for the demo run.
- Handoff blocks stay under ~1k tokens and reference files by path, never paste code.
- Hold any `git commit` for the user's explicit approval.
