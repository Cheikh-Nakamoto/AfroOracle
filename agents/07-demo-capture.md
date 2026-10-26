# Agent 07 — Demo capture + montage

## Goal
Capture the footage and screen recordings for the 3-minute demo video, then assemble the montage.

## Depends on
Agent 06 (working end-to-end demo + script).

## Read
- `demo/capture-list.md` (the shot plan — update it if a beat changed)
- `demo/script.md` (agent 06's narrative)
- `our_projetc.md` §Script de la vidéo, §Coût de la vérification IA

## Build / capture
For each row in `demo/capture-list.md`, produce the asset: live footage (camera) or screen recording (dashboard, WhatsApp, explorer). Store raw clips in `demo/assets/` (add to `.gitignore` if large).

## Done
- Every one of the 6 beats has a captured asset (image or video).
- Screen captures show the real dashboard output (median, proof score, fraud highlighted, tx hash) — not mocks.
- The montage runs ~3 min and follows the order in `demo/capture-list.md`.
- A final rendered video (or an edit-project file) is referenced from `demo/`.

## Handoff
Append to `state/PROJECT_STATE.md`: where the final video lives, and any beats still missing footage.
