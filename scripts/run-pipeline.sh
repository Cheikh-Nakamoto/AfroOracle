#!/usr/bin/env bash
#
# run-pipeline.sh — AfriOracle supervisor loop.
#
# Launches each worker agent in `agents/` in execution order, honoring the
# dependency graph from PIPELINE.md:
#
#   00 (bootstrap)  ->  01 + 02 + 03 (parallel, disjoint files)
#                    ->  04 (server)  ->  05 (frontend)
#                    ->  06 (integration)  ->  07 (demo capture)
#
# Each worker is a fresh non-interactive `claude -p` run scoped to the repo.
# Workers never share files; they transfer state only through the handoff
# block they append to state/PROJECT_STATE.md.
#
# Usage:
#   ./scripts/run-pipeline.sh            # run all agents from 00
#   FROM=04 ./scripts/run-pipeline.sh    # resume from agent 04
#   ALLOWED_TOOLS="Read Write Edit Bash Glob Grep" ./scripts/run-pipeline.sh
#
# Env:
#   CLAUDE_BIN      path to the CLI (default: claude)
#   ALLOWED_TOOLS   tools each worker may use (default: read/write/bash/search)
#   FROM            first agent number to run (default: 00)
#   VERBOSE         set to 1 to stream worker output instead of logging to file

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLAUDE_BIN="${CLAUDE_BIN:-claude}"
ALLOWED_TOOLS="${ALLOWED_TOOLS:-Read Write Edit Bash Glob Grep}"
FROM="${FROM:-00}"
VERBOSE="${VERBOSE:-0}"
LOGFILE="$REPO/state/run.log"

# Run workers on Meta Model API (Muse Spark) instead of the default model.
# Load META_API_KEY from .env and point the CLI at Meta's Anthropic-compatible
# Messages API. Override with META_BASE_URL / META_MODEL if needed.
if [[ -f "$REPO/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$REPO/.env"
  set +a
fi
if [[ -n "${META_API_KEY:-}" ]]; then
  export ANTHROPIC_BASE_URL="${META_BASE_URL:-https://api.meta.ai}"
  export ANTHROPIC_API_KEY="$META_API_KEY"
  export ANTHROPIC_MODEL="${META_MODEL:-muse-spark-1.3}"
fi

# Ordered list of agent numbers, in execution order. Numbers that may run in
# parallel are grouped on one line; the script `wait`s between lines.
ORDER=(
  "00"
  "01 02 03"
  "04"
  "05"
  "06"
  "07"
)

log() { printf '[pipeline] %s\n' "$*"; }

run_agent() {
  local num="$1"
  local taskfile
  taskfile="$(cd "$REPO/agents" && ls "$num"-*.md 2>/dev/null | head -n1)"
  if [[ -z "$taskfile" ]]; then
    log "error: no task file for agent $num (agents/$num-*.md)"; return 1
  fi
  taskfile="$REPO/agents/$taskfile"

  local prompt
  prompt="You are worker agent $num of the AfriOracle pipeline. "
  prompt+="Read AGENTS.md, PIPELINE.md, spec.md, and your task file $taskfile. "
  prompt+="Execute the task file exactly: build ONLY the files it names, run its "
  prompt+="done-criteria/tests, then append your handoff block to state/PROJECT_STATE.md. "
  prompt+="Do not edit any file owned by another agent. Reference files by path in your handoff; never paste code."

  log "launching agent $num ($(basename "$taskfile"))"
  if [[ "$VERBOSE" == "1" ]]; then
    ( cd "$REPO" && "$CLAUDE_BIN" -p "$prompt" --allowedTools "$ALLOWED_TOOLS" )
  else
    ( cd "$REPO" && "$CLAUDE_BIN" -p "$prompt" --allowedTools "$ALLOWED_TOOLS" ) \
      >>"$LOGFILE" 2>&1
  fi
  log "agent $num finished"
}

mkdir -p "$REPO/state"
: > "$LOGFILE"

for group in "${ORDER[@]}"; do
  pids=()
  for num in $group; do
    if [[ "$num" < "$FROM" ]]; then
      log "skipping agent $num (FROM=$FROM)"
      continue
    fi
    run_agent "$num" &
    pids+=("$!")
  done
  # Wait for every agent in this parallel group before the next line.
  for pid in "${pids[@]:-}"; do
    wait "$pid"
  done
done

log "pipeline complete — see state/PROJECT_STATE.md and state/run.log"
