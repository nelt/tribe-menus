# Plan : bibliothèque de plats

- **Nom** : `bibliotheque`
- **Date** : 2026-10-29
- **Statut** : brouillon, première version. Décisions D1, D3 à D7, D11 et D13 prises par le développeur le 2026-10-29 ; les autres sont des propositions, à confirmer à la revue du plan.

## Objectif

Premier plan du métier : un membre constitue depuis son téléphone la bibliothèque de plats de sa tribu, avec le référentiel d'ingrédients qui se forme au fil de la saisie, et règle la taille de la tribu qui fixe les parts proposées. Le planning et les listes de courses s'appuieront dessus.

À la fin du plan :

- une fois connecté, le membre voit la barre d'onglets **Planning**, **Plats**, **Courses** (D11) ;
- l'onglet **Plats** montre la bibliothèque, par ordre alphabétique, et la recherche sur les mots du nom et des ingrédients, sans tenir compte de la casse ni des accents (P3), dans le navigateur (D1) ;
- **Nouveau** ouvre la création d'un plat (P1) : nom, parts (proposées à la taille de la tribu, T3), ingrédients avec quantité facultative (D4), unité proposée et autocomplétion sur le référentiel, avec création d'un ingrédient à la volée (P4) ; toucher un plat l'ouvre pour le modifier (P2, D19) ; deux modifications croisées ne s'écrasent pas (D5) ;
- l'écran **Tribu**, ouvert depuis l'en-tête du Planning, règle la taille de la tribu (T3, D3) ;
- les scénarios de la bibliothèque, du référentiel et de la taille de la tribu passent, contre l'API ou dans le navigateur selon ce qu'ils prouvent (D1, D14), y compris le compartimentage entre tribus (ENF-02) ;
- la version `v0.2.0` est publiée et déployée en production (D24).

La conception (modèle, découpage du code, API, échanges) est dans [`docs/conception/bibliotheque-de-plats.md`](../conception/bibliotheque-de-plats.md), avec ses diagrammes de classes et de séquence (D13). Ce plan dit dans quel ordre on la réalise et ce qui reste à décider.

## Contexte et contraintes

- **À lire avant de commencer** : ADR 0002, 0003 (points 4 et 7), 0004 (points 3, 7 et 9), 0005, 0008, 0012 (point 6) ; `docs/specs/fonctionnalites.md` (concepts, P1 à P4, T1 à T3, Q5b, Q8, Q19, Q20), `glossaire.md`, `conventions-gherkin.md`, `features/bibliotheque-plats.feature`, `tribu.feature`, `compartimentage-tribus.feature` ; `docs/design/README.md` (« Couleurs des plats », « Navigation », « Planning et plats ») et les maquettes `Doux-Bibliotheque`, `Doux-Plat`, `Doux-Tribu` ; le document de conception.
- **En parallèle de la fin du plan `recette`** : ce plan démarre avant la release `v0.1.0`. Aucun de ses lots n'est fusionné avant l'étiquette `v0.1.0`, sans quoi la première release contiendrait un métier à moitié fait ; ses archives de PR ne vont en recette qu'après la séance S4 (D17). La recette est partagée : y déployer une PR de ce plan remplace la version qu'on y prouve.
- **Où se fait le travail** : le lot A, documentation seule, peut se faire depuis une session cloud ; les lots B à D se font dans le Dev Container. La session cloud ne peut ni compiler ni tester le projet (modules Go et site de recette hors de sa liste d'accès), et la CI ne remplace pas l'écriture des tests avant le code (ADR 0005).
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, avec `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`bibliotheque/B`) et les stories ; `make ci` avant de pousser.
- **Une seule dépendance d'exécution nouvelle**, `golang.org/x/text/unicode/norm`, pour la décomposition Unicode de la normalisation des noms (D9), que la bibliothèque standard n'a pas ; elle est justifiée dans l'ADR 0027 (ADR 0002, point 5). Rien de nouveau côté npm (ADR 0004, point 8) ; le lien entre Playwright et les `.feature` passe par le parseur Gherkin que godog apporte déjà (D14).
- **Vocabulaire** : le code suit `glossaire.md` (`dish`, `dish library`, `ingredient`, `ingredient catalog`, `reference servings`, `default unit`, `color`, `tribe size`, unités et familles) ; les termes nouveaux y sont ajoutés au lot A, avant le code : révision (*revision*), nom normalisé (*normalized name*), ligne d'ingrédient (*dish ingredient*), recherche (*search*), proposition (*suggestion*), onglet (*tab*).
- **Une migration** de la base de la tribu (`0007`) : revenir en arrière sur une version antérieure demandera la restauration de l'instantané (ADR 0012, point 6 ; plan `recette`, D7). Le registre et la base de limitation ne changent pas.
- **Hors-ligne hors du plan** (D18) : il a son propre sujet dans la feuille de route. Ici, une modification sans réseau échoue avec « Réseau nécessaire » (ENF-01, PT-04) et ne change rien ; la consultation hors ligne viendra du cache des réponses `GET dishes` et `GET ingredients`, que D1 et D2 rendent suffisant.
- **Rien d'une autre tribu** (ENF-02) : chaque requête lit la seule base de la tribu de sa session (ADR 0003) ; les identifiants de plats sont aléatoires (D10), pour qu'un identifiant d'une autre tribu ne désigne jamais un plat de la sienne.
- **Texte saisi par un membre affiché pour la première fois** : noms de plats et d'ingrédients. Le rendu reste échappé (Lit, ADR 0004, point 9) et `webcheck` est complété avant (`revue-securite`, constat 10).

### Scénarios couverts

| Fichier | Scénario | Preuve | Lot |
| --- | --- | --- | --- |
| `bibliotheque-plats.feature` | P1 : créer un plat avec ses ingrédients ; autre nombre de parts ; teinte la moins utilisée ; la couleur ne change pas ; nom obligatoire ; plat sans ingrédient | API | B |
| | P1, P2, P4 : scénarios nouveaux du lot A (nom déjà pris, ingrédient sans quantité, ingrédient en double, modification croisée) | API | B |
| | P2 : un plat ne peut pas être supprimé | API, absence de route (ADR 0005, point 4) | B |
| | P2 : la modification s'applique aux repas où il est servi | **reste en attente** : planning et listes | — |
| | P3 : rechercher un plat (plan du scénario, 7 exemples) | navigateur, `@ui` | C |
| | P4 : un nom qui ne diffère que par la casse, les accents ou les espaces désigne le même ingrédient | API | B |
| | P4 : créer un nouvel ingrédient pendant la saisie | API pour l'ajout au référentiel ; à scinder au lot A si « il m'est proposé » doit se prouver dans le navigateur | B |
| | P4 : l'autocomplétion propose les ingrédients existants ; ignore la casse et les accents ; l'unité par défaut est proposée | navigateur, `@ui` | D |
| `tribu.feature` | T1 : les plats sont partagés ; T2 : mêmes droits ; T3 : la taille vaut 4 par défaut ; T3 : la taille est proposée pour un nouveau plat | API | B |
| | T1 : le référentiel est partagé (« pim » proposé) | navigateur, `@ui` | D |
| | T1 : le planning est partagé ; T3 : la taille fixe les parts des plats servis | **restent en attente** : planning | — |
| `compartimentage-tribus.feature` | ENF-02 : la bibliothèque est propre à la tribu ; le référentiel est propre à la tribu ; un même nom d'ingrédient est indépendant ; accès croisé refusé (lire, modifier, supprimer) ; la tribu d'une requête est celle de la session | API (D15) | B |
| | ENF-02 : toutes les données sont compartimentées | **reste en attente** : ses exemples planning et listes | — |

## Étapes

### Lot A : specs, conception et ADR (`feature/specs-bibliotheque-de-plats`)

Documentation seule, une fois ce plan revu et ses décisions confirmées.

- [ ] **1. `fonctionnalites.md`** : décisions nouvelles, à la suite de Q24 : révision et refus d'une modification croisée (D5), quantité facultative et unité absente sans quantité (D4), une ligne par ingrédient (D6), nom de plat unique au sens du nom normalisé (D7), ligatures dans la normalisation (D8, complète Q20), ordre alphabétique de la bibliothèque (D22), limites (D12, D20). P1, P2 et P4 accordés ; concept « Plat » : quantités facultatives.
- [ ] **2. Scénarios**, selon `conventions-gherkin.md` : nom de plat déjà pris (P1) ; ingrédient sans quantité (P1) ; même ingrédient deux fois refusé (P1, P4) ; modification croisée refusée, avec le plat à jour (P2) ; tag `@ui` sur les scénarios prouvés dans le navigateur (tableau ci-dessus) ; scission éventuelle de « Créer un nouvel ingrédient pendant la saisie ». Les nouveaux titres entrent dans `acceptance/pending.txt`.
- [ ] **3. `glossaire.md`** : les termes nouveaux de « Contexte et contraintes ».
- [ ] **4. ADR 0027, recherche et autocomplétion dans le navigateur** (D1, D2, D9) : bibliothèque et référentiel chargés en entier ; règle de normalisation en Go et en TypeScript, tenue d'accord par une table de cas commune ; `golang.org/x/text/unicode/norm` en dépendance d'exécution, justifiée selon l'ADR 0002, point 5. Il précise l'ADR 0005, point 2 : P3 et P4 sont des règles métier prouvées dans le navigateur.
- [ ] **5. ADR 0028, scénarios `@ui` exécutés par Playwright** (D14) : ferme `socle`, D5. Export des scénarios `@ui` par un outil Go, pas exécutés par godog, même règle de `pending.txt`.
- [ ] **6. `docs/design/README.md`** : barre d'onglets avec Courses inactif « À venir » et Planning provisoire (D11) ; modification d'un plat depuis la bibliothèque, sur l'écran de création (D19) ; bandeau d'une modification croisée (D5) ; écran Tribu réduit à la taille (D3) ; messages d'erreur (nom obligatoire, nom déjà pris, ingrédient en double, quantité invalide, réseau nécessaire). Écarts à reporter dans le canevas de travail par le développeur s'il le souhaite.
- [ ] **7. Document de conception** : statut « retenu » ; corrections de la revue.
- [ ] **8. Feuille de route** : plan « en cours ».

### Lot B : domaine et API (`feature/bibliotheque-api`)

- [ ] **9. Migration `0007`** de la base de la tribu : colonne `size` de `tribe` (4 par défaut, de 1 à 20, D12) ; tables `ingredients`, `dishes`, `dish_ingredients` (quantité et unité toutes deux présentes ou toutes deux absentes, une ligne par ingrédient dans un plat, D4, D6 ; index uniques sur les noms normalisés, D7) ; sans auteur ni date de modification par membre (D23). Requêtes sqlc, `make generate`.
- [ ] **10. Règles pures de `internal/dish`**, tests en tableaux d'abord : `NormalizeName` sur la table de cas commune (`testdata`, D9) ; unités, familles, unité de base, lecture d'une quantité saisie (virgule ou point, trois décimales au plus dans l'unité de la ligne, strictement positive, D20) ; `PickColor` (Q19, ordre de la palette de `docs/design/README.md`) ; validation d'un plat (nom obligatoire et limité, lignes, doublons, D6, D20).
- [ ] **11. `dish.Library`** derrière une interface `Store` : liste, création, modification avec révision (D5), référentiel ; résolution des ingrédients par nom normalisé et unité par défaut (Q20, D4) dans une transaction d'écriture ; adaptateur sqlc dans `internal/storage`. Taille de la tribu dans `internal/tribe`.
- [ ] **12. API** : `GET dishes`, `POST dishes` (parts facultatives : taille de la tribu, D16), `PUT dishes/{id}` (409 `dish_changed` avec le plat à jour), `GET ingredients`, `GET tribe`, `PUT tribe/size` ; pas de `DELETE` ; 404 identique pour un identifiant inconnu ou d'une autre tribu ; corps limité à 16 Kio pour les plats (D20), 4 Kio ailleurs comme aujourd'hui.
- [ ] **13. Étapes godog** des scénarios « API » du tableau (D15), retirés de `pending.txt`.
- [ ] **14. `make seed`** : les huit plats de la maquette `Doux-Bibliotheque` dans la tribu `demo` (D21). La tribu de recette, créée par `admin seed -members`, n'en reçoit pas.

### Lot C : navigation, bibliothèque et scénarios `@ui` (`feature/bibliotheque-ecran`)

- [ ] **15. `webcheck`** : `unsafeStatic`, `unsafeMathML`, `setHTMLUnsafe`, `srcdoc`, `createContextualFragment` ajoutés aux échappatoires refusées (`revue-securite`, constat 10), avant le premier écran qui affiche un texte saisi.
- [ ] **16. Scénarios `@ui`** (D14) : outil Go qui exporte les scénarios `@ui` des `.feature` (plans du scénario déroulés) en JSON pour Playwright ; registre d'étapes en TypeScript ; un scénario `@ui` sans étape connue échoue, et `pending.txt` vaut dans les deux sens comme pour godog ; godog ignore les scénarios `@ui`.
- [ ] **17. `web/src/dishes.ts`**, module pur testé par `node:test` : `normalizeName` sur la même table de cas que Go (D9), `searchDishes` (mots, préfixes, tous les termes), tri alphabétique (D22).
- [ ] **18. Barre d'onglets et routeur** : pages Planning (l'accueil provisoire actuel, avec la pastille de la tribu), Plats, Courses inactif « À venir » (D11) ; chemins en français sous la tribu (`plats`, `plats/nouveau`, `plats/<id>`, `tribu`).
- [ ] **19. Écran Bibliothèque** d'après `Doux-Bibliotheque` : nombre de plats ou de résultats, carte d'un plat (initiale dans sa couleur, nom, parts, ingrédients, termes trouvés mis en valeur), « Aucun plat ne correspond », « Nouveau » ; erreur réseau.
- [ ] **20. P3 sous Chromium et WebKit**, retiré de `pending.txt`.

### Lot D : fiche d'un plat et écran Tribu (`feature/bibliotheque-fiche-plat`)

- [ ] **21. Stepper de parts** : shadow DOM, champ de formulaire par `ElementInternals` (ADR 0004, point 3).
- [ ] **22. `dishes.ts`** : `suggestIngredients` (sans ceux déjà dans le plat, D6), lecture et affichage des quantités (D20), avec leurs tests.
- [ ] **23. Écran d'un plat** d'après `Doux-Plat`, en création et en modification (D19) : nom, parts, lignes (quantité facultative, unité dans la liste fermée, ingrédient en combobox ARIA, « Créer « … » », unité par défaut proposée, retrait d'une ligne), « Enregistrer le plat » ; erreurs de l'API ; modification croisée : bandeau et plat à jour (D5) ; « Réseau nécessaire » (D18).
- [ ] **24. Écran Tribu** d'après `Doux-Tribu`, réduit au nom, au nombre de membres actifs et au stepper de la taille (D3), ouvert par la pastille de l'en-tête du Planning ; « Se déconnecter » reste sur le Planning provisoire.
- [ ] **25. P4 et T1 (référentiel partagé) sous Chromium et WebKit**, retirés de `pending.txt`.

### Recette et release

- [ ] **26. Recette**, après S4 et sur l'archive de la PR du lot D (D17) : création, recherche et modification d'un plat sur Android et, si possible, sur un iPhone ; deux appareils qui modifient le même plat ; taille de la tribu ; retour arrière avec restauration (migration `0007`).
- [ ] **27. Release `v0.2.0`** selon `RELEASING.md`, déployée en production (D24).
- [ ] **28. Clôture** : statut « terminé » ; feuille de route (plan terminé, points reportés à jour) ; document de conception ; `CLAUDE.md` (stack : dossier `conception/`, paquet `internal/dish`) ; `CHANGELOG.md`.

## Critères de validation

- `make ci` vert à chaque PR ; `pending.txt` ne garde de ce plan que les scénarios marqués « reste en attente » ci-dessus.
- Les règles de normalisation de Go et de TypeScript passent la même table de cas.
- En recette, sur téléphone : un plat créé est retrouvé par la recherche, modifié, et la modification croisée d'un second appareil est refusée sans rien perdre.
- En production, le développeur saisit ses vrais plats.

### Ce que ce plan ne prouve pas

- Que la modification d'un plat s'applique aux repas où il est servi (P2) et que la taille fixe les parts des plats servis (T3) : avec le planning et les listes de courses.
- La consultation hors ligne de la bibliothèque et du référentiel : avec le plan hors-ligne.
- Le vrai Safari sur iPhone, si aucun n'est disponible en recette.

## Décisions

Le 2026-10-29, avec le développeur.

- **D1. Recherche (P3) et autocomplétion (P4) dans le navigateur**, comme la maquette : elles fonctionneront hors ligne sans réécriture. Écartée : des routes de recherche et de suggestions dans l'API, prouvées par godog, mais limitées hors ligne aux recherches déjà faites et à réécrire au plan hors-ligne. Conséquences : D2, D9, D14 ; P3 et P4 sont les premiers scénarios prouvés dans le navigateur.
- **D3. Taille de la tribu (T3) dans ce plan**, avec un écran Tribu réduit au stepper ; le plan des membres le complétera. Écartée : une taille fixe à 4 jusqu'au plan des membres.
- **D4. Quantité facultative** pour un ingrédient d'un plat (« sel »), comme pour un article ajouté (Q13). Sans quantité, pas d'unité. L'unité par défaut d'un ingrédient est la première unité saisie avec lui ; tant qu'aucune ligne n'en a eu, il n'en a pas (précise Q20). Ce que devient une ligne sans quantité dans les courses (C4) est laissé au plan des listes (questions ouvertes). Écartée : une quantité obligatoire.
- **D5. Une modification croisée est refusée.** Un plat porte une révision, augmentée à chaque modification ; un enregistrement fait sur une révision dépassée reçoit `409 dish_changed` et le plat à jour ; l'écran le dit (« Ce plat a été modifié entre-temps ») et montre le plat à jour, à modifier de nouveau. Écartée : le dernier enregistrement gagne, qui écrase sans prévenir.
- **D6. Une ligne par ingrédient dans un plat** ; l'ingrédient déjà dans le plat n'est plus proposé. Écartée : une ligne par famille d'unités.
- **D7. Deux plats ne portent pas le même nom**, au sens du nom normalisé : « Un plat porte déjà ce nom ». Écartée : des noms en double, comme les listes (Q17).
- **D11. Trois onglets dès ce plan** : Planning, l'accueil provisoire actuel jusqu'au plan du planning ; Plats ; Courses inactif, « À venir ». Écartés : deux onglets, ou pas de barre d'onglets avant le planning.
- **D13. Diagrammes de classes et de séquence en Mermaid, dans un document de conception durable** par domaine, `docs/conception/`, tenu à jour par les plans suivants ; le plan y renvoie. Écartés : les diagrammes dans le plan, qui se fige une fois terminé.

Propositions, à confirmer à la revue.

- **D2. Bibliothèque et référentiel chargés en entier** à l'ouverture de l'onglet Plats et de l'écran d'un plat (`GET dishes`, `GET ingredients`). Une tribu compte quelques dizaines à quelques centaines de plats : quelques dizaines de kilo-octets. Pas de pagination en V1.
- **D8. Ligatures dans le nom normalisé** : œ devient oe, æ devient ae, en plus de la casse, des accents et des espaces (Q20). Sans cela, « boeuf » ne trouve pas « bœuf haché », et « Boeuf » et « Bœuf » seraient deux ingrédients.
- **D9. La règle de normalisation existe en Go et en TypeScript**, tenue d'accord par une table de cas commune, un fichier JSON lu par `go test` et par `node:test`. Go décide de l'identité d'un ingrédient et de l'unicité d'un nom de plat ; TypeScript cherche et propose. Même parade que pour le motif des noms d'une archive (plan `recette`, lot E). Écarté : une seule implémentation côté serveur, que D1 exclut. *Le 2026-10-29, à la revue du plan (point 1), avec le développeur : la décomposition Unicode se fait en Go par `golang.org/x/text/unicode/norm`, du projet Go, déjà dans `go.sum` par les outils et désormais dépendance d'exécution ; dans le navigateur, par `String.prototype.normalize`. Écartée : une table des lettres latines accentuées écrite dans le projet, sans dépendance, mais qui laisse telle quelle toute lettre absente de la table.*
- **D10. Identifiants de plats aléatoires** (128 bits, texte), et non des entiers : avec une base par tribu, l'entier d'un plat d'une autre tribu désignerait souvent un plat de la sienne, ce que le scénario « Accès croisé refusé » interdit de confondre avec un refus. Ils se prêtent aussi à une création hors ligne, si elle vient un jour. Les ingrédients ne sont jamais désignés par leur identifiant dans l'API, mais par leur nom.
- **D12. Taille de la tribu de 1 à 20.**
- **D14. Scénarios `@ui` exécutés par Playwright à partir des `.feature`** (ferme `socle`, D5) : un outil Go du projet lit les `.feature` avec le parseur Gherkin de godog, déjà en dépendance, et exporte les scénarios `@ui` en JSON ; un registre d'étapes en TypeScript les exécute ; godog les ignore ; `pending.txt` vaut pour les deux, dans les deux sens. Les `.feature` restent la source unique (ADR 0005, point 1). Écartés : `playwright-bdd` ou `@cucumber/gherkin` en npm (dépendances nouvelles) ; des tests Playwright écrits à la main qui citent les titres (le scénario serait recopié). ADR 0028.
- **D15. Contre l'API, « une recherche ne renvoie rien » et « ne fait pas partie des propositions » se prouvent par l'absence dans `GET dishes` et `GET ingredients`** : recherche et propositions ne portent que sur ces réponses. Cela suffit au compartimentage (ENF-02) ; la règle de recherche elle-même est prouvée dans le navigateur (P3, P4).
- **D16. Parts facultatives dans `POST dishes`** : absentes, le plat est défini pour la taille de la tribu. L'écran les envoie toujours ; le scénario T3 « proposée pour un nouveau plat » se prouve ainsi contre l'API.
- **D17. Rien de ce plan n'est fusionné avant l'étiquette `v0.1.0`**, et ses archives ne vont en recette qu'après S4 : la recette sert encore aux preuves de la PR #59 et à la release. Les PR peuvent être ouvertes et relues avant.
- **D18. Pas de hors-ligne dans ce plan** : une modification sans réseau affiche « Réseau nécessaire » et ne change rien (ENF-01, PT-04). Le cache des réponses, l'effacement à la déconnexion et le scénario « Consultation hors ligne de la bibliothèque de plats » viennent avec le plan hors-ligne.
- **D19. Modifier un plat** : toucher sa carte dans la bibliothèque ouvre l'écran de la maquette `Doux-Plat`, titré du nom du plat, avec les mêmes champs. Il n'y a pas de maquette dédiée ; à dessiner si le développeur le souhaite.
- **D20. Limites** : nom de plat de 60 caractères, nom d'ingrédient de 40, 40 lignes par plat ; quantité strictement positive, trois décimales au plus dans l'unité de la ligne, saisie avec une virgule ou un point ; corps d'une requête sur un plat limité à 16 Kio (4 Kio aujourd'hui pour toute l'API, trop peu pour 40 lignes).
- **D21. `make seed`** ajoute à la tribu `demo` les huit plats de la maquette `Doux-Bibliotheque`, pour le développement et les états des écrans ; rien en mode serveur.
- **D22. Bibliothèque par ordre alphabétique** du nom (`Intl.Collator` en français), dans les résultats d'une recherche aussi.
- **D23. Ni auteur ni historique des modifications d'un plat** : les specs n'en demandent pas (T2 : mêmes droits pour tous), et le journal d'audit ne porte que sur les membres et les sessions (EF-07). Minimisation des données.
- **D24. Release `v0.2.0`** à la fin du plan, déployée en production, où seule la tribu du développeur existe (`recette`, D1).

## Questions ouvertes

- **Ligne sans quantité dans les courses** (D4) : article calculé sans quantité, un seul par ingrédient quel que soit le nombre de plats servis ? À trancher au plan des listes de courses, avec un scénario.
- **Maquettes** : modification d'un plat (D19), bandeau d'une modification croisée (D5), onglet Courses inactif (D11) n'ont pas de maquette. Les dessiner dans le canevas avant le lot C, ou accepter l'interprétation écrite au lot A ?
- **« Se déconnecter »** : sur le Planning provisoire, comme aujourd'hui ; sa place définitive (écran Tribu, « Mes appareils ») revient au plan des membres.
- **Affichage des unités** : « 3 pièce » comme la maquette, ou un pluriel (« 3 pièces ») et des abréviations (« c. à soupe ») ? À fixer au lot A dans `docs/design/README.md`.

## Notes d'exécution

*Aucune pour l'instant.*
