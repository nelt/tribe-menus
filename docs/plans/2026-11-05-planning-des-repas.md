# Plan : planning des repas

- **Nom** : `planning`
- **Date** : 2026-11-05
- **Statut** : en cours (lot B). Décisions D1 à D8 prises par le développeur le 2026-11-05 au cadrage ; D9 à D20 proposées : la PR du plan (#86) a été fusionnée sans commentaire, le lot A les applique, à confirmer à sa revue.

## Objectif

Deuxième plan du métier : un membre organise depuis son téléphone les repas de la semaine de sa tribu, en y plaçant des plats de la bibliothèque, chacun avec son nombre de parts. Avec la bibliothèque (`v0.2.0`), c'est la première application viable à montrer ; les listes de courses viennent dans un plan à part, ensuite (feuille de route, décision du 2026-11-05).

À la fin du plan :

- l'onglet **Planning** remplace l'accueil provisoire par le planning de `Doux-Main` : 7 jours à partir d'aujourd'hui, deux repas par jour, midi et soir (R1), les semaines précédentes et suivantes (R5) ;
- toucher un repas ouvre son détail, d'après `Doux-Repas` : on y ajoute un ou plusieurs plats de la bibliothèque, à la taille de la tribu (R2), on règle leurs parts (R3), on les retire (R4) ; un plat déjà servi dans le repas n'est pas proposé (Q21) ; les repas passés restent modifiables (R5) ;
- « Se déconnecter » passe sur l'écran Tribu (D5) ;
- le planning est partagé entre les membres de la tribu et seulement eux (T1, ENF-02) ; la taille de la tribu fixe les parts des plats servis ajoutés ensuite (T3) ;
- la version `v0.3.0` est publiée et déployée en production (D7).

La conception (modèle, découpage du code, API, échanges) est dans [`docs/conception/planning-des-repas.md`](../conception/planning-des-repas.md), avec ses diagrammes de classes et de séquence (`bibliotheque`, D13). Ce plan dit dans quel ordre on la réalise et ce qui reste à décider.

## Contexte et contraintes

- **À lire avant de commencer** : ADR 0002, 0003, 0004 (points 3, 6, 7 et 9), 0005, 0008, 0012 (point 6), 0027, 0028, 0029 ; `docs/specs/fonctionnalites.md` (concepts Repas et Plat servi, R1 à R5, T1, T3, Q1, Q2, Q5, Q8, Q21), `glossaire.md`, `conventions-gherkin.md` (« Dates et repas »), `features/planning-repas.feature`, `tribu.feature`, `compartimentage-tribus.feature` ; `docs/design/README.md` (« Navigation », « Planning et plats », écarts du plan `bibliotheque`) et les maquettes `Doux-Main`, `Doux-Repas`, `Doux-Tribu` ; le document de conception de ce plan et celui de la bibliothèque, dont ce plan réutilise la copie des collections et la recherche.
- **Après `bibliotheque`** : `v0.2.0` est en production, la PR de clôture (#85) est fusionnée. La recette n'a pas d'autre preuve en cours : les archives de PR de ce plan peuvent y aller dès le lot B.
- **Où se fait le travail** : le lot A, documentation seule, peut se faire depuis une session cloud ; les lots B à D se font dans le Dev Container (la session cloud ne peut ni compiler ni tester le projet).
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, avec `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`planning/B`) et les stories ; `make ci` avant de pousser.
- **Aucune dépendance nouvelle**, ni en Go ni en npm : les dates se calculent avec `time` en Go et `Date` dans le navigateur (D4), l'horloge des scénarios `@ui` avec celle de Playwright (`page.clock`).
- **Vocabulaire** : le code suit `glossaire.md` (`meal plan`, `meal`, `mealtime: lunch / dinner`, `served dish`, `servings`) ; termes nouveaux ajoutés au lot A, avant le code : jour (*day*), plage de jours (*day range*), case de repas (*meal tile*), ajout d'un plat à un repas (*serve a dish*), retrait (*remove*), copie du planning (*cached meal plan*), proposition de plat (*dish suggestion*).
- **Une migration** de la base de la tribu (`0008`) : revenir en arrière sur `v0.2.0` demandera la restauration de l'instantané (ADR 0012, point 6 ; plan `recette`, D7). Le registre et la base de limitation ne changent pas.
- **Hors-ligne hors du plan** (D13), comme pour la bibliothèque : une modification sans réseau affiche « Réseau nécessaire » et ne change rien (ENF-01, PT-04, Q16).
- **Rien d'une autre tribu** (ENF-02) : chaque requête lit la seule base de la tribu de sa session (ADR 0003) ; les identifiants des plats servis sont aléatoires (D10) ; un plat d'une autre tribu ne peut pas être servi, son identifiant étant inconnu de la base.
- **Aucun texte saisi nouveau** : le planning n'affiche que des noms de plats, déjà échappés (Lit, ADR 0004, point 9) et déjà contrôlés par `webcheck`.

### Scénarios couverts

| Fichier | Scénario | Preuve | Lot |
| --- | --- | --- | --- |
| `planning-repas.feature` | R1 : afficher les repas de 7 jours à partir d'aujourd'hui | navigateur, `@ui`, horloge au 5 octobre 2026 | C |
| | R2 : ajouter un plat à un repas ; servir plusieurs plats dans un même repas | API | B |
| | R2 : un même plat ne peut pas être ajouté deux fois, **scindé** au lot A (D6) : le refus de l'API | API | B |
| | R2 : … et « un plat déjà servi dans le repas ne m'est pas proposé » | navigateur, `@ui` | D |
| | R3 : modifier le nombre de parts d'un plat servi | API | B |
| | R4 : retirer un plat d'un repas ; un plat retiré puis ajouté de nouveau repart de 4 parts | API | B |
| | R5 : naviguer vers la semaine suivante, vers la semaine précédente | navigateur, `@ui` | C |
| | R5 : les repas passés restent modifiables | API | B |
| | Scénarios nouveaux du lot A : parts de 1 à 50 et hors de ces bornes, repas de 2000 à 2099 et hors de ces bornes, plat inconnu, ordre d'ajout, plat servi entre-temps, dernier réglage des parts, parts d'un plat retiré entre-temps (Q33, Q35) | API | B |
| `tribu.feature` | T1 : le planning est partagé entre membres ; T3 : la taille de la tribu fixe les parts des plats servis ajoutés ensuite | API | B |
| `compartimentage-tribus.feature` | ENF-02 : « Le planning est propre à la tribu », **tiré** au lot A de « Toutes les données sont compartimentées » (D6) ; « Un plat servi d'une autre tribu ne peut être ni modifié ni retiré », « Un plat d'une autre tribu ne peut pas être servi » (nouveaux) | API | B |
| | ENF-02 : toutes les données sont compartimentées | **reste en attente** : ses exemples listes, membres et sessions | — |
| `bibliotheque-plats.feature` | P2 : la modification d'un plat s'applique aux repas où il est servi | **reste en attente** : listes de courses | — |
| `authentification.feature` | ENF-01 : consultation hors ligne (du planning) ; les modifications autres que les listes nécessitent le réseau | **restent en attente** : plan hors-ligne | — |

## Étapes

### Lot A : specs, design et conception (`feature/specs-planning-des-repas`)

Documentation seule, une fois ce plan revu et ses décisions confirmées.

- [x] **1. `fonctionnalites.md`** : décisions nouvelles, à la suite de Q31 : Q32, un plat retiré puis ajouté de nouveau est un nouveau plat servi (D2) ; Q33, écritures croisées sur un repas (D3) ; Q34, « aujourd'hui » est la date de l'appareil (D4) ; Q35, limites et ordre (parts de 1 à 50, dates de 2000 à 2099, plats d'un repas dans l'ordre d'ajout, D4, D11). R2 et R4 accordés ; concept « Plat servi » : identité propre.
- [x] **2. Scénarios**, selon `conventions-gherkin.md` : scission du scénario R2 « Un même plat ne peut pas être ajouté deux fois à un repas » en un refus contre l'API et « Un plat déjà servi dans un repas ne m'est pas proposé » (`@ui`) ; tags `@ui` sur R1 et les deux scénarios de navigation de R5 ; scénarios nouveaux contre l'API (tableau ci-dessus) ; dans `compartimentage-tribus.feature`, l'exemple « le planning » sorti de « Toutes les données sont compartimentées » en un scénario « Le planning est propre à la tribu », et « Un plat servi d'une autre tribu ne peut être ni modifié ni retiré » (D6). Les nouveaux titres entrent dans `acceptance/pending.txt`, les scénarios `@ui` y restent (ADR 0028, point 4).
- [x] **3. `glossaire.md`** : les termes nouveaux de « Contexte et contraintes ».
- [x] **4. `docs/design/README.md`** : le Planning remplace l'accueil provisoire (« Écarts » du plan `bibliotheque` mis à jour) ; interprétations de ce que les maquettes ne montrent pas (D8) : plage de jours et son titre à cheval sur deux mois ou deux années, « Revenir à aujourd'hui », chargement et erreur de réseau, bibliothèque vide dans le détail d'un repas, ordre et recherche des propositions (D12), messages (plat déjà servi, plat retiré entre-temps, réseau nécessaire) ; « Se déconnecter » en bas de l'écran Tribu (D5) ; chemins (D14). Écarts à reporter dans le canevas de travail par le développeur s'il le souhaite.
- [x] **5. Document de conception** : accordé aux specs, au glossaire et au design du lot.
- [x] **6. Feuille de route** : plan « en cours ».

### Lot B : domaine et API (`feature/planning-api`)

- [x] **7. Migration `0008`** de la base de la tribu : table `served_dishes` (identifiant aléatoire, date `AAAA-MM-JJ`, moment `lunch` ou `dinner`, plat, parts de 1 à 50, rang d'ajout ; index unique sur la date, le moment et le plat, Q21 ; clé étrangère vers `dishes`) ; sans auteur ni date de modification (D18) ; collection `meal_plan` déclarée dans `storage.Collections`, avec ses trois déclencheurs (ADR 0029, point 2), que le test de schéma vérifie. Requêtes sqlc, `make generate`.
- [x] **8. Règles pures de `internal/mealplan`**, tests en tableaux d'abord : lecture d'une date (`AAAA-MM-JJ`, date réelle, de 2000-01-01 à 2099-12-31, D4), d'un moment, des parts (de 1 à 50, la règle des parts d'un plat) ; aucune notion d'« aujourd'hui » dans le serveur (D4).
- [x] **9. `mealplan.Plan`** derrière une interface `Store`, tests d'abord : liste par pages (D1), ajout d'un plat à un repas (parts absentes : taille de la tribu au moment de l'ajout, D9 ; plat inconnu ; plat déjà servi, avec les plats servis du repas à jour, D3), réglage des parts (dernier enregistrement gagnant, D3), retrait ; dans une transaction d'écriture qui ramène la version à « avant plus un » (`writeCollections`, conception de la bibliothèque) ; adaptateur `MealPlanStore` dans `internal/storage` (D15).
- [x] **10. API** : `GET meal-plan` par pages, servie par `internal/collection` (ETag, `304`, `412`) ; `POST meal-plan` (201 ; 400 `invalid_date`, `invalid_mealtime`, `invalid_servings`, `unknown_dish` ; 409 `dish_already_served` avec les plats servis du repas) ; `PUT meal-plan/{id}/servings` (200 ; 404 `not_found`) ; `DELETE meal-plan/{id}` (200 avec les versions ; 404 `not_found`, D19) ; réponses d'écriture avec les versions des collections ; 404 identique pour un identifiant inconnu ou d'une autre tribu ; corps limité à 4 Kio.
- [x] **11. Étapes godog** des scénarios « API » du tableau, retirés de `pending.txt`.
- [x] **12. `make seed`** : la semaine de `Doux-Main` dans la tribu `demo`, à partir du jour du seed (D16). La tribu de recette n'en reçoit pas.

### Lot C : planning de la semaine (`feature/planning-semaine`)

- [ ] **13. `web/src/meal-plan.ts`**, module pur testé par `node:test` : date du jour de l'appareil (D4), plage de 7 jours et ses voisines, titre de la plage (« 5 – 11 octobre », « 28 septembre – 4 octobre », « 29 décembre 2026 – 4 janvier 2027 »), cases de repas d'une plage à partir des plats servis et de la bibliothèque, total des parts d'un repas.
- [ ] **14. Copie du planning** : `web/src/planning.ts`, une copie générique de la collection `meal_plan` (`collections.ts`, ADR 0029, point 4), vérifiée avec la bibliothèque à chaque visite du Planning ou d'un repas et à chaque retour au premier plan (D15) ; mise à jour par la réponse d'une écriture. Rien n'est écrit sur l'appareil.
- [ ] **15. Écran Planning** d'après `Doux-Main`, à la place de l'accueil provisoire : en-tête (titre, pastille de la tribu vers l'écran Tribu), plage et ses flèches, « Revenir à aujourd'hui » (D8), « Aujourd'hui » sur le jour courant, cases de repas (liserés midi et soir, pilules des plats et leurs parts, case vide « + Ajouter »), chargement et erreur de réseau ; chemins `?du=AAAA-MM-JJ` (D14) ; changement de jour au retour au premier plan (D17). `home-screen.ts` disparaît.
- [ ] **16. « Se déconnecter »** en bas de l'écran Tribu, bouton au contour rouge tomate (D5), avec le test d'écran qui le suivait sur l'accueil.
- [ ] **17. R1 et R5 (navigation) sous Chromium et WebKit**, horloge de Playwright au lundi 5 octobre 2026, retirés de `pending.txt`.

### Lot D : détail d'un repas (`feature/planning-repas`)

- [ ] **18. `meal-plan.ts`** : propositions de plats d'un repas (bibliothèque sans les plats déjà servis, Q21 ; ordre alphabétique sans recherche, résultats de `searchDishes` avec, D12), avec leurs tests.
- [ ] **19. Écran d'un repas** d'après `Doux-Repas`, au chemin `repas/AAAA-MM-JJ/midi|soir` (D14) : jour et moment, total des parts, plats servis (initiale dans sa couleur, nom, « Défini pour N parts », stepper de parts, retrait), « Aucun plat pour ce repas », « Ajouter un plat » (recherche et propositions, phrase sur les plats déjà servis) ; parts enregistrées à chaque pas, le dernier seul envoyé après un enregistrement en cours (comme la taille de la tribu) ; messages de D3 (plat déjà servi, plat retiré entre-temps, la copie prenant le repas à jour) ; « Réseau nécessaire » (D13).
- [ ] **20. R2 « Un plat déjà servi dans un repas ne m'est pas proposé » sous Chromium et WebKit**, retiré de `pending.txt`.

### Recette et release

- [ ] **21. Recette** sur l'archive de la PR de release `release/v0.3.0` (`RELEASING.md`, étape 6), qui contient tout le plan : check-list « Séance de recette » ci-dessous.
- [ ] **22. Release `v0.3.0`** selon `RELEASING.md`, déployée en production (D7).
- [ ] **23. Clôture** : statut « terminé » ; feuille de route (plan terminé, points reportés à jour) ; document de conception ; `CLAUDE.md` (stack : paquet `internal/mealplan`, version en service) ; `CHANGELOG.md`.

### Séance de recette (étape 21)

Par le développeur, sur l'archive de la PR `release/v0.3.0`, CI verte (artefact gardé trois jours, sinon `gh run rerun <identifiant>`). Deux appareils : Android (Chrome) et un second, iPhone si possible, sinon un ordinateur. La tribu de recette n'a ni plat ni planning : commencer par créer trois plats. Les demandes de code partagent la limite de dix par heure et par adresse IP (ENF-01).

Déploiement :

- [ ] `sudo tribe-menus-deploy --env recette --pr <numéro>` : la migration `0008` s'applique à la tribu de recette ; connexion depuis les deux appareils ; trois plats créés (« Chili con carne », « Chili végétarien », « Ratatouille »).

Planning de la semaine (R1, R5, D4, D8, D17) :

- [ ] L'onglet Planning ouvre les 7 jours à partir d'aujourd'hui, « Aujourd'hui » sur le jour courant, midi et soir pour chaque jour, cases vides en pointillé.
- [ ] Semaine suivante, puis précédente deux fois : titres des plages justes (à cheval sur deux mois si la date s'y prête) ; « Revenir à aujourd'hui » ramène à la plage du jour.
- [ ] Pastille de l'en-tête → écran Tribu ; « Se déconnecter » y est, et ramène à la connexion (D5) ; reconnexion.

Détail d'un repas (R2, R3, R4, Q21, D9, D12) :

- [ ] Toucher le repas de demain soir : « Aucun plat pour ce repas », propositions par ordre alphabétique ; « chili » ne garde que les deux chilis.
- [ ] Ajouter « Chili con carne » : il est servi pour la taille de la tribu, et n'est plus proposé (« Les plats déjà servis dans ce repas ne sont pas proposés… ») ; ajouter « Chili végétarien » : deux plats servis, total des parts juste.
- [ ] Parts réglées à 3 et 1 par le stepper, à petits pas rapides : retour au Planning, les pilules montrent 3 et 1 ; rouvrir : gardé.
- [ ] Retirer « Chili con carne » : il disparaît du repas, reste dans la bibliothèque ; l'ajouter de nouveau : taille de la tribu (R4).
- [ ] Un repas de la semaine précédente : ajout et parts fonctionnent (R5).

Taille de la tribu (T3) :

- [ ] Taille passée à 6 ; un plat ajouté ensuite est servi pour 6, ceux déjà servis gardent leurs parts.

Deux appareils (T1, D3, D15) :

- [ ] A ajoute un plat à un repas ; B, laissé en arrière-plan sur le Planning, revient au premier plan : le plat apparaît sans recharger la page.
- [ ] Le même repas ouvert sur les deux appareils : A ajoute « Ratatouille », puis B la choisit dans ses propositions encore périmées : message « « Ratatouille » est déjà servi dans ce repas. », et le repas à jour.
- [ ] A retire un plat ; B, sur le même repas sans l'avoir quitté, en change les parts : message « Ce plat a été retiré du repas entre-temps. », et le repas à jour.

Sans réseau (D13) :

- [ ] Mode avion, puis ajout d'un plat, réglage des parts, retrait : « Réseau nécessaire », rien n'est changé une fois le réseau revenu.

iPhone, s'il y en a un :

- [ ] Les étapes ci-dessus sous Safari, au moins la navigation, l'ajout et les parts ; si possible, les points reportés de Safari (connexion, bibliothèque, `304` sur `GET dishes` et `GET meal-plan`).

Retour arrière avec restauration (migration `0008`, ADR 0016, point 2) :

- [ ] `sudo tribe-menus-deploy --env recette --rollback` : refusé, avec la date de l'instantané et ce qui serait perdu.
- [ ] `sudo tribe-menus-deploy --env recette --rollback --restore-data` : `v0.2.0` revient, avec l'accueil provisoire ; les plats créés avant l'instantané sont là, le planning a disparu.
- [ ] De nouveau `sudo tribe-menus-deploy --env recette --pr <numéro>` : la migration repasse, le planning est vide, un plat s'ajoute à un repas.

Fin de séance :

- [ ] `sudo tribe-menus-check report`, remis avec les observations ; défauts éventuels corrigés par des PR ordinaires, la PR de release ensuite mise à jour avec `main`.

## Critères de validation

- `make ci` vert à chaque PR ; `pending.txt` ne garde de ce plan que les scénarios marqués « reste en attente » ci-dessus.
- En recette, sur téléphone : une semaine se remplit de plats de la bibliothèque, leurs parts se règlent, un plat se retire ; un second appareil voit le planning au retour sur l'application, et ses écritures croisées ne perdent rien sans le dire.
- En production, le développeur planifie ses vrais repas.

### Ce que ce plan ne prouve pas

- Que la modification d'un plat s'applique aux repas où il est servi (P2) : avec les listes de courses, qui calculent les quantités.
- La consultation hors ligne du planning : avec le plan hors-ligne.
- Le coût du rechargement complet d'un planning de plusieurs années (D1) : mesuré plus tard, le delta est laissé au plan hors-ligne.
- Le vrai Safari sur iPhone, si aucun n'est disponible en recette.

## Décisions

Le 2026-11-05, au cadrage, avec le développeur.

- **D1. Le planning est une collection chargée en entier**, `meal_plan`, servie par `internal/collection` selon l'ADR 0029 tel quel : pages de 200 plats servis, dans l'ordre des identifiants, gardées en entier dans une copie en mémoire, vérifiée par un `304`. Une tribu de quatre personnes sert de l'ordre de 700 plats par an, une centaine d'octets chacun : quelques dizaines de Ko par an, trois ou quatre pages. La navigation entre semaines ne coûte aucune requête, et le plan hors-ligne trouvera le planning comme la bibliothèque. Écartées : une fenêtre par semaine (`?from=…&to=…`), avec un ETag par période, qui demandait de préciser l'ADR 0029 ; une collection bornée dans le temps (trois mois passés), qui rendait les semaines plus anciennes vides ou demandait une seconde route.
- **D2. Un plat retiré puis ajouté de nouveau à un repas est un nouveau plat servi**, avec un nouvel identifiant, parti de la taille de la tribu (R4) : sa configuration est perdue (Q5). Les listes de courses le traiteront comme tel : une liste qui avait reçu l'ancien pourra recevoir le nouveau (Q22). Q32. Écarté : retrouver le plat servi d'avant, qui demandait de garder les plats servis retirés.
- **D3. Écritures croisées sur un repas**, sans révision : il n'y a pas de temps réel (Q6), et un stepper ne se prête pas à un refus. Q33.
  - **parts** : le dernier enregistrement l'emporte ;
  - **ajout d'un plat déjà servi** dans le repas (par un autre membre, entre-temps) : refusé, `409 dish_already_served`, avec les plats servis du repas à jour, que la copie prend et que l'écran montre avec « « … » est déjà servi dans ce repas. » ;
  - **parts ou retrait d'un plat servi qu'un autre membre a retiré** : `404 not_found` ; l'écran dit « Ce plat a été retiré du repas entre-temps. » pour les parts, rien pour un retrait (il est fait), et la copie se vérifie.
  - Écartée : une révision par repas et un bandeau, comme pour un plat (Q25), lourde pour une écriture qui ne fait rien perdre d'important.
- **D4. « Aujourd'hui » est la date de l'appareil**, lue par le navigateur ; le serveur ne connaît pas « aujourd'hui » : il stocke et accepte toute date réelle de 2000-01-01 à 2099-12-31, passée comme future (R5, Q2), sans fuseau (`AAAA-MM-JJ`). Deux membres dans des fuseaux différents voient chacun leur « aujourd'hui », sur le même planning. Q34, Q35. Écartée : un fuseau fixé par tribu (Europe/Paris), qui demandait un réglage et une horloge du serveur dans les écrans.
- **D5. « Se déconnecter » en bas de l'écran Tribu**, en bouton au contour rouge tomate, à la place qu'occupera « Quitter la tribu » dans `Doux-Tribu` ; l'accueil provisoire disparaît. Ferme la question ouverte de `bibliotheque` ; le plan des membres pourra la déplacer vers « Mes appareils ». Écartés : l'en-tête du Planning, chargé ; attendre le plan des membres, qui laissait l'application sans déconnexion.
- **D6. Preuves** : affichage et navigation (R1, R5) dans le navigateur, `@ui` ; écritures et refus (R2 à R4, R5 « repas passés », T1, T3) contre l'API. Le scénario « Un même plat ne peut pas être ajouté deux fois à un repas » est **scindé**, comme P4 au plan `bibliotheque` : le refus de l'API contre l'API, « ne m'est pas proposé » dans le navigateur. L'exemple « le planning » est **tiré** du plan du scénario « Toutes les données sont compartimentées », que godog juge en entier et que ses exemples listes, membres et sessions gardent en attente ; avec lui, un scénario d'accès croisé à un plat servi.
- **D7. Quatre lots, puis `v0.3.0` en production** : specs, design et conception ; domaine et API ; planning de la semaine ; détail d'un repas. Le planning est utilisable en production avant les listes de courses. Écartée : une seule release avec les listes.
- **D8. Pas de nouvelle maquette** : `Doux-Main` et `Doux-Repas` suffisent ; ce qu'elles ne montrent pas (titre d'une plage à cheval sur deux mois, « Revenir à aujourd'hui », chargement et erreur, bibliothèque vide dans un repas, ordre des propositions, messages) est écrit en interprétation dans `docs/design/README.md` au lot A, à valider à la revue. Le développeur peut les dessiner dans Design avant le lot C ou D s'il le souhaite.

Proposées le 2026-11-05, à confirmer à la revue du plan.

- **D9. Parts d'un plat ajouté : la taille de la tribu au moment de l'ajout, lue par le serveur.** `POST meal-plan` sans parts les prend dans la même transaction que l'ajout ; l'écran ne les envoie pas. T3 « La taille de la tribu fixe les parts des plats servis ajoutés ensuite » se prouve donc contre l'API, et un écran resté ouvert pendant un changement de taille ne sert pas l'ancienne. Différent de `bibliotheque`, D16, où l'écran montre les parts avant l'enregistrement ; ici, le plat est ajouté d'un geste et ses parts se règlent ensuite.
- **D10. Identifiants aléatoires** (128 bits, texte) pour les plats servis, comme pour les plats (`bibliotheque`, D10) : un identifiant d'une autre tribu ne désigne jamais un plat servi de la sienne.
- **D11. Plats d'un repas dans l'ordre d'ajout**, sur la case de repas comme dans le détail ; un plat retiré puis ajouté passe en dernier (D2). Écarté : l'ordre alphabétique, qui déplacerait un plat sous le doigt.
- **D12. Propositions d'un repas** : sans recherche, toute la bibliothèque par ordre alphabétique (Q30), sans les plats déjà servis (Q21) ; avec une recherche, les résultats de `searchDishes` (P3, mêmes règles que l'onglet Plats). Toucher une proposition l'ajoute aussitôt. Bibliothèque vide : « Aucun plat dans la bibliothèque », avec un lien vers « Nouveau » de l'onglet Plats. Pas de création d'un plat depuis le repas (aucune maquette).
- **D13. Pas de hors-ligne dans ce plan**, comme `bibliotheque`, D18 : ajout, parts et retrait affichent « Réseau nécessaire » sans réseau et ne changent rien ; le cache et la consultation hors ligne viennent avec le plan hors-ligne.
- **D14. Chemins** sous la tribu : la racine montre les 7 jours à partir d'aujourd'hui ; `?du=AAAA-MM-JJ` une autre plage, que le retour depuis un repas retrouve ; `repas/AAAA-MM-JJ/midi` et `repas/AAAA-MM-JJ/soir` le détail d'un repas. Une date illisible ou hors des bornes de D4 ramène à la racine.
- **D15. Code** : paquet `internal/mealplan` (logique pure derrière une interface `Store`, ADR 0002, 0008), adaptateur `MealPlanStore` dans `internal/storage`, comme `DishStore` ; collection `meal_plan` ; côté navigateur, `meal-plan.ts` (module pur), `planning.ts` (copie du planning sur `CachedCollection`), écrans `planning-screen.ts` et `meal-screen.ts`. Le Planning et le détail d'un repas lisent aussi la copie de la bibliothèque (`library.ts`) pour les noms, couleurs et parts de référence des plats : les deux copies sont vérifiées ensemble.
- **D16. `make seed`** ajoute à la tribu `demo` les plats servis de `Doux-Main` (lundi à jeudi), à partir du jour du seed, pour le développement et les états des écrans ; « Restes », absent des huit plats de la bibliothèque `demo`, est remplacé par l'un d'eux. Rien en mode serveur.
- **D17. Changement de jour** : au retour au premier plan, si la date de l'appareil a changé et que le Planning montre la plage d'aujourd'hui (la racine), il passe à la nouvelle ; une autre plage (`?du=`) reste affichée.
- **D18. Ni auteur ni historique des plats servis**, comme `bibliotheque`, D23 : T2, mêmes droits pour tous ; minimisation des données.
- **D19. Retrait d'un plat servi inconnu : `404 not_found`**, comme pour un identifiant d'une autre tribu (ENF-02, « la réponse est la même que pour un identifiant qui n'existe pas ») ; l'écran le traite comme un retrait fait (D3).
- **D20. Pas de nouvel ADR** : le plan applique les ADR 0027 (recherche dans le navigateur, sur la copie de la bibliothèque) et 0029 (collection chargée en entier, D1) sans les changer. Ses décisions de domaine (D2 à D4) vont dans `fonctionnalites.md`.

## Questions ouvertes

- **Croissance du planning** (D1) : à mesurer après quelques mois de production ; si le rechargement complet coûte, le delta d'une collection (`?since=<version>`, plan `bibliotheque`, questions ouvertes) le règle, au plan hors-ligne ou avant.
- **Plat servi et listes de courses** : ce qu'une liste retient d'un plat servi pour savoir qu'il a changé (C8 : parts modifiées, plat retiré) — révision du plat servi, ou parts gardées dans la provenance ; à trancher au plan des listes. D2 fixe déjà l'identité.
- **Mise en page d'ordinateur** : le Planning reste une colonne de téléphone, comme la bibliothèque (point reporté de la feuille de route).

## Notes d'exécution

**Lot A** (2026-11-05, session cloud, `feature/specs-planning-des-repas`), à valider à la revue :

- **D9 à D20** appliquées telles que proposées ; la fusion de la PR du plan ne valait pas revue explicite, d'où la confirmation demandée dans la PR du lot.
- **Scénarios au-delà de la liste de l'étape 2** : « Les plats d'un repas sont dans l'ordre où ils ont été ajoutés » (Q35, D11, avec le cas du plat retiré puis remis, qui passe en dernier), et deux plans de scénario pour les bornes des dates (2000 à 2099, D4), en plus de ceux des parts. L'étape « contient, dans cet ordre : » est nouvelle ; « contient : » ne juge pas l'ordre. 13 titres nouveaux dans `acceptance/pending.txt`, vérifiés contre les `.feature` (les outils Go ne s'installent pas dans la session cloud : `make ci` reste à lancer dans le Dev Container avant la fusion).
- **`conventions-gherkin.md`** : le premier du mois s'écrit `1er` (« le samedi 1er janvier 2000 »).
- **Glossaire** : en plus des termes de « Contexte et contraintes », rang (*rank*, champ de l'API) et aujourd'hui (*today*, `today()` de `meal-plan.ts`).
- **Design, interprétations** (D8) : année dans le titre d'une plage hors de l'année en cours ; « Revenir à aujourd'hui » à la place de « À partir d'aujourd'hui » ; jours et cases affichés une fois les deux copies chargées ; retrait sans confirmation ; « Tous les plats de la bibliothèque sont déjà dans ce repas. » ; bandeaux d'avertissement, comme pour la modification croisée d'un plat, pour les deux messages de Q33.
- **« Se déconnecter »** (D5) : bouton au contour rouge tomate, comme sur l'accueil provisoire. Dans `Doux-Tribu`, « Quitter la tribu » est un texte rouge tomate dans un contour neutre `#E7E1EE`, à côté de « Mes appareils » ; à trancher à la revue si le bouton doit plutôt suivre la maquette.

**Lot B** (2026-11-05, Dev Container, `feature/planning-api`), à valider à la revue :

- **Interfaces du domaine** un peu différentes du diagramme de la conception, qui est mis à jour : `Tx.NextRank` donne le rang d'un ajout (le plus grand de la tribu, plus un : un rang peut resservir après le retrait du dernier plat ajouté, l'ordre d'un repas reste juste) ; `Tx.SetServings` rend le plat servi modifié (`UPDATE … RETURNING`) et `Tx.Delete` une erreur `ErrNotFound`, au lieu d'un booléen ; `Store.Version` sans nom de collection, le planning n'en a qu'une.
- **Bornes vérifiées deux fois** : par `ParseDate` (date réelle, format exact) et par la table (`CHECK`, dont `date(date) IS date` contre un 30 février), comme les parts ; l'index unique (date, moment, plat) garde Q21 même pour une écriture qui ne passerait pas par le domaine.
- **`POST meal-plan` accepte des parts** (1 à 50), que l'écran n'envoie pas (D9) : l'API de la conception les prévoit, les étapes godog s'en servent pour poser « servi pour 3 parts ». Un réglage des parts à la même valeur augmente quand même la version : le dernier enregistrement l'emporte (D3), sans comparaison.
- **Revue de la PR #87, point 3** : `TestServeSetAndRemove` prouve le `404 not_found` d'un retrait et d'un réglage des parts sur un plat servi déjà retiré de la même tribu.
- **Étapes godog** : « j'ajoute le plat "…" au repas du … » crée le plat s'il manque à la bibliothèque, pour T1 (« Ratatouille », absente du contexte) ; « nous sommes le … » règle l'horloge du scénario, que le serveur ne lit pas (D4) ; « je ne vois aucun élément de la tribu "…" » vérifie aussi le planning consulté (identifiants des plats servis et des plats de l'autre tribu). 21 titres retirés de `acceptance/pending.txt` ; restent de ce plan les quatre scénarios `@ui` (lots C et D) et ceux marqués « reste en attente ».
- **`make seed`** : « Pizza maison » remplace « Restes » le premier jour à midi (D16) ; les dates partent du jour du seed, dans le fuseau du poste.

**Revue de la PR #87**, points à reprendre plus tard :

- **Plages aux bornes de 2000 à 2099** (point 2, lot C) : `?du=2099-12-28` montre des jours de 2100, dont le repas serait refusé par l'API (`400 invalid_date`, affiché « Réseau nécessaire ») ; « Semaine suivante » mène alors à une date hors bornes, qui ramène à aujourd'hui sans le dire ; de même avant 2000. Cas très improbable. À trancher au lot C, avec un test `node:test` dans `meal-plan.ts` ; proposition du relecteur : flèche inactive quand la plage voisine sortirait des bornes, cases hors bornes inactives.
- **Retrait d'un plat servi déjà retiré** (point 3, Q33, D19) : aucun scénario ne prouve le `404 not_found` pour un plat servi retiré de la même tribu, seul l'accès d'une autre tribu le fait. Au lot B, un test du handler le prouve ; au lot D, un état d'écran sur l'API simulée montre que le plat disparaît sans message.

**Revue de la PR #88**, point à reprendre plus tard :

- **Après un `409 dish_already_served`, la copie se vérifie** (point 1, lot D, étape 19) : la réponse porte le repas à jour mais pas les versions ; si `planning.ts` ne faisait que prendre ce repas, la copie garderait une version plus ancienne que son contenu (sans perte : la vérification suivante rechargerait tout). `planning.ts` prend le repas, puis vérifie la copie, comme après un `404` ; un état d'écran sur l'API simulée le montre.
