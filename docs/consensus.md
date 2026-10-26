# Consensus — AfriOracle

Full consensus spec, verified against `backend/consensus.go` and `backend/consensus_test.go`
(`go test -count=1 .` passes). See also the domain rules in `our_projetc.md` (couches 2–3);
where they disagree on demo scope, `spec.md` and this code win.

## Formula

```
weight = tierWeight[tier] × reputation(rep_score).consensusMult
```

`func (s Submission) Weight()`; automatic nodes (`ProofScore == nil`) skip reputation and
use a neutral multiplier of 1.0, so a tier-3 automatic node weighs exactly 0.7.

## Tier weights (`tierWeight`)

| Tier | Weight |
| --- | --- |
| 1 — Institutionnel | 3.0 |
| 2 — Terrain | 1.0 |
| 3 — Automatique | 0.7 |

Unknown tiers (e.g. 0) weigh 0 — they can never count toward consensus.

## Reputation grid (`reputation`)

| Score | Level | Consensus mult | Reward mult |
| --- | --- | --- | --- |
| 90–100 | Expert | ×2.0 | ×1.8 |
| 75–89 | Confirmé | ×1.5 | ×1.3 |
| 60–74 | Actif | ×1.0 | ×1.0 |
| 40–59 | Débutant | ×0.5 | ×0.5 |
| 0–39 | Non qualifié | ×0.0 | ×0.0 |

## Proof-score thresholds (`proofScore` in `backend/ai.go`)

| Condition | Score | Effect |
| --- | --- | --- |
| Photo bytes already seen (sha256 in `fingerprints`) | 12 (+ `DuplicatePhoto: true`) | Excluded (fraud path) |
| Unreadable slate (`photo_price == nil`) | 40 | Excluded |
| Slate/submitted gap ≤ 2% | 94 | Accepted |
| Gap ≤ 10% | 70 | Accepted |
| Otherwise | 20 | Excluded |

Mock `ObjectsFound` is `["ardoise de prix", "sac de mil"]`; `Model` is `"mock"` without
`META_API_KEY`, else `"Muse Spark 1.3 + SAM 3.1"`.

## Acceptance / exclusion (`Accepted`)

- Automatic node (`ProofScore == nil`): **always accepted** (tier-3 source hash).
- Otherwise accepted iff `ProofScore >= 50` (boundary 50 accepted; 49 rejected — covered by test).
- Excluded submissions get `Status "excluded"`, `Note "preuve rejetée (score < 50)"`,
  `Reward 0`, `ScoreDelta -2`. No stake slashing exists in this prototype.

## Weighted median (`weightedMedian`)

Over accepted submissions with weight > 0 only; caller guarantees ≥ 1. Values sorted
ascending; the first value whose cumulative weight reaches `total/2` wins.

## Collecte rule (`handleConsensus` in `backend/main.go`)

Fewer than 3 accepted, weight > 0 submissions → HTTP **422**
`{error: "collecte rule: minimum 3 independent submissions required"}`. No median is computed.

## Confidence and validation (`runConsensus`)

- `deviation = |value − median| / median × 100`.
- `confidence = round(weight within 15% of median / total accepted weight × 100)`.
- `validated = confidence >= 66`.
- Outliers (> 15%) get `Note = contextAgent(outliers)` — currently the fixed stub string
  `"écart > 15 % de la médiane; revue manuelle recommandée"` (or `""` when none).

## Rewards (`runConsensus(subs, baseReward)` with `baseReward = 5.0`)

- Tier 3 accepted: flat `0.5`.
- Other accepted: `5.0 × rewardMult` from the reputation grid.
- Excluded: `0` with `ScoreDelta −2`; accepted outliers keep their reward (stub agent
  never suspends it in code).

## Worked golden example (exact, from `consensus_test.go`)

Submissions: A(2, rep 88, 187, proof 94) · B(2, 72, 191, 91) · C(2, 65, 185, 89) ·
FAO(3, auto `nil`, 190) · D(2, 61, 240, proof 12).

| Node | Weight | Running total (sorted) |
| --- | --- | --- |
| C 185 | 1.0 × 1.0 = 1.0 | 1.0 |
| A 187 | 1.0 × 1.5 = 1.5 | 2.5 ≥ 2.1 → **median 187** |
| FAO 190 | 0.7 | — |
| B 191 | 1.0 | — |
| D 240 | excluded (12 < 50) | — |

Total accepted weight 4.2, half 2.1 → median **187**. All four accepted values within
15% → confidence **100**, validated. Rewards: **A = 5 × 1.3 = 6.5, B = 5, C = 5,
FAO = 0.5, D = 0** with D `ScoreDelta −2`.

The live `scripts/demo.sh` run adds honest agents E(186)/F(188)/G(189)/H(184), all within
2% of 187 — the median and confidence stay 187/100 while showing the fraudster D
(byte-identical reuse of A's photo → 12 → excluded) is not load-bearing.
