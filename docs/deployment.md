# Deployment — AfriOracle

Build/run/test/deploy, verified against `.env.example`, `backend/go.mod`,
`frontend/package.json`, `scripts/demo.sh`, and `scripts/run-pipeline.sh`.

## Environment variables (see `.env.example`)

| Var | Purpose | Unset behavior |
| --- | --- | --- |
| `META_API_KEY` | Meta AI models (Muse Spark, Voice Transcribe, SAM 3.1) | Deterministic mock path (dev default) |
| `POLYGON_RPC` / `PRIVATE_KEY` / `CONTRACT_ADDRESS` | Live Polygon Amoy publish (chain ID 80002 required) | Simulated `"0x"+sha256` tx, `chain: "simulated"` |
| `ORACLE_ZONE` / `ORACLE_PRODUIT` | Publication keys passed to `AgriculturalData.publish` | `Kaolack` / `millet` |
| `INSURANCE_THRESHOLD_X100` | Parametric insurance floor passed to deployment script | `20000` (200 FCFA/kg) |
| `DATABASE_URL` | PostgreSQL DSN | In-memory mock store (current code has no Postgres wiring) |
| `WHATSAPP_TOKEN` / `WHATSAPP_VERIFY_TOKEN` | WhatsApp Cloud API | Webhook handshake fails closed (403) |
| `PORT` | Backend port (read in `backend/main.go`) | `8000` |

Copy to `.env` for local runs; `PRIVATE_KEY` must be a dedicated test wallet funded
only with Amoy test POL. Never commit it. A configured publisher refuses to send on a
chain whose ID is not 80002.

## Backend (module `afrioracle`, Go; `go 1.22`)

```bash
cd backend && go build -o /tmp/afrioracle . && PORT=8000 ./afrioracle
# or for iteration: cd backend && go run .
```

The backend uses `go-ethereum` for the receipt-checked `AgriculturalData.publish` call.

## Frontend (Next.js 14.2.5, React 18)

```bash
cd frontend && npm i && npm run dev
```

`frontend/app/page.tsx` calls a hardcoded `API = "http://localhost:8000"`, so the
backend must run on port 8000. Buttons: "Lancer le consensus" (`POST /api/consensus`,
shows the 422 error inline when sources are short) and "Interroger l'assistant"
(`GET /api/ask`); fraud rows (proof < 50) render red.

## End-to-end demo

```bash
./scripts/demo.sh            # from the repo root; BASE default http://localhost:8000
BASE=http://localhost:8001 ./scripts/demo.sh
```

Starts its own backend (built binary, mock AI path) if nothing answers at `$BASE`,
then submits 8 honest agents + FAO + fraudster D, runs consensus, and asks the
assistant. Asserts median 187, confidence 100, all D outcomes `excluded`, and the
answer contains 187 without estimating. Temp files: `/tmp/afrioracle-demo-bin`,
`/tmp/afrioracle-demo.log`, `/tmp/afrioracle-{submit,consensus}.json`.

## Tests

```bash
cd backend && go test -count=1 .
```

Covers the Kaolack golden case (median 187, conf 100, rewards A=6.5 B=5 C=5 FAO=0.5
D=0, D excluded with delta −2), the automatic-node weight (0.7), and the proof
threshold boundary (49 rejected / 50 accepted).

## Live Amoy run (demo day)

1. Fund a dedicated test wallet with Amoy test POL. Set `POLYGON_RPC` and `PRIVATE_KEY`
   in the shell environment; leave the private key out of source control.
2. Run `./scripts/deploy-amoy.sh`. It compiles the contracts with solc 0.8.20, refuses
   any chain ID other than 80002, deploys both contracts, and prints addresses and tx
   hashes. Copy `agricultural_data_address` to `CONTRACT_ADDRESS`.
3. Keep the same publisher key for backend writes: `AgriculturalData` assigns its
   deployer as the only publisher. Set `ORACLE_ZONE` / `ORACLE_PRODUIT` if changing
   the demo's defaults.
4. Start from a **fresh backend** for the demo, run `./scripts/demo.sh`, and trigger a
   validated consensus. Open the returned tx at `https://amoy.polygonscan.com`.

The automated chain tests use go-ethereum's in-process simulated EVM, including the
oracle publication receipt and insurance payout calculation. They do not deploy
contracts to public Amoy or validate a live RPC/wallet.

## Pipeline reruns

```bash
./scripts/run-pipeline.sh            # full worker run from agent 00
FROM=04 ./scripts/run-pipeline.sh    # resume from agent 04
```

Knobs: `CLAUDE_BIN` (default `claude`), `ALLOWED_TOOLS`, `VERBOSE=1`.
