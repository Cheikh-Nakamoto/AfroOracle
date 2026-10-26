# Demo walkthrough — AfriOracle (3 minutes)

The 3-minute video flow, per `demo/script.md` and `demo/capture-list.md`.
Prerequisite for every screen beat: `./scripts/demo.sh` passes
(median **187**, confidence **100**, fraudster D excluded, assistant quotes 187).
Figures below are the golden numbers the script asserts.

## The six beats

| # | Time | Kind | What it shows |
| --- | --- | --- | --- |
| 1 | 0:00 Problem | [LIVE] | Market b-roll + voiceover: an insurer cannot indemnify a farmer without a trustworthy local price |
| 2 | 0:30 Agent submits | [LIVE]+[SCREEN] | Agent sends WhatsApp voice note ("Le prix du mil à Kaolack est de 187 francs CFA le kilo") + slate photo; chat screen recording (mic/camera web page is the fallback) |
| 3 | 1:00 Proof on dashboard | [SCREEN] | Dashboard (`GET /api/state`): submissions with proof scores; fraud row **D, score 12, red** — reused photo, reward 0 |
| 4 | 1:45 Publication | [SCREEN] | Click "Lancer le consensus" → median **187 FCFA/kg**, confidence **100**, rewards A=6.5 B=5 C=5 (+E–H=5 each) FAO=0.5 D=0, tx hash + `aiReportHash`; then the tx in the Amoy explorer |
| 5 | 2:15 Insurance + assistant | [SCREEN] | `ParametricInsurance.check(zone, date)` reads 187 (no payout — price holds the floor); farmer asks "prix du mil à Kaolack ?" → "187 FCFA/kg, publié…, confiance 100", quoted, never estimated |
| 6 | 2:45 Cost | [SCREEN] | Static card from `our_projetc.md`: Transcribe 30 s $0.0015 + Spark $0.007 + SAM $0.0025 = **≈ $0.011/submission** (< 1% of base reward) |

## Golden numbers (what "pass" looks like)

- `POST /api/consensus` → `median 187`, `confidence 100`, `validated true`,
  `chain "simulated"` in dev, `value_x100 18700`.
- Every `D` outcome: `Status "excluded"`, `Reward 0`, `ScoreDelta −2`.
- `GET /api/ask` answer contains `187` and no estimation language.

## Remaining day-of shoot list

From `demo/capture-list.md` + `demo/montage.md` (status at time of writing):

- [ ] Beats 1–2 [LIVE]: real market b-roll + agent phone footage (stills only: `demo/assets/beat1-title.svg`, `beat2a-live.svg`).
- [ ] Beats 2b–5 video: screen recordings (stills with real data already in `demo/assets/`; `raw/{state,consensus,ask}.json` hold the verified snapshots).
- [ ] Beat 4 explorer: live Amoy tx page — requires the finished `go-ethereum` wiring + `POLYGON_RPC`/`PRIVATE_KEY` (see `docs/deployment.md`).
- [ ] Render the final ~3-min mp4 per `demo/montage.md` (order/timings/voiceover) and drop it under `demo/`; record the path in `state/PROJECT_STATE.md`.

Shoot order (cheapest first): beats 3–6 screens → beat 2 screen → beats 1–2 live.
Record at 1080p+ in a desktop browser so scores and hashes stay legible.
