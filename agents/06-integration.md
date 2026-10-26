# Agent 06 — Integration + demo

## Goal
Prove the whole flow end to end and produce the 3-minute demo script that feeds the capture/montage.

## Depends on
All of 01–05.

## Read
- `spec.md` (all), `our_projetc.md` §Script de la vidéo
- `state/PROJECT_STATE.md` — every prior handoff block

## Build
- `scripts/demo.sh` — end-to-end run: submit the 8 agents + the scripted fraudster via `/api/submit`, then `/api/consensus`, then `/api/ask`.
- `demo/script.md` — the 3-minute video script: the six beats expanded shot-by-shot, each beat tagged with what must be captured (live footage or screen capture). This is the input to agent 07's `demo/capture-list.md`.

## Done
- `cd backend && go test .` passes (golden case from agent 03).
- `scripts/demo.sh` publishes a median, flags the fraudster, and the assistant answers with the published value (never an estimate).
- `demo/script.md` covers the doc's six beats (0:00 problem → 2:45 cost) and tags each with its capture need.

## Handoff
Final update to `state/PROJECT_STATE.md`: how to run the demo start-to-finish, the published median, and a pointer to agent 07 for capture/montage.
