# Point d’avancement — Blockchain AfriOracle

**Branche :** `feat/polygon-amoy-publish`

## Ce qui est prêt

- Le backend de publication refuse d’envoyer une transaction si le RPC n’annonce pas
  Polygon Amoy (chain ID `80002`).
- `scripts/deploy-amoy.sh` compile les deux contrats avec Solidity 0.8.20, les déploie
  sur Amoy, attend les reçus et affiche les adresses et transactions.
- L’assurance refuse maintenant de calculer une indemnité si aucun prix n’a été publié
  pour la date demandée (`price not published`).
- Les contrats rejettent les règlements à prix nul, les confiances au-dessus de 100, une
  adresse oracle nulle et un seuil d’assurance nul.
- Chaque prix publié doit aussi avoir sa zone, son produit, sa date, sa source et ses
  deux hashes de preuve renseignés.
- Les clés de stockage utilisent un encodage non ambigu des champs, évitant qu’une
  concaténation identique écrase un autre couple marché/date.
- `cd backend && go run ./cmd/check-insurance` lit le résultat de
  `ParametricInsurance.check` en lecture seule, sans clé privée.
- `AgriculturalData.getPriceRecord` permet de relire le prix et ses métadonnées de
  preuve (source, hash du consensus et hash du rapport IA) directement on-chain.

## Scénario de démonstration

Avec un seuil de `20000` (200 FCFA/kg) et un prix publié de `18700` (187 FCFA/kg), le
contrat calcule `1300`, soit 13 FCFA/kg en unités ×100. Le contrat actuel ne transfère
pas de tokens : il retourne seulement le montant calculé. Aucun token AFRO, ERC-20 ou
NFT n’a été ajouté au prototype.

## Vérifications et prochaine étape

`go test ./...`, `go vet ./...`, `sh -n scripts/deploy-amoy.sh` et la compilation
solc 0.8.20 passent. Les contrats n’ont pas encore été déployés publiquement : il
faut un RPC Amoy et un wallet de test financé en POL de test. Toute ancienne instance
oracle devra être redéployée pour utiliser le nouveau calcul de clé. Après le déploiement,
configurer `CONTRACT_ADDRESS` pour les publications et `INSURANCE_CONTRACT_ADDRESS`
pour lire l’assurance.
