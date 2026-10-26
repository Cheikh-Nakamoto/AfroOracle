# AfriOracle

Decentralized oracle for African market data — demo: the price of millet in Kaolack, from a field agent's voice note to a parametric insurance contract reading the published value.

## Documentation

| Guide | Contents |
| --- | --- |
| [`docs/architecture.md`](docs/architecture.md) | System overview, data flow, component responsibilities, storage model, failure modes |
| [`docs/consensus.md`](docs/consensus.md) | Weighted median, tier weights, reputation grid, proof thresholds, worked golden example |
| [`docs/api.md`](docs/api.md) | Complete endpoint reference with request/response field names |
| [`docs/contracts.md`](docs/contracts.md) | `AgriculturalData` + `ParametricInsurance`, ABI, Amoy deploy notes |
| [`docs/deployment.md`](docs/deployment.md) | Env vars, build/run, demo script, tests, live Amoy steps |
| [`docs/demo.md`](docs/demo.md) | 3-min demo walkthrough, golden numbers, day-of shoot list |

## Architecture

```
 field agent (voice/photo)          automatic node (source hash)
        │ WhatsApp / web mic             │ API
        ▼                                ▼
 ┌─────────────┐  transcript   ┌──────────────────┐
 │ Muse Voice  │──────────────▶│ Muse Spark 1.3   │── JSON {produit, prix, marché}
 │ Transcribe  │               │ extraction       │
 └─────────────┘               └──────────────────┘
        photo                         │
        ▼                             ▼
 ┌──────────────────┐  score   ┌──────────────────┐   median    ┌───────────────────┐
 │ Muse Spark slate │─────────▶│ Go backend       │────────────▶│ AgriculturalData  │
 │ + SAM 3.1 + hash │  <50 out │ weighted median  │  publish    │ (Polygon Amoy)    │
 └──────────────────┘          │ + context agent  │             └───────────────────┘
                               └──────────────────┘                      │ getPrice
                                        │ assistant (/api/ask)          ▼
                                        ▼                    ┌───────────────────┐
                               ┌──────────────────┐          │ ParametricInsurance│
                               │ Next.js dashboard│          │ .check(zone, date) │
                               └──────────────────┘          └───────────────────┘
```

| Component | Code | Notes |
| --- | --- | --- |
| Go backend (`net/http`, stdlib only) | `backend/main.go`, `backend/ai.go`, `backend/consensus.go` | Mock AI when `META_API_KEY` unset; CORS `*` on `/api/*` |
| Solidity contracts (0.8.20, Polygon Amoy) | `contracts/AgriculturalData.sol`, `contracts/ParametricInsurance.sol`, `contracts/abi.json` | Settlement only: `valueX100`, confidence, `aiReportHash` |
| Next.js dashboard | `frontend/app/page.tsx` | Calls `http://localhost:8000`; shows submissions, proof scores, median, tx, rewards, assistant answer |
| WhatsApp webhook | `POST/GET /webhook/whatsapp` in `backend/main.go` | Cloud API ingress; mic/camera web page is the fallback |
| Hybrid storage | on-chain settlement / off-chain content-addressed proof media / PostgreSQL operational store | `DATABASE_URL` unset → in-memory mock (dev); real Postgres only for the demo run |

## Quickstart

Env vars (see `.env.example`):

| Var | Purpose |
| --- | --- |
| `META_API_KEY` | Meta AI models (unset = deterministic mock path) |
| `POLYGON_RPC` / `PRIVATE_KEY` | Live Amoy publish (unset = simulated `"0x"+sha256` tx) |
| `DATABASE_URL` | PostgreSQL DSN (unset = in-memory mock) |
| `WHATSAPP_TOKEN` / `WHATSAPP_VERIFY_TOKEN` | WhatsApp Cloud API |
| `PORT` | Backend port (default `8000`) |

Backend (module `afrioracle`, Go):

```bash
cd backend && go build -o /tmp/afrioracle . && PORT=8000 ./afrioracle
# or: cd backend && go run .
```

Frontend (Next.js 14):

```bash
cd frontend && npm i && npm run dev
```

End-to-end demo (starts its own backend if none answers at `$BASE`, default `http://localhost:8000`):

```bash
./scripts/demo.sh
```

Tests:

```bash
cd backend && go test -count=1 .
```

## API surface

| Method + path | Purpose |
| --- | --- |
| `POST /api/submit` | Submit an observation (`node_id`, `tier`, `rep_score`, `value`, `photo_price?`, `channel?`, `transcript?`, `photo_b64?`) → `201 {status, submission, proof}` or `200 {status:"clarify", question}` |
| `POST /api/consensus` | Run weighted median → `200 {median, confidence, validated, outcomes, tx, chain, value_x100, aiReportHash}`, or `422` under 3 independent submissions |
| `GET /api/state` | Inspect `{submissions, publications, rewards, fingerprints}` |
| `GET /api/ask?q=...` | Assistant answer from published data only; never estimates |
| `POST /webhook/whatsapp` | Meta Cloud API message ingress |
| `GET /webhook/whatsapp` | Verification handshake (`hub.verify_token` vs `WHATSAPP_VERIFY_TOKEN`) |
| `GET /` | Placeholder HTML (the real dashboard is the Next.js build) |

## Consensus model

- **Weighted median** over accepted submissions (`backend/consensus.go`: `weightedMedian`, `runConsensus`).
- **Tier weights**: T1 = 3.0, T2 = 1.0, T3 = 0.7. Weight = tier × reputation multiplier.
- **Reputation grid**: 90+ Expert (2.0, 1.8) · 75+ Confirmé (1.5, 1.3) · 60+ Actif (1.0, 1.0) · 40+ Débutant (0.5, 0.5) · else Non qualifié (0, 0).
- **Proof-score exclusion**: score < 50 → excluded, reward 0, `ScoreDelta` −2. Duplicate photo → score 12; unreadable slate → 40; else gap ≤2% → 94, ≤10% → 70, else 20.
- **Automatic nodes** (`ProofScore == nil`, tier 3) are always accepted at tier weight 0.7.
- **Collecte rule**: consensus returns 422 with fewer than 3 accepted submissions.
- **Confidence**: weight share within 15% of the median; validated when ≥ 66.

## Demo numbers (golden case)

Kaolack millet run in `scripts/demo.sh`: **median 187, confidence 100**, rewards **A=6.5 B=5 C=5 FAO=0.5 D=0** — fraudster D reuses A's photo (score 12 → excluded, reward 0, delta −2). `chainPublish` reports `chain: "simulated"` in dev vs `"polygon-amoy"` live. The assistant quotes the published 187 and never estimates; with no publication it says so.

## Demo-day TODO

- Set `META_API_KEY` + `POLYGON_RPC`/`PRIVATE_KEY` for real transcription and a live Amoy tx in beat 4 (dev shows `chain: simulated`).
- Render the final ~3-min mp4 and drop it under `demo/` (see `demo/script.md`, `demo/capture-list.md`, `demo/montage.md`).
