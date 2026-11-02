# BE-10 — Repeatable Polygon Amoy contract deployment

## Goal
Provide a small, guarded deployment path for the two existing demo contracts.
Do not add a token; the current demo asset is the read-only parametric insurance
scenario.

## Build
- `scripts/deploy-amoy.sh`
- `backend/cmd/deploy-amoy/main.go`
- `backend/cmd/deploy-amoy/main_test.go`
- `README.md`
- `docs/contracts.md`
- `docs/deployment.md`
- `.env.example`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- Script compiles the checked-in Solidity sources with solc 0.8.20 and runs the Go
  deployer using the resulting bytecode.
- Deployer requires RPC + private key, refuses any chain ID except 80002 before
  sending, deploys `AgriculturalData` then `ParametricInsurance`, waits for successful
  receipts, and prints both addresses and tx hashes without exposing key material.
- Insurance threshold is configurable in ×100 units and defaults to 20000.
- Local build and tests pass; no public transaction is sent without supplied Amoy
  credentials.
