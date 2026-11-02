# BE-12 — Read the parametric insurance result on Amoy

## Goal
Add a read-only CLI to inspect `ParametricInsurance.check` for a zone and date after
the oracle has published a price.

## Build
- `backend/cmd/check-insurance/main.go`
- `backend/cmd/check-insurance/main_test.go`
- `docs/contracts.md`
- `docs/deployment.md`
- `README.md`
- `docs/blockchain-progress.md` (short French teammate handoff)
- `state/PROJECT_STATE.md` (handoff only)

## Done
- CLI requires an Amoy RPC and insurance contract address, but no private key.
- It refuses networks other than chain ID 80002, checks the contract address has code,
  and reads the threshold plus `check(zone,date)`.
- Output shows exact ×100 values and human-readable FCFA amounts; a missing oracle
  publication returns the contract's clear error.
- Unit tests cover network guard, date validation, and amount formatting; Go checks pass.
