# Plan : intégration continue (GitHub Actions)

- **Date** : 2026-09-30
- **Statut** : brouillon

## Objectif

Chaque PR et chaque push sur `main` lancent `make tools` puis `make ci` sur un runner GitHub. Le job `ci` devient la vérification exigée pour fusionner. Le Dev Container est reconstruit chaque semaine pour vérifier qu'il fonctionne toujours. La mise en œuvre suit l'ADR 0013.

Ce plan est exécuté par **Claude Code dans le Dev Container**. La validation (push, PR, lecture des exécutions) est faite par le développeur depuis l'hôte.

**Hors périmètre** :
- `release.yml` : il produit l'archive de l'ADR 0012 (binaire, `site/`, `deploy/`, empreinte), mais `deploy/` et `RELEASING.md` n'existent pas encore. Il arrivera avec le plan de déploiement (ADR 0015, 0016).
- L'application des réglages de [securite-depot.md](../securite-depot.md), qui se fait au passage en public (voir « Dépôt privé » ci-dessous).

## Contexte et contraintes

- À lire avant de commencer : ADR 0009, 0010, 0011, 0012, 0013, 0016, 0018, `docs/securite-depot.md` et `CLAUDE.md`.
- **Git** : branche `feature/ci`. Un commit par étape, avec `git commit -s` (DCO, ADR 0011). **Ne jamais pousser** (ADR 0018).
- **Dépôt privé pour l'instant** (conditions GitHub vérifiées le 2026-09-30) :
  - Minutes : 2 000 par mois avec GitHub Free, 3 000 avec Pro ; illimitées une fois le dépôt public. Estimation : 300 à 500 minutes par mois (environ 45 exécutions de `ci.yml` de 5 à 10 minutes, plus 4 constructions du Dev Container). Sans moyen de paiement, les jobs sont bloqués une fois le quota épuisé ; rien n'est facturé.
  - Stockage : 500 Mo d'artefacts avec Free ; 10 Go de cache par dépôt, décomptés à part.
  - **Dependency review** et **CodeQL** exigent GitHub Code Security sur un dépôt privé. `dependency-review.yml` est écrit maintenant, mais son job ne s'exécute que si le dépôt est public (`if: ${{ !github.event.repository.private }}`). CodeQL, en configuration par défaut, s'active dans les réglages au passage en public ; aucun fichier n'est nécessaire.
  - **Rulesets** (protection de `main`, vérification `ci` obligatoire) : sur un dépôt personnel privé, ils demandent GitHub Pro. Avec Free, ils s'appliquent au passage en public.
- **Versions**, relevées le 2026-09-30. Actions épinglées par empreinte de commit complète, avec la version en commentaire (`uses: actions/checkout@3d3c42e… # v7.0.1`). Tout ajout hors de cette liste est à signaler plutôt qu'à faire.

  | Outil | Version | Empreinte / où elle est fixée |
  | --- | --- | --- |
  | `actions/checkout` | v7.0.1 | `3d3c42e5aac5ba805825da76410c181273ba90b1` |
  | `actions/setup-go` | v7.0.0 | `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` |
  | `actions/setup-node` | v7.0.0 | `820762786026740c76f36085b0efc47a31fe5020` |
  | `actions/cache` | v6.1.0 | `55cc8345863c7cc4c66a329aec7e433d2d1c52a9` |
  | `actions/upload-artifact` | v7.0.1 | `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` |
  | `actions/dependency-review-action` | v5.0.0 | `a1d282b36b6f3519aa1f3fc636f609c47dddb294` |
  | actionlint | 1.7.12 | `go.mod`, directive `tool` (`github.com/rhysd/actionlint/cmd/actionlint`) |
  | `@devcontainers/cli` | 0.89.0 | appelé par `npx --yes @devcontainers/cli@0.89.0` dans `devcontainer.yml` |

  Vérifier chaque empreinte au moment de l'écrire (`git ls-remote` ou API GitHub) : le tableau sert de référence, il ne remplace pas la vérification.
- **Sécurité des workflows** (ADR 0013, point 5) :
  - `permissions: contents: read` au niveau du workflow ;
  - checkout avec `persist-credentials: false` ;
  - aucun secret ; aucun déclencheur `pull_request_target` ;
  - pas d'expression `${{ github.event.* }}` fournie par l'utilisateur (titre, branche) interpolée dans un `run:` ; passer par `env:` si besoin.
- **Pas de logique dans le Makefile** (ADR 0010) et, par extension, peu de logique dans les workflows : au-delà de quelques lignes, un programme Go dans `internal/tools/`, comme `webcheck`.
- Code et commentaires en anglais ; noms de jobs et d'étapes en anglais (ADR 0008).

## Étapes

- [ ] **1. actionlint dans `make lint`.**
  - Ajouter actionlint comme directive `tool` dans `go.mod`, puis `go tool actionlint` en dernière ligne de la cible `lint`. Sans fichier dans `.github/workflows/`, la commande doit passer ; sinon, ne l'ajouter qu'à l'étape 3.
  - `make lint`. Commit.

- [ ] **2. Contrôle DCO : `internal/tools/dcocheck`.**
  - Programme Go qui lit les commits d'un intervalle (`dcocheck <base>..<head>`, via `git log --format=...`) et échoue en listant ceux qui n'ont pas de ligne `Signed-off-by:` correspondant à l'auteur (nom et adresse). Les commits de fusion sont ignorés.
  - Logique pure (analyse d'un message et de l'auteur) séparée de l'appel à `git`, testée en tableau de cas : ligne présente, absente, adresse différente, plusieurs lignes, fusion.
  - Ajouter `CONTRIBUTING.md` (ADR 0011, point 7) : licence, DCO, `git commit -s`, `make ci` avant de pousser. Court, en français.
  - `make lint test`. Commit.

- [ ] **3. `.github/workflows/ci.yml`.**
  - Déclencheurs : `pull_request`, `push` sur `main`, `workflow_dispatch`, `schedule` hebdomadaire (lundi, heure creuse).
  - `concurrency` : groupe `ci-${{ github.ref }}`, `cancel-in-progress` pour les PR seulement.
  - Un job `ci` sur `ubuntu-24.04`, `timeout-minutes: 20` :
    1. checkout (`fetch-depth: 0` pour le contrôle DCO) ;
    2. `setup-go` (`go-version-file: go.mod`, cache activé) ;
    3. `setup-node` (`node-version-file: .nvmrc`, cache npm sur `web/package-lock.json`) ;
    4. cache des navigateurs Playwright (`~/.cache/ms-playwright`, clé = OS + version de `@playwright/test` lue dans `web/package-lock.json`) ;
    5. sur une PR seulement : `go run ./internal/tools/dcocheck` sur `base.sha..head.sha`, passés par `env:` ;
    6. `make tools`, puis `make ci` avec `VERSION=pr-<numéro>` sur une PR ;
    7. sur une PR réussie : artefact `tribe-menus-pr-<numéro>` contenant `bin/tribe-menus` et `site/`, conservé 3 jours ;
    8. en cas d'échec : artefact `playwright-report` (`web/playwright-report/`, `web/test-results/`), conservé 3 jours.
  - `make lint` (actionlint). Commit.

- [ ] **4. `.github/workflows/devcontainer.yml`.**
  - Déclencheurs : `schedule` hebdomadaire, `workflow_dispatch`, `pull_request` avec `paths: .devcontainer/**`.
  - Job `devcontainer`, `timeout-minutes: 30` : checkout, `setup-node`, puis `npx --yes @devcontainers/cli@0.89.0 up --workspace-folder .`. `postCreateCommand` lance `make tools`, dont l'échec fait échouer l'étape. Ensuite, `devcontainer exec ... go version` et `node --version` pour la trace.
  - `make lint`. Commit.

- [ ] **5. `.github/workflows/dependency-review.yml`.**
  - Sur `pull_request`. Job conditionné à un dépôt public (voir « Dépôt privé ») ; `permissions: contents: read`.
  - `fail-on-severity: moderate` ; licences : liste `allow-licenses` compatible AGPL-3.0-or-later (MIT, BSD-2-Clause, BSD-3-Clause, Apache-2.0, ISC, 0BSD, MPL-2.0, LGPL, GPL-3.0, AGPL-3.0, Unlicense, CC0-1.0, BlueOak-1.0.0). La liste est à valider par le développeur.
  - `make lint`. Commit.

- [ ] **6. Documentation.**
  - `docs/securite-depot.md` : ajouter à la liste du passage en public l'activation de CodeQL (configuration par défaut) et la vérification que `dependency-review` s'exécute.
  - `CLAUDE.md` : retirer « `actionlint` rejoindra `lint` avec la CI » ; mentionner les workflows et `dcocheck`.
  - `CHANGELOG.md` : ligne de la PR sous « Non publié ».
  - Cocher les étapes de ce plan, noter les écarts. Commit.

- [ ] **7. Validation par le développeur**, depuis l'hôte :
  - `git push`, puis ouverture de la PR ;
  - `ci` au vert ; noter sa durée (à froid, puis avec caches) dans les notes d'exécution ;
  - lancer `devcontainer.yml` à la main (`workflow_dispatch`) : au vert ;
  - `dependency-review` apparaît comme ignoré (dépôt privé) ;
  - l'artefact de la PR est téléchargeable et contient le binaire et `site/`.

## Notes d'exécution

- ...

## Critères de validation

- Sur la PR : `ci` passe ; un commit sans `Signed-off-by` le fait échouer, avec un message qui nomme le commit (à essayer sur une branche jetable).
- Une modification qui casse `gofmt`, un test ou un workflow (actionlint) fait échouer `ci`.
- Un nouveau commit poussé sur la PR annule l'exécution en cours.
- `devcontainer.yml` passe en exécution manuelle.
- Aucun workflow n'a de permission en écriture ni n'utilise de secret.
- Consommation après un mois : relevée dans Settings > Billing, cohérente avec l'estimation.

## Questions ouvertes

- **Artefact de PR** : binaire et `site/` pour l'instant, sans `deploy/` ni empreinte. L'archive complète de l'ADR 0012 viendrait avec `release.yml` et une cible `make dist` (ou `make archive`) à créer à ce moment-là. D'accord pour ce découpage ?
- **Mise à jour des actions épinglées** : sans Dependabot (version updates pour `github-actions`), les empreintes vieilliront sans alerte. L'ADR 0013 n'en parle pas. Ajouter `.github/dependabot.yml` (actions, Go, npm), avec un ADR ou une note à l'ADR 0013 ?
- **Workflows planifiés** : GitHub les désactive après 60 jours sans activité sur un dépôt public. Acceptable pour ce projet ?
- **Liste de licences** de `dependency-review` : à valider.
- **Formule GitHub** : Free ou Pro ? Avec Pro, les rulesets de `main` peuvent être appliqués dès maintenant, en privé.
