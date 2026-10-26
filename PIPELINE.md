# Pipeline

Spec-driven, multi-agent, context-budgeted. One supervisor (you, the human, or a chief-of-staff agent) dispatches a set of worker agents, each with one narrow task. Workers never share files, so they cannot conflict; they transfer state through compact handoff blocks, never through conversation transcripts.

## Principles

1. **Single source of truth.** `spec.md` holds the interfaces and acceptance criteria. `our_projetc.md` holds the domain rules. Nothing else defines behavior; agents read these, not each other's memories.
2. **Interface first.** The contract ABI and the Python module interfaces are written in `spec.md` before any worker runs, so downstream workers code against a file, not against another worker's output.
3. **Disjoint file ownership.** Each worker owns a unique set of files. Two workers never edit the same file.
4. **Structured handoff.** A worker ends by appending one block to `state/PROJECT_STATE.md`:

```markdown
## <agent> — <status>

- **done**: <what was built, files>
- **decisions_made**: <choices + why>
- **files_changed**: <paths>
- **open_questions**: <anything blocking the next agent>
- **next_task**: <pointer to the next agent file>
```

5. **Flat hierarchy.** One level of workers under the supervisor. No agent orchestrating other agents.
6. **Concrete done-criteria.** Every task file states checkable acceptance criteria, not "implement the feature".

## Agent roster

| # | Agent | Files owned | Depends on |
| --- | --- | --- | --- |
| 00 | [bootstrap](agents/00-bootstrap.md) | `.env.example`, `.gitignore` | — |
| 01 | [contracts](agents/01-contracts.md) | `contracts/*.sol` | spec §Contracts |
| 02 | [backend-ai](agents/02-backend-ai.md) | `backend/ai.go` | spec §AI |
| 03 | [backend-consensus](agents/03-backend-consensus.md) | `backend/consensus.go`, `backend/consensus_test.go` | spec §Consensus |
| 04 | [backend-server](agents/04-backend-server.md) | `backend/main.go`, `backend/go.mod` | 01, 02, 03 interfaces |
| 05 | [frontend](agents/05-frontend.md) | `frontend/*` (Next.js) | 04 API shape |
| 06 | [integration](agents/06-integration.md) | `scripts/demo.sh`, `demo/script.md` | all |
| 07 | [demo-capture](agents/07-demo-capture.md) | `demo/capture-list.md`, footage | 06 |

## Execution order

Serialize where files depend, parallelize where they do not:

1. `00-bootstrap` (commit scaffold, env).
2. `01-contracts` (ABI first) **and** `02-backend-ai` **and** `03-backend-consensus` — all three in parallel (disjoint files; interfaces already in `spec.md`).
3. `04-backend-server` after 01–03 (imports their interfaces).
4. `05-frontend` after 04 (needs the API/state shape).
5. `06-integration` after 05 (end-to-end + demo).
6. `07-demo-capture` after 06 (capture footage + montage).

## Running the pipeline

[`scripts/run-pipeline.sh`](scripts/run-pipeline.sh) is the supervisor loop: it launches each worker as a fresh non-interactive `claude -p` run, scoped to the repo, in the order above. Disjoint-file agents (`01`/`02`/`03`) run in parallel; the rest serialize. Each worker reads only `AGENTS.md` + `PIPELINE.md` + `spec.md` + its task file + the files it owns, then appends its handoff block to `state/PROJECT_STATE.md`.

```bash
./scripts/run-pipeline.sh            # full run from 00
FROM=04 ./scripts/run-pipeline.sh    # resume from agent 04
```

Env knobs: `CLAUDE_BIN` (CLI path, default `claude`), `ALLOWED_TOOLS` (worker tool scope, default `Read Write Edit Bash Glob Grep`), `VERBOSE=1` (stream output instead of logging to `state/run.log`).

## Storage

Hybrid on-chain/off-chain. Settlement data (published price, median, confidence, `aiReportHash`) lives on-chain; proof media (voice notes, photos) is content-addressed off-chain with only the hash on-chain; submissions/reputation are relational in PostgreSQL (`DATABASE_URL`, in-memory mock when unset). See `spec.md` §Storage for the full table and rules.

## Context budget

- A worker reads: `AGENTS.md`, `PIPELINE.md`, `spec.md`, its own task file, and the files it touches. That is the entire context. No full-repo exploration.
- Handoff blocks are the only cross-agent memory; keep them under ~1k tokens.
- Reuse the mock AI path during development so the $150 budget goes to the demo, not to development.
