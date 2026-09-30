# Plan : environnement de développement sur le poste

- **Date** : 2026-09-29
- **Statut** : en cours (étape 10, validation par le développeur)

## Objectif

Pouvoir développer et tester en local avec VS Code et Claude Code : dans le Dev Container, `make tools` puis `make dev` servent une application squelette sur `http://localhost:8080`, et `make lint`, `make test`, `make e2e`, `make vuln` et `make build` passent.

Ce plan est exécuté par **Claude Code dans le Dev Container**, sur le poste. Le socle du conteneur (`.devcontainer/`, `.nvmrc`, `.claude/settings.json`) est déjà dans le dépôt ; la préparation du poste (Docker, VS Code, Git, jeton) est décrite dans `docs/poste-de-developpement.md`.

**Hors périmètre** : la CI (ADR 0013), qui fera l'objet d'une PR séparée s'appuyant sur les mêmes cibles ; `make seed`, `make generate` et `make acceptance`, qui arrivent avec la première story leur donnant un contenu (base SQLite, requêtes sqlc, scénarios godog) ; la sous-commande `admin`, le fichier de configuration et l'activation de socket (ADR 0015).

## Contexte et contraintes

- À lire avant de commencer : ADR 0002, 0004, 0005, 0006, 0008, 0009, 0010, 0012, 0018 et `CLAUDE.md`.
- **Git** : travailler sur la branche `feature/environnement-de-developpement`. Un commit par étape, avec `git commit -s` (DCO, ADR 0011). **Ne jamais pousser** : le push se fait depuis l'hôte (ADR 0018), `git push` est refusé par `.claude/settings.json`.
- **Dépendances** : versions exactes, relevées le 2026-09-29. Tout ajout hors de cette liste est à signaler plutôt qu'à faire.

  | Outil | Version | Où elle est fixée |
  | --- | --- | --- |
  | Go | 1.27.1 | `go.mod` : `go 1.27`, `toolchain go1.27.1` |
  | sqlc | 1.31.1 | `go.mod`, directive `tool` (`github.com/sqlc-dev/sqlc/cmd/sqlc`) |
  | staticcheck | 0.8.1 | `go.mod`, directive `tool` (`honnef.co/go/tools/cmd/staticcheck`) |
  | govulncheck | 1.8.0 | `go.mod`, directive `tool` (`golang.org/x/vuln/cmd/govulncheck`) |
  | Node | 24.21.0 | `.nvmrc` (déjà présent) |
  | lit | 3.3.3 | `web/package.json`, `dependencies` |
  | typescript | 6.0.3 | `web/package.json`, `devDependencies` |
  | esbuild | 0.28.2 | `web/package.json`, `devDependencies` |
  | @playwright/test | 1.63.0 | `web/package.json`, `devDependencies` |
  | @types/node | dernière 24.x | `web/package.json`, `devDependencies` |

  actionlint arrivera avec la PR de CI.
- **Pas de logique dans le Makefile** (ADR 0010) : une ou deux lignes par cible ; au-delà, un programme Go ou un script npm du projet.
- Code en anglais, commentaires compris ; textes affichés en français (ADR 0008).

## Étapes

- [x] **0. Vérifier le socle** (sans commit). `go version` affiche `go1.27.x`, `node --version` `v24.21.0`, `claude --version` `2.1.284`, `sqlite3 --version` répond. `ssh-add -l` ne trouve pas d'agent. Signaler tout écart avant de continuer.

- [x] **1. Module Go et outils.**
  - `go mod init github.com/nelt/tribe-menus`, puis `go 1.27` et `toolchain go1.27.1`.
  - `go get -tool` pour sqlc, staticcheck et govulncheck, aux versions du tableau.
  - Directive `ignore ./web/node_modules` (Go 1.25+), pour que `./...` ne parcoure pas les dépendances npm.
  - Vérifier : `go tool sqlc version`, `go tool staticcheck -version`, `go tool govulncheck -version`.
  - Commit : `go.mod`, `go.sum`.

- [x] **2. Front : dépendances et construction.**
  - `web/package.json` : `private`, `"type": "module"`, licence `AGPL-3.0-or-later`, versions exactes (`npm install --save-exact --ignore-scripts`). Scripts : `build` (`node scripts/build.mjs`), `watch` (`node scripts/build.mjs --watch`), `typecheck` (`tsc --noEmit`), `test` (`node --test "src/**/*.test.ts"`, Node 24 exécute le TypeScript sans étape de build), `e2e` (`playwright test`).
  - `web/tsconfig.json` : `strict`, `noUncheckedIndexedAccess`, `noImplicitOverride`, `exactOptionalPropertyTypes`, `moduleResolution: Bundler`, `allowImportingTsExtensions` (imports en `.ts`, exigés par Node), `verbatimModuleSyntax`, `useDefineForClassFields: false` (recommandé par Lit sans décorateurs), `noEmit`. Inclut `src`, `e2e`, `playwright.config.ts`.
  - `web/scripts/build.mjs` (API esbuild) : vide `web/dist` sauf `.gitkeep`, assemble `src/main.ts` en ESM, copie `src/index.html` et `src/app.css` ; `--watch` : contexte esbuild en surveillance, sourcemaps en ligne, sans minification.
  - `web/dist/.gitkeep` versionné, le reste de `web/dist` ignoré (ADR 0008, point 4 ; `.gitignore` déjà prêt).
  - Commit : `web/package.json`, `web/package-lock.json`, `web/tsconfig.json`, `web/scripts/`.

- [x] **3. Front : squelette.**
  - `web/src/index.html` : gabarit `html/template` (rendu par le serveur) avec `<base href="{{.Base}}">`, feuille `app.css` et `<script type="module" src="main.js">` ; aucun script en ligne (CSP, ADR 0004).
  - `web/src/route.ts` : module pur, `tribeIdFromPath(pathname)` qui extrait l'identifiant de `/tribes/<identifiant>/…` (décodé), `undefined` sinon.
  - `web/src/route.test.ts` : tests `node:test` en tableau de cas (racine, sans barre finale, sous-chemin, identifiant encodé, `/tribes/`, `/`, `/tribesx/…`).
  - `web/src/main.ts` : élément `mt-app` (Lit, shadow DOM, `static styles`) qui affiche « Melting Tribe » et l'identifiant de la tribu. Squelette provisoire, remplacé plus tard par le routeur (ADR 0004).
  - Vérifier : `npm run typecheck`, `npm test`, `npm run build`.
  - Commit.

- [x] **4. Serveur Go.**
  - `web/embed.go`, paquet `web` : `//go:embed all:dist` et `Dist() (fs.FS, error)`.
  - `internal/server` : `New(Config) (http.Handler, error)`, avec `Config{Web fs.FS; Site fs.FS; Dev bool; Logger *slog.Logger}`.
    - `GET /tribes/{tribe}` redirige (301) vers `/tribes/{tribe}/` ;
    - `GET /tribes/{tribe}/{rest...}` sert le fichier de `Web` s'il existe (hors `index.html`), répond 404 pour un chemin avec extension absent, et sinon rend `index.html` avec `Base = /tribes/<identifiant échappé>/` (routage côté client) ;
    - `X-Robots-Tag: noindex` sous `/tribes/` (ADR 0006, point 6) ;
    - `GET /` sert `Site` s'il est fourni (développement uniquement, Caddy s'en charge en production) ;
    - en-têtes communs : `Content-Security-Policy: default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` ;
    - production : gabarit analysé une fois au démarrage, erreur explicite si le front n'a pas été construit ; développement : gabarit relu à chaque requête, 503 « front en cours de construction » s'il manque.
  - Tests `go test` en tableau (`httptest`, `fstest.MapFS`), en mode production et en mode développement : chaque route ci-dessus, échappement de l'identifiant, présence de la CSP, absence de site en production, front manquant.
  - `cmd/tribe-menus` : sous-commandes `version` (affiche `tribe-menus <version> (<commit>)`, variables `version` et `commit` renseignées par `-ldflags -X`, ADR 0012) et `serve` (`-addr`, par défaut `localhost:8080` ; `-dev` : front lu dans `<root>/web/dist` et site dans `<root>/site` ; `-root`, par défaut `.`). Arrêt propre sur SIGINT/SIGTERM, `ReadHeaderTimeout`, journalisation `log/slog`. Fonction `run(args, stdout, stderr) int` testée.
  - `site/index.html` (page provisoire, titre « Melting Tribe ») et `site/robots.txt` (`Disallow: /tribes/`).
  - Commit.

- [x] **5. Contrôle `webcheck`** (ADR 0004, point 9 ; PT-13).
  - `internal/tools/webcheck` : programme qui parcourt `web/src` (ou le dossier passé en argument), fichiers `.ts`, `.js`, `.mjs`, `.html`, et signale `fichier:ligne` pour `unsafeHTML`, `unsafeSVG`, `innerHTML`, `outerHTML`, `insertAdjacentHTML`, `document.write`/`writeln` (mots entiers). Code de sortie 1 s'il trouve quelque chose.
  - Test sur un `fstest.MapFS` (cas positifs, faux positifs évités, extensions ignorées).
  - Commit.

- [x] **6. Scénario Playwright de fumée.**
  - `web/playwright.config.ts` : projets `chromium` (appareil « Pixel 7 ») et `webkit` (« iPhone 15 ») ; `webServer` qui construit le front puis lance `go run ../cmd/tribe-menus serve -dev -root .. -addr localhost:8090` ; rapport HTML jamais ouvert automatiquement.
  - `web/e2e/squelette.spec.ts` : la racine affiche le titre du site public ; `/tribes/demo/planning` affiche l'application avec « tribu demo ». Le contenu étant dans un shadow root, passer par `getByText`/`getByRole` (qui traversent les shadow roots ouverts).
  - Commit.

- [x] **7. Makefile** (ADR 0010) : `.PHONY`, commentaire `##` par cible, `help` par défaut.

  | Cible | Contenu |
  | --- | --- |
  | `help` | liste les cibles à partir des commentaires `##` |
  | `tools` | `go mod download` ; `npm ci --ignore-scripts` dans `web/` ; `npx playwright install --with-deps chromium webkit` |
  | `dev` | esbuild en surveillance et `go run ./cmd/tribe-menus serve -dev`, arrêtés ensemble par Ctrl-C |
  | `lint` | `gofmt -l` (échec si sortie non vide), `go vet ./...`, `go tool staticcheck ./...`, `npm run typecheck`, `go run ./internal/tools/webcheck` |
  | `test` | `go test ./...` ; `npm test` |
  | `e2e` | `npm run e2e` |
  | `vuln` | `go tool govulncheck ./...` ; `npm audit` |
  | `build` | `npm run build` ; `CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/tribe-menus ./cmd/tribe-menus`, avec `VERSION ?= dev` et `COMMIT` tiré de `git rev-parse --short HEAD` |
  | `ci` | `lint test e2e vuln build`, dans cet ordre |

  Lancer chaque cible, puis `make ci`. Commit.

- [x] **8. Dev Container : installation à la création.** Ajouter `"postCreateCommand": "make tools"` dans `.devcontainer/devcontainer.json`. Commit. La vérification se fait à l'étape 10 (reconstruction par le développeur).

- [x] **9. Documentation.**
  - `CLAUDE.md`, section Stack : retirer « le projet n'est pas encore initialisé », lister les cibles réellement disponibles et celles à venir, ajouter une section « Conventions Go » courte (erreurs retournées et enveloppées avec `%w`, pas de `panic` hors `main`, tests en tableaux de cas, logique métier sans HTTP ni SQL).
  - `CHANGELOG.md` : ajouter la ligne de cette PR sous « Non publié ».
  - Cocher les étapes de ce plan et passer son statut à « terminé » une fois l'étape 10 validée.
  - Commit.

- [ ] **10. Validation par le développeur**, hors de Claude Code :
  - depuis l'hôte : `git push`, puis ouverture de la PR ;
  - « Rebuild Container » : `make tools` s'exécute sans erreur à la création ;
  - critères de validation ci-dessous.

## Notes d'exécution

- `@types/node` : 24.19.0. TypeScript 6 ne charge plus les `@types` implicitement : `"types": ["node"]` ajouté à `web/tsconfig.json`.
- Build de production : sourcemap dans un fichier séparé (ADR 0012) ; noms de fichiers sans empreinte pour l'instant (`main.js`), à reprendre avec le service worker. En surveillance, `index.html` et `app.css` sont recopiés à chaque reconstruction, mais leur modification seule n'en déclenche pas.
- Serveur : la redirection vers `/tribes/<identifiant>/` conserve la chaîne de requête ; sous-commande `help` en plus de `version` et `serve`.
- `webcheck` : tests complétés par deux petits dossiers `testdata/` (dont un avec un `innerHTML` volontaire) pour vérifier la sortie et le code de retour.
- Playwright : intitulés des tests en anglais (ADR 0008), `reuseExistingServer` et `forbidOnly` selon `CI`.
- Makefile : textes de `make help` en anglais. `make dev` compile le serveur (`bin/tribe-menus-dev`) avant de lancer la surveillance esbuild, au lieu de `go run` en parallèle : sinon `//go:embed` peut lister un fichier de `web/dist` que la surveillance supprime au démarrage (erreur rencontrée à la validation).
- `CHANGELOG.md` : la ligne de la PR, déjà présente, a été complétée plutôt que doublée.

## Critères de validation

- `make dev` : `http://localhost:8080/` affiche la page publique, `http://localhost:8080/tribes/demo/` l'application ; une modification de `web/src/main.ts` est visible au rechargement de la page.
- `make ci` passe dans le conteneur ; `bin/tribe-menus version` affiche la version et le commit.
- Dans le conteneur, `git push` échoue (refusé par Claude Code, et sans identifiants dans un terminal) ; depuis l'hôte, il fonctionne.
- Claude Code lance les cibles `make` sans demander d'autorisation.

## Questions ouvertes

- Version de `gopls` : installée par l'extension Go à sa dernière version, puis figée par `go.toolsManagement.autoUpdate: false`. L'épingler par une directive `tool` si des écarts apparaissent.
- Version de l'extension VS Code Claude Code : épinglée à 2.1.284, en supposant qu'elle suit celle de la CLI. À corriger si VS Code ne trouve pas cette version.
- Passage à TypeScript 7 (compilateur natif) : à évaluer dans une PR dédiée.
- Pare-feu sortant du conteneur (ADR 0018) : reporté.
