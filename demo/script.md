# AfriOracle — 3-minute demo video script

Product: millet price in Kaolack, from a field agent's voice note to a
parametric insurance contract reading the published value on-chain.
Prerequisite: `./scripts/demo.sh` passes (median 187, confidence 100,
fraudster D excluded, assistant quotes 187). Capture at 1080p+; keep
on-screen text legible. This script feeds agent 07's `demo/capture-list.md`.

## Beat 1 — 0:00–0:30 · The problem [LIVE]

- **Shot 1a** [LIVE]: b-roll of a real market stall (millet sacks, price
  slate, crowd). A staged stall works for a hackathon.
- **Voiceover**: "An insurer cannot indemnify a farmer without a trustworthy
  local price. African markets have no reliable, auditable price data."
- **On screen**: market scene, then title card "AfriOracle — le prix du mil
  à Kaolack, vérifié et publié on-chain".
- **Capture need**: camera footage, natural sound, 10–15 s usable.

## Beat 2 — 0:30–1:00 · The agent submits [LIVE] + [SCREEN]

- **Shot 2a** [LIVE]: agent filmed on the market sends a WhatsApp voice note
  ("Le prix du mil à Kaolack est de 187 francs CFA le kilo") plus a photo of
  the price slate.
- **Shot 2b** [SCREEN]: the WhatsApp chat — voice note + slate photo going
  out. Fallback per spec: the mic/camera web page if the Cloud API number is
  not validated in time.
- **Voiceover**: "Huit agents simulés complètent les vraies soumissions —
  et un fraudeur scénarisé rejoue une photo déjà vue."
- **Capture need**: phone screen recording of the WhatsApp send; camera
  footage of the agent. In `demo.sh` this is the 9× `POST /api/submit`
  sequence (A…H + FAO + fraudster D reusing A's photo).

## Beat 3 — 1:00–1:45 · Proof on the dashboard [SCREEN]

- **Shot 3** [SCREEN]: Next.js dashboard (`frontend/`, backed by
  `GET /api/state`) loading live — submissions with node, value, proof
  score; transcription of the voice note; then the fraud row (D, score 12)
  highlighted red.
- **Voiceover**: "Muse Spark lit l'ardoise, SAM 3.1 détecte les objets, le
  serveur compare l'empreinte : la photo réutilisée tombe à 12 et D est
  exclu, récompense zéro."
- **Capture need**: desktop-browser recording (not a phone) so scores stay
  legible. Zoom on the red fraud row.

## Beat 4 — 1:45–2:15 · Publication [SCREEN]

- **Shot 4a** [SCREEN]: click "Lancer le consensus" (`POST /api/consensus`)
  → median **187 FCFA/kg**, confidence **100**, rewards
  (A=6.5, B=C=E=F=G=H=5, FAO=0.5, D=0), tx hash + `aiReportHash`.
- **Shot 4b** [SCREEN]: the transaction opened in the Polygon Amoy explorer
  (`https://amoy.polygonscan.com`). Dev runs show `chain: simulated`; shoot
  this beat on the real demo run (`META_API_KEY`, `POLYGON_RPC`,
  `PRIVATE_KEY` set) so the explorer page is live.
- **Voiceover**: "La médiane pondérée est publiée : 187, confiance 100 —
  la source de vérité est on-chain, la preuve reste hors-chaîne."
- **Capture need**: dashboard recording of the click + result; explorer tab.

## Beat 5 — 2:15–2:45 · Insurance + assistant [SCREEN]

- **Shot 5a** [SCREEN]: `ParametricInsurance.check(zone, date)` reading the
  published 187 (`contracts/` on Amoy; event/log on screen).
- **Shot 5b** [SCREEN]: a farmer asks the WhatsApp assistant
  ("prix du mil à Kaolack ?") and gets
  "Prix du mil à Kaolack : 187 FCFA/kg, publié…, confiance 100" —
  quoted from published data, never estimated. Same answer is served by
  `GET /api/ask` in the demo run.
- **Voiceover**: "Le contrat d'assurance lit la valeur ; l'assistant ne
  répond qu'avec des données publiées — jamais d'estimation."
- **Capture need**: contract-call screen + WhatsApp assistant chat.

## Beat 6 — 2:45–3:00 · The cost [SCREEN]

- **Shot 6** [SCREEN]: one static cost card built from `our_projetc.md`
  §Coût de la vérification IA: Transcribe 30 s → $0.0015, Spark
  (4000 in + 500 out) → $0.007, SAM 1 photo → $0.0025,
  **total ≈ $0.011 per submission** — under 1 % of a base reward.
- **Voiceover**: "Environ un centime de vérification par soumission."
- **Capture need**: static graphic; hold 5 s over the closing title.

## Shoot order (cheapest first)

1. Beats 3–6 screens (reproducible any time via `demo.sh`).
2. Beat 2 screen (WhatsApp send / web fallback page).
3. Beats 1–2 live (one market visit covers both).
