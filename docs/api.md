# API reference — AfriOracle backend

Complete endpoint reference, derived field-by-field from `backend/main.go`
(structs `SubmitReq`, `StateSub`, `Publication`, `RewardEntry`, `Outcome`).
CORS (`Access-Control-Allow-Origin: *`, methods `GET, POST, OPTIONS`, header
`Content-Type`; `OPTIONS` → 204) applies to `/api/*` only — not to
`/webhook/whatsapp` or `/`.

## `POST /api/submit`

Submit one observation. Non-POST → 405. Malformed JSON → 400.

Request body (`SubmitReq` json tags):

| Field | Type | Notes |
| --- | --- | --- |
| `node_id` | string | Agent/node identifier |
| `tier` | int | 1/2/3; tier 3 = automatic node (skips proof scoring) |
| `rep_score` | int | 0–100 reputation score |
| `value` | float64 | Typed price; `0` means "absent" (falls back to transcript parsing) |
| `photo_price` | number \| null | Price read off the slate; `null`/absent → score 40 |
| `channel` | string | e.g. `"app"`, `"whatsapp"`, `"api"` |
| `transcript` | string | Voice-note transcript; empty → `transcribe("")` default sentence |
| `photo_b64` | string | Base64 photo bytes (decode errors ignored → empty bytes) |

Responses:

- `200 {status: "clarify", question}` — no usable price (question is
  `"le kilo ou le sac ?"`). Nothing is stored.
- `201 {status: "accepted", submission, proof}` — `submission` uses **Go field names**
  (`NodeID, Tier, RepScore, Value, ProofScore, Channel` — `Submission` has no json tags);
  `proof` uses Go names (`Score, SlatePrice, DuplicatePhoto, ObjectsFound, Model`).
  The stored record starts with `status: "pending"` until consensus runs.

## `POST /api/consensus`

Run the weighted median over all stored submissions. Non-POST → 405.

- `< 3` accepted, weight > 0 submissions → **422**
  `{error: "collecte rule: minimum 3 independent submissions required"}`.
- Else `200`:

| Field | Notes |
| --- | --- |
| `median` | Weighted median (float) |
| `confidence` | 0–100 weight share within 15% |
| `validated` | `confidence >= 66` |
| `outcomes` | Array with Go field names: `NodeID, Status, Note, Reward, ScoreDelta, Deviation, Weight` |
| `tx` | `"0x"+sha256` tx hash (simulated or Amoy label — see below) |
| `chain` | `"simulated"` (dev) or `"polygon-amoy"` (live keys set) |
| `value_x100` | `int64(median*100 + 0.5)` — the on-chain integer |
| `aiReportHash` | `"0x"+sha256` of `{median, confidence, outcomes: count}` |

Side effects: appends a publication, back-fills every submission's status/reward/score
delta matched by `NodeID`. No request body is read.

## `GET /api/state`

Inspect everything. `200`:

| Field | Shape |
| --- | --- |
| `submissions` | `StateSub` with json tags: `node_id, tier, rep_score, value, proof_score, channel, status, reward, score_delta` |
| `publications` | `median, confidence, validated, tx, chain, value_x100, aiReportHash, timestamp` (unix) |
| `rewards` | From last consensus: `node_id, reward, score_delta` |
| `fingerprints` | Array of photo sha256 hex strings |

## `GET /api/ask?q=...`

Assistant answer, oracle data only, never estimates. `200 {answer}`:

- No publication yet → `"Aucune donnée publiée pour le moment. Je n'estime jamais un prix moi-même."`
- Else → `"Prix du mil à Kaolack : {median} FCFA/kg, publié le {HH:MM}, {acceptedCount} sources, confiance {conf}. Tx : {tx}"`
  where `acceptedCount` counts stored submissions with `status == "accepted"`.
  The `q` parameter is accepted but does not change the answer.

## `GET /webhook/whatsapp` — verification handshake

- `hub.verify_token == WHATSAPP_VERIFY_TOKEN` → returns `hub.challenge` verbatim (200).
- Mismatch → 403 `"verification failed"`.

## `POST /webhook/whatsapp` — message ingress (text-only)

Parses Cloud-API-shaped JSON at `entry[].changes[].value.messages[].text.body`
(other methods → 405; empty body → bare 200). A non-empty text body is submitted as
`SubmitReq{Transcript: body, Channel: "whatsapp"}` and the `processSubmit` result is
returned directly (201 accepted or 200 clarify). Media download and replies are not
implemented; the created submission has tier 0 → weight 0.

## `GET /` — placeholder

`text/html`: `"AfriOracle dashboard — See frontend/ for the Next.js dashboard."`
The real dashboard is the Next.js build (`frontend/`), which calls the API above.
