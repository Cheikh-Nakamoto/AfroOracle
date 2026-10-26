# Contracts — AfriOracle

On-chain settlement layer, verified against `contracts/AgriculturalData.sol`,
`contracts/ParametricInsurance.sol`, and `contracts/abi.json` (keys:
`AgriculturalData`, `ParametricInsurance`, `IAgri`). Solidity **0.8.20**,
target network **Polygon Amoy** (testnet). Compiles with solc 0.8.20 (via npx).

## `AgriculturalData.sol` — settlement store

Stores **only settlement data**: value (×100), timestamp, confidence, source, consensus
hash, AI report hash. Media bytes and raw submissions never go on-chain.

```solidity
struct Record {
    uint256 valueX100;
    uint256 timestamp;
    uint256 confidence;
    string source;
    string consensusHash;
    string aiReportHash;
}
```

| Function | Access | Notes |
| --- | --- | --- |
| `constructor()` | — | Sets `publisher = msg.sender` |
| `publish(zone, produit, date, valueX100, confidence, source, consensusHash, aiReportHash)` | `onlyPublisher` | Key = `keccak256(zone, produit, date)`; stamps `block.timestamp`; emits `Published(zone, produit, date, valueX100, confidence, source, consensusHash, aiReportHash)` |
| `getPrice(zone, produit, date)` | view, anyone | Returns `(valueX100, timestamp, confidence)` |
| `setRainfall(zone, date, valueX100)` | `onlyPublisher` | Key = `keccak256(zone, date)` |
| `getRainfall(zone, date)` | view, anyone | Returns `uint256` (0 when unset) |

`onlyPublisher` reverts `"only publisher"` for anyone else. For the demo, the backend
calls `publish("Kaolack", …, 18700, …)` — value ×100, so 187 FCFA/kg → `18700`.

## `ParametricInsurance.sol` — demo consumer

```solidity
interface IAgri {
    function getPrice(string zone, string produit, string date)
        external view returns (uint256 valueX100, uint256 timestamp, uint256 confidence);
}
```

- `constructor(_oracle, _thresholdX100)` stores the oracle address, the insurer
  (`msg.sender`), and the insured floor price ×100.
- `check(zone, date)` (view) reads `oracle.getPrice(zone, "millet", date)` — the produit
  `"millet"` is hardcoded — and returns `0` when `price >= thresholdX100` (floor held,
  no indemnification), else `thresholdX100 - price` as the parametric payout in ×100 units.
- Note: `spec.md` §Contracts phrases the trigger as "price > threshold"; the code pays
  when the price falls **below** the insured floor. The code governs.

## ABI

`contracts/abi.json` holds the three ABIs (`AgriculturalData`, `ParametricInsurance`,
`IAgri`) for the backend's `go-ethereum` wiring. The wiring itself is still a `TODO` in
`backend/main.go:chainPublish` — dev runs emit a simulated `"0x"+sha256` tx.

## Deploy notes (Polygon Amoy)

1. Compile with solc **0.8.20** (matches `pragma solidity ^0.8.20`).
2. Deploy `AgriculturalData`; the deployer becomes `publisher` (phase-1 stand-in for the
   foundation multisig described in `our_projetc.md`).
3. Deploy `ParametricInsurance` with the oracle address and the floor, e.g. a threshold
   below 18700 to show "no payout" on the golden run.
4. Fund the deployer with Amoy testnet MATIC; verify the `Published` event for the
   Kaolack publication, then open the tx at `https://amoy.polygonscan.com` (beat 4).
