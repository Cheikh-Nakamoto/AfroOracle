# BE-13 — Validate settlement values and insurance constructor inputs

## Goal
Enforce basic data invariants in the contracts so a publisher or deployer cannot
create invalid settlement records or a broken insurance consumer.

## Read
- `spec.md` §Contracts and §Acceptance criteria
- `agents/01-contracts.md`
- `contracts/AgriculturalData.sol`
- `contracts/ParametricInsurance.sol`
- `backend/internal/model/chain_test.go`

## Build
- `contracts/AgriculturalData.sol`
- `contracts/ParametricInsurance.sol`
- `backend/internal/model/chain_test.go`
- `backend/internal/model/testdata/AgriculturalData.bin`
- `backend/internal/model/testdata/ParametricInsurance.bin`
- `docs/contracts.md`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- Oracle reverts for zero price and confidence above 100; valid records still publish.
- Insurance constructor rejects a zero oracle address and a zero threshold.
- Local EVM tests cover each rejection and compile-generated test bytecode matches
  both updated contracts (Solidity 0.8.20).
- Go tests and vet pass.
