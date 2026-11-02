# BE-11 — Reject insurance checks without a published price

## Goal
Prevent `ParametricInsurance.check` from treating an unset oracle record (price 0)
as a real market crash and calculating a false payout.

## Read
- `spec.md` §Contracts and §Acceptance criteria
- `agents/01-contracts.md`
- `contracts/ParametricInsurance.sol`
- `backend/internal/model/chain_test.go`

## Build
- `contracts/ParametricInsurance.sol`
- `backend/internal/model/chain_test.go`
- `backend/internal/model/testdata/ParametricInsurance.bin` (compiled from the contract with solc 0.8.20)
- `docs/contracts.md`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- `check(zone,date)` reverts with `price not published` when the oracle timestamp is
  zero; valid published prices preserve the current threshold and payout behavior.
- The EVM test checks the no-data revert and the existing 18700 → 1300 payout case.
- Test bytecode is regenerated with solc 0.8.20 and `go test ./...` passes.
