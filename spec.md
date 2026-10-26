# AfriOracle — Demo Implementation Spec

Single source of truth for the hackathon prototype. The full protocol is in `our_projetc.md`; this file scopes it to the one demo flow and pins the interfaces.

## Problem Statement

An insurer cannot indemnify a farmer without a trustworthy local price. African markets lack reliable, auditable price data. AfriOracle publishes a consensus price on-chain: field agents submit observations, AI verifies the proof, a weighted median is published, and a consumer contract reads it.

## Solution (demo scope)

One flow, end to end: **the price of millet in Kaolack**, from a field agent's voice note to a parametric insurance contract reading the published value.

## Out of scope (not built)

AFRO token, DAO governance, land registry, company valuation, full reward distribution, production deployment, Wolof transcription, mobile app. Documented in `our_projetc.md`, not implemented here.

## Demo components

1. **Input** — WhatsApp Cloud API, with a mic/camera web page as fallback.
2. **Transcription + extraction** — Muse Voice Transcribe, then Muse Spark → JSON.
3. **Proof score** — Muse Spark reads the price slate, SAM 3.1 detects expected objects, server checks photo fingerprint.
4. **Consensus** — Go weighted median + context agent (web search).
5. **Chain** — `AgriculturalData.sol` + `ParametricInsurance.sol` on Polygon Amoy.
6. **Output** — dashboard (Next.js) + WhatsApp assistant.
7. **Video** — live footage + screen captures assembled per [`demo/capture-list.md`](demo/capture-list.md).

## Stack

- **Backend**: Go, package `main` in `backend/`, module `afrioracle`. Stdlib `net/http`, `encoding/json`, `crypto/sha256`; one external dep `github.com/ethereum/go-ethereum` for the chain. No web framework.
- **Contracts**: Solidity 0.8.20, Polygon Amoy.
- **Frontend**: Next.js (React + TypeScript) in `frontend/`.

## Storage (hybrid: on-chain + off-chain)

Data is classified by trust requirement, not stored in one place:

| Data | Store | Notes |
| --- | --- | --- |
| Published price, median, confidence, `aiReportHash` per submission | On-chain (Polygon Amoy) | Settlement only — the immutable source of truth |
| Voice notes, price-slate photos (proof media) | Off-chain, content-addressed | Hash the bytes; only the hash/CID goes on-chain |
| Submissions, nodes, reputation, consensus runs | PostgreSQL (`DATABASE_URL`) | Backend's operational read/aggregation store |
| Historical price feed | PostgreSQL (timestamp column) | Time-series; TimescaleDB is a v2 option |

Rules:

- Never store media bytes, raw submissions, or aggregation on-chain (SSTORE is expensive).
- On-chain storage is minimal: final price + proof hashes only.
- `reportHash` is the tamper-evidence link: off-chain content is verified against the on-chain hash.
- Development: if `DATABASE_URL` is unset, the server uses an in-memory store (mock path). Real Postgres only for the demo run.

## Interfaces (the seams)

Structs carry `json` tags matching the HTTP shapes (`node_id`, `rep_score`, `proof_score`, `value_x100`, `aiReportHash`, …).

### `backend/ai.go` — AI adapters (mock when no key)

```go
var mock = os.Getenv("META_API_KEY") == ""

type ExtractResult struct {
    Produit           string
    Prix              *float64
    Unite             string
    Marche            string
    NeedsConfirmation bool
    Question          string
}

type ProofResult struct {
    Score          int
    SlatePrice     *float64
    DuplicatePhoto bool
    ObjectsFound   []string
    Model          string
}

func transcribe(audioHint string) string
func extract(transcript string, typedValue *float64) ExtractResult
func proofScore(photoPrice *float64, submitted float64, photoBytes []byte, seen map[string]bool) ProofResult
func reportHash(payload any) string   // "0x" + sha256
```

Mock rules: duplicate photo → score 12; unreadable slate (`photoPrice == nil`) → score 40; else gap ≤2% → 94, ≤10% → 70, else 20.

### `backend/consensus.go` — consensus + reputation

```go
var tierWeight = map[int]float64{1: 3.0, 2: 1.0, 3: 0.7}

func reputation(score int) (level string, consensusMult, rewardMult float64)
// 90+ Expert (2.0, 1.8); 75+ Confirmé (1.5, 1.3); 60+ Actif (1.0, 1.0); 40+ Débutant (0.5, 0.5); else Non qualifié (0,0)

type Submission struct {
    NodeID     string
    Tier       int
    RepScore   int
    Value      float64
    ProofScore *int   // nil = automatic node (source hash) -> always accepted
    Channel    string
}
func (s Submission) Weight() float64
func (s Submission) Accepted() bool

type Outcome struct {
    NodeID, Status, Note string
    Reward, ScoreDelta, Deviation, Weight float64
}
type ConsensusResult struct { Median float64; Confidence int; Validated bool; Outcomes []Outcome }

func weightedMedian(subs []Submission) float64   // caller guarantees >=1 accepted
func runConsensus(subs []Submission, baseReward float64) ConsensusResult
```

### `backend/main.go` — HTTP server (net/http)

```
POST /api/submit        {node_id, tier, rep_score, value, photo_price?, channel?, transcript?, photo_b64?}
                        -> 201 {status, submission, proof} | 200 {status:"clarify", question}
POST /api/consensus     -> 200 {median, confidence, validated, outcomes, tx, chain, value_x100, aiReportHash}
                           | 422 if < 3 sources
POST /webhook/whatsapp  Meta Cloud API (media download + reply)
GET  /webhook/whatsapp  verification handshake
GET  /api/state         -> {submissions, publications, rewards, fingerprints}
GET  /api/ask?q=...     -> assistant answer (oracle data only, never estimates)
GET  /                  -> dashboard (Next.js build)
```

- CORS: set `Access-Control-Allow-Origin: *` on `/api/*` so the Next.js dev server can call it.
- `chainPublish(result, report)`: simulated tx (`"0x" + sha256`) when `POLYGON_RPC`/`PRIVATE_KEY` unset; otherwise `go-ethereum` call to `AgriculturalData.publish`.

### Contracts (Polygon Amoy)

```solidity
// AgriculturalData — value stored x100
publish(zone, produit, date, valueX100, confidence, source, consensusHash, aiReportHash)
getPrice(zone, produit, date) -> (valueX100, timestamp, confidence)
getRainfall(zone, date)

// ParametricInsurance
check(zone, date) -> payout   // price > thresholdX100
```

## Acceptance criteria (done-criteria)

- Collecte rule enforced: consensus returns 422 under 3 independent submissions.
- Automatic node (`ProofScore == nil`) accepted at tier weight 0.7.
- Proof score < 50 → excluded, reward 0, ScoreDelta −2.
- Reused photo → score 12 → excluded (fraud path).
- Doc example reproduces exactly: median **187**, confidence **100**, rewards A=6.5 B=5 C=5 FAO=0.5 D=0.
- `chainPublish` labels simulated tx vs real chain.
- Assistant answers only from published data; if none, says so and never estimates.
- Dashboard renders submissions, proof scores, median, tx, rewards, assistant answer.
- `demo/capture-list.md` maps each of the 6 video beats to a concrete asset (live footage or screen capture) with what is on screen.

## Testing decisions

- Test at the public interfaces above (the seams). No private-function tests.
- Golden case: the doc's Kaolack example, in `backend/consensus_test.go` via `go test .`.
- Expected values come from the doc, not recomputed from the code.
