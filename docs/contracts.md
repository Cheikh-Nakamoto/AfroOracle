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
`IAgri`). The Go backend embeds the oracle ABI at
`backend/internal/model/AgriculturalData.abi.json`; a test checks it against the
canonical ABI. With `POLYGON_RPC`, `PRIVATE_KEY` and `CONTRACT_ADDRESS` configured,
the backend sends `publish`, waits for a successful receipt, and returns the real tx
hash. Without all three values, it uses a deterministic simulated hash.

## Deploy notes (Polygon Amoy)

1. Compile both contracts with solc **0.8.20** (matches `pragma solidity ^0.8.20`).
2. Deploy `AgriculturalData` to Polygon Amoy. Its deployer becomes `publisher`; use a
   dedicated test wallet funded only with Amoy test POL.
3. Deploy `ParametricInsurance` with the oracle address and a threshold in ×100 units.
   For a 200 FCFA/kg floor, pass `20000`; a published price of 18700 yields a demo
   payout of 1300 (13 FCFA/kg in ×100 representation).
4. Set `POLYGON_RPC`, `PRIVATE_KEY` and `CONTRACT_ADDRESS` in the local environment.
   Optionally set `ORACLE_ZONE` and `ORACLE_PRODUIT`; defaults are `Kaolack` and
   `millet`. The backend refuses a configured publisher connection unless its chain
   ID is Polygon Amoy (80002).
5. Run a validated consensus and inspect the successful transaction and `Published`
   event at `https://amoy.polygonscan.com`.

The backend publisher wallet must be the contract's `publisher` address. The insurance
contract's `check` method is read-only: the current demo computes a payout amount but
does not transfer tokens or represent a funded insurance policy.
