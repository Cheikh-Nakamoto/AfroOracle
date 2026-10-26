# AfriOracle Network — Guide du Protocole v2.0 (Édition Meta Hackathon)

Oct 18, 2026 · @Cheikh Mounirou Coly Diouf

## Nouveautés de la v2.0

La v2.0 ajoute une couche d'intelligence artificielle, construite sur les modèles Meta, à trois points du pipeline : la collecte, la validation et la consommation. Elle corrige aussi dix incohérences et une faille de conception de la v1.0, et définit une démo réalisable en 10 jours.

Le principe directeur ne change pas : la vérité vient du consensus entre nœuds indépendants. L'IA prépare et vérifie les preuves, l'humain signe, le consensus décide. Aucun modèle n'écrit directement une valeur on-chain.

- **Collecte** : soumission par message vocal et par WhatsApp, extraction automatique des bilans institutionnels.
- **Validation** : chaque photo reçoit un score de preuve, et un agent de contexte explique les écarts avant toute pénalité.
- **Consommation** : agriculteurs et coopératives interrogent l'oracle en langage naturel sur WhatsApp.
- **Cohérence** : une seule grille de réputation, une seule table de pénalités, un exemple chiffré qui respecte ses propres règles.

## Positionnement pour le Meta Global AI Developer Hackathon

AfriOracle concourt dans le thème « Build for Your World » : un problème vécu de première main, l'absence de données fiables sur les marchés africains, résolu avec les modèles Meta. Le [hackathon](https://dev.meta.ai/events/global-hackathon) est virtuel, dure du 26 octobre au 4 novembre 2026 et distribue 1 M$ de prix.

**Track recommandé : Track 1 — Indie.** AfriOracle est aujourd'hui une idée, sans partenaire ni revenu. Le track Indie est jugé sur la créativité, l'ambition technique et le fait que le projet fonctionne : trois critères où un prototype d'oracle vérifié par IA peut se démarquer. Le track Startup attend un produit réel qui tient en production. Si vous êtes plusieurs, vérifiez dans le règlement officiel, publié avant le 26 octobre, qu'une équipe peut concourir en Indie.

**Dates clés (heure de Dakar)**

- Candidature : avant le 26 octobre à 3h59 (25 octobre 23h59 heure de New York), ou plus tôt si les 20 000 places sont prises.
- Soumission du projet : 5 novembre à 2h00 (4 novembre 21h00 heure de New York).
- Finale et annonce des gagnants : 9 novembre.

La candidature est notée à parts égales sur deux critères : la clarté et la faisabilité de l'idée, et la compétence technique montrée dans un projet IA déjà réalisé.

**Modèles Meta utilisés**

| Modèle | Rôle dans AfriOracle | Coût indicatif |
| --- | --- | --- |
| [Muse Spark 1.3](https://dev.meta.ai/models/muse-spark/) | Agent de validation : lit photos et documents, structure les soumissions, explique les écarts, rédige les rapports | 1,25 $ / 4,25 $ par million de tokens (entrée / sortie) |
| [Muse Voice Transcribe](https://dev.meta.ai/models/muse-voice-transcribe/) | Transcription des soumissions vocales des agents terrain | 0,18 $ par heure d'audio |
| [SAM 3.1](https://dev.meta.ai/models/sam-3-1/) | Détection des objets attendus dans les photos de preuve (ardoise de prix, sacs, étals) | 2,50 $ pour 1 000 images |

Muse Spark 1.3 existe aussi en version « contributor » à 0,10 $ / 0,20 $, mais ses données servent à améliorer les produits Meta. AfriOracle traite des données KYC et financières : la version standard est obligatoire pour ces flux.

## Vue globale du pipeline

Une donnée traverse cinq couches, de l'observation au contrat qui la lit. L'IA intervient à la collecte, au consensus et à la consommation, sans jamais publier de valeur.

&#91;embedded content: pipeline v2.0 · 5 couches, 3 points d'appui IA\]

Le seul lien entre l'IA et la chaîne est le hash de son rapport, publié avec chaque valeur pour que n'importe qui puisse l'auditer.

## Couche 1 — Collecte des données

La v2.0 garde les quatre familles de sources et ajoute deux canaux de soumission : le message vocal et WhatsApp. Un agent qui sait parler et prendre une photo peut désormais contribuer, même sans maîtriser un formulaire.

| Famille | Exemples de sources | Type de donnée | Canal |
| --- | --- | --- | --- |
| Financière et économique | BCEAO, BRVM, CREPMF, banques | Taux, indices, bilans | API, portail institutionnel |
| Agricole et climatique | Coopératives, FAO, satellites NASA/ESA | Prix des récoltes, météo | Terrain (app, WhatsApp, vocal), API |
| Foncière et cadastrale | DGID, mairies, notaires | Valeur des terrains, propriété | Terrain, portail institutionnel |
| Identité et crédit | État civil, Wave, Orange Money | Solvabilité, flux mobiles | API, terrain |

### Soumission terrain assistée par l'IA

1. L'agent envoie un message vocal et une photo, dans l'app ou sur WhatsApp.
2. Muse Voice Transcribe transcrit l'audio. Le biais de vocabulaire du modèle reçoit la liste des produits et des marchés de la zone (mil, arachide, niébé, Kaolack…), sans réentraînement.
3. Muse Spark transforme la transcription en donnée structurée : produit, prix, unité, marché.
4. S'il manque une information, l'agent reçoit une question (« le kilo ou le sac ? »).
5. L'agent confirme le récapitulatif, puis signe. L'IA ne signe jamais à sa place.

Sur WhatsApp, la signature est faite par une clé gérée côté serveur, liée au numéro vérifié par KYC. C'est moins sûr qu'une clé détenue par l'agent : ces soumissions gardent un poids normal, mais un nœud ne peut pas passer au niveau Expert par ce seul canal.

### Documents institutionnels

Le cabinet d'audit charge son rapport sur le portail. Muse Spark, qui lit nativement les documents, extrait les valeurs et indique pour chacune la page source. L'institution relit le tableau extrait, corrige si besoin, puis signe avec sa clé institutionnelle.

### Règle de collecte maintenue

Une donnée n'est jamais acceptée d'une seule source : 3 sources indépendantes minimum pour les données de marché, 2 institutions indépendantes pour une valorisation d'entreprise.

**Question ouverte** : les pages de Muse Voice Transcribe ne listent pas les langues prises en charge. Le wolof, le pulaar et le sérère sont à tester dès le premier jour ; à défaut, la démo utilise le français.

## Couche 2 — Réseau de nœuds

Le poids d'un nœud dans le consensus est désormais défini par une seule formule : poids du tier × multiplicateur de réputation. La v1.0 donnait des chiffres incompatibles selon les sections.

```latex
\text{poids} = \text{poids du tier} \times \text{multiplicateur de réputation}
```

| Tier | Qui | Admission | Stake minimum | Poids du tier |
| --- | --- | --- | --- | --- |
| 1 — Institutionnel | Cabinet d'audit agréé ONECCA, banque agréée BCEAO, établissement public, institution régulée CREPMF | Statuts, audit technique, contrat, formation, vote du conseil Tier 1 | 30 000 à 100 000 AFRO | 3 |
| 2 — Terrain | Tout majeur avec smartphone : agents de coopérative, courtiers, journalistes, étudiants | KYC, formation de 2 h, test de qualification | 500 AFRO | 1 |
| 3 — Automatique | Logiciel opéré par un développeur du réseau, une source par nœud | Revue du code, test sur 7 jours | À définir | 0,7 |

Un nœud institutionnel pèse donc 3 fois un nœud terrain et environ 4,3 fois un nœud automatique, à réputation égale. La v1.0 annonçait « 5 fois », ce que les poids ne permettaient pas.

### Règles terrain conservées

- Chaque nœud terrain est assigné à une zone de 50 km de rayon et ne soumet que pour cette zone.
- Chaque soumission est croisée avec au moins 2 autres nœuds de la même zone.
- Le score de réputation démarre à 50 sur 100. Le badge de confiance s'obtient à 75 (niveau Confirmé), seuil unique dans tout le document.

### Sources des nœuds automatiques

| Source | Fréquence | Vérification |
| --- | --- | --- |
| Taux XOF/USDC (BCEAO et marchés) | Toutes les heures | Hash de la page source officielle |
| Indices BRVM | Quotidienne, à la clôture | 2 nœuds minimum |
| Prix CBOT/ICE | Quotidienne | API officielle hashée |
| Météo satellite | Toutes les 6 h | Coordonnées GPS vérifiées |
| Journal officiel, décrets | À la publication | Extraction par Muse Spark et hash du document |

## Couche 3 — Consensus et validation

La valeur publiée reste la médiane pondérée des soumissions, mais seules les soumissions dont la preuve a été vérifiée y entrent. L'IA filtre les preuves et explique les écarts ; elle ne fixe jamais la valeur.

La médiane est conservée parce qu'une valeur extrême ne la déplace pas : pour \[185, 188, 190, 192, 10 000\], la médiane vaut 190 et la moyenne 2 151.

### Niveau 1 — Consensus statistique, avec vérification de la preuve

1. **Fenêtre de collecte** : 30 min à 2 h selon la donnée. Le consensus n'est jamais calculé avant la fin de la fenêtre.
2. **Score de preuve (nouveau)** : chaque soumission terrain reçoit un score de 0 à 100. Muse Spark lit le prix sur la photo et le compare à la valeur saisie. SAM 3.1 vérifie la présence des objets attendus (« ardoise de prix », « sac de mil »). Le serveur contrôle aussi le GPS de la zone et l'empreinte de la photo, pour repérer une image déjà soumise.
3. **Filtre** : une soumission avec un score de preuve inférieur à 50 sort du calcul et part en revue manuelle.
4. **Médiane pondérée** : calculée avec les poids de la couche 2.
5. **Détection des écarts** : toute valeur à plus de 15 % de la médiane est marquée.
6. **Validation** : la donnée est validée si au moins 66 % du poids total est à moins de 15 % de la médiane.
7. **Publication** : la valeur part vers le smart contract avec le hash du rapport de consensus.

### Agent de contexte (nouveau)

Quand la dispersion dépasse 15 %, un agent Muse Spark cherche une cause avant toute pénalité. Il consulte l'actualité par recherche web (Tabaski, inondation, fermeture de frontière), les autres zones et l'historique du produit. Il produit un rapport et une recommandation : valider, prolonger la fenêtre ou passer au niveau 2.

L'agent peut suspendre une pénalité en attendant une revue, mais il ne peut ni modifier une valeur ni en publier une. Ses consignes et la version du modèle sont publiées et hashées : chacun peut auditer ce qu'il a reçu et produit. Le texte lu sur les photos et le web est traité comme une donnée, jamais comme une instruction.

### Niveau 2 — Validation institutionnelle

Inchangé pour les valorisations d'entreprise, le foncier et la solvabilité. Une signature institutionnelle est obligatoire, et une contre-valeur à plus de 10 % de la médiane terrain déclenche l'arbitrage.

### Niveau 3 — Arbitrage DAO

Déclenché par un écart de plus de 20 % entre institutions et terrain, un désaccord entre institutions, un soupçon de manipulation ou une contestation formelle. Le vote dure 7 jours. Nouveau : l'agent de contexte rédige un dossier neutre pour les votants, avec les arguments et les preuves des deux parties.

### Pénalités et score : une seule table

| Écart à la médiane | Pénalité sur le stake | Effet sur le score |
| --- | --- | --- |
| 0 à 5 % | Aucune | +0,5 |
| 5 à 10 % | Aucune | +0,2 |
| 10 à 15 % | Avertissement ; le 3e coûte 1 % | −1 |
| 15 à 25 % | 3 % | −3 |
| 25 à 50 % | 10 % | −10 |
| Plus de 50 % | 25 % et suspension de 30 jours | −30 et enquête |
| Preuve rejetée (score de preuve < 50) | Aucune, revue manuelle | −2 |
| Manipulation prouvée | 100 % et exclusion | Remis à 0 |

Un écart que l'agent de contexte juge plausible reste sans pénalité jusqu'à la revue.

### Anti-collusion corrigée

La v1.0 mettait en quarantaine les valeurs identiques soumises en 5 minutes. Or des agents honnêtes sur le même marché relèvent souvent le même prix : la règle aurait puni la bonne foi.

En v2.0, un groupe est mis en quarantaine seulement si deux conditions sont réunies :

- il s'écarte de plus de 15 % des sources indépendantes (autres zones, nœuds automatiques, historique) ;
- ses membres partagent un signal de lien : même appareil, même photo, même réseau ou même parrain.

## Couche 4 — Smart contracts oracle

Les contrats restent sur Polygon ; la v2.0 ajoute à chaque publication le hash du rapport IA, pour que la vérification des preuves soit auditable comme le reste. Pour le hackathon, tout est déployé sur le testnet Polygon.

| Contrat | Rôle | Changement v2.0 |
| --- | --- | --- |
| PriceOracle.sol | Prix validés : change, matières premières, indices | Champ `aiReportHash` |
| AgriculturalData.sol | Prix agricoles locaux et météo pour l'assurance paramétrique | Champ `aiReportHash` |
| CompanyValuation.sol | Valorisation certifiée des PME avec signature institutionnelle | Hash du tableau extrait par l'IA et validé par l'institution |
| LandRegistry.sol | Titres fonciers et évaluations immobilières | Inchangé |
| NodeRegistry.sol | Nœuds autorisés, stakes, scores de réputation | Inchangé |
| RewardDistributor.sol | Calcul et versement des récompenses après chaque consensus | Ignore les soumissions à preuve rejetée |
| GovernanceDAO.sol | Votes et arbitrages | Vote sur la version du modèle et des consignes de l'agent |

### Ce qui est stocké à chaque publication

La valeur (multipliée par 100), l'horodatage, le score de confiance, la source principale, le hash du rapport de consensus et le hash du rapport IA. Les rapports complets restent hors chaîne, sur IPFS, et leur hash garantit qu'ils n'ont pas été modifiés.

### Qui signe la transaction

En phase 1, la publication est signée par le multisig de la fondation ; en phase 2, par les validateurs du réseau. La v1.0 affirmait qu'aucun humain n'approuve le processus : c'est vrai du calcul, pas de la signature en phase 1.

### Garanties aux consommateurs

Immuabilité, traçabilité jusqu'à la session de consensus, âge de la donnée exposé et score de confiance de 0 à 100 : inchangés.

## Couche 5 — Marketplace, API et WhatsApp

La v2.0 ajoute un sixième mode d'accès : un assistant WhatsApp qui répond aux agriculteurs et aux coopératives à partir des seules données publiées par l'oracle. Les cinq modes de la v1.0 sont conservés.

| Mode d'accès | Pour qui | Coût | Latence |
| --- | --- | --- | --- |
| Lecture on-chain directe | Développeurs Solidity / Web3 | Frais de gas Polygon | Temps réel |
| API REST publique | Développeurs Web2 / Web3 | Tier gratuit limité, puis plans en AFRO | < 500 ms |
| WebSocket temps réel | Applications à mise à jour continue | Plan premium en AFRO | < 100 ms |
| SDK JavaScript / Python | Intégration rapide | SDK gratuit, coût de l'API | Selon l'API |
| Abonnement institutionnel | Banques, assureurs, fonds | Contrat annuel en AFRO | Prioritaire |
| **Assistant WhatsApp (nouveau)** | Agriculteurs, coopératives, commerçants | Gratuit pour les questions de prix | Quelques secondes |

### Comment l'assistant répond

1. L'utilisateur écrit ou dit : « Prix du mil à Kaolack ? »
2. Muse Spark identifie le produit et la zone, puis appelle l'API de l'oracle comme un outil.
3. Il répond avec la valeur, la date de publication, le score de confiance et le lien de la transaction.
4. Si l'oracle n'a pas de donnée récente, il le dit. Il n'estime jamais un prix lui-même.

### Cas d'usage et fonctions

| Cas d'usage | Fonction appelée |
| --- | --- |
| Tokenisation PME | `CompanyValuation.getValuation(address)` |
| Assurance agricole | `AgriculturalData.getRainfall(zone, date)` |
| Tokenisation foncière | `LandRegistry.getPropertyData(id)` |
| DeFi et échanges | `PriceOracle.getRate('XOF', 'USDC')` |

La v1.0 appelait `PriceOracle.getCompanyValuation`, alors que la valorisation est stockée dans CompanyValuation.sol. Le coût d'une requête API reste d'environ 0,001 AFRO.

## Système de récompenses

Une seule grille de réputation fixe désormais le poids dans le consensus et le multiplicateur de récompense. La vérification IA coûte environ 1 centime de dollar par soumission, soit moins de 1 % d'une récompense de base.

### Grille de réputation unique

| Score | Niveau | Poids dans le consensus | Multiplicateur de récompense | Contraintes |
| --- | --- | --- | --- | --- |
| 90 à 100 | Expert | ×2 | ×1,8 | Reporting renforcé ; clé détenue par l'agent obligatoire |
| 75 à 89 | Confirmé | ×1,5 | ×1,3 | Badge de confiance |
| 60 à 74 | Actif | ×1 | ×1 | Toutes catégories |
| 40 à 59 | Débutant | ×0,5 | ×0,5 | — |
| 0 à 39 | Non qualifié | 0 | 0 | Accès suspendu, formation à repasser |

Les effets des écarts sur le score sont dans la table de la couche 3. Autres événements : 3 non-soumissions −2, dispute DAO gagnée +3, perdue −5, 3 mois réguliers sans incident +5.

### Sources de revenus du réseau

| Source | Montant | Répartition |
| --- | --- | --- |
| Requêtes API | 0,001 AFRO par requête | 70 % nœuds, 20 % trésor DAO, 10 % fondation |
| Abonnements institutionnels | Selon contrat | 60 % nœuds concernés, 25 % trésor, 15 % fondation |
| Certification PME | 500 AFRO par certification initiale | 50 % auditeur, 30 % trésor, 20 % fondation |
| Pénalités | Selon la faute | 50 % aux nœuds précis, 50 % trésor |
| Émission programmée | 2 % par an pendant 10 ans | Nœuds terrain actifs |

### Récompenses par tier

| Tier | Récompense | Montant |
| --- | --- | --- |
| 1 | Disponibilité mensuelle | 200 à 2 000 AFRO selon le volume |
| 1 | Certification validée | 250 AFRO |
| 1 | Bonus précision (conformité > 90 %) | ×1,5 sur le mois |
| 1 | Arbitrage DAO | 50 AFRO par dispute |
| 2 | Soumission validée | 1 à 10 AFRO selon le type ; 5 AFRO pour un prix agricole de marché |
| 2 | Réputation | Multiplicateur de la grille ci-dessus |
| 2 | Zone sous-couverte | +50 % de la base |
| 2 | Régularité 3 / 6 / 12 mois | +10 % / +20 % / +35 % |
| 3 | Donnée validée | 0,5 AFRO |
| 3 | Disponibilité > 99 % | 100 AFRO par mois et par source |

### Coût de la vérification IA par soumission terrain

| Étape | Hypothèse | Coût |
| --- | --- | --- |
| Muse Voice Transcribe | Message de 30 s | 0,0015 $ |
| Muse Spark 1.3 | 4 000 tokens en entrée, 500 en sortie | 0,007 $ |
| SAM 3.1 | 1 photo | 0,0025 $ |
| **Total** |  | **≈ 0,011 $** |

À 0,31 $ l'AFRO (hypothèse de la v1.0), une base de 5 AFRO vaut 1,55 $ : la vérification en coûte 0,7 %. Les volumes de tokens sont des estimations, à mesurer pendant le hackathon.

### Simulation de revenu mensuel

| Profil | Activité | Revenu estimé | Équivalent |
| --- | --- | --- | --- |
| Terrain débutant | 5 soumissions par semaine, score 55 | 80 à 120 AFRO | 15 000 à 23 000 FCFA |
| Terrain confirmé | 15 par semaine, score 82 | 500 à 700 AFRO | 95 000 à 133 000 FCFA |
| Terrain expert, zone exclusive | 20+ par semaine, score 93 | 1 200 à 1 800 AFRO | 228 000 à 342 000 FCFA |
| Cabinet d'audit, 20 certifications | 20 par mois et disponibilité | 5 500 à 7 000 AFRO | 1,05 M à 1,33 M FCFA |
| Développeur, 5 sources | Disponibilité > 99 %, 3 000 données | 800 à 1 200 AFRO | 152 000 à 228 000 FCFA |

Hypothèse : 1 AFRO ≈ 190 FCFA (0,31 $). Les revenus réels dépendent du cours du token et du volume du réseau.

## Exemple complet : le prix du mil à Kaolack

Le prix publié est de 187 FCFA/kg, à 9h31, après avoir écarté une soumission frauduleuse que la médiane seule aurait laissé passer. Sans le filtre IA, cette fraude aurait déplacé la valeur à 190 FCFA et n'aurait été traitée que comme un simple écart, sans enquête pour manipulation.

### Les soumissions (lundi, fenêtre de 9h00 à 9h30)

| Nœud | Score | Canal | Valeur saisie | Score de preuve | Poids |
| --- | --- | --- | --- | --- | --- |
| Agent A | 88 | App, photo de l'ardoise | 187 | 94 | 1 × 1,5 = 1,5 |
| Agent B | 72 | WhatsApp vocal et photo | 191 | 91 | 1 × 1 = 1 |
| Agent C | 65 | App, photo | 185 | 89 | 1 × 1 = 1 |
| Nœud automatique FAO | — | API | 190 | Hash de la source | 0,7 |
| Agent D | 61 | App, photo | 240 | 12 | Exclu |

Le score de preuve de l'agent D est de 12 : l'ardoise sur sa photo affiche 190, et l'empreinte de l'image correspond à une photo soumise la semaine précédente.

### Le déroulé

1. **9h00 à 9h30 — Collecte.** Les quatre agents et le nœud FAO soumettent. L'agent B dicte son prix sur WhatsApp ; Muse Spark lui demande de confirmer l'unité, puis il signe.
2. **Dès réception — Vérification.** Chaque photo est lue par Muse Spark et analysée par SAM 3.1. L'agent D passe en revue manuelle.
3. **9h30 — Médiane pondérée.** Valeurs triées avec leurs poids : 185 (1), 187 (1,5), 190 (0,7), 191 (1). Le poids total vaut 4,2 ; la moitié, 2,1, est atteinte à 187. Toutes les valeurs sont à moins de 5 % de 187 : accord de 100 %, score de confiance de 100.
4. **9h31 — Publication.** AgriculturalData reçoit 18 700, l'horodatage, la confiance, la source « Kaolack-Mil-Hebdo », le hash du rapport de consensus et celui du rapport IA.
5. **9h31 — Récompenses.** Agent A : 5 × 1,3 = 6,5 AFRO. Agents B et C : 5 AFRO chacun. Nœud FAO : 0,5 AFRO. Agent D : rien, −2 points, et l'enquête pour manipulation est ouverte. Total : 17 AFRO.
6. **9h31 — Consommation.** L'assurance agricole constate que le prix est au-dessus du seuil : pas d'indemnisation. L'application de microfinance met à jour son taux. Un agriculteur demande sur WhatsApp « prix du mil à Kaolack ? » et reçoit : 187 FCFA/kg, publié à 9h31, 4 sources, confiance 100.

Avec la soumission de l'agent D incluse, la médiane pondérée serait passée à 190 FCFA (poids total 5,2, moitié 2,6 atteinte à 190).

## Gouvernance

La v2.0 soumet les paramètres de l'IA au même vote que les paramètres économiques : un modèle qui filtre les preuves ne doit pas changer sans décision collective.

| Décision | Qui vote | Quorum | Délai |
| --- | --- | --- | --- |
| Mise à jour du protocole | Détenteurs AFRO et équipe | 30 % de la supply, majorité 80 % | 30 jours |
| Partenariat institutionnel majeur | Détenteurs AFRO | 25 % de la supply, majorité 75 % | 21 jours |
| Distribution des récompenses | Détenteurs AFRO | 20 % de la supply, majorité 66 % | 21 jours |
| **Version du modèle ou consignes de l'agent (nouveau)** | Détenteurs AFRO et nœuds actifs | 15 % de la supply | 14 jours |
| **Seuil du score de preuve (nouveau)** | Détenteurs AFRO | 10 % de la supply | 7 jours |
| Nouvelle source de données | Détenteurs AFRO et nœuds actifs | 15 % de la supply | 14 jours |
| Seuils de pénalité | Détenteurs AFRO | 10 % de la supply | 7 jours |
| Admission d'un nœud institutionnel | Conseil des nœuds Tier 1 | 66 % du conseil | 7 jours |

**Décentralisation progressive** : phase 1 (0 à 6 mois), équipe fondatrice et conseil consultatif ; phase 2 (7 à 18 mois), multisig élargi aux nœuds institutionnels ; phase 3 (19 à 36 mois), DAO complète.

## Périmètre de la démo hackathon

La démo montre un seul flux de bout en bout : le prix du mil, de la note vocale de l'agent jusqu'au contrat d'assurance qui le lit. Le token AFRO, la DAO, le foncier et la valorisation PME sont présentés dans ce document, pas construits.

### Ce qui est construit

- **Entrée** : WhatsApp via l'API Cloud, avec une page web (micro et caméra) en secours.
- **Transcription et extraction** : Muse Voice Transcribe, puis Muse Spark avec une sortie JSON.
- **Score de preuve** : Muse Spark lit l'ardoise, SAM 3.1 détecte les objets, le serveur compare l'empreinte de l'image.
- **Consensus** : service Python avec médiane pondérée et agent de contexte doté de la recherche web.
- **Chaîne** : AgriculturalData.sol et un contrat d'assurance paramétrique de démonstration, sur le testnet Polygon.
- **Sortie** : tableau de bord (soumissions, scores, médiane, transaction) et assistant WhatsApp.

Huit agents simulés complètent les vraies soumissions, dont un fraudeur scénarisé. Les récompenses sont versées en AFRO de test.

### Plan

&#91;embedded content: plan de la démo · 6 étapes de travail, 2 échéances\]

Si le temps manque, l'agent de contexte est coupé avant tout autre élément : le flux de bout en bout passe en premier.

### Script de la vidéo (3 minutes)

1. **0:00** — Le problème : un assureur ne peut pas indemniser un agriculteur sans prix local fiable.
2. **0:30** — Un agent filmé sur un vrai marché envoie une note vocale et une photo sur WhatsApp.
3. **1:00** — Le tableau de bord affiche la transcription, le score de preuve, puis la fraude détectée.
4. **1:45** — La médiane est publiée ; la transaction s'ouvre dans l'explorateur Polygon.
5. **2:15** — Le contrat d'assurance lit la valeur ; un agriculteur interroge l'assistant WhatsApp.
6. **2:45** — Le coût : environ 1 centime de dollar de vérification par soumission.

### Risques

| Risque | Parade |
| --- | --- |
| Validation du compte WhatsApp Business trop longue | Numéro de test de l'API Cloud dès le premier jour (à confirmer dans la console Meta) ; page web en secours |
| Wolof non pris en charge par la transcription | Démo en français, wolof dans la feuille de route |
| Ardoises manuscrites mal lues | Seuil de preuve réglable et revue manuelle |
| Délai trop court | Couper l'agent de contexte, garder le flux complet |

## Journal des modifications v1.0 → v2.0

### Ajouts

| Zone | Ajout |
| --- | --- |
| Couche 1 | Soumission vocale et WhatsApp ; extraction des bilans par Muse Spark |
| Couche 3 | Score de preuve par photo ; agent de contexte ; dossier d'arbitrage pour la DAO |
| Couche 4 | Hash du rapport IA à chaque publication |
| Couche 5 | Assistant WhatsApp pour les agriculteurs |
| Gouvernance | Vote sur la version du modèle, les consignes et le seuil de preuve |

### Corrections

| Zone | v1.0 | v2.0 |
| --- | --- | --- |
| Bonus de réputation | Trois versions : ×1,3 / ×1,8 (2.2), +30 % / +80 % (6.3), ×1,5 dans l'exemple | Une grille : ×1,3 (75–89), ×1,8 (90–100) |
| Seuil du badge | 80 (2.2) ou 75 (6.3) | 75 |
| Poids Tier 1 / Tier 3 | « 5 fois », alors que 3 ÷ 0,7 ≈ 4,3 | 4,3 fois |
| Effet sur le score | Deux tables contradictoires (10–15 % : +0,2 ou −2) | Une seule table, couche 3 |
| Fenêtre de consensus | Exemple à 9h07, fenêtre minimale de 30 min | Consensus à 9h30 |
| Médiane de l'exemple | Médiane simple (188,5) | Médiane pondérée (187) |
| Récompenses de l'exemple | B et C « base normale » mais 6,5 et 5,5 AFRO | 5 AFRO chacun |
| Équivalents FCFA | Calculés à ≈ 310 FCFA l'AFRO, annoncé à 190 | Recalculés à 190 FCFA |
| Approbation humaine | « Aucun humain n'approuve », malgré le multisig de phase 1 | Formulation corrigée |
| Fonction de valorisation | `PriceOracle.getCompanyValuation` | `CompanyValuation.getValuation` |
| Anti-collusion | Quarantaine des valeurs identiques en 5 min | Déviation de groupe et signal de lien |

## Sources

- [Meta Global AI Developer Hackathon](https://dev.meta.ai/events/global-hackathon) — dates, tracks, thèmes, critères de sélection
- [Muse Spark 1.3](https://dev.meta.ai/models/muse-spark/) — capacités multimodales, appel d'outils, tarifs
- [Muse Voice Transcribe](https://dev.meta.ai/models/muse-voice-transcribe/) — transcription en streaming, biais de vocabulaire, tarif
- [SAM 3.1](https://dev.meta.ai/models/sam-3-1/) — détection et segmentation par texte, tarif
