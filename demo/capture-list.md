# Demo Capture List — AfriOracle (3-min video)

Shot-by-shot plan for the montage. Two asset kinds: **[LIVE]** camera footage, **[SCREEN]** screen/device recording. Capture at 1080p+; keep on-screen text legible (zoom where needed). Raw clips go in `demo/assets/`.

| # | Beat (time) | Asset | What to capture | On screen |
| --- | --- | --- | --- | --- |
| 1 | 0:00 Problem | [LIVE] | B-roll of a market stall; voiceover "no reliable local price" | market scene |
| 2 | 0:30 Agent submits | [LIVE] | Agent on the market filming: sends a voice note + photo on WhatsApp | agent's phone, market |
| 2 | 0:30 (cut) | [SCREEN] | WhatsApp: voice note + photo of the price slate being sent | WhatsApp chat |
| 3 | 1:00 Dashboard | [SCREEN] | Dashboard loading `/api/state`: transcription, proof score, fraud row highlighted red | dashboard |
| 4 | 1:45 Publication | [SCREEN] | Click "Lancer le consensus" → median, confidence, tx hash; then the Polygon Amoy explorer tx page | dashboard + explorer |
| 5 | 2:15 Insurance + assistant | [SCREEN] | Assistant answer on WhatsApp ("187 FCFA/kg, publié, confiance 100"); optional: contract `check` event | WhatsApp assistant |
| 6 | 2:45 Cost | [SCREEN] | Cost card: ≈$0.011/submission breakdown (transcribe + Spark + SAM) | cost graphic |

## Capture notes

- **Beat 2 live**: film the agent on a real market (the doc's "un vrai marché"). A staged stall works for a hackathon.
- **Beats 3–4 screen**: record the dashboard in a desktop browser, not a phone, so proof scores and the tx hash stay legible.
- **Beat 4 explorer**: Amoy testnet — open the published tx at `https://amoy.polygonscan.com`.
- **Beat 6**: render the doc's "Coût de la vérification IA" table (`our_projetc.md` §Coût) as one static card.

## Montage order (3 min)

1. Problem (0:00–0:30) → 2. Agent submits (0:30–1:00) → 3. Dashboard proof (1:00–1:45) → 4. Publication + explorer (1:45–2:15) → 5. Insurance + assistant (2:15–2:45) → 6. Cost (2:45–3:00).
