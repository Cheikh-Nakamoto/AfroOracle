# AfriOracle — Montage du demo 3 min (edit-project)

Fichier de montage : ordre, durées, assets, voix off. Durée totale ≈ 3:00.
Stills SVG 1080p dans `demo/assets/` ; snapshots API bruts dans `demo/assets/raw/`
(`state.json`, `consensus.json`, `ask.json` — valeurs réelles de `./scripts/demo.sh` :
médiane 187, confiance 100, D exclu score 12, tx `0x6783…cb7d25` [simulated]).

Statut : **stills écrans = données réelles capturées** ; [LIVE] = placeholders,
b-roll marché à tourner ; enregistrements vidéo d'écran + run Amoy réel à shooter
(`META_API_KEY`, `POLYGON_RPC`, `PRIVATE_KEY`).

| # | Time | Asset | Voix off / note |
| --- | --- | --- | --- |
| 1 | 0:00–0:30 | `assets/beat1-title.svg` (+ b-roll marché à tourner) | « Un assureur ne peut indemniser sans prix local fiable. » Titre « le prix du mil à Kaolack, vérifié et publié on-chain ». |
| 2 | 0:30–1:00 | `assets/beat2a-live.svg` (placeholder) → `assets/beat2b-whatsapp.svg` | Agent envoie note vocale + photo ardoise ; 8 agents honnêtes + FAO, D rejoue la photo de A. |
| 3 | 1:00–1:45 | `assets/beat3-dashboard.svg` (zoom rangée D rouge) | Spark lit l'ardoise, SAM détecte, empreinte comparée : D tombe à 12, exclu, récompense 0. |
| 4 | 1:45–2:15 | `assets/beat4-publication.svg` → onglet `https://amoy.polygonscan.com` (run réel) | « Médiane pondérée 187, confiance 100 — vérité on-chain, preuve hors-chaîne. » Récompenses A=6.5, B–H=5, FAO=0.5, D=0. |
| 5 | 2:15–2:45 | `assets/beat5-assistant.svg` (+ event `check` contrat, run réel) | Contrat lit 187 ; l'assistant ne cite que le publié, jamais d'estimation. |
| 6 | 2:45–3:00 | `assets/beat6-cost.svg` (fixe 5 s + titre de fin) | « Environ un centime de vérification par soumission. » Total ≈ $0.011. |

## Reste à shooter (jour démo)

1. Beats 1–2a : une visite au marché (étals + agent qui envoie).
2. Beats 2b–5 vidéo : screen recordings du dashboard / WhatsApp / explorer
   pendant le run réel (remplacent les stills, mêmes valeurs attendues).
3. Rendu final ~3 min → déposer le mp4 sous `demo/` et pointer son chemin ici.
