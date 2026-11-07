# Plan : tablette et ordinateur

- **Nom** : `grands-ecrans`
- **Date** : 2026-11-07
- **Statut** : brouillon. Demandé par le développeur le 2026-11-07 ; D9 prise par le développeur le 2026-11-07 ; D1 à D8 et D10 à D14 proposées, à confirmer à la revue de ce plan. Place dans la feuille de route proposée (D1) : après `courses`, avant `courses-suite`.

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
- **Déjà adapté** : le site public (`site/site.css`, mise en page d'ordinateur à partir de 900 px) et l'écran de connexion (`ConnexionBureau`, à partir de 960 px). Ce plan aligne ces seuils sur les siens (D3) sans redessiner ces pages.
- **Après `courses`** : rien des lots B à D n'est fusionné avant l'étiquette `v0.4.0` (comme `courses`, D21) ; la PR de ce plan et le lot A, documentation seule, peuvent l'être avant. Les archives de PR de ce plan ne vont en recette qu'une fois la recette de `courses` close.
- **Où se fait le travail** : le lot A (maquettes dans Design, specs, ADR) depuis une session cloud ; les lots B à D dans le Dev Container.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`grands-ecrans/B`) ; `make ci` avant de pousser.
- **Aucune dépendance nouvelle**, ni en Go ni en npm : CSS (grilles, requêtes de média) et `matchMedia`.
- **Pas de détection de l'appareil** : l'iPad se présente comme un Mac (point reporté « iPad vu comme un ordinateur », EF-04), une fenêtre d'ordinateur peut être étroite, une tablette Android peut avoir une souris. La mise en page suit la fenêtre (D2).
- **Les écrans à venir** (sélection d'articles de `courses-suite`, membres et appareils, bandeau hors ligne) se dessinent ensuite dans les trois formats (D13) : c'est la raison de placer ce plan avant eux (D1).
- **Recette** : le développeur recette sur Android (Chrome) et sur un ordinateur Ubuntu (Chrome), sans appareil Apple. Le développeur a une tablette Android (2026-11-07), pour le paysage tactile, la rotation et le clavier virtuel ; l'iPad n'est prouvé que par WebKit sous Playwright, comme l'iPhone.

## Décisions

Proposées le 2026-11-07, à confirmer à la revue de ce plan, sauf D9, prise par le développeur le 2026-11-07.

- **D1. Place dans la feuille de route** : après `courses` (`v0.4.0`), avant `courses-suite`, qui passe en `v0.6.0`.
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

  La condition de hauteur garde un téléphone en paysage (844 × 390) en compact. Le contenu de l'étendu est borné en largeur (de l'ordre de 1440 px) et centré. La connexion passe à deux colonnes à partir de l'étendu (aujourd'hui 960 px), le site public garde 900 px : il n'a pas de session et pas de rail.
- **D4. Navigation** : en compact, la barre d'onglets en bas, inchangée ; en moyen et en étendu, un **rail vertical à gauche** (Planning, Plats, Courses : icône et libellé ; onglet actif en aplat rouge tomate, comme aujourd'hui), avec en haut le symbole et la pastille de la tribu, qui ouvre l'écran Tribu. Le rail reste à portée du pouce gauche sur une tablette tenue en paysage.
- **D5. Deux volets (liste et détail) en étendu**, sous les mêmes chemins qu'aujourd'hui :
  - `plats` : la bibliothèque à gauche, à droite « Choisissez un plat ou créez-en un » ; `plats/<id>` et `plats/nouveau` : la bibliothèque à gauche, le plat ouvert à droite, sa carte marquée dans la bibliothèque ;
  - `courses` et `courses/<id>` : le choix des listes à gauche, la liste à droite ; `courses/listes` n'a plus d'écran propre en étendu (il montre la liste principale) ;
  - en compact et en moyen, un seul volet, comme aujourd'hui.
- **D6. Planning en étendu : un semainier**, 7 colonnes (les jours), 2 rangées (midi, soir), l'en-tête de la plage au-dessus ; le détail d'un repas s'ouvre **en panneau latéral à droite** par-dessus le semainier (sur un voile léger), sous le chemin `repas/…` ; le fermer revient au semainier. En moyen, la liste des jours d'aujourd'hui, élargie. *Variantes à comparer dans Design* : détail en dialogue centré ; semainier rétréci à côté d'un volet fixe.
- **D7. Les feuilles deviennent des dialogues centrés** en moyen et en étendu (même `dialog`, même contenu, coins de 26 px sur les quatre côtés, largeur bornée vers 440 px) ; en compact, elles restent en bas de l'écran.
- **D8. Rien ne se perd à la rotation** : passer d'un format à l'autre garde l'écran affiché, le chemin, la saisie en cours (plat, article, nom d'une liste, période des courses des repas) et la position de défilement du volet de détail. Moyen technique (ADR, D12) : le volet de détail occupe toujours la même place dans le rendu ; le volet de liste est ajouté à côté, ou masqué par le CSS, mais l'élément du détail n'est jamais recréé par un changement de format.
- **D9. Changer d'élément dans le volet de liste** pendant une saisie (un autre plat, une autre liste) : comme sur téléphone, quand on revient par « Plats » : la saisie non enregistrée est abandonnée, sans confirmation (décision du développeur, 2026-11-07).
- **D10. Interaction à la souris et au clavier** : survol (liserés et fonds légèrement renforcés) et curseur sur ce qui se touche, seulement sous `(hover: hover) and (pointer: fine)` ; focus visible partout (`:focus-visible`) ; ordre de tabulation rail puis volet de liste puis détail ; Échap ferme un dialogue ou le panneau d'un repas. Pas de raccourcis clavier dans ce plan.
- **D11. Clavier virtuel** : sur une tablette en paysage, il ne doit pas faire basculer la mise en page en compact. Les navigateurs visés redimensionnent la zone visible, pas la fenêtre de mise en page (comportement par défaut de Safari et de Chrome sur Android) ; on le vérifie en recette, et le bouton fixé en bas d'un formulaire (« Enregistrer le plat ») reste visible au-dessus du clavier. Si un navigateur bascule, le format est figé tant qu'un champ a le focus.
- **D12. Un ADR « Mises en page selon la taille de la fenêtre »** (0030) : D2, D3, D7, D8 ; seuils en constantes TypeScript (`layout.ts`) et en requêtes de média du CSS, un test vérifiant qu'ils concordent (la CSP interdit tout style en ligne, les requêtes de média ne lisent pas de variables CSS) ; un composant de mise en page qui place rail, volet de liste et détail ; projets Playwright par format.
- **D13. Les plans suivants dessinent chaque écran dans les trois formats.** Le principe 4 du design devient : « Usage principal sur téléphone ; tablette et ordinateur adaptés » ; les maquettes de l'application gardent le 390 × 844, et ajoutent un réglage `format` (compact, moyen, étendu) ou une maquette par grand format, selon ce que Design permet (lot A).
- **D14. Preuves** : une exigence **ENF-03** et un fichier `formats-ecran.feature`, quelques scénarios `@ui` (semainier, deux volets, rotation sans perte de saisie, dialogue centré) exécutés dans les projets Playwright de la tablette et de l'ordinateur ; les scénarios `@ui` existants restent sous les téléphones (Pixel 7, iPhone 15), plus un projet « ordinateur » sous Chromium. Les tests d'écran existants (états des écrans sur une API simulée) gagnent les formats moyen et étendu. Durée de `make e2e` à mesurer (question ouverte 3).

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

- [ ] **1. Maquettes dans Design**, canevas de travail « Menus de la semaine » : rail de navigation ; Planning en semainier et détail d'un repas en panneau (D6, avec ses variantes) ; bibliothèque et plat en deux volets, dont l'état vide « Choisissez un plat » ; choix des listes et liste en deux volets ; courses des repas en moyen ; dialogue centré ; chaque fois en tablette paysage (1180 × 820), tablette portrait (820 × 1180) et ordinateur (1440 × 900). Validation par le développeur.
- [ ] **2. Report dans le dépôt** : maquettes dans `docs/design/maquettes/` ; `docs/design/README.md` : principe 4 (D13), section « Formats d'écran » (formats et seuils, rail, deux volets, semainier, dialogues, survol et focus), navigation accordée ; canevas de référence refait (V4).
- [ ] **3. ENF-03** dans `exigences-non-fonctionnelles.md` et `features/formats-ecran.feature` (D14), selon `conventions-gherkin.md` ; titres dans `acceptance/pending.txt`.
- [ ] **4. ADR 0030** « Mises en page selon la taille de la fenêtre » (D12).
- [ ] **5. Feuille de route** : plan « en cours ».

### Lot B : formats, navigation et dialogues (`feature/formats-et-rail`)

Tous les écrans utilisables dans les trois formats, une colonne au besoin ; pas encore de deux volets ni de semainier.

- [ ] **6. `web/src/layout.ts`**, module pur testé par `node:test` : formats et seuils (D3), avec le test de concordance avec `app.css` (D12) ; suivi du format par `matchMedia`.
- [ ] **7. Composant de mise en page** : rail ou barre d'onglets selon le format (D4), pastille de la tribu dans le rail, emplacement fixe du détail (D8) ; colonne des écrans élargie en moyen.
- [ ] **8. Dialogues centrés** en moyen et en étendu (D7).
- [ ] **9. Souris et clavier** : survol, curseur, `:focus-visible`, ordre de tabulation, Échap (D10).
- [ ] **10. Playwright** : projets « tablette portrait », « tablette paysage » (Chromium et WebKit) et « ordinateur » (Chromium) ; tests d'écran existants dans les nouveaux formats ; scénarios `@ui` du rail et des dialogues retirés de `pending.txt`.

### Lot C : planning en semainier (`feature/semainier`)

- [ ] **11. Semainier** en étendu (D6) : 7 colonnes, midi et soir, cases et pilules d'après les maquettes, en-tête de la plage et flèches ; bornes de 2000 à 2099 comme aujourd'hui ; jours élargis en moyen.
- [ ] **12. Détail d'un repas en panneau** (D6), sous le même chemin ; fermeture par la croix, Échap ou le voile, retour à la même plage ; rien de perdu à la rotation (D8).
- [ ] **13. Scénarios `@ui`** du semainier, retirés de `pending.txt`.

### Lot D : deux volets pour les plats et les courses (`feature/deux-volets`)

- [ ] **14. Plats** en deux volets (D5) : bibliothèque et recherche à gauche, plat ouvert ou « Choisissez un plat ou créez-en un » à droite, carte du plat ouvert marquée ; changement de plat pendant une saisie (D9).
- [ ] **15. Courses** en deux volets (D5) : choix des listes à gauche, liste à droite ; courses des repas dans le volet de détail ; « Nouvelle liste » ouvre la liste créée à droite.
- [ ] **16. Rotation** : scénario « Tourner la tablette ne perd pas la saisie » sur un plat et sur un article (D8), retirés de `pending.txt`.

### Recette et release

- [ ] **17. Recette** sur l'archive de la PR `release/v0.5.0` : check-list ci-dessous.
- [ ] **18. Release `v0.5.0`** selon `RELEASING.md`, déployée en production.
- [ ] **19. Clôture** : statut « terminé » ; feuille de route (point reporté « Mise en page d'ordinateur » retiré) ; `CLAUDE.md` ; `CHANGELOG.md`.

### Séance de recette (étape 17)

Par le développeur, guidé pas à pas, sur trois appareils : téléphone Android, tablette Android, ordinateur.

- [ ] Téléphone : rien n'a changé, en portrait ni en paysage (le paysage reste en compact).
- [ ] Tablette en portrait : rail, colonnes élargies, dialogues centrés.
- [ ] Tablette en paysage : semainier, détail d'un repas en panneau ; deux volets des plats et des courses ; cibles confortables au doigt.
- [ ] Rotation pendant la saisie d'un plat, d'un article et d'une période : rien n'est perdu, l'écran reste le même.
- [ ] Clavier virtuel ouvert en paysage : la mise en page ne bascule pas, « Enregistrer le plat » reste visible (D11).
- [ ] Ordinateur, fenêtre large puis réduite jusqu'au téléphone : passage d'un format à l'autre sans perte ; survol et focus ; tout au clavier, Échap compris.
- [ ] Connexion sur la tablette et l'ordinateur.

## Critères de validation

- `make ci` et `make e2e` verts, scénarios de `formats-ecran.feature` sortis de `pending.txt`.
- Recette passée sur les trois appareils ; iPad et Safari sur Mac prouvés par WebKit seul, ajoutés au point reporté « Safari sur un vrai iPhone » de la feuille de route.
- Aucun écran du téléphone modifié, hors le pied de page et la pastille de la tribu s'ils bougent en moyen et en étendu.

## Questions ouvertes

1. *Fermée le 2026-11-07* : le développeur recette sur une tablette Android.
2. *Fermée le 2026-11-07* : saisie abandonnée sans confirmation, comme sur téléphone (D9).
3. **Projets Playwright** : tous les scénarios `@ui` sous tous les formats, ou seulement ceux de `formats-ecran.feature` (D14) ? Dépend de la durée de `make e2e` en CI.
4. **Semainier sur une tablette de 1024 px** : 7 colonnes d'environ 130 px avec le rail ; les noms de plats sont fortement tronqués. Seuil du semainier plus haut que celui de l'étendu (par exemple 1180 px), et jours sur deux rangées en dessous ?
5. **Orientation de la PWA installée** : le manifeste (plan hors-ligne) ne fige pas l'orientation (`orientation: any`), à noter dans ce plan-là.
