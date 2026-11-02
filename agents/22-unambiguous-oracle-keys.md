# BE-14 — Use unambiguous keys for oracle records

## Goal
Prevent collisions between different zone/product/date tuples in the oracle's packed
string storage keys.

## Read
- `spec.md` §Contracts and §Storage
- `agents/01-contracts.md`
- `contracts/AgriculturalData.sol`
- `backend/internal/model/chain_test.go`

## Build
- `contracts/AgriculturalData.sol`
- `backend/internal/model/chain_test.go`
- `backend/internal/model/testdata/AgriculturalData.bin` (compiled with solc 0.8.20)
- `docs/contracts.md`
- `state/PROJECT_STATE.md` (handoff only)

## Done
- Price keys hash `abi.encode(zone, product, date)` and rainfall keys hash
  `abi.encode(zone, date)`.
- A local EVM test verifies that tuples with colliding concatenated strings remain
  separate for both price and rainfall records.
- Solidity compiles with 0.8.20; `go test ./...` and `go vet ./...` pass.
