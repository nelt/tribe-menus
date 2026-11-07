# Plan : listes de courses

- **Nom** : `courses`
- **Date** : 2026-11-06
- **Statut** : en cours (lot D). Décisions D1 à D3 prises par le développeur le 2026-11-06 au cadrage ; D4 à D24 proposées, confirmées le 2026-11-07 par la fusion de la PR du plan (#92) et l'accord du développeur au lancement du lot A, D3 retenue telle qu'écrite.

## Objectif

Troisième plan du métier : un membre tient depuis son téléphone les listes de courses de sa tribu, y ajoute des articles à la main et les courses des repas d'une période, calculées à partir du planning et de la bibliothèque, les coche au fil des courses et déclare les courses faites. Avec la bibliothèque (`v0.2.0`) et le planning (`v0.3.0`), c'est l'application viable à montrer (feuille de route, décision du 2026-11-05). Le recalcul d'une liste périmée et la sélection d'articles suivent dans le plan `courses-suite` (D1).

À la fin du plan :

- l'onglet **Courses** est actif et ouvre la liste principale, d'après `Doux-Liste` (C1) ; chaque tribu a sa liste « Courses », y compris celles qui existent déjà (D14) ;
- le **choix des listes**, d'après `Doux-ChoixListes`, crée, renomme, supprime une liste vide non principale et désigne la liste principale (C1, C2, C12) ;
- on y **ajoute des articles à la main**, avec l'autocomplétion du référentiel, et on les retire (C3) ;
- on y **ajoute les courses des repas** d'une période, d'après `Doux-CoursesRepas` : quantités ajustées aux parts, additionnées, converties et arrondies, plats servis déjà reçus ignorés (C4, P2) ; une ligne calculée se déplie vers ses plats et repas (C5) ;
- on **coche** les articles et on **déclare les courses faites** (C6, C11) ;
- les listes sont partagées entre les membres de la tribu et seulement eux (T1, ENF-02) ;
- la version `v0.4.0` est publiée et déployée en production.

La conception (modèle, calcul, découpage du code, API, échanges) est dans [`docs/conception/listes-de-courses.md`](../conception/listes-de-courses.md), avec ses diagrammes de classes et de séquence (`bibliotheque`, D13). Ce plan dit dans quel ordre on la réalise et ce qui reste à décider.

## Contexte et contraintes

- **À lire avant de commencer** : ADR 0002, 0003 (point 7, quantités en unités de base), 0004 (points 3, 6, 7 et 9), 0005, 0008, 0012 (point 6), 0027, 0028, 0029 ; `docs/specs/fonctionnalites.md` (concepts Liste de courses, Liste principale, Article, Plats servis reçus ; « Ajout des courses d'une période », « Règle de calcul », « Unités, conversions et arrondis » ; C1 à C12, Q10 à Q18, Q22 à Q24, Q26, Q31), `glossaire.md` (« Listes de courses »), `conventions-gherkin.md`, `features/liste-courses.feature`, `tribu.feature`, `compartimentage-tribus.feature`, le scénario P2 de `bibliotheque-plats.feature` ; `docs/design/README.md` (« Listes de courses », « Navigation ») et les maquettes `Doux-Liste`, `Doux-ArticleMain`, `Doux-ChoixListes`, `Doux-CoursesRepas` ; les documents de conception de la bibliothèque et du planning, dont ce plan réutilise les quantités, le référentiel, la copie des collections et le planning.
- **Après `planning`** : sa recette et `v0.3.0` sont en cours au moment de ce plan. Rien des lots B à E n'est fusionné avant l'étiquette `v0.3.0` (D21), comme `bibliotheque`, D17 ; la PR de ce plan et le lot A, documentation seule, peuvent l'être avant. Les archives de PR de ce plan ne vont en recette qu'une fois la recette de `planning` close. *Le 2026-11-07 : recette de `planning` close, `v0.3.0` posée et en production ; les lots B à E peuvent être fusionnés, leurs archives aller en recette.*
- **Où se fait le travail** : le lot A peut se faire depuis une session cloud ; les lots B à E dans le Dev Container (la session cloud ne peut ni compiler ni tester le projet).
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, avec `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`courses/B`) et les stories ; `make ci` avant de pousser.
- **Aucune dépendance nouvelle**, ni en Go ni en npm : le calcul se fait en entiers, en unités de base (ADR 0003, point 7) ; le calendrier de période est écrit dans le projet, avec `Date`.
- **Vocabulaire** : le code suit `glossaire.md` (`shopping list`, `main list`, `item`, `computed item`, `manual item`, `item source`, `contributed quantity`, `received served dishes`, `add meal shopping`, `check out`) ; termes nouveaux ajoutés au lot A, avant le code : article sans quantité (*unquantified item*), période (*period*), choix des listes (*list picker*), copie des listes (*cached shopping lists*), destination des courses (*target list*).
- **Une migration** de la base de la tribu (`0009`), qui crée aussi la liste principale de chaque tribu existante (D14) : revenir en arrière sur `v0.3.0` demandera la restauration de l'instantané (ADR 0012, point 6 ; plan `recette`, D7). Le registre et la base de limitation ne changent pas.
- **Hors-ligne hors du plan** (D2) : une liste affichée reste lisible tant que l'onglet reste ouvert, la copie étant en mémoire ; toute écriture, coche comprise, demande le réseau et affiche « Réseau nécessaire » sans lui. C'est un écart temporaire à Q7 et Q16 : le plan hors-ligne vient après `courses-suite`, et avant d'inviter la famille (feuille de route).
- **Rien d'une autre tribu** (ENF-02) : chaque requête lit la seule base de la tribu de sa session (ADR 0003) ; les identifiants des listes et des articles sont aléatoires (D10).
- **Un texte saisi nouveau** : le nom d'une liste, affiché par Lit (ADR 0004, point 9) et soumis aux règles des noms (D30 de `bibliotheque`, D12). Les articles ajoutés sont des ingrédients du référentiel, aux règles déjà en place (Q13, Q31).

### Scénarios couverts

| Fichier | Scénario | Preuve | Lot |
| --- | --- | --- | --- |
| `liste-courses.feature` | C1 : une tribu naît avec une liste principale ; changer de liste principale ; il y a toujours exactement une liste principale | API | B |
| | C1 : l'onglet Courses ouvre la liste principale | navigateur, `@ui` | D |
| | C2 : créer une liste vide ; le nom d'une liste est obligatoire ; renommer une liste | API | B |
| | C2 : ordre des listes | navigateur, `@ui` | D |
| | C3 : ajouter un article à la main ; sa quantité est facultative ; un nouvel article enrichit le référentiel ; retirer un article ajouté | API | B |
| | C4 : période et liste par défaut | navigateur, `@ui`, horloge au 5 octobre 2026 | E |
| | C4 : la date de fin ne peut pas précéder la date de début, **scindé** au lot A (D16) : le refus de l'API | API | C |
| | C4 : … et le bouton inactif avec son message | navigateur, `@ui` | E |
| | C4 : toutes les règles de calcul (liste choisie, période, parts, additions, conversions, familles, arrondis, plat sans ingrédient, articles déjà présents, coche retirée, article ajouté non fusionné, plats servis déjà reçus, achetés, reçus par deux listes) | API | C |
| | C5 : un article indique le nombre de plats servis dont il provient | API | C |
| | C5 : déplier un article ; un article ajouté ne se déplie pas | navigateur, `@ui` | E |
| | C6 : cocher un article | API | B |
| | C11 : les trois scénarios (courses faites, liste utilisable ensuite, planning intact) | API | B (C pour « ne touchent pas au planning ») |
| | C12 : les trois scénarios (liste vide supprimée, liste non vide, liste principale) | API | B |
| | Scénarios nouveaux du lot A : nom d'une liste (même nom, espaces, 40 caractères, refus) et 20 listes (D12), 300 articles à la main (D12), deux articles ajoutés du même ingrédient (D23), ordre des articles (D13), coche croisée et coche d'un article retiré (D8), courses faites croisées (D9), suppressions croisées et écritures sur une liste supprimée (D11), retrait d'un article déjà retiré (D11) | API | B ; *« Une écriture sur une liste supprimée entre-temps est refusée » : C, pour son exemple des courses des repas (notes du lot B)* |
| | Scénarios nouveaux du lot A : période refusée (fin avant début, plus de 31 jours, hors de 2000 à 2099) et acceptée (un jour, 31 jours, D12), masses au gramme (D7), ingrédient sans quantité (D3), 300 articles par les courses des repas (D12) | API | C |
| | C7, C8, C9, C10 | **restent en attente** : hors-ligne (D2), `courses-suite` (D1) | — |
| `bibliotheque-plats.feature` | P2 : la modification d'un plat s'applique aux repas où il est servi | API | C |
| `tribu.feature` | T1 : « Les listes de courses sont partagées entre membres » (nouveau, lot A) | API | B |
| `compartimentage-tribus.feature` | ENF-02 : « Les listes de courses sont propres à la tribu », **tiré** au lot A de « Toutes les données sont compartimentées » (D16) ; « Une liste d'une autre tribu ne peut être ni lue, ni modifiée, ni supprimée », « Un article d'une autre tribu ne peut être ni coché ni retiré » (nouveaux) | API | B |
| | ENF-02 : toutes les données sont compartimentées | **reste en attente** : ses exemples membres et sessions | — |
| `authentification.feature` | ENF-01 : les modifications autres que les listes de courses nécessitent le réseau | **reste en attente** : plan hors-ligne | — |

## Étapes

### Lot A : specs, design et conception (`feature/specs-listes-de-courses`)

Documentation seule, une fois ce plan revu et ses décisions confirmées.

- [x] **1. `fonctionnalites.md`** : Q26 fermée et décisions nouvelles, à la suite de Q35 : Q36, ingrédient sans quantité dans les courses (D3) ; Q37, arrondis des masses et des volumes et quantité apportée (D7) ; Q38, limites (D12) ; Q39, écritures croisées sur une liste (D9, D11) ; Q40, ordre des articles (D13). Concept « Article » accordé (article sans quantité) ; C4 précisé (période de 31 jours au plus).
- [x] **2. Scénarios**, selon `conventions-gherkin.md` : scission de « La date de fin ne peut pas précéder la date de début » en un refus contre l'API et l'écran (`@ui`) ; tags `@ui` sur « L'onglet Courses ouvre la liste principale », « Ordre des listes », « Période et liste par défaut », « Déplier un article… », « Un article ajouté ne se déplie pas » ; scénarios nouveaux contre l'API (tableau ci-dessus) ; T1 pour les listes ; dans `compartimentage-tribus.feature`, l'exemple « les listes de courses » sorti en un scénario « Les listes de courses sont propres à la tribu », et l'accès croisé à une liste (D16). Les nouveaux titres entrent dans `acceptance/pending.txt`.
- [x] **3. `glossaire.md`** : les termes nouveaux de « Contexte et contraintes ».
- [x] **4. `docs/design/README.md`** : l'onglet Courses devient actif (« Écarts » du plan `bibliotheque` mis à jour) ; interprétations de ce que les maquettes ne montrent pas (D17) : ligne sans quantité, chargement et erreur de réseau, liste supprimée entre-temps, refus des limites, « N plats servis ignorés » après un ajout, ordre des articles ; ce que ce plan laisse de côté dans les maquettes jusqu'à `courses-suite` et au plan hors-ligne (bandeau de liste périmée, pastille « Repas modifiés », « Sélectionner » et mode sélection, bandeau hors ligne) ; chemins (D15) ; messages.
- [x] **5. Document de conception** : accordé aux specs, au glossaire et au design du lot.
- [x] **6. Feuille de route** : plan « en cours ».

### Lot B : listes et articles, domaine et API (`feature/listes-api`)

- [x] **7. Migration `0009`** de la base de la tribu : tables `shopping_lists` (identifiant aléatoire, nom, rang de création, liste principale marquée par un index unique partiel), `shopping_items` (identifiant aléatoire, liste, ingrédient, quantité en unités de base et unité facultatives, coché, ajouté ou calculé), `shopping_item_sources` (provenance : plat servi, plat, date, moment, parts, quantité apportée, sans clé étrangère vers `served_dishes`, D5), `shopping_received` (liste, plat servi, plat, parts servies, parts de référence du plat, D5) et `shopping_received_lines` (liste, plat servi, rang, ingrédient, quantité et unité facultatives, D5), sans clé étrangère vers `served_dishes` ; la liste « Courses », principale, insérée par la migration (D14) ; sans auteur ni date (D22) ; collection `shopping_lists` déclarée dans `storage.Collections`, avec les déclencheurs de ses cinq tables (ADR 0029, point 2), que le test de schéma vérifie. Requêtes sqlc, `make generate`.
- [x] **8. Règles pures de `internal/shopping`**, tests en tableaux d'abord : nom d'une liste (règles des noms, 40 caractères, D12), limites (20 listes, 300 articles par liste), ordre des listes (principale, puis création) et des articles (D13).
- [x] **9. `shopping.Lists`** derrière une interface `Store`, tests d'abord : créer, renommer, désigner la principale, supprimer (liste non vide ou principale refusées, D11) ; article ajouté (ingrédient du référentiel trouvé ou créé par son nom normalisé, quantité et unité facultatives selon Q31, unité par défaut fixée si l'ingrédient n'en a pas, Q20), retrait d'un article ajouté (un article calculé ne se retire pas) ; coche (dernier enregistrement gagnant) ; courses faites sur les articles vus cochés (D9) ; adaptateur `ShoppingStore` dans `internal/storage`, écritures par `writeCollections`.
- [x] **10. API** : `GET shopping-lists` par pages, servie par `internal/collection` ; `POST shopping-lists`, `PUT shopping-lists/{id}/name`, `PUT shopping-lists/main`, `DELETE shopping-lists/{id}` ; `POST shopping-lists/{id}/items`, `DELETE shopping-lists/{id}/items/{item}`, `PUT shopping-lists/{id}/items/{item}/checked` ; `POST shopping-lists/{id}/checkout` ; quantités en texte avec leur unité, au format des lignes des plats, un article calculé dans la plus petite unité de sa famille (conception, « API » ; revue #92, point 2) ; réponses d'écriture avec la liste à jour et les versions des collections touchées ; `404` identique pour une liste ou un article inconnu, supprimé entre-temps ou d'une autre tribu ; corps limité à 4 Kio, sauf celui de `checkout`, limité à 16 Kio pour ses 300 identifiants (revue #92, point 1), avec un test de `checkout` à 300 identifiants. Codes d'erreur dans la conception.
- [x] **11. Étapes godog** des scénarios « API » du lot B, retirés de `pending.txt`.

### Lot C : courses des repas, calcul et API (`feature/courses-des-repas-api`)

- [x] **12. Calcul pur dans `internal/shopping`**, tests en tableaux d'abord, à partir des exemples de la spec : quantité d'une provenance (quantité du plat × parts servies ÷ parts de référence, arrondie à l'unité de base supérieure), addition par ingrédient et famille d'unités, arrondi après l'addition (pièces, cuillères, pincées à l'entier supérieur ; masses au gramme et volumes au millilitre supérieurs, D7), lignes sans quantité (D3), versement dans les articles calculés existants, décoche d'un article dont la quantité augmente, articles ajoutés jamais fusionnés. Table de cas dans `internal/shopping/testdata/`, que le lot E reprend pour l'affichage des quantités apportées.
- [x] **13. Ajout des courses des repas** dans `shopping.Lists` : période lue (dates de 2000 à 2099, fin au plus tôt le jour du début, 31 jours au plus, D12), plats servis de la période lus dans la même transaction que les plats et leurs lignes (P2 : l'état actuel du plat), plats servis déjà reçus ignorés et comptés (Q22), nouveaux plats servis reçus enregistrés avec leurs parts, les parts de référence du plat et ses lignes au moment du calcul, même sans ingrédient (D5) ; limite de 300 articles vérifiée sur le résultat, refus de toute l'opération au-delà.
- [x] **14. API** : `POST shopping-lists/{id}/meal-shopping` (200 avec la liste à jour, le nombre de plats servis ajoutés et ignorés, les versions ; 400 `invalid_period`, `period_too_long` ; 404 ; 409 `too_many_items`) ; provenances dans `GET shopping-lists`.
- [x] **15. Étapes godog** des scénarios « API » du lot C, dont P2 et « Les courses faites ne touchent pas au planning », retirés de `pending.txt`.
- [x] **16. `make seed`** : à la tribu `demo`, les courses de la semaine du seed dans « Courses », quelques articles ajoutés, une liste « Marché » (D20). La tribu de recette n'en reçoit pas.

### Lot D : onglet Courses et choix des listes (`feature/onglet-courses`)

- [x] **17. `web/src/shopping.ts`**, module pur testé par `node:test` : ordre des listes et des articles (D13), libellé d'un article (quantité affichée dans la plus grande unité de sa famille qui donne au moins 1, règles d'affichage des quantités du design), compte « N plats », état de « Courses faites » (combien partent, combien restent), raisons d'une suppression impossible.
- [x] **18. Copie des listes** : `web/src/shopping-lists.ts`, une copie générique de la collection `shopping_lists` (`collections.ts`), vérifiée à chaque visite de l'onglet Courses et à chaque retour au premier plan, mise à jour par la réponse d'une écriture ; vérifiée avec la bibliothèque pour les noms et couleurs des plats des provenances (lot E). Rien n'est écrit sur l'appareil.
- [x] **19. Écran d'une liste** d'après `Doux-Liste` et `Doux-ArticleMain` : en-tête (nom de la liste, pastille « principale », ouverture du choix des listes), articles (case à cocher, nom, quantité, pastille « ajouté » et croix de retrait), cochés en bas et barrés, « Ajouter un article à la main » (ingrédient en combobox sur le référentiel, comme l'écran d'un plat ; quantité et unité facultatives), « Courses faites » et sa feuille de confirmation, bandeau vert ensuite ; liste vide ; chargement et erreur de réseau ; « Réseau nécessaire » pour toute écriture (D2) ; liste supprimée entre-temps (D11). L'onglet Courses devient actif.
- [x] **20. Choix des listes** d'après `Doux-ChoixListes` : listes dans l'ordre de C2, nombre d'articles et part cochée, « Nouvelle liste », feuille « ⋮ » (renommer, définir comme principale, supprimer, inactif avec sa raison) ; refus des limites (D12).
- [x] **21. C1 et C2 sous Chromium et WebKit** (onglet Courses, ordre des listes), retirés de `pending.txt`.

### Lot E : courses des repas à l'écran (`feature/ecran-courses-des-repas`)

- [x] **22. `shopping.ts`** : période par défaut (de demain à J+7, date de l'appareil, Q2, Q34), grille d'un mois (lundi en premier), jours ayant des repas, nombre de jours, plats servis de la période déjà reçus par la liste choisie (« … seront ignorés », Q22), lignes dépliées d'un article (repas, plat, parts, quantité apportée), avec leurs tests.
- [x] **23. Écran des courses des repas** d'après `Doux-CoursesRepas`, au chemin `courses/repas` (D15) : calendrier de période (bornes, jours intermédiaires, aujourd'hui, points des repas), message et bouton inactif quand la fin précède le début ou que la période dépasse 31 jours, destination (feuille de choix, pastille « principale »), ligne des plats servis déjà reçus ; « Ajouter à « … » » revient à la liste, avec un bandeau qui dit combien de plats servis ont été ajoutés et ignorés ; « Ajouter les courses des repas » sur l'écran d'une liste.
- [ ] **24. Dépliage d'un article calculé** (C5) : pastille « N plats » et chevron, lignes dépliées d'après `Doux-Liste` (initiale du plat dans sa couleur, plat et parts, jour et moment aux couleurs midi et soir, quantité apportée) ; un article ajouté ne se déplie pas.
- [ ] **25. C4 et C5 sous Chromium et WebKit**, horloge de Playwright au lundi 5 octobre 2026, retirés de `pending.txt`.

### Recette et release

- [ ] **26. Recette** sur l'archive de la PR de release `release/v0.4.0` (`RELEASING.md`, étape 6), qui contient tout le plan : check-list « Séance de recette » ci-dessous.
- [ ] **27. Release `v0.4.0`** selon `RELEASING.md`, déployée en production.
- [ ] **28. Clôture** : statut « terminé » ; feuille de route (plan terminé, `courses-suite` prêt à cadrer, points reportés à jour) ; document de conception ; `CLAUDE.md` (stack : paquet `internal/shopping`, version en service) ; `CHANGELOG.md`.

### Séance de recette (étape 26)

Par le développeur, sur l'archive de la PR `release/v0.4.0`, CI verte (artefact gardé trois jours, sinon `gh run rerun <identifiant>`). Deux appareils : Android (Chrome) et un second, iPhone si possible, sinon un ordinateur. La tribu de recette a des plats et un planning laissés par la recette de `planning` ; sinon, créer « Chili con carne » (500 g de bœuf haché, 1 oignon, sel sans quantité, pour 4 parts) et « Ratatouille », et les servir dans la semaine. Les demandes de code partagent la limite de dix par heure et par adresse IP (ENF-01).

Déploiement :

- [ ] `sudo tribe-menus-deploy --env recette --pr <numéro>` : la migration `0009` s'applique ; l'onglet Courses est actif et ouvre « Courses », vide, marquée « principale » (C1, D14).

Listes (C1, C2, C12, D11, D12) :

- [ ] Choix des listes : créer « Marché », puis « Drive » ; ordre « Courses », « Marché », « Drive » ; « Drive » devient principale : elle passe en tête et l'onglet l'ouvre ; renommer « Marché » en « Marché du samedi ».
- [ ] Une liste sans nom, ou un nom de plus de 40 caractères : refusés avec leur message.
- [ ] « Supprimer la liste » inactif sur la principale et sur une liste non vide, avec la raison ; actif et sans confirmation sur une liste vide.
- [ ] Remettre « Courses » comme principale pour la suite.

Articles à la main et coches (C3, C6, C11, D9) :

- [ ] Ajouter « pain » pour 2 pièces, « lessive » sans quantité, puis un ingrédient nouveau : il est proposé ensuite dans l'écran d'un plat (C3).
- [ ] Cocher « pain » : barré, en bas de la liste ; décocher, cocher de nouveau.
- [ ] « Courses faites » : la feuille dit 1 article retiré, 1 restant ; « lessive » reste, non cochée ; bandeau vert ; retirer « lessive » par sa croix.

Courses des repas (C4, C5, P2, D3, D7) :

- [ ] « Ajouter les courses des repas » : période de demain à J+7, liste « Courses » ; points sous les jours ayant des repas ; date de fin avant la date de début : message et bouton inactif.
- [ ] Ajouter : les quantités sont celles attendues (bœuf haché selon les parts, oignon arrondi à l'entier, « sel » en ligne sans quantité) ; bandeau « N plats servis ajoutés ».
- [ ] Toucher « bœuf haché » : les plats et repas concernés, avec leurs parts et la quantité apportée ; toucher de nouveau : replié. « pain » ne se déplie pas.
- [ ] Ajouter de nouveau la même période : rien ne change, « N plats servis déjà reçus ignorés ».
- [ ] Servir un plat de plus dans la période, ajouter de nouveau : seule sa part s'ajoute ; un article coché dont la quantité augmente est décoché.
- [ ] Modifier la quantité d'un ingrédient d'un plat servi plus tard dans la semaine, puis ajouter les courses de ce seul jour à « Marché » : la quantité nouvelle (P2).
- [ ] Une période de plus de 31 jours : message et bouton inactif.

Deux appareils (T1, D9, D11) :

- [ ] A coche un article ; B, laissé en arrière-plan sur la liste, revient au premier plan : la coche apparaît sans recharger.
- [ ] B ouvre « Courses faites » (feuille ouverte), A coche un autre article, B confirme : seul l'article que B voyait coché part ; l'autre reste, coché.
- [ ] A ajoute un article à « Marché du samedi » pendant que B, sur le choix des listes resté ouvert sur une liste vue vide, la supprime : refus, « Cette liste n'est plus vide. », la liste à jour.
- [ ] A supprime une liste vide que B a ouverte ; B y ajoute un article : « Cette liste a été supprimée entre-temps. », retour à la liste principale.

Sans réseau (D2) :

- [ ] Mode avion sur une liste affichée : elle reste lisible ; cocher, ajouter un article, « Courses faites » : « Réseau nécessaire », rien n'est changé une fois le réseau revenu.

iPhone, s'il y en a un :

- [ ] Les étapes ci-dessus sous Safari, au moins l'onglet, l'ajout des courses des repas, la coche et les courses faites ; si possible, les points reportés de Safari.

Retour arrière avec restauration (migration `0009`, ADR 0016, point 2) :

- [ ] `sudo tribe-menus-deploy --env recette --rollback` : refusé, avec la date de l'instantané et ce qui serait perdu.
- [ ] `sudo tribe-menus-deploy --env recette --rollback --restore-data` : `v0.3.0` revient, onglet Courses inactif ; le planning d'avant l'instantané est là.
- [ ] De nouveau `sudo tribe-menus-deploy --env recette --pr <numéro>` : la migration repasse, « Courses » existe, vide.

Fin de séance :

- [ ] `sudo tribe-menus-check report`, remis avec les observations ; défauts éventuels corrigés par des PR ordinaires, la PR de release ensuite mise à jour avec `main`.

## Critères de validation

- `make ci` vert à chaque PR ; `pending.txt` ne garde de ce plan que les scénarios marqués « reste en attente » ci-dessus.
- En recette, sur téléphone : une liste reçoit les courses de la semaine, justes au gramme près, et des articles à la main ; on coche, on déclare les courses faites ; un second appareil voit les coches au retour sur l'application, et les écritures croisées ne perdent rien sans le dire.
- En production, le développeur fait ses vraies courses avec la liste.

### Ce que ce plan ne prouve pas

- Le recalcul d'une liste périmée (C8) et la sélection d'articles (C9, C10) : `courses-suite`, sur le modèle posé ici (D5).
- Les coches et les ajouts sans réseau (C7, Q18) : plan hors-ligne.
- Le vrai Safari sur iPhone, si aucun n'est disponible en recette.

## Décisions

Le 2026-11-06, au cadrage, avec le développeur.

- **D1. Deux plans.** `courses` : listes, articles à la main, courses des repas, coches, courses faites, suppression (C1 à C6, C11, C12), puis `v0.4.0`. `courses-suite` ensuite : liste périmée et recalcul (C8), sélection, déplacement et nouvelle liste à partir d'une sélection (C9, C10), puis `v0.5.0`. Le modèle de ce plan prévoit le recalcul : provenances gardées avec ce qu'elles ont apporté (D5). Écartés : un plan et deux releases ; un plan et une release, plus long avant la production.
- **D2. Pas de hors-ligne dans ce plan**, comme `bibliotheque`, D18, et `planning`, D13 : une liste affichée reste lisible tant que l'onglet reste ouvert (copie en mémoire) ; cocher, ajouter, retirer, déclarer les courses faites et toute action sur les listes affichent « Réseau nécessaire » sans réseau et ne changent rien. Les scénarios C7 restent en attente. Écartés : une file des coches en mémoire, perdue au rechargement ; C7 complet (IndexedDB, file persistante, rejeu Q18), qui empiète sur le plan hors-ligne.
- **D3. Un ingrédient sans quantité devient un article calculé sans quantité.** Q36, ferme Q26. Une seule ligne par ingrédient sans quantité, distincte des lignes avec quantité du même ingrédient (comme une famille d'unités à part) ; ses provenances sont dépliables (C5), sans quantité apportée ; jamais arrondi. Un article sans quantité qui gagne une provenance n'est pas décoché : sa quantité n'augmente pas (« il faut en racheter » ne s'applique pas ; confirmé le 2026-11-07). Écartés : l'ignorer, ce qui fait oublier de vérifier le placard ; le rattacher à une ligne avec quantité du même ingrédient, qui mélange deux sens.

Proposées le 2026-11-06, à confirmer à la revue du plan.

- **D4. Les listes sont une collection chargée en entier**, `shopping_lists`, servie par `internal/collection` selon l'ADR 0029 tel quel. Un élément est une liste avec ses articles, leurs provenances et ses plats servis reçus, comme un plat porte ses lignes ; pages de 20 listes, dans l'ordre des identifiants. Une coche fait recharger toutes les listes sur les autres appareils : quelques dizaines de Ko au plus (D12), acceptable sans temps réel (Q6) ; le delta reste au plan hors-ligne. Écartées : une collection par liste, qui demandait de préciser l'ADR 0029 ; articles et listes en deux collections, qui obligeait à les recoudre dans le navigateur.
- **D5. Une liste garde ce qu'elle a reçu de chaque plat servi** (revue #92, point 3), pour que `courses-suite` puisse dire qu'elle est périmée (C8) et la recalculer sans recréer ce qui a été acheté. Deux enregistrements, sans clé étrangère vers `served_dishes` : un plat servi retiré du planning (R4) laisse les siens lisibles jusqu'au recalcul.
  - **Provenance** d'un article calculé : le plat servi, le plat, la date, le moment, les parts et la quantité apportée, au moment du calcul. Le dépliage (C5) la lit telle quelle.
  - **Plat servi reçu** (Q22) : le plat servi, le plat, les parts servies, **les parts de référence du plat** (revue #92, point 5) et **les lignes du plat au moment du calcul** (ingrédient et quantité avec son unité, ou sans quantité), y compris celles dont les articles sont partis depuis. Gardé même pour un plat sans ingrédient (« Restes »). Le nom du plat n'y est pas : un renommage ne change rien.
  - **Ce que `courses-suite` en tirera** : un plat servi reçu a changé si le plat servi a été retiré, si ses parts ont changé, si les parts de référence du plat ont changé (la quantité apportée en dépend : quantité × parts servies ÷ parts de référence), ou si les lignes actuelles du plat diffèrent des lignes gardées (ligne ajoutée, retirée, quantité ou unité changée, P2). Les lignes gardées distinguent une ligne ajoutée au plat, absente du calcul, d'une ligne dont l'article est parti par les courses faites, présente au calcul et que le recalcul ne doit pas recréer (spec, « Recalcul d'une liste »).
  - **Un plat servi reçu qui n'a plus aucune provenance dans la liste** (ses articles sont tous partis, ou il n'a jamais rien apporté) ne rend pas la liste périmée, quoi qu'il lui arrive ensuite : le recalcul ne recrée pas un article acheté, il n'y aurait rien à recalculer. Par conséquent, une ligne ajoutée à « Restes » après l'ajout des courses n'est pas signalée ; on l'obtient sur une autre liste (Q22). À confirmer au cadrage de `courses-suite`, qui décide aussi si le recalcul crée l'article d'une ligne ajoutée au plat ; les données gardées permettent l'un et l'autre.
  - Ferme la question ouverte du plan `planning` (« Plat servi et listes de courses »). Écartés : la révision du plat, qui change aussi pour un renommage ; une empreinte des lignes, qui dit qu'elles ont changé sans dire laquelle est nouvelle ; une provenance réduite à l'identifiant du plat servi, illisible après son retrait ; une ligne ajoutée à un plat qui ne rendrait pas la liste périmée, contraire au concept « Liste périmée » de la spec.
  - Ces données restent dans le serveur : l'élément de liste de l'API ne porte que les identifiants des plats servis reçus (`received`), dont l'écran des courses des repas a besoin.
- **D6. Le calcul se fait dans le serveur**, dans `internal/shopping`, en fonctions pures, dans la transaction d'écriture qui lit le planning, les plats et la liste : deux membres qui ajoutent les courses en même temps ne versent pas deux fois les mêmes plats servis (Q22). Le navigateur ne calcule aucune quantité d'article ; il montre avant l'ajout combien de plats servis de la période la liste a déjà reçus, d'après les copies du planning et des listes. Écarté : le calcul dans le navigateur, qui demandait les mêmes règles en TypeScript et une écriture de tous les articles.
- **D7. Arrondis.** Q37. Une quantité apportée se calcule en entiers en unités de base (milligramme, millilitre, millième de pièce, de cuillère, de pincée ; ADR 0003, point 7), arrondie à l'unité de base supérieure ; l'article additionne les quantités apportées, puis arrondit : pièces, cuillères et pincées à l'entier supérieur (Q4), masses au **gramme** supérieur, pour ne pas afficher « 166,667 g » ; les volumes, dont l'unité de base est déjà le millilitre, ne changent pas (revue #92, pour information). L'affichage suit la règle de la spec (plus grande unité de la famille qui donne au moins 1 : 1,5 kg, 750 g, 75 cl). Dans le dépliage (C5), la quantité apportée est montrée sans l'arrondi de l'article, au gramme supérieur pour une masse, au millilitre pour un volume, au centième pour les autres (« 0,75 pièce »).
- **D8. Coche sans révision** : `PUT …/checked { checked }`, le dernier enregistrement l'emporte, comme les parts d'un plat servi (`planning`, D3). Un article supprimé entre-temps (liste supprimée, courses faites) : `404`, la copie se vérifie, sans message (la coche n'a plus d'objet).
- **D9. Courses faites sur les articles vus cochés.** La requête porte les identifiants des articles que le membre voyait cochés en confirmant ; le serveur retire ceux qui le sont encore. Un article coché par un autre membre entre-temps reste, coché ; un article décoché entre-temps reste, non coché. La réponse dit combien sont partis, et la liste à jour. Q39. Écarté : retirer tous les articles cochés au moment de la requête, qui retire ce que la feuille de confirmation n'a pas annoncé.
- **D10. Identifiants aléatoires** (128 bits, texte) pour les listes et les articles, comme pour les plats et les plats servis.
- **D11. Écritures croisées sur une liste**, sans révision. Q39.
  - **renommer, désigner la principale** : le dernier enregistrement l'emporte ; il y a toujours exactement une principale (index unique partiel et transaction) ;
  - **supprimer une liste devenue non vide** entre-temps : `409 list_not_empty` avec la liste à jour, « Cette liste n'est plus vide. » ; **devenue principale** : `409 main_list` ;
  - **toute écriture sur une liste supprimée** entre-temps : `404` ; l'écran dit « Cette liste a été supprimée entre-temps. » et ouvre la liste principale ;
  - **retirer un article ajouté déjà retiré** : rien n'est signalé.
- **D12. Limites.** Q38. Nom d'une liste de 40 caractères au plus, mêmes règles que les noms de plats et d'ingrédients (espaces réduites, caractère de contrôle refusé, nom invisible refusé comme un nom vide ; `bibliotheque`, D30), deux listes pouvant porter le même nom (Q17) ; 20 listes par tribu ; 300 articles par liste, vérifiés aussi après un ajout de courses, refusé en entier au-delà ; période de 31 jours au plus, de 2000 à 2099 : un article calculé ne se retire pas à la main (hors périmètre V1), une année versée par erreur ne pourrait sortir qu'en la cochant. À revoir après quelques semaines de production.
- **D13. Ordre des articles** : non cochés d'abord, cochés en bas (design) ; dans chaque groupe, ordre alphabétique du nom de l'ingrédient, au sens du nom normalisé (Q29), articles calculés et ajoutés mêlés, pour retrouver un article au rayon ; à ingrédient égal, avec quantité avant sans quantité, ajoutés après calculés. Q40. Écarté : l'ordre d'ajout, qui disperse les ingrédients d'un même rayon.
- **D14. La liste principale « Courses » est créée par la migration `0009`**, dans toute base de tribu : les tribus existantes la reçoivent au déploiement, les tribus nouvelles à leur création, qui passe par les migrations (C1 « Une tribu naît avec une liste principale »). Aucun code d'initialisation à part. Son identifiant a le format des autres, 32 caractères hexadécimaux en minuscules (`lower(hex(randomblob(16)))`, comme `randomID`), ce que le test de schéma vérifie (revue #92, pour information).
- **D15. Chemins** sous la tribu : `courses` la liste principale ; `courses/<id>` une autre liste ; `courses/listes` le choix des listes ; `courses/repas?liste=<id>` les courses des repas, vers la liste d'où l'on vient. Une liste inconnue ramène à `courses`.
- **D16. Preuves** : écritures, règles de calcul et refus contre l'API ; ce que montre l'écran (onglet, ordre des listes, période par défaut, bouton inactif, dépliage) dans le navigateur, `@ui`. « La date de fin ne peut pas précéder la date de début » est **scindé**, comme P4 et R2 aux plans précédents. L'exemple « les listes de courses » est **tiré** de « Toutes les données sont compartimentées », comme le planning (`planning`, D6). « Il y a toujours exactement une liste principale » se prouve contre l'API : aucune route ne retire ce rôle, `DELETE` de la principale est refusé, `PUT shopping-lists/main` en désigne une autre.
- **D17. Maquettes du plan** : d'abord sans nouvelle maquette, `Doux-Courses` (état « à jour »), `Doux-ListesCourses` et `Doux-AjoutCourses` suffisant, avec des interprétations dans `docs/design/README.md`. *Révisée le 2026-11-07 par le développeur, avant la réalisation : maquettes complètes du plan, dessinées dans le canevas de travail d'après ces interprétations, validées et reportées au lot A : `Doux-Liste`, `Doux-ArticleMain`, `Doux-ChoixListes`, `Doux-CoursesRepas`, chacune avec ses états (réglage `etat`). Les anciennes restent la référence des états périmée, sélection et hors ligne, pour `courses-suite` et le plan hors-ligne.*
- **D18. Code** : paquet `internal/shopping` (règles pures et calcul derrière une interface `Store`, ADR 0002, 0008), qui lit les quantités et familles d'unités de `internal/dish` ; adaptateur `ShoppingStore` dans `internal/storage` ; collection `shopping_lists` ; côté navigateur, `shopping.ts` (module pur), `shopping-lists.ts` (copie sur `CachedCollection`), écrans `shopping-list-screen.ts`, `list-picker-screen.ts` et `meal-shopping-screen.ts` ; combobox d'ingrédient de l'écran d'un plat reprise.
- **D19. Pas de nouvel ADR** : le plan applique les ADR 0003 (point 7), 0027 et 0029 sans les changer. Ses décisions de domaine (D3, D7, D9, D11 à D13) vont dans `fonctionnalites.md`.
- **D20. `make seed`** ajoute à la tribu `demo` les courses de la semaine du seed dans « Courses », « pain » et « lessive » ajoutés à la main, et une liste « Marché » avec un article, pour le développement et les états des écrans. Rien en mode serveur.
- **D21. Rien des lots B à E n'est fusionné avant l'étiquette `v0.3.0`**, pour que la PR de release de `planning` ne porte que le planning ; la PR de ce plan et le lot A peuvent l'être avant, leurs lignes de `CHANGELOG.md` partant alors dans les notes de `v0.3.0` comme documentation. *Tenue : `v0.3.0` est posée le 2026-11-07 ; la PR de ce plan, fusionnée après, a sa ligne dans les notes de `v0.4.0`.*
- **D22. Ni auteur ni historique des listes et des articles**, comme `bibliotheque`, D23, et `planning`, D18 : pas d'historique des courses faites (Q14), mêmes droits pour tous (T2).
- **D23. Article ajouté** : son ingrédient est désigné par son nom, trouvé dans le référentiel par son nom normalisé ou créé (Q13, Q20, Q29) ; deux articles ajoutés du même ingrédient peuvent coexister dans une liste, en lignes distinctes, comme après un déplacement (C9). L'écran propose l'unité par défaut de l'ingrédient, comme pour un plat.
- **D24. La période par défaut et « aujourd'hui » viennent de l'appareil** (Q2, Q34) : de demain à J+7 inclus ; le serveur ne reçoit que des dates.

## Questions ouvertes

- **Croissance des listes** (D4) : une liste peut garder des articles non cochés longtemps ; les provenances s'accumulent ; et `received`, que la V1 ne vide jamais, gagne de l'ordre de 25 Ko par an pour une liste servie chaque semaine (revue #92, point 4), rechargés à chaque coche sur les autres appareils. L'estimation de D4 n'en tient pas compte. À mesurer en production ; si la taille gêne, `received` peut sortir de l'élément de liste au cadrage de `courses-suite` (seul l'écran des courses des repas s'en sert), ou le delta d'une collection le régler, au plan hors-ligne.
- **Ordre des articles par rayon** : D13 range par ordre alphabétique ; un classement par rayon demanderait une donnée de plus sur l'ingrédient. Hors V1, à rediscuter après usage.
- **Recalcul et dépliage** (`courses-suite`) : D5 garde ce qu'il faut ; restent à décider, au cadrage de `courses-suite`, si le dépliage montre une provenance dont le plat servi a été retiré comme telle avant le recalcul, si le recalcul crée l'article d'une ligne ajoutée au plat, et si un plat servi reçu sans provenance restante reste hors de la péremption (D5).

## Notes d'exécution

**Revue de la PR du plan (#92)**, 2026-11-06 :

- **Point 4, à noter** : `received` grandit sans limite (environ 14 plats servis par semaine, 25 Ko par an et par liste), et avec lui le rechargement de toutes les listes à chaque coche (D4). Reporté dans « Questions ouvertes », « Croissance des listes » ; à mesurer en production, à reprendre au cadrage de `courses-suite`.
- **Point 6 (vérification des corrections, 2026-11-07), à noter** : un changement sans effet sur les articles peut rendre une liste périmée, contre le scénario C8 « Un changement sans effet sur les articles ne rend pas la liste périmée » : une quantité réécrite dans une autre unité de sa famille (500 g en 0,5 kg), ou les parts changées d'un plat servi dont toutes les lignes sont sans quantité. Les données gardées (D5) suffisent pour comparer en unités de base ou comparer les quantités apportées ; règle de comparaison à fixer au cadrage de `courses-suite`, sans changer le format.
- **Pour information, gardé pour `courses-suite`** : un ajout de courses fait à la mauvaise liste ne s'efface qu'en cochant puis en déclarant les courses faites, et ses plats servis restent reçus par cette liste (D12, Q22) ; acceptable en V1.

**Lot A** (`feature/specs-listes-de-courses`), 2026-11-07, depuis une session cloud :

- **Décisions** : D4 à D24 tenues pour confirmées par la fusion de #92, avec l'accord du développeur ; D3 retenue telle qu'écrite (Q36).
- **Q36 à Q40** dans `fonctionnalites.md` ; « Unités, conversions et arrondis » accordé à Q37 (masses au gramme supérieur, ce que la spec écartait jusque-là : « les masses et les volumes ne sont pas arrondis à l'entier ») ; « Ajout des courses d'une période » complété (sans quantité, plats servis reçus sans ingrédient, 300 articles).
- **Scénarios ajoutés au-delà de la liste du plan**, à valider à la revue : « Deux articles ajoutés du même ingrédient restent distincts » (D23), « Un article d'une autre tribu ne peut être ni coché ni retiré » (ENF-02, le `404` identique pour un article), « Deux listes peuvent porter le même nom » (Q17), « Les espaces en trop sont retirées du nom d'une liste » et « Un nom de liste de 40 caractères est accepté » (D12). La coche d'un article retiré entre-temps (D8) n'affiche aucun message ; elle est prouvée à part de « Une écriture sur une liste supprimée entre-temps est refusée », qui ne porte que sur les écritures d'une liste vide (ajout d'un article, renommage, liste principale, courses des repas).
- **Scénario scindé** : « La date de fin ne peut pas précéder la date de début » garde son titre, devient `@ui` (bouton inactif et message) ; le refus de l'API est « Une période dont la date de fin précède la date de début est refusée ». 29 titres nouveaux dans `acceptance/pending.txt` ; l'exemple « les listes de courses » quitte « Toutes les données sont compartimentées », qui reste en attente pour les membres et les sessions.
- **Design**, interprétations à valider à la revue : navigation par mois dans le calendrier des courses des repas, bornée de 2000 à 2099 ; « Du » puis « Au » au toucher d'un jour ; bouton d'ajout actif sur une période sans repas, avec « Aucun plat servi ajouté » ensuite ; formulaire d'ajout à la main laissé ouvert après un ajout ; « Définir comme liste principale » absent de la feuille de la principale ; bandeau d'erreur « Réseau nécessaire » en haut de la liste pour toute écriture (D2) ; libellés des bandeaux après un ajout de courses et des courses faites, d'après la maquette et la conception.
- **Maquettes** (D17 révisée, 2026-11-07) : à la demande du développeur, quatre écrans dessinés dans le canevas de travail (rangée G), validés, puis reportés dans `docs/design/maquettes/` ; le README accordé à ce qu'elles montrent : progression sous l'en-tête et résumé des cartes repris des maquettes existantes, cases vides pendant le chargement, « Indiquez un article. », phrase d'une période sans repas, « Ajout en cours… », croix du bandeau vert, message d'une période invalide sous le calendrier.
- **Conception** : statut retenu, Q36 à Q40 cités, `SortItems` (Q40) dans les règles et ordre des articles dans la réponse de l'API.

**Lot B** (`feature/listes-api`), 2026-11-07, Dev Container, à valider à la revue :

- **Interfaces du domaine** différentes du diagramme de la conception, mis à jour : le `Tx` écrit par opérations ciblées (`InsertList`, `RenameList`, `SetMain`, `DeleteList`, `InsertItem`, `SetChecked`, `DeleteItem`) au lieu de réécrire toute la liste par `SaveList` ; `Lists` lit la liste, vérifie, puis écrit. Le lot C ajoute au `Tx` ce qu'écrit le calcul. `SetMain` sur la liste déjà principale n'écrit rien : la version ne change pas.
- **Règles partagées avec les plats** : `internal/dish` exporte la règle des noms (`CleanName`, D30 de `bibliotheque`), celle d'une ligne saisie (`ParseLine`) et la résolution d'un ingrédient dans le référentiel (`ResolveIngredient`, sur l'interface `dish.Catalog`) ; les plats s'en servent aussi, sans changement de comportement. Côté stockage, `catalogTx` porte les trois méthodes du référentiel, pour `DishStore` et `ShoppingStore`.
- **Codes d'erreur d'un article ajouté** : ceux d'une ligne de plat (`ingredient_name_required`, `unit_required`, `invalid_unit`), et non `ingredient_name_empty` et `unknown_unit` qu'écrivait la conception : même règle, mêmes messages à l'écran (lot D). Conception corrigée.
- **Ordre des listes** (étape 8) : pas de fonction Go, l'API servant les listes dans l'ordre des identifiants (D4) ; il est fait par `shopping.ts` (lot D, étape 17). `SortItems` (Q40) range les articles de chaque liste lue et de chaque réponse d'écriture.
- **Provenances et plats servis reçus lus dès ce lot** dans `GET shopping-lists` et les réponses d'écriture (étape 14 pour le lot C) : le schéma les porte, le lot C les écrit. Une provenance par plat servi et par article (clé primaire) ; supprimer une liste supprime ses plats servis reçus et leurs lignes, retirer un article ses provenances.
- **Schéma** : bornes vérifiées aussi par la table (`CHECK` des unités, des dates réelles, des parts ; au plus une liste principale par l'index unique partiel) ; `SetMain` retire le rôle avant de le donner, l'index étant vérifié instruction par instruction. Migration testée sur une base créée avant `0009` (D14).
- **Corps de `checkout`** : un test à 300 identifiants, au-delà des 4 Kio ; un corps de plus de 16 Kio est refusé (revue #92, point 1).
- **Étapes godog** : 35 titres retirés de `acceptance/pending.txt` (31 de `liste-courses.feature`, T1, et trois ENF-02).
  - « Une liste d'une autre tribu ne peut être ni lue, ni modifiée, ni supprimée » passe dès ce lot : son exemple « ajouter les courses des repas à » reçoit, pour la liste d'une autre tribu comme pour un identifiant inconnu, le `405` d'une route qui n'existe pas encore ; le lot C la remplace par la route, comme `bibliotheque` l'a fait pour « lire » un plat.
  - « Une écriture sur une liste supprimée entre-temps est refusée » **reste en attente jusqu'au lot C** : son exemple des courses des repas attend la route et son `404`.
  - « aucun message ne m'est présenté » est prouvé contre l'API par un `200`, ou un `404 not_found` que l'écran prend pour fait, sans message (D8, D11).
  - « la liste "…" contenant 1 pièce de "oignon" provenant du "Chili con carne" du … » (« Ordre des articles », puis les scénarios du lot C) sert le plat dans le repas pour les parts qui apportent cette quantité, puis écrit en base l'article calculé, sa provenance et le plat servi reçu avec les lignes du plat, comme le calcul les laissera.
  - « la requête est refusée » et « la réponse est la même que pour un identifiant qui n'existe pas » servent aussi aux listes ; « je ne vois aucun élément de la tribu "…" » vérifie aussi les listes consultées (identifiants des listes et des articles, ingrédients des articles de l'autre tribu).

**Revue de la PR #98**, 2026-11-07 :

- **Point 1, corrigé** : `main` fusionné dans la branche (conflit de `CHANGELOG.md` avec la PR #97), job `ci` vert sur la PR.
- **Point 2, à traiter au lot D, étape 18** : une écriture qui ne change pas une collection renvoie quand même sa version courante (`writeCollections` les renvoie toutes) ; une copie en retard d'une version, un autre membre ayant écrit entre-temps, la prend pour la sienne et reçoit ensuite des `304` jusqu'à la prochaine écriture. Chaque écriture des listes renvoie `ingredients`, que seul l'ajout d'un article change : un ingrédient créé par un autre membre, puis une coche, et l'autocomplétion ne le montre plus avant un rechargement, si la copie des listes passe cette version à celle du référentiel comme `library.ts` pour un plat. Défaut déjà présent en production pour la bibliothèque (plat enregistré sans ingrédient nouveau). Cas limite : `checkout` sans article retiré pendant une écriture sur une autre liste ; sans effet pour `PUT shopping-lists/main` sur la liste déjà principale. Correction proposée : `writeCollections` ne renvoie que les collections dont la version a changé (ADR 0029, point 3, « les collections qu'elle a touchées ») ; `library.ts` gère déjà l'absence de `ingredients` ; la conception écrira alors `{ versions }` sans `shopping_lists` pour la liste déjà principale.
- **Point 3, à traiter au lot D, étape 17** : l'ordre des articles n'a pas de table de cas commune à Go et TypeScript, alors que l'écran retrie la liste renvoyée par l'API. Mettre les cas de `TestSortItems` dans `internal/shopping/testdata/` et les faire lire par les deux tests, comme ceux de `internal/dish/testdata/`.
- **Pour information, gardé pour plus tard** : les clés étrangères de `shopping_item_sources.dish_id` et `shopping_received.dish_id` vers `dishes` seront à reprendre par un plan qui permettrait de supprimer un plat, comme D5 pour `served_dishes`. Au lot C, étape 15, vérifier que l'exemple « ajouter les courses des repas à » de l'accès croisé à une liste compare bien deux `404` : il passe aujourd'hui sur deux `405`, et rien ne signalerait l'oubli.

**Lot C** (`feature/courses-des-repas-api`), 2026-11-07, Dev Container, à valider à la revue :

- **Période** : `ParsePeriod` lit ses bornes par `mealplan.ParseDate`, la règle des dates du planning (`internal/shopping` importe donc `internal/mealplan` pour ses règles pures, pas pour `Plan`) ; une borne illisible ou hors de 2000 à 2099 répond `400 invalid_date`, le code du planning, et non `invalid_period`, réservé à une fin avant le début (message « La date de fin doit suivre la date de début. ») ; `period_too_long` au-delà de 31 jours. Conception corrigée.
- **Écritures du calcul** : le `Tx` gagne `ServedDishes` (plats servis de la période, par repas, midi avant soir, puis rang, avec les parts de référence et les lignes actuelles de leurs plats, P2), `UpdateItem` (quantité et coche d'un article calculé), `InsertSource` et `InsertReceived` (plat servi reçu et lignes du plat, D5), au lieu du `SaveList` du diagramme. Le calcul pur (`addMealShopping`) rend la liste après, les articles créés ou changés avec les provenances qu'ils gagnent, et les plats servis reçus.
- **Quantité d'un article calculé** : recalculée à chaque ajout à partir de toutes ses provenances, anciennes et nouvelles, puis arrondie (conception, étape 5) ; elle est donc toujours l'arrondi de la somme des quantités apportées. Un ingrédient est reconnu par son nom normalisé (Q29), unique dans le référentiel.
- **Rien d'écrit sans plat servi retenu** (période sans repas, ou tous déjà reçus) : la version des listes ne change pas, la réponse donne `added: 0`.
- **`dish.Unit.Factor`** exporté : l'arrondi de l'article se fait à l'unité entière la plus petite de sa famille (gramme, millilitre, pièce, cuillère, pincée), dont le facteur est celui de `internal/dish`.
- **Table de cas** `internal/shopping/testdata/meal-shopping.json` : quantités apportées (`Contribution`) et arrondi d'un article (`RoundItem`), au format de l'API.
- **À traiter au lot E, étapes 22 et 24** : `shopping.ts` lit cette table, comme `dishes.test.ts` lit celles de `internal/dish/testdata/`, pour l'affichage des quantités apportées du dépliage (C5). L'API donne une quantité apportée sans l'arrondi de l'article, à l'unité de base près (« 166.667 » `g`, « 0.334 » `piece`) ; l'écran l'arrondit selon D7 : au gramme supérieur pour une masse, au millilitre pour un volume, au centième pour les autres. Ajouter à la table, pour chaque cas, la quantité affichée attendue.
- **Étapes godog** : 30 titres retirés de `acceptance/pending.txt` (29 de `liste-courses.feature`, dont « Une écriture sur une liste supprimée entre-temps est refusée » et « Les courses faites ne touchent pas au planning », et P2).
  - « la période » est la période par défaut d'après l'horloge du scénario (de demain à J+7, D24), et « dans la période » sert le plat à son premier soir.
  - « chacun de ces plats » sert les plats définis par les étapes qui précèdent immédiatement ; le « Chili con carne » du Contexte, défini juste avant le scénario, en fait partie, sans effet sur les lignes que ces scénarios vérifient (godog ne dit pas qu'une étape vient du Contexte).
  - L'accès croisé à une liste échoue si l'identifiant inconnu reçoit un `405` : chaque exemple compare bien deux `404` (revue #98, pour information).
- **`make seed`** : « Courses » reçoit les courses du jour du seed à J+6, la semaine de `Doux-Main`, puis « pain » (à côté du « pain » calculé de la soupe à l'oignon, D23) et « lessive » ; « Marché » reçoit 500 g de fraises.

**Lot D** (`feature/onglet-courses`), 2026-11-07, Dev Container, à valider à la revue :

- **Revue #98, point 2, traité** : `writeCollections` ne renvoie plus que les versions des collections que l'écriture a changées (ADR 0029, point 3) ; `PUT shopping-lists/main` sur la liste déjà principale et un ajout de courses sans plat servi renvoient des `versions` vides. Effet aussi pour la bibliothèque : un plat enregistré sans ingrédient nouveau ne renvoie plus `ingredients`, ce que `library.ts` gérait déjà. Conceptions de la bibliothèque et des listes accordées.
- **Revue #98, point 3, traité** : table de cas `internal/shopping/testdata/sort-items.json`, lue par `TestSortItems` et par `shopping.test.ts` ; les noms normalisés sont comparés point de code par point de code, comme Go compare les octets UTF-8, et non selon l'ordre de la langue. L'écran trie de nouveau les articles reçus.
- **Combobox d'ingrédient** : sortie de l'écran d'un plat en un contrôleur Lit, `ingredient-combobox.ts` (champ, propositions, clavier, fermeture après le geste), que l'écran d'un plat et le formulaire d'un article utilisent ; l'écran d'un plat ne change pas, ses tests le prouvent. Un article ajouté ne retire aucune proposition (D23).
- **Copie des listes** : coche montrée aussitôt, puis la liste renvoyée ; un `409` fait vérifier la copie ; un `404` aussi, qui dit ensuite si la liste a été supprimée (D11) ou seulement l'article (D8) ; un ingrédient créé fait vérifier le référentiel, l'élément de liste ne portant pas son unité par défaut. Vérifiée à chaque visite de l'onglet et au retour au premier plan sur l'onglet Courses.
- **CSP** : la politique de l'application (`default-src 'self'`) refuse les styles en ligne ; les barres de progression sont des `progress` natifs et les largeurs du squelette de chargement viennent de la feuille de style. Un test des écrans vérifie qu'aucun refus de la CSP n'apparaît.
- **Laissé au lot E** : la pastille « N plats » est là, sans chevron ni dépliage ; « Ajouter les courses des repas » n'est pas encore sur l'écran d'une liste ; `courses/repas` mène pour l'instant à la liste principale (D15).
- **Interprétations de l'écran** dans `docs/design/README.md`, « Réalisation de l'onglet Courses et du choix des listes » : les sept validées par le développeur le 2026-11-07, sur captures de l'écran réel.
- **Preuves** : `shopping.test.ts`, `shopping-lists.test.ts` (copie et lecture des réponses), `router.test.ts` ; états des écrans sur l'API simulée (`e2e/ecrans-courses.spec.ts`) ; C1 « L'onglet Courses ouvre la liste principale » et C2 « Ordre des listes » sous Chromium et WebKit (`e2e/steps-shopping.ts`), retirés de `acceptance/pending.txt`.

**Revue de la PR #100**, 2026-11-07 :

- **Point 1, corrigé** : une coche qui échoue sans réseau ne remet plus la liste entière d'avant, ce qui effaçait une coche réussie entre-temps ; seul l'article touché revient, et seulement si aucune réponse de l'API ne l'a remplacé depuis. Tests dans `shopping-lists.test.ts`.
- **Pour information** : `versions` réduites vérifiées pour les autres clients ; « Courses faites » juste après une coche, conforme à D9 ; `npm test` sous Node 22 hors de `.nvmrc`.
- **Remarqué en passant, à reprendre au plan hors-ligne** (décision du développeur, 2026-11-07) : pendant qu'une coche attend sa réponse, la réponse à une autre écriture remplace la liste entière ; l'article en attente s'affiche un instant dans son état d'avant, jusqu'à sa propre réponse. Rien n'est perdu. La file des coches du plan hors-ligne (C7) devra garder les coches en attente par-dessus les listes reçues.

*(à compléter au fil des lots)*
