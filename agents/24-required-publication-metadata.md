# BE-16 — Require settlement identity and proof metadata

## Goal
Reject on-chain price publications that omit the market identity, source, or audit
hashes needed to identify and verify the settlement record.

## Build
- `contracts/AgriculturalData.sol`
- `backend/internal/model/chain_test.go`
- `backend/internal/model/testdata/AgriculturalData.bin` (solc 0.8.20)
- `docs/contracts.md`
- `docs/blockchain-progress.md`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- `publish` rejects empty zone, product, date, source, consensus hash, or AI report hash.
- Local EVM regression tests assert the specific revert reasons.
- The updated Solidity compiles with solc 0.8.20; Go tests and vet pass.
