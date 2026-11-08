# 0030. Mises en page selon la taille de la fenêtre

- **Date** : 2026-11-08
- **Statut** : accepté

## Contexte

L'application est dessinée pour le téléphone (`docs/design/README.md`, principe 4) : plus large, chaque écran reste une colonne de 480 px (`web/src/app.css`), et seule la barre d'onglets s'étend. Le plan `grands-ecrans` l'adapte à la tablette, en portrait et en paysage, et à l'ordinateur (ENF-03) : rail de navigation, planning en semainier, deux volets pour les plats et les courses, dialogues centrés à la place des feuilles (maquettes `GE-*`, validées le 2026-11-08).

Les forces en présence :

- **L'appareil ne se détecte pas de façon fiable** : l'iPad se présente comme un Mac, une fenêtre d'ordinateur peut être étroite, une tablette Android peut avoir une souris.
- **La CSP interdit tout style en ligne** (ADR 0004) : TypeScript ne peut pas poser de largeur ni de position, seulement des classes et des attributs.
- **Les requêtes de média ne lisent pas de variables CSS**, et TypeScript ne peut pas les suspendre : un format décidé par le CSS ne peut pas être figé, alors que le plan veut le figer si un navigateur fait basculer une tablette en compact à l'ouverture du clavier virtuel (D11 ; revue #104, point 3).
- **Rien ne doit se perdre en changeant de format** (D8) : la saisie d'un formulaire n'existe que dans son composant Lit, en mémoire, jusqu'à l'enregistrement ; un composant recréé la perd.
- **Les composants Lit rendent dans le DOM principal** (`createRenderRoot` renvoie l'élément, pas de shadow DOM) : le CSS global de l'application s'applique à tous les écrans.
- **Les scénarios `@ui` et les tests d'écran tournent sous deux téléphones** (Pixel 7, iPhone 15 ; ADR 0028, point 6) ; les grands formats demandent d'autres projets Playwright, et la CI d'une PR doit rester courte.

Décidé avec le développeur les 2026-11-07 et 2026-11-08 (plan `grands-ecrans`, D2, D3, D7, D8, D11, D12, D14).

## Décision

1. **Trois formats, décidés par la fenêtre, jamais par l'appareil** :

   | Format | Condition |
   | --- | --- |
   | `compact` | moins de 720 px de large, ou moins de 560 px de haut |
   | `moyen` | de 720 à 1023 px de large, 560 px de haut au moins |
   | `etendu` | 1024 px de large et 560 px de haut au moins |

   La condition de hauteur garde un téléphone en paysage en compact. L'interaction (survol, curseur) ne dépend pas du format : elle suit `(hover: hover) and (pointer: fine)`, en requête de média du CSS.
2. **`web/src/layout.ts` calcule le format et le pose en attribut sur la racine du document** : `data-format="compact|moyen|etendu"` sur `<html>`. Les seuils sont des constantes de ce module, et nulle part ailleurs ; un module pur (format d'une largeur et d'une hauteur), testé par `node:test`, et un suivi par `matchMedia` sur les seuils, qui met l'attribut à jour. L'attribut est posé avant le premier rendu de l'application, dans `main.ts`, et vaut pour tous ses écrans, connexion et chargement compris.
3. **Le CSS de l'application lit l'attribut, pas de requête de média de largeur** : `:root[data-format="etendu"] .…`. C'est compatible avec la CSP (un attribut, pas un style), les seuils n'ont qu'une source, et le format peut être figé : si l'ouverture du clavier virtuel fait passer une tablette de `moyen` ou `etendu` à `compact` alors qu'un champ a le focus, `layout.ts` garde le format tant que le focus reste dans un champ (D11). La connexion, qui fait partie de la SPA, lit l'attribut comme le reste : elle passe à deux colonnes (`ConnexionBureau`) en format étendu, et sa requête de média à 960 px disparaît. **Seule exception : le site public** (`site/site.css`, mise en page d'ordinateur à partir de 900 px), hors de la SPA, statique et sans script, garde ses requêtes de média (revue #106, point 7).
4. **Un composant de mise en page place le rail ou la barre d'onglets, le volet de liste et le détail.** Le **volet de détail occupe toujours la même place dans le rendu** : le volet de liste est ajouté à côté de lui ou masqué par le CSS, mais un changement de format ne recrée jamais l'élément du détail (D8). La saisie en cours, le chemin et la position de défilement du détail survivent ainsi à la rotation sans être copiés hors des composants. Les composants qui affichent un même écran différemment selon le format (semainier ou liste de jours) lisent le format par `layout.ts` ; ils ne détruisent pas pour autant un formulaire en cours.
   *Mise en œuvre (plan `grands-ecrans`, lots C et D, 2026-11-08 ; revue #107, point 3), validée par le développeur : pas de composant à part. Un module pur, `web/src/panes.ts`, dit pour une page et un format le volet de liste et le détail affichés, et ce qu'une navigation ferait perdre (D9) ; `mt-app` rend chacun à sa place dans `.app-content`, le détail sous une clé qui ne dépend pas du format, et le CSS les place d'après `data-format`. La décision est inchangée : un changement de format ne recrée jamais le détail. Son défilement passe de la fenêtre au volet, et inversement, d'après la position lue juste avant le changement (`onFormatWillChange`).*
5. **Les feuilles deviennent des dialogues centrés en moyen et en étendu** (D7) : même élément `dialog`, même contenu, seuls ses styles changent avec l'attribut.
6. **Formats des tests Playwright** : chaque projet a un format. Sept projets :

   | Projet | Navigateur | Fenêtre | Format |
   | --- | --- | --- | --- |
   | Pixel 7 | Chromium | celle de l’appareil Playwright | `compact` |
   | iPhone 15 | WebKit | celle de l’appareil Playwright | `compact` |
   | tablette en portrait | Chromium | 820 × 1180 | `moyen` |
   | tablette en portrait | WebKit | 820 × 1180 | `moyen` |
   | tablette en paysage | Chromium | 1180 × 820 | `etendu` |
   | tablette en paysage | WebKit | 1180 × 820 | `etendu` |
   | ordinateur | Chromium | 1440 × 900 | `etendu` |

   La tablette, en portrait comme en paysage, tourne sous WebKit en plus de Chromium : c'est la seule preuve pour l'iPad, faute d'appareil Apple en recette, rotation comprise.
   - Un scénario `@ui` qui ne vaut que dans certains formats porte un ou plusieurs tags `@compact`, `@moyen`, `@etendu` ; **sans tag de format, il vaut pour tous** (`docs/specs/conventions-gherkin.md`). L'outil d'export (`internal/tools/uiexport`) transmet les tags ; le registre d'étapes ne lance un scénario que dans les projets de ses formats.
   - Un test d'écran (`*.spec.ts`) propre à un format le déclare par un `test.skip` selon le format du projet (`ecrans-onglets.spec.ts`, barre d'onglets en bas : compact seul).
7. **Deux niveaux d'exécution** (D14) : `make e2e`, sur chaque PR, lance tous les scénarios `@ui` et tests d'écran sous les deux téléphones, et seulement les scénarios de `formats-ecran.feature` sous la tablette et l'ordinateur ; `make e2e-formats`, sur la PR de release et chaque nuit, lance tout dans tous les projets. Le changement des workflows se pousse depuis l'hôte (ADR 0020, point 2). Les durées mesurées au lot B sont notées ci-dessous.
8. **`acceptance/pending.txt` juge un scénario `@ui` d'après tous les projets de ses formats** : il est passé quand il passe dans chacun d'eux. `make e2e-formats` applique la règle de l'ADR 0028 dans les deux sens. `make e2e`, qui ne lance qu'une partie des projets pour les scénarios sans tag de format, ne juge que dans un sens : un scénario hors de la liste qui échoue le fait échouer ; un scénario de la liste qui passe sur les seuls projets lancés ne le fait pas échouer, puisqu'il n'a pas été essayé partout.

## Alternatives envisagées

- **Requêtes de média dans le CSS, seuils recopiés en TypeScript et test de leur concordance** (proposition initiale de D12) : écartée le 2026-11-08 (revue #104, point 3). Deux sources des seuils, et un format qu'on ne peut pas figer pendant la saisie au clavier virtuel.
- **Détecter l'appareil** (agent utilisateur, `pointer: coarse`) : écartée. L'iPad se présente comme un Mac, et une tablette peut avoir une souris ; la mise en page suit ce que l'on voit, la taille de la fenêtre.
- **Requêtes de conteneur** (`@container`) : écartées pour ce plan. La mise en page dépend de la fenêtre entière (rail ou barre, un ou deux volets), pas de la largeur d'un composant ; elles restent possibles plus tard à l'intérieur d'un volet.
- **Copier la saisie en cours hors des composants** (mémoire commune, `sessionStorage`) pour la restaurer après une rotation : écartée. À faire pour chaque formulaire, présent et à venir, avec la position de défilement ; un formulaire oublié perdrait sa saisie. La règle du point 4 protège tous les écrans par construction.
- **Exécuter tous les formats sur chaque PR** : écartée (D14), pour garder la CI d'une PR courte ; répartir les tests sur plusieurs machines reste possible si la durée l'exige.

## Conséquences

- **Positif** : une seule source des seuils ; un format qu'on peut figer ; aucune dépendance nouvelle ; la rotation ne perd rien par construction ; les écrans à venir se dessinent dans les trois formats (D13) et héritent du composant de mise en page.
- **Négatif** : la mise en page dépend du JavaScript, qui pose l'attribut : sans lui, l'application ne fonctionne de toute façon pas (SPA, ADR 0004). Un instant sans attribut est évité en le posant avant le premier rendu. Les sélecteurs du CSS s'allongent (`:root[data-format="…"]`). Sept projets Playwright au lieu de deux (cinq nouveaux), et une cible `make e2e-formats` à ajouter (`CLAUDE.md`, `make help`, ADR 0010).
- **Précise l'ADR 0028** : formats des projets, tags de format des scénarios `@ui`, et jugement de `pending.txt` d'après les projets d'un scénario et selon la cible (points 6 à 8 ci-dessus).
- **Précise l'ADR 0004** : la mise en page de l'application passe par un attribut de la racine, lu par le CSS, et non par des requêtes de média de largeur.
- **Durées** (lot B, 2026-11-08), dans le Dev Container (8 cœurs) : `make e2e` 1 min 55 s (405 tests, dont 75 sautés : format hors projet, ou WebKit) ; `make e2e-formats` 6 min 30 s (1 085 tests, dont 134 sautés). Sur la CI (`ubuntu-24.04`, 2 workers Playwright), PR #107 au commit `4c24714` : `make e2e` 2,2 min (405 tests : 330 réussis, 75 sautés), job `ci` complet 4 min 40 s. `make e2e-formats` sur la CI : à relever au premier lancement d'`e2e-formats.yml`.
