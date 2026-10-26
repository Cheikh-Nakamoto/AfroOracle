# Agent 00 — Bootstrap

## Goal
Secure the scaffold: make the first git commit and set up the environment files the other agents need.

## Depends on
Nothing. Runs first.

## Read
- `AGENTS.md`, `PIPELINE.md`, `spec.md`
- `our_projetc.md` (skim, for context)

## Build
- `.gitignore` — ignore `.env`, `node_modules/`, `.next/`, `backend/afrioracle` (build binary), `demo/assets/`, `*.log`, `*.db`.
- `.env.example` — document (with placeholder values, never real secrets):

```
META_API_KEY=          # Meta AI models (Muse Spark, Voice Transcribe, SAM 3.1)
POLYGON_RPC=           # Polygon Amoy RPC URL
PRIVATE_KEY=           # deployer key (testnet only)
DATABASE_URL=          # PostgreSQL DSN for submissions/publications (unset = in-memory mock)
WHATSAPP_TOKEN=        # Meta WhatsApp Cloud API access token
WHATSAPP_VERIFY_TOKEN= # webhook verification token
```

## Done
- `.gitignore` and `.env.example` exist with the fields above.
- `git status` shows the scaffold (skills/, AGENTS.md, PIPELINE.md, spec.md, agents/, state/) plus `our_projetc.md` and `README.md`.
- `go version` is 1.22+ and `node --version` is 20+.

## Handoff
Append to `state/PROJECT_STATE.md`: commit hash, confirmation the env vars are documented, any deviation.
