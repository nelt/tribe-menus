# 0028. Scénarios `@ui` exécutés par Playwright à partir des `.feature`

- **Date** : 2026-11-03
- **Statut** : accepté

## Contexte

L'ADR 0005 fait des `.feature` la source unique des critères d'acceptation, exécutés tels quels, et prouve dans le navigateur, par Playwright, les seuls scénarios dont le sens tient à l'interface (tag `@ui`). Il laissait « à préciser » le moyen de faire lire les `.feature` par Playwright ; le plan `socle` l'a reporté (D5) à la première story dont un scénario ne se prouve que dans le navigateur. Les tests Playwright existants (parcours de connexion, états des écrans) sont des tests d'écran ordinaires, sans lien avec un scénario.

Le plan `bibliotheque` apporte ces premiers scénarios : la recherche (P3) et l'autocomplétion (P4) se calculent dans le navigateur (ADR 0027), la taille proposée pour un nouveau plat (T3) et la fraîcheur de la copie en mémoire (T1) n'ont de sens qu'à l'écran.

Côté Go, godog lit les `.feature` avec le parseur Gherkin officiel, déjà dans `go.sum`, et ignore déjà les tags `@manuel` et `@ui` (`acceptance/acceptance_test.go`). Côté npm, le projet n'ajoute aucune dépendance sans nécessité (ADR 0004, point 8).

Décidé avec le développeur le 2026-10-29 (plan `bibliotheque`, D14) ; ferme `socle`, D5.

## Décision

1. **Un outil Go du projet exporte les scénarios `@ui`** : il lit les `.feature` de `docs/specs/features/` avec le parseur Gherkin que godog utilise (`github.com/cucumber/gherkin/go`), déroule les plans du scénario en un scénario par exemple, et écrit en JSON, pour chacun, le fichier, le titre, les tags et les étapes (texte, tableau de données). Le JSON est produit avant chaque exécution de Playwright, jamais modifié à la main ni versionné.
2. **Un registre d'étapes en TypeScript** (`web/e2e/`) associe à chaque formulation d'étape une fonction Playwright, par expression régulière ; un test Playwright par scénario exporté enchaîne ses étapes. Les formulations suivent `docs/specs/conventions-gherkin.md`, comme pour godog ; les définitions d'étapes des deux exécuteurs restent distinctes (ADR 0005, « Négatif »).
3. **Un scénario `@ui` dont une étape n'a pas de définition échoue**, comme sous godog en mode strict.
4. **`acceptance/pending.txt` vaut pour les deux exécuteurs, dans les deux sens** : un scénario `@ui` qui échoue hors de la liste, ou qui passe en y figurant, fait échouer `make e2e` ; la liste ne peut que se vider. Chaque exécuteur ne juge que les lignes des scénarios qu'il exécute : godog celles des scénarios sans `@ui` ni `@manuel`, Playwright celles des scénarios `@ui`.
5. **godog ignore les scénarios `@ui`**, comme aujourd'hui.
6. **Les scénarios `@ui` s'exécutent sous Chromium et WebKit**, comme les tests d'écran (`web/playwright.config.ts`), contre le serveur de test de `make e2e`.
7. **L'outil suit la version du parseur que godog impose.** Il importe le parseur directement : `github.com/cucumber/gherkin/go/v<N>` devient une dépendance directe de `go.mod`, à la version que requiert godog, et n'est jamais monté seul. Une montée de godog qui change la version majeure du parseur change l'import de l'outil dans la même PR. Il en va de même pour `github.com/cucumber/messages/go/v<N>`, dont l'outil utilise les types que rend le parseur : dépendance directe, à la version que requiert godog (plan `bibliotheque`, lot C, revue #77, point 5). Dépendance de développement : l'outil n'entre pas dans le binaire.
8. **Le tag `@ui` est posé quand le plan d'une story fixe le niveau de preuve de ses scénarios** (`docs/specs/conventions-gherkin.md`), pas seulement à l'implémentation.

## Alternatives envisagées

- **`playwright-bdd` ou `@cucumber/gherkin` en npm** : écartés. Dépendances nouvelles côté npm, pour un besoin que le parseur déjà présent côté Go couvre.
- **Des tests Playwright écrits à la main qui citent le titre du scénario** : écartés. Le scénario serait recopié dans le test, et une spec modifiée ne changerait pas le test (ADR 0005, point 1).
- **Exécuter les scénarios `@ui` par godog, en pilotant le navigateur** (chromedp, playwright-go) : écarté par l'ADR 0005 (Chrome seul, portage communautaire).
- **Un fichier d'attente distinct pour Playwright** : écarté. Une seule liste dit ce qui reste à faire, quel que soit l'exécuteur.

## Conséquences

- **Positif** : les `.feature` restent la source unique, y compris pour le navigateur ; aucune dépendance npm nouvelle ; une seule liste d'attente.
- **Négatif** : un outil et un registre d'étapes à écrire et maintenir ; une dépendance directe du parseur Gherkin, liée à la version de godog.
- **Transition** : jusqu'à l'outil (plan `bibliotheque`, étape 16), aucun exécuteur ne lit les scénarios `@ui`. Ceux que le lot A du plan a marqués `@ui` sortent donc de `pending.txt`, puisque godog ne les exécute pas, et y reviennent avec l'outil.
- **Les tests d'écran existants** (`connexion.spec.ts`, `ecrans-connexion.spec.ts`) restent des tests ordinaires : le parcours de connexion n'est pas relié à un scénario (écart assumé de `socle`, D5).
- **Précise l'ADR 0005**, « À préciser à l'implémentation » : moyen de faire lire les `.feature` par Playwright, et moment où le tag `@ui` est posé.
