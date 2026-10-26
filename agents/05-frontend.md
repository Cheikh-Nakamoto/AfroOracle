# Agent 05 — Frontend: dashboard (Next.js)

## Goal
Build the Next.js dashboard that renders the full flow for the demo video.

## Depends on
Agent 04's `/api/state` shape (read its handoff block in `state/PROJECT_STATE.md`).

## Read
- `spec.md` §HTTP interface, §Acceptance criteria
- `state/PROJECT_STATE.md` — the `/api/state` JSON shape recorded by agent 04

## Build
- `frontend/*` only — a Next.js (App Router, TypeScript) app. Nothing else.

The dashboard page:
- Fetches the Go backend (`http://localhost:8000/api/state`, CORS is enabled) and renders submissions (node, value, proof score), highlighting fraud (`proof_score < 50`) in red.
- A "Lancer le consensus" button POSTs `/api/consensus`, then shows the median, confidence, tx, `aiReportHash`, and per-node rewards.
- Calls `/api/ask?q=prix du mil à Kaolack ?` and renders the assistant answer.

## Done
- `cd frontend && npm run dev` shows the dashboard with live state and no console errors.
- The fraud row is visibly flagged.
- Median/confidence/tx/rewards and the assistant answer render after a consensus run.

## Handoff
None (leaf agent). Append a short "done" note to `state/PROJECT_STATE.md`.
