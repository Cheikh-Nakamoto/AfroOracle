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
| `publish(zone, produit, date, valueX100, confidence, source, consensusHash, aiReportHash)` | `onlyPublisher` | Requires nonempty market identity/source/hashes, positive price, and confidence ≤ 100; key = `keccak256(abi.encode(zone, produit, date))`; stamps `block.timestamp`; emits `Published(...)` |
| `getPrice(zone, produit, date)` | view, anyone | Returns `(valueX100, timestamp, confidence)` using the same tuple key |
| `getPriceRecord(zone, produit, date)` | view, anyone | Returns settlement fields, including `source`, `consensusHash`, and `aiReportHash` |
| `setRainfall(zone, date, valueX100)` | `onlyPublisher` | Key = `keccak256(abi.encode(zone, date))` |
| `getRainfall(zone, date)` | view, anyone | Returns `uint256` (0 when unset), using the same tuple key |

`onlyPublisher` reverts `"only publisher"` for anyone else. For the demo, the backend
calls `publish("Kaolack", …, 18700, …)` — value ×100, so 187 FCFA/kg → `18700`.

## `ParametricInsurance.sol` — demo consumer

```solidity
interface IAgri {
    function getPrice(string zone, string produit, string date)
        external view returns (uint256 valueX100, uint256 timestamp, uint256 confidence);
}
```

- `constructor(_oracle, _thresholdX100)` rejects a zero oracle address or zero
  threshold, then stores the oracle address, insurer (`msg.sender`), and insured floor
  price ×100.
- `check(zone, date)` (view) reads `oracle.getPrice(zone, "millet", date)` — the produit
  `"millet"` is hardcoded — and reverts with `price not published` when no record exists.
  It returns `0` when `price >= thresholdX100` (floor held, no indemnification), else
  `thresholdX100 - price` as the parametric payout in ×100 units.
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

This version uses unambiguous `abi.encode` storage keys and adds `getPriceRecord`.
If an older `AgriculturalData` instance was already deployed, deploy this version as a
new contract and update `CONTRACT_ADDRESS`; existing deployed contracts cannot be changed.

1. Set `POLYGON_RPC` to an Amoy RPC URL and `PRIVATE_KEY` to a dedicated test wallet
   funded with Amoy test POL. This deploys both contracts from that wallet.
2. Run `./scripts/deploy-amoy.sh`. It compiles Solidity **0.8.20**, verifies chain ID
   80002 before submitting, waits for both deployment receipts, and prints the contract
   addresses and transaction hashes as JSON.
3. The default insurance floor is `20000` (200 FCFA/kg); set
   `INSURANCE_THRESHOLD_X100` to change it. A price of 18700 yields a computed payout
   of 1300 (13 FCFA/kg in ×100 representation).
4. Copy the printed `agricultural_data_address` to `CONTRACT_ADDRESS` in the backend
   environment. The backend publisher wallet must be the same wallet that deployed
   `AgriculturalData`, because its constructor sets that wallet as `publisher`.
5. Run a validated consensus and inspect the successful transaction and `Published`
   event at `https://amoy.polygonscan.com`.
6. To read the insurance result, set `INSURANCE_CONTRACT_ADDRESS` to the printed
   insurance address and run `cd backend && go run ./cmd/check-insurance`. Set
   `ORACLE_DATE` if checking a publication date other than today UTC. The command is
   read-only and does not need `PRIVATE_KEY`.

The backend publisher wallet must be the contract's `publisher` address. The insurance
contract's `check` method is read-only: the current demo computes a payout amount but
does not transfer tokens or represent a funded insurance policy.
