# Plan : tablette et ordinateur

- **Nom** : `grands-ecrans`
- **Date** : 2026-11-07
- **Statut** : en cours (lot D). Demandé par le développeur le 2026-11-07 ; D1, D6, D9 et D14 prises par le développeur le 2026-11-07 ; les autres décisions confirmées par le développeur le 2026-11-08, D12 modifiée (format posé en attribut de la racine, revue #104, point 3). Place dans la feuille de route (D1) : après `courses`, avant `courses-suite`.

## Objectif

L'application est dessinée pour le téléphone (`docs/design/README.md`, principe 4) : plus large, chaque écran reste une colonne de 480 px au milieu de la fenêtre, et la barre d'onglets s'étend seule en bas. La recette de `bibliotheque` l'a relevé (point reporté de la feuille de route : « Mise en page d'ordinateur… à maquetter dans Design, puis un plan »).

Ce plan adapte l'interface à deux usages de plus :

- **la tablette**, en portrait et surtout en **paysage** : posée dans la cuisine, sur la table au moment de faire le planning, dans le chariot pour les courses ; tactile, avec le clavier virtuel ;
- **l'ordinateur** : le planning de la semaine et la bibliothèque se préparent volontiers sur un grand écran, à la souris et au clavier.

À la fin du plan :

- toute l'application, connexion comprise, s'utilise sans défilement horizontal ni zone vide démesurée de 320 px à 2560 px de large, en portrait comme en paysage ;
- sur un grand écran, la navigation passe sur le côté, et les écrans qui s'y prêtent montrent plus à la fois : le **planning en semainier** (7 jours côte à côte, midi et soir), la **bibliothèque à côté du plat** ouvert, le **choix des listes à côté de la liste** ;
- tourner une tablette ne perd ni la saisie en cours ni l'écran affiché ;
- les cibles restent de 44 px partout (une tablette en paysage est tactile) ; survol, curseur et focus visible s'ajoutent à la souris et au clavier ;
- la version `v0.5.0` est publiée et déployée en production.

Aucun changement du métier, des données ni de l'API.

## Contexte et contraintes

- **À lire avant de commencer** : ADR 0004 (SPA Lit, CSP sans style en ligne), 0005, 0027 (copies en mémoire des collections), 0028 (scénarios `@ui`, projets Playwright) ; `docs/design/README.md` (principes, tokens, navigation, composants, « Écarts » de chaque plan) ; `docs/specs/exigences-non-fonctionnelles.md` ; `web/src/app.css` (colonne `.screen` de 480 px, barre d'onglets, connexion d'ordinateur à partir de 960 px) ; `web/src/router.ts`, `app.ts`, `tab-bar.ts`.
- **Déjà adapté** : le site public (`site/site.css`, mise en page d'ordinateur à partir de 900 px) et l'écran de connexion (`ConnexionBureau`, à partir de 960 px). Ce plan fait suivre à la connexion les formats de l'application (D3, D12) et laisse au site public son seuil, sans redessiner ces pages.
- **Après `courses`** : rien des lots B à D n'est fusionné avant l'étiquette `v0.4.0` (comme `courses`, D21) ; la PR de ce plan et le lot A, documentation seule, peuvent l'être avant. Les archives de PR de ce plan ne vont en recette qu'une fois la recette de `courses` close.
- **Où se fait le travail** : le lot A (maquettes dans Design, specs, ADR) depuis une session cloud ; les lots B à D dans le Dev Container.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`grands-ecrans/B`) ; `make ci` avant de pousser.
- **Aucune dépendance nouvelle**, ni en Go ni en npm : CSS (grilles, sélecteurs d'attribut) et `matchMedia`.
- **Pas de détection de l'appareil** : l'iPad se présente comme un Mac (point reporté « iPad vu comme un ordinateur », EF-04), une fenêtre d'ordinateur peut être étroite, une tablette Android peut avoir une souris. La mise en page suit la fenêtre (D2).
- **Les écrans à venir** (sélection d'articles de `courses-suite`, membres et appareils, bandeau hors ligne) se dessinent ensuite dans les trois formats (D13) : c'est la raison de placer ce plan avant eux (D1).
- **Recette** : le développeur recette sur Android (Chrome) et sur un ordinateur Ubuntu (Chrome), sans appareil Apple. Le développeur a une tablette Android (2026-11-07), pour le paysage tactile, la rotation et le clavier virtuel ; l'iPad n'est prouvé que par WebKit sous Playwright, comme l'iPhone.

## Décisions

Proposées le 2026-11-07 et confirmées par le développeur le 2026-11-08, sauf D1, D6, D9 et D14, prises par le développeur le 2026-11-07 ; D12 modifiée le 2026-11-08.

- **D1. Place dans la feuille de route** : après `courses` (`v0.4.0`), avant `courses-suite`, qui passe en `v0.6.0` (décision du développeur, 2026-11-07).
  - *Pour* : chaque écran construit avant ce plan sera repris ; `courses-suite` remplace la barre d'onglets par les actions de sélection (`Doux-CoursesSelection`), précisément ce que ce plan déplace ; les membres et appareils n'existent pas encore à l'écran ; l'application « viable à montrer » se montre aussi sur un ordinateur ; les écrans des courses sont frais.
  - *Contre* : retarde `courses-suite` et le hors-ligne, donc l'invitation de la famille, d'un plan. Ce plan ne bloque pas l'invitation.
  - *Autres places possibles* : après `courses-suite` (la sélection serait reprise ici, petite reprise) ; ou juste avant l'invitation de la famille (le plus de reprises, membres compris).
- **D2. La mise en page suit la taille de la fenêtre, jamais l'appareil.** Trois formats, décidés par la largeur et la hauteur de la fenêtre ; l'interaction (survol, curseur) suit `(hover: hover) and (pointer: fine)`, indépendamment du format.
- **D3. Trois formats et leurs seuils** (à ajuster sur les maquettes) :

  | Format | Condition | Appareils typiques | Mise en page |
  | --- | --- | --- | --- |
  | **compact** | moins de 720 px de large, **ou moins de 560 px de haut** | téléphone, en portrait et en paysage | celle d'aujourd'hui |
  | **moyen** | de 720 à 1023 px de large, 560 px de haut au moins | tablette en portrait, fenêtre étroite d'ordinateur | rail de navigation, une colonne élargie |
  | **étendu** | 1024 px de large et 560 px de haut au moins | tablette en paysage, ordinateur | rail de navigation, deux volets, semainier |

  La condition de hauteur garde un téléphone en paysage (844 × 390) en compact. Le contenu de l'étendu est borné en largeur (de l'ordre de 1440 px) et centré. La connexion passe à deux colonnes en format étendu (aujourd'hui 960 px), d'après l'attribut de D12 comme le reste de l'application ; le site public garde 900 px : il est hors de l'application, sans session ni rail (revue #106, point 7).
- **D4. Navigation** : en compact, la barre d'onglets en bas, inchangée ; en moyen et en étendu, un **rail vertical à gauche** (Planning, Plats, Courses : icône et libellé ; onglet actif en aplat rouge tomate, comme aujourd'hui), avec en haut le symbole et la pastille de la tribu, qui ouvre l'écran Tribu. Le rail reste à portée du pouce gauche sur une tablette tenue en paysage. Barre et rail sont la même navigation pour les lecteurs d'écran, nommée « Onglets », avec les mêmes liens : les étapes des scénarios qui ouvrent un onglet (`web/e2e/steps-*.ts`) valent dans tous les formats (revue #104, point 1).
- **D5. Deux volets (liste et détail) en étendu**, sous les mêmes chemins qu'aujourd'hui :
  - `plats` : la bibliothèque à gauche, à droite « Choisissez un plat ou créez-en un » ; `plats/<id>` et `plats/nouveau` : la bibliothèque à gauche, le plat ouvert à droite, sa carte marquée dans la bibliothèque ;
  - `courses` et `courses/<id>` : le choix des listes à gauche, la liste à droite ; `courses/listes` n'a plus d'écran propre en étendu (il montre la liste principale) ;
  - en compact et en moyen, un seul volet, comme aujourd'hui.
- **D6. Planning en étendu : un semainier**, 7 colonnes (les jours), 2 rangées (midi, soir), l'en-tête de la plage au-dessus. **Chaque colonne fait au moins 160 px** environ ; quand les 7 ne tiennent pas (tablette de 1024 px : environ 5 jours visibles), le semainier **défile sur le côté**, et se cale sur un jour quand on le lâche (`scroll-snap`), l'en-tête et les flèches de semaine restant fixes (décision du développeur, 2026-11-07, plutôt qu'un seuil du semainier à 1180 px). À la souris, le défilement horizontal doit rester possible sans pavé tactile (boutons ou molette, à voir dans Design) ; le détail d'un repas s'ouvre **en panneau latéral à droite** par-dessus le semainier (sur un voile léger), sous le chemin `repas/…` ; le fermer revient au semainier. En moyen, la liste des jours d'aujourd'hui, élargie. *Variantes à comparer dans Design* : détail en dialogue centré ; semainier rétréci à côté d'un volet fixe.
- **D7. Les feuilles deviennent des dialogues centrés** en moyen et en étendu (même `dialog`, même contenu, coins de 26 px sur les quatre côtés, largeur bornée vers 440 px) ; en compact, elles restent en bas de l'écran.
- **D8. Rien ne se perd à la rotation** : passer d'un format à l'autre garde l'écran affiché, le chemin, la saisie en cours (plat, article, nom d'une liste, période des courses des repas) et la position de défilement du volet de détail. Moyen technique (ADR, D12) : le volet de détail occupe toujours la même place dans le rendu ; le volet de liste est ajouté à côté, ou masqué par le CSS, mais l'élément du détail n'est jamais recréé par un changement de format.
- **D9. Saisie non enregistrée, en moyen et en étendu : confirmation avant de l'abandonner** (décision du développeur, 2026-11-07, qui remplace celle de la même date, « comme sur téléphone »). Quand un formulaire contient une saisie non enregistrée (un plat créé ou modifié, un article à la main, le nom d'une liste), toute navigation dans l'application qui la ferait perdre (autre élément du volet de liste, rail, lien « Plats », « Nouveau ») ouvre un dialogue de confirmation : « Abandonner la modification ? », « Ce que vous avez saisi sera perdu. », « Abandonner » en aplat, « Continuer la modification ». Une saisie identique à ce qui est enregistré n'est pas une saisie à confirmer. En compact, rien ne change : pas de confirmation. Les courses des repas n'en ont pas (une période se refait d'un geste). Le format compte au moment de la navigation : une saisie commencée sur téléphone puis tournée en paysage sur une tablette est protégée. Seules les navigations dans l'application sont gardées : ni le retour arrière du navigateur ni la fermeture de l'onglet ne demandent de confirmation (décision du développeur, 2026-11-07).
- **D10. Interaction à la souris et au clavier** : survol (liserés et fonds légèrement renforcés) et curseur sur ce qui se touche, seulement sous `(hover: hover) and (pointer: fine)` ; focus visible partout (`:focus-visible`) ; ordre de tabulation rail puis volet de liste puis détail ; Échap ferme un dialogue ou le panneau d'un repas. Pas de raccourcis clavier dans ce plan.
- **D11. Clavier virtuel** : sur une tablette en paysage, il ne doit pas faire basculer la mise en page en compact. Les navigateurs visés redimensionnent la zone visible, pas la fenêtre de mise en page (comportement par défaut de Safari et de Chrome sur Android) ; on le vérifie en recette, et le bouton fixé en bas d'un formulaire (« Enregistrer le plat ») reste visible au-dessus du clavier. Si un navigateur bascule, le format est figé tant qu'un champ a le focus (possible parce que le format est un attribut posé par `layout.ts`, D12).
- **D12. Un ADR « Mises en page selon la taille de la fenêtre »** (0030) : D2, D3, D7, D8 ; **le format est calculé par `layout.ts`** (seuils en constantes TypeScript, suivi par `matchMedia`) **et posé en attribut sur la racine** (`data-format="compact|moyen|etendu"`), que le CSS de l'application lit par des sélecteurs d'attribut au lieu de requêtes de média : une seule source des seuils, un format qu'on peut figer (D11), compatible avec la CSP (un attribut, pas un style en ligne) ; l'attribut est posé avant le premier rendu ; seul le site public, hors de l'application, garde ses requêtes de média ; la connexion lit l'attribut (revue #106, point 7) (décision du développeur, 2026-11-08, revue #104, point 3, qui remplace des requêtes de média dans le CSS et un test de concordance des seuils) ; un composant de mise en page qui place rail, volet de liste et détail ; projets Playwright par format.
- **D13. Les plans suivants dessinent chaque écran dans les trois formats.** Le principe 4 du design devient : « Usage principal sur téléphone ; tablette et ordinateur adaptés » ; les maquettes de l'application gardent le 390 × 844, et ajoutent un réglage `format` (compact, moyen, étendu) ou une maquette par grand format, selon ce que Design permet (lot A).
- **D14. Preuves** : une exigence **ENF-03** et un fichier `formats-ecran.feature`, quelques scénarios `@ui` (semainier et son défilement, deux volets, rotation sans perte de saisie, dialogue centré, confirmation avant d'abandonner une saisie, D9). Exécution en deux niveaux, pour garder la CI d'une PR courte (décision du développeur, 2026-11-07) :
  - **sur chaque PR** (`make e2e`, job `ci`) : tous les scénarios `@ui` et tests d'écran sous les téléphones (Pixel 7, iPhone 15), comme aujourd'hui, plus les seuls scénarios de `formats-ecran.feature` sous la tablette (portrait et paysage) et l'ordinateur ; quelques minutes de plus ;
  - **sur la PR de release et chaque nuit** (`make e2e-formats`, nouvelle cible) : tous les scénarios `@ui` et tests d'écran sous tous les formats, avant la recette. Le changement de `.github/workflows/` se pousse depuis l'hôte (ADR 0020, point 2).
  - Répartir les tests sur plusieurs machines de la CI reste possible plus tard, si la durée l'exige. Durées mesurées au lot B et notées dans l'ADR.
  - **Formats d'un scénario ou d'un test** (revue #104, point 1) : chaque projet Playwright a un format ; sept projets : Pixel 7 (Chromium) et iPhone 15 (WebKit), format compact ; tablette en portrait, 820 × 1180, sous Chromium et sous WebKit, format moyen ; tablette en paysage, 1180 × 820, sous Chromium et sous WebKit, et ordinateur, 1440 × 900, sous Chromium, format étendu (revue #106, point 4 : la tablette en portrait sous WebKit aussi, pour prouver la rotation sur un iPad). Un scénario `@ui` qui ne vaut que dans certains formats porte un ou plusieurs tags `@compact`, `@moyen`, `@etendu` ; **sans tag de format, il vaut pour tous**. L'export (`internal/tools/uiexport`) transmet les tags, et le registre d'étapes ne lance un scénario que dans les projets de ses formats. Un test d'écran (`*.spec.ts`) propre à un format le déclare par un `test.skip` selon le format du projet (par exemple `ecrans-onglets.spec.ts`, barre d'onglets en bas : compact seul).
  - **`pending.txt` juge toujours un scénario en entier** (ADR 0028, point 4) : un scénario est passé quand il passe dans **tous les projets de ses formats** ; il peut sortir de la liste seulement alors. `make e2e-formats`, qui lance tous les projets, applique la règle dans les deux sens : un scénario qui échoue hors de la liste, ou qui passe partout en y figurant, le fait échouer. `make e2e`, qui ne lance qu'une partie des projets pour les scénarios sans tag de format, ne juge que dans un sens : un scénario hors de la liste qui échoue le fait échouer ; un scénario de la liste qui passe sur ses seuls projets ne le fait pas échouer, puisqu'il n'a pas été essayé partout.
  - **L'ADR 0030 précise l'ADR 0028** sur ces deux points : tags de format, jugement de `pending.txt` d'après les projets d'un scénario et selon la cible.

## Écrans concernés

| Écran | Compact | Moyen | Étendu |
| --- | --- | --- | --- |
| Connexion, code, chargement | inchangés | colonne centrée | deux colonnes (`ConnexionBureau`) |
| Planning (`Doux-Main`) | inchangé | jours élargis | semainier (D6) |
| Détail d'un repas (`Doux-Repas`) | inchangé | colonne élargie | panneau latéral (D6) |
| Bibliothèque, plat (`Doux-Bibliotheque`, `Doux-Plat`, `Doux-PlatModif`) | inchangés | colonne élargie | deux volets (D5) |
| Liste, choix des listes (`Doux-Liste`, `Doux-ChoixListes`, `Doux-ArticleMain`) | inchangés | colonne élargie | deux volets (D5) |
| Courses des repas (`Doux-CoursesRepas`) | inchangé | calendrier et destination côte à côte | dans le volet de détail |
| Tribu (`Doux-Tribu`) | inchangé | colonne centrée | colonne centrée |
| Feuilles (courses faites, « ⋮ », destination) | en bas | dialogue centré (D7) | dialogue centré |

## Étapes

### Lot A : design, specs et ADR (`feature/specs-grands-ecrans`)

Depuis une session cloud, une fois ce plan revu et ses décisions confirmées.

- [x] **1. Maquettes dans Design**, canevas de travail « Menus de la semaine » : rail de navigation ; dialogue « Abandonner la modification ? » (D9) ; Planning en semainier et détail d'un repas en panneau (D6, avec ses variantes) ; bibliothèque et plat en deux volets, dont l'état vide « Choisissez un plat » ; choix des listes et liste en deux volets ; courses des repas en moyen ; dialogue centré ; chaque fois en tablette paysage (1180 × 820), tablette portrait (820 × 1180) et ordinateur (1440 × 900). Validation par le développeur. *Fait le 2026-11-08* : neuf maquettes `GE-*` (planning, plats, courses ; tablette paysage, ordinateur, tablette portrait), validées par le développeur, détail d'un repas en panneau latéral retenu (le dialogue centré et le volet fixe sont écartés) ; après validation, « Enregistrer le plat » fixé en bas du volet du plat, comme sur téléphone et comme le suppose D11.
- [x] **2. Report dans le dépôt** : maquettes dans `docs/design/maquettes/` ; `docs/design/README.md` : principe 4 (D13), section « Formats d'écran » (formats et seuils, rail, deux volets, semainier, dialogues, survol et focus), navigation accordée ; canevas de référence refait (V4). *Fait le 2026-11-08* : maquettes `GE-*`, section « Formats d'écran », canevas « Melting Tribe · design V4 · 2026-11-08 » au commit `df8522e` ; largeur du volet de liste continue, `clamp(380px, 30vw, 420px)`, sans seuil (revue #106, point 5).
- [x] **3. ENF-03** dans `exigences-non-fonctionnelles.md` et `features/formats-ecran.feature` (D14), selon `conventions-gherkin.md` ; tags de format `@compact`, `@moyen`, `@etendu` ajoutés à `conventions-gherkin.md` (sans tag, tous les formats ; à poser quand le plan d'une story fixe le niveau de preuve, comme `@ui`) et posés sur les scénarios de `formats-ecran.feature` ; titres dans `acceptance/pending.txt`.
- [x] **4. ADR 0030** « Mises en page selon la taille de la fenêtre » (D12). *Fait le 2026-11-08* : `docs/adr/0030-mises-en-page-selon-la-taille-de-la-fenetre.md`, qui précise les ADR 0004 et 0028 ; la connexion passe à deux colonnes en format étendu, d'après l'attribut `data-format` (revue #106, point 7).
- [x] **5. Feuille de route** : plan « en cours » (2026-11-08).

### Lot B : formats, navigation et dialogues (`feature/formats-et-rail`)

Tous les écrans utilisables dans les trois formats, une colonne au besoin ; pas encore de deux volets ni de semainier.

- [x] **6. `web/src/layout.ts`**, module pur testé par `node:test` : formats et seuils (D3) ; suivi du format par `matchMedia` et attribut `data-format` sur la racine, figé tant qu'un champ a le focus si le clavier virtuel fait basculer (D11, D12). *Fait le 2026-11-08* : `formatOf`, `nextFormat` (un format moyen ou étendu gardé quand la fenêtre passe en compact avec le focus dans un champ de saisie, suivi de nouveau quand le focus le quitte), `trackFormat` appelé par `main.ts` avant le premier rendu, `currentFormat` et `onFormatChange` pour les composants.
- [x] **7. Composant de mise en page** : rail ou barre d'onglets selon le format (D4), pastille de la tribu dans le rail, emplacement fixe du détail (D8) ; colonne des écrans élargie en moyen. *Fait le 2026-11-08* : `mt-tab-bar` rend la même navigation « Onglets » dans tous les formats, placée par le CSS d'après `data-format` (barre en bas, ou rail de 96 px avec le symbole et la pastille de la tribu, qui quitte alors l'en-tête du Planning) ; l'application rend toujours la navigation, puis l'écran dans `.app-content` (masquée en compact sur les écrans ouverts depuis un onglet, comme avant) ; colonne de 680 px en moyen et en étendu jusqu'aux lots C et D ; « Enregistrer le plat » à droite du rail ; connexion en deux colonnes en étendu, sa requête de média à 960 px retirée.
- [x] **8. Dialogues centrés** en moyen et en étendu (D7). *Fait le 2026-11-08* : `dialog.sheet`, 440 px au plus, coins de 26 px aux quatre angles, par le seul CSS.
- [x] **9. Souris et clavier** : survol, curseur, `:focus-visible`, ordre de tabulation, Échap (D10). *Fait le 2026-11-08* : survol (fond `--color-hover`, liseré renforcé des cartes) et curseur sous `(hover: hover) and (pointer: fine)` ; `:focus-visible` existait ; navigation avant l'écran dans le document, donc rail puis écran à la tabulation ; Échap ferme les dialogues (`showModal`, natif) ; le panneau d'un repas suit au lot C.
- [x] **10. Playwright** : cinq projets nouveaux, sept en tout (D14) : « tablette portrait » et « tablette paysage », chacun sous Chromium et sous WebKit, et « ordinateur » sous Chromium, chacun avec son format ; tags de format lus par l'export et le registre d'étapes, `test.skip` des tests d'écran propres à un format (`ecrans-onglets.spec.ts` en compact), jugement de `pending.txt` selon la cible (D14), avec leurs tests ; `make e2e` limite les nouveaux projets aux scénarios de `formats-ecran.feature`, `make e2e-formats` y passe tout (D14) ; job de la PR de release et job de nuit, poussés depuis l'hôte ; durées mesurées ; scénarios `@ui` du rail et des dialogues retirés de `pending.txt`. *Fait le 2026-11-08* : projets `phone-chromium`, `phone-webkit`, `tablet-portrait-*`, `tablet-landscape-*`, `computer-chromium`, leur format en métadonnée ; `scenarios.spec.ts` saute un scénario hors de ses formats et note l'échec d'un scénario en attente au lieu d'échouer ; `e2e/pending-reporter.ts` fait échouer l'exécution quand un scénario en attente passe dans tous les projets de ses formats (règle pure dans `e2e/formats.ts`, testée par `node:test`) ; `ecrans-formats.spec.ts` (320, 720, 1023, 1024 et 2560 px, revue #106, point 6) ; `ecrans-onglets.spec.ts` en compact seul ; workflow `e2e-formats.yml` (PR `release/…`, chaque nuit, à la demande), dernier commit de la branche, poussé depuis l'hôte ; cinq scénarios retirés de `pending.txt`. Durées dans l'ADR 0030.

### Lot C : planning en semainier (`feature/planning-hebdo`)

- [x] **11. Semainier** en étendu (D6) : 7 colonnes de 160 px au moins, midi et soir, cases et pilules d'après les maquettes, défilement horizontal calé sur un jour quand elles ne tiennent pas, en-tête de la plage et flèches fixes ; bornes de 2000 à 2099 comme aujourd'hui ; jours élargis en moyen. *Fait le 2026-11-08* : mêmes jours et mêmes cases qu'en compact, disposés par le CSS d'après `data-format` (grille de 7 colonnes `minmax(160px, 1fr)`, `scroll-snap`, hauteur de la fenêtre) ; « Menus » et la plage sur une ligne ; boutons « Jours précédents » et « Jours suivants » tant qu'il reste des jours de ce côté (`scrollEdges` dans `layout.ts`), qui font défiler d'un jour ; une autre semaine repart de son premier jour ; en moyen, la colonne de 680 px du lot B suffit.
- [x] **12. Détail d'un repas en panneau** (D6), sous le même chemin ; fermeture par la croix, Échap ou le voile, retour à la même plage ; rien de perdu à la rotation (D8). *Fait le 2026-11-08* : voir les notes d'exécution du lot C.
- [x] **13. Scénarios `@ui`** du semainier, retirés de `pending.txt`. *Fait le 2026-11-08* : cinq scénarios (semainier, défilement, 7 jours sur l'ordinateur, panneau, Échap) ; `ecrans-semainier.spec.ts` (défilement gardé à la fermeture, focus, voile, copies vérifiées, autre semaine, rotation), fenêtre de tablette posée par le test pour tourner sur chaque PR.

### Lot D : deux volets pour les plats et les courses (`feature/deux-volets`)

- [x] **14. Plats** en deux volets (D5) : bibliothèque et recherche à gauche, plat ouvert ou « Choisissez un plat ou créez-en un » à droite, carte du plat ouvert marquée. *Fait le 2026-11-08* : voir les notes d'exécution du lot D.
- [x] **14 bis. Confirmation avant d'abandonner une saisie** (D9) : état « saisie non enregistrée » du plat, de l'article à la main et du nom d'une liste (comparé à ce qui est enregistré) ; garde sur les navigations de l'application en moyen et en étendu ; dialogue « Abandonner la modification ? » ; scénarios `@ui` retirés de `pending.txt`. *Fait le 2026-11-08* : `inputDiffers` (`dishes.ts`), `unsaved` des trois écrans, garde dans `mt-app` d'après `lostBy` (`panes.ts`) ; cinq scénarios retirés.
- [x] **15. Courses** en deux volets (D5) : choix des listes à gauche, liste à droite ; courses des repas dans le volet de détail ; « Nouvelle liste » ouvre la liste créée à droite. *Fait le 2026-11-08* : et courses des repas avec le calendrier à gauche en moyen et en étendu (reporté du lot B).
- [x] **16. Rotation** : scénario « Tourner la tablette ne perd pas la saisie » sur un plat et sur un article (D8), retirés de `pending.txt`. *Fait le 2026-11-08* : et `ecrans-volets.spec.ts` (même élément du plat, saisie et défilement gardés dans les deux sens).

### Recette et release

- [ ] **17. Recette** sur l'archive de la PR `release/v0.5.0` : check-list ci-dessous.
- [ ] **18. Release `v0.5.0`** selon `RELEASING.md`, déployée en production.
- [ ] **19. Clôture** : statut « terminé » ; feuille de route (point reporté « Mise en page d'ordinateur » retiré) ; `CLAUDE.md` ; `CHANGELOG.md`.

### Séance de recette (étape 17)

Par le développeur, guidé pas à pas, sur trois appareils : téléphone Android, tablette Android, ordinateur.

- [ ] Téléphone : rien n'a changé, en portrait ni en paysage (le paysage reste en compact).
- [ ] Tablette en portrait : rail, colonnes élargies, dialogues centrés.
- [ ] Tablette en paysage : semainier, son défilement au doigt calé sur un jour, détail d'un repas en panneau ; deux volets des plats et des courses ; cibles confortables au doigt.
- [ ] Rotation pendant la saisie d'un plat, d'un article et d'une période : rien n'est perdu, l'écran reste le même.
- [ ] Clavier virtuel ouvert en paysage : la mise en page ne bascule pas, « Enregistrer le plat » reste visible (D11).
- [ ] Ordinateur, fenêtre large puis réduite jusqu'au téléphone : passage d'un format à l'autre sans perte ; survol et focus ; tout au clavier, Échap compris.
- [ ] Tablette et ordinateur : un plat modifié non enregistré, puis un autre plat, le rail ou « Nouveau » : « Abandonner la modification ? » ; « Continuer la modification » garde la saisie, « Abandonner » la perd. Même essai sur le téléphone : pas de confirmation.
- [ ] Connexion sur la tablette et l'ordinateur.
- [ ] Ordinateur, fenêtre large mais de moins de 560 px de haut (fenêtre réduite en hauteur, ou portable 1366 × 768 avec des barres d'outils) : l'application passe en compact, barre d'onglets en bas (D3).

## Critères de validation

- `make ci` et `make e2e` verts, scénarios de `formats-ecran.feature` sortis de `pending.txt`.
- Recette passée sur les trois appareils ; iPad et Safari sur Mac prouvés par WebKit seul, ajoutés au point reporté « Safari sur un vrai iPhone » de la feuille de route.
- En compact, aucun écran ne change.

## Notes d'exécution

- **Revue #104, point 3, tranché le 2026-11-08** (D12, reporté dans l'ADR 0030, points 2 et 3) : D11 prévoit, si un navigateur bascule à l'ouverture du clavier virtuel, de figer le format tant qu'un champ a le focus ; or D12 faisait décider la mise en page par des requêtes de média du CSS, que TypeScript ne peut pas suspendre. Option relevée par la revue, retenue : `layout.ts` pose un attribut sur la racine (`data-format="etendu"`), que le CSS lit au lieu de requêtes de média ; le format peut alors être figé, le test de concordance des seuils devient inutile, et c'est compatible avec la CSP (un attribut, pas un style).
- **Revue #104, point 4 (lot B, étape 10)** : la nouvelle cible `make e2e-formats` entre dans la liste des cibles de `CLAUDE.md` et dans `make help` (ADR 0010).
- **Revue #106, point 6 (lot B, étape 10)** : rien ne prouve encore l'absence de défilement horizontal de 320 à 2560 px (ENF-03) : aucun projet ne descend sous 390 px ni ne dépasse 1440 px. Un test d'écran parcourt quelques largeurs (320, 720, 1023, 1024, 2560) et vérifie `scrollWidth <= clientWidth` sur chaque écran.

- **Lot B, mis en œuvre le 2026-11-08** :
  - **Tous les scénarios `@ui` existants passent déjà dans les sept projets** (`make e2e-formats`) : la colonne élargie suffit à les faire valoir en moyen et en étendu. Seuls des tests d'écran supposaient le compact (pas de navigation sur un écran ouvert depuis un onglet, un seul lien « Planning » ou « Plats ») : ils lisent le format de leur projet (`tabsOnSubScreen`) ou cherchent le lien de retour dans l'écran (`getByRole("main")`) ; le toucher d'un bouton (`tap`) est sauté sans écran tactile (ordinateur).
  - **La navigation précède l'écran dans le document, en compact aussi** (D10 : rail, puis écran) : l'ordre de lecture et de tabulation du téléphone change (onglets avant l'écran), pas son affichage.
  - **Une seule lecture de la tribu** : le rail et le Planning la demandent ensemble ; `Api.getTribe` partage la lecture en cours et annonce chaque tribu lue ou changée (`tribeEvent`), que suit la pastille du rail (sinon « Réessayer » sur le Planning, en grand format, ne la faisait pas apparaître). En compact, le rail ne la lit pas.
  - **Parcours de connexion réel sous le seul téléphone Chromium** : sous quatre projets Chromium, il dépassait les demandes de code de l'heure ; les écrans de la connexion sont prouvés dans tous les formats par `ecrans-connexion.spec.ts`.
  - **Pas encore fait** : courses des repas avec le calendrier et la destination côte à côte en moyen (« Écrans concernés ») : une colonne d'ici le lot D, qui place les courses des repas dans le volet de détail.
  - **Revue #107, point 3 : le composant de mise en page n'existe pas encore**, malgré l'étape 7 cochée. Au lot B, D8 tient plus simplement : `mt-app` rend toujours la navigation puis l'écran dans `.app-content`, et rien ne réagit au format. Le composant qui place le volet de liste et le détail (ADR 0030, point 4) reste à faire au lot D (étapes 14 et 15), où les scénarios de rotation (étape 16) le prouveront.
  - **Revue #107, lien d'évitement** : sur téléphone, un lecteur d'écran entend les trois onglets avant chaque écran (la navigation précède l'écran, D8 et D10). Un lien « Aller au contenu », comme sur le site public, est possible ; à juger à la recette (étape 17), décision du développeur du 2026-11-08.

- **Lot C, mis en œuvre le 2026-11-08** :
  - **Le Planning reste sous le panneau, le même élément** : en étendu, sur `repas/…`, l'application rend le Planning de la plage de retour (la dernière affichée, ou celle qui tient le repas) à la première place du contenu, puis le repas à la seconde, toujours dans `.meal-panel` ; le CSS en fait un panneau de 440 px sur un voile en étendu, et rien ailleurs. Le semainier garde ainsi son défilement à la fermeture, et une rotation ne recrée jamais l'écran du repas (D8) : en compact et en moyen, le Planning disparaît et le repas reste, avec sa recherche.
  - **Couvert, le Planning est `inert` et `aria-hidden`** : un seul `main` et un seul titre de niveau 1, ceux du repas (Playwright ne tient pas compte d'`inert`) ; le rail reste accessible. Pas de rôle `dialog` sur le panneau, qui n'est pas modal pour le rail. Découvert, le Planning vérifie les copies, comme à chaque visite (plan `planning`, D15), puisqu'il n'est pas recréé.
  - **Focus et clavier** (D10) : le repas ouvert depuis le semainier met le focus sur « Fermer le repas » ; fermé, le focus revient à sa case ; Échap le ferme, sauf si un dialogue est ouvert. La croix est un lien vers la plage de retour, comme « Planning », masqué en étendu.
  - **Tests d'écran du repas en étendu** : le retour au Planning passe par la croix (`mealBackLink`, `api-mock.ts`).
  - **Branche `feature/planning-hebdo`** au lieu de `feature/semainier` : « semainier » contient `main`, et la règle `git push *main*` de `.claude/settings.json` en refusait le push (décision du développeur, 2026-11-08) ; règle de nommage des branches ajoutée à `CLAUDE.md`.

- **Lot D, mis en œuvre le 2026-11-08** :
  - **Pas de composant de mise en page à part** (revue #107, point 3 ; ADR 0030, point 4, note de mise en œuvre validée par le développeur le 2026-11-08) : `web/src/panes.ts`, module pur testé par `node:test`, dit pour une page et un format quel volet de liste (bibliothèque, choix des listes) et quel détail (plat, liste, courses des repas, invitation « Choisissez un plat ») s'affichent ; `mt-app` rend chacun à sa place dans `.app-content` (Planning, volet de liste, détail, autre écran), le détail sous une clé qui ne dépend pas du format. Un changement de format ajoute ou retire le volet de liste et ne recrée jamais le détail (D8), comme le Planning sous le panneau d'un repas au lot C. Le CSS place les deux volets d'après `data-format` et la classe `app-content--panes`.
  - **Ce qu'une navigation ferait perdre** (D9) se déduit des mêmes fonctions (`lostBy`) : un volet dont le contenu change. Ouvrir une autre liste en étendu garde donc le nom d'une liste en cours de création dans le choix des listes, sans confirmation ; quitter l'onglet la demande. Seuls les liens de l'application sont gardés (`#followLink`), jamais le retour arrière du navigateur. Le dialogue met le focus sur « Continuer la modification », l'action sans perte ; Échap et le voile valent « Continuer ».
  - **Un seul `main`** : à côté d'un détail, la bibliothèque et le choix des listes sont une région (`section`, rôle `main` quand ils sont seuls), titrée en niveau 2 ; le détail garde son `main` et son titre de niveau 1 (la maquette met un `main` autour des deux volets et le titre du plat en niveau 2 ; écart gardé, décision du développeur, 2026-11-08). Seuls le rôle et le titre changent avec le format : le champ de recherche ou le nom d'une nouvelle liste ne sont pas recréés.
  - **Défilement à la rotation** (D8) : en un volet, c'est la fenêtre qui défile, en deux volets le détail ; `mt-app` reporte la position de l'un à l'autre au changement de format.
  - **Après l'enregistrement d'un plat en étendu**, le plat reste ouvert avec « Plat enregistré. », comme la maquette (décision du développeur, 2026-11-08) ; un plat créé s'ouvre sous son chemin `plats/<id>`. Le message disparaît à la première modification. En compact et en moyen, retour à la bibliothèque, comme avant.
  - **Tests d'écran** : en étendu, plusieurs se lisaient sur un seul volet (deux liens « Nouveau », deux alertes de réseau, titre de la liste devenu un titre seul) ; ils cherchent dans le `main` ou lisent le format (`twoPanes`, `api-mock.ts`). L'étape « j'ouvre le choix des listes » ne touche plus le titre en étendu.

## Questions ouvertes

1. *Fermée le 2026-11-07* : le développeur recette sur une tablette Android.
2. *Fermée le 2026-11-07* : confirmation avant d'abandonner une saisie en moyen et en étendu, pas en compact (D9).
3. *Fermée le 2026-11-07* : deux niveaux d'exécution, chaque PR et release ou nuit (D14).
4. *Fermée le 2026-11-07* : semainier qui défile sur le côté, colonnes de 160 px au moins (D6).
5. **Orientation de la PWA installée** : le manifeste (plan hors-ligne) ne fige pas l'orientation (`orientation: any`), à noter dans ce plan-là.
6. *Fermée le 2026-11-07* : navigations de l'application seulement, ni retour arrière du navigateur ni fermeture de l'onglet (D9).
