# BE-15 — Read the full published oracle record

## Goal
Expose the source and tamper-evidence hashes already stored by `AgriculturalData`
through a read-only contract function.

## Read
- `spec.md` §Contracts and §Storage
- `contracts/AgriculturalData.sol`
- `contracts/abi.json`
- `backend/internal/model/AgriculturalData.abi.json`
- `backend/internal/model/chain_test.go`

## Build
- `spec.md`
- `contracts/AgriculturalData.sol`
- `contracts/abi.json`
- `backend/internal/model/AgriculturalData.abi.json`
- `backend/internal/model/chain_test.go`
- `backend/internal/model/testdata/AgriculturalData.bin` (solc 0.8.20)
- `docs/contracts.md`
- `docs/blockchain-progress.md`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- Add `getPriceRecord(zone, product, date)` returning value, timestamp, confidence,
  source, consensus hash and AI report hash; preserve the existing `getPrice` function.
- Update the canonical spec and both ABI copies; the ABI drift test passes.
- Local EVM test verifies the full record after publication; Go tests and vet pass.
