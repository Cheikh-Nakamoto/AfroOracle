# BE-09 — Polygon Amoy publishing guard and deployment guide

## Goal
Make the backend refuse to publish to an unexpected EVM network and bring the
Polygon deployment documentation in line with the existing receipt-checked publisher.

## Build
- `backend/internal/model/chain.go`
- `backend/internal/model/chain_test.go`
- `.env.example`
- `README.md`
- `docs/architecture.md`
- `docs/api.md`
- `docs/demo.md`
- `docs/contracts.md`
- `docs/deployment.md`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- Production `NewPublisher` sends only when the connected chain ID is Polygon Amoy
  (80002); refusal occurs before signing or sending a transaction.
- The local simulated-EVM contract tests remain supported through an unexported
  test constructor.
- Environment variables document the deployed oracle address and publication keys.
- Docs accurately distinguish local simulated-EVM tests from a public Amoy run.
