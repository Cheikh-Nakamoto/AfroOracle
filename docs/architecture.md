# Architecture — AfriOracle

System overview, data flow, component responsibilities, storage model, failure modes.
All claims verified against `backend/*.go`, `contracts/*.sol`, `frontend/app/page.tsx`, `scripts/demo.sh`.

## Data flow (the one demo flow: millet price in Kaolack)

```
field agent (voice + photo)              automatic node (source hash)
 │ WhatsApp Cloud API / web mic fallback  │ POST /api/submit, tier 3, no photo
 ▼                                         ▼
transcribe() → extract() → proofScore() → Submission → runConsensus() → chainPublish() → AgriculturalData.publish
 (ai.go)        (ai.go)      (ai.go)        (main.go)    (consensus.go)    (main.go)         (Polygon Amoy)
                                                        │                                       │ getPrice
                                                        ▼                                       ▼
                                              Next.js dashboard ← /api/state        ParametricInsurance.check()
                                              assistant ← /api/ask
```

Step by step (`backend/main.go` + `backend/ai.go`):

1. **Submit** — `POST /api/submit` with `SubmitReq{node_id, tier, rep_score, value,
   photo_price?, channel?, transcript?, photo_b64?}`. Empty transcript falls back to
   `transcribe("")` (default French sentence). `extract()` prefers the typed `value`
   when non-zero, else parses the first number from the transcript, else returns
   `NeedsConfirmation` with question `"le kilo ou le sac ?"` (→ HTTP 200 clarify, nothing stored).
2. **Proof score** — tier 3 submits skip scoring (`ProofScore == nil`, always accepted).
   All others go through `proofScore(photo_price, value, photo_b64 bytes, fingerprints)`.
   Score < 50 marks the submission for exclusion at consensus time.
3. **Consensus** — `POST /api/consensus` runs `runConsensus(subs, 5.0)` over all stored
   submissions, computes median/confidence/outcomes, calls `chainPublish`, appends a
   `Publication`, and back-fills each submission's `status`/`reward`/`score_delta`.
4. **Publish** — `chainPublish` returns `("0x"+sha256, "simulated")` in dev; see Failure modes.
5. **Consume** — `ParametricInsurance.check(zone, date)` reads the price on-chain;
   `GET /api/ask` quotes the last publication; the dashboard renders `/api/state`.

AI involvement (never writes on-chain directly): transcription + extraction + slate
reading are real Meta models only when `META_API_KEY` is set; otherwise deterministic
mocks in `backend/ai.go`. The only on-chain trace of AI work is `aiReportHash`.

## Component responsibilities

| Component | File(s) | Owns |
| --- | --- | --- |
| AI adapters | `backend/ai.go` | `transcribe`, `extract`, `proofScore`, `reportHash`, `modelName` |
| Consensus | `backend/consensus.go` | `reputation`, `Submission{Weight,Accepted}`, `weightedMedian`, `runConsensus`, `contextAgent` (stub) |
| HTTP server + state | `backend/main.go` | routes, `processSubmit`, `chainPublish`, in-memory store (`submissions`, `publications`, `fingerprints`, `lastOutcomes`, mutex-guarded) |
| Settlement | `contracts/AgriculturalData.sol` | `Record`, `publish`, `getPrice`, `setRainfall`/`getRainfall`, `onlyPublisher` |
| Consumer | `contracts/ParametricInsurance.sol` | `check(zone, date)` payout vs `thresholdX100` |
| Dashboard | `frontend/app/page.tsx` | submissions table, consensus block, assistant answer; `API = "http://localhost:8000"` |
| E2E demo | `scripts/demo.sh` | 9 submits → consensus → ask, with assertions |

## Storage model (hybrid)

| Data | Store | Notes |
| --- | --- | --- |
| Published price, median, confidence, `aiReportHash` | On-chain (Polygon Amoy) | Settlement only; `Record{valueX100,timestamp,confidence,source,consensusHash,aiReportHash}` |
| Voice notes, slate photos (proof media) | Off-chain, content-addressed | Only the sha256 fingerprint / hash goes on-chain; media bytes are never stored on-chain |
| Submissions, publications, rewards, fingerprints | PostgreSQL (`DATABASE_URL`) | Spec target; **current code uses an in-memory store** (mock path) — Postgres is not wired yet |
| Historical price feed | PostgreSQL timestamp column | v2 option per spec (TimescaleDB); not implemented |

Rules honored: no media bytes on-chain; on-chain storage is final price + proof hashes;
`reportHash` (`"0x"+sha256` of canonical JSON) is the tamper-evidence link.

## Failure modes (known, in code)

- **No reset endpoint.** Submissions accumulate across runs on a reused server; `demo.sh`
  assertions tolerate this, but day-of demos should start a fresh backend.
- **`chainPublish` is a stub.** With `POLYGON_RPC`/`PRIVATE_KEY` set it labels the tx
  `"polygon-amoy"` but still returns `reportHash(report)` — the real `go-ethereum`
  call to `AgriculturalData.publish` is a `TODO` in `backend/main.go`, and `go.mod`
  has no external dependency yet.
- **`contextAgent` is a stub.** Returns `""` with no outliers, else a fixed French note;
  no web search.
- **WhatsApp ingress is text-only.** `POST /webhook/whatsapp` parses
  `entry[].changes[].value.messages[].text.body`; media download + reply are not
  implemented, and the resulting tier-0 submission has weight 0 (missing tier key).
- **`value: 0` is unsubmittable as typed.** `processSubmit` treats `Value == 0` as
  "no typed value" and falls back to transcript parsing.
- **Single-process state.** The in-memory store is lost on restart and not shared
  between instances; fine for the demo, not for production.
