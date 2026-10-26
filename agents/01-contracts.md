# Agent 01 — Contracts

## Goal
Write the Solidity contracts and the ABI (the on-chain interface every consumer reads).

## Depends on
`spec.md` §Contracts.

## Read
- `spec.md` §Contracts, §Storage, §Acceptance criteria
- `our_projetc.md` Couche 4 (what is stored at publication)

## Build
- `contracts/AgriculturalData.sol` — `publish`, `getPrice`, `getRainfall`; struct `Record{value, timestamp, confidence, source, consensusHash, aiReportHash}`; only `publisher` may publish; value stored ×100; `Published` event.
- `contracts/ParametricInsurance.sol` — `IAgri` interface, `check(zone, date)` returns payout when price > threshold.
- `contracts/abi.json` — the ABI for both contracts (downstream agents read this file, never re-derive the interface).

Storage rule (from `spec.md` §Storage): store settlement data only — value, timestamp, confidence, source, consensusHash, `aiReportHash`. Never media bytes or raw submissions; `aiReportHash` carries the off-chain proof link.

## Done
- Both contracts compile with solc 0.8.20.
- `abi.json` lists `publish`, `getPrice`, `getRainfall`, `check`, and their input/output types.
- `aiReportHash` field is present in `Record` and in the `publish` signature.

## Handoff
Append to `state/PROJECT_STATE.md`: `abi.json` path, the exact `publish`/`getPrice` signatures, deploy notes (constructor args, network).
