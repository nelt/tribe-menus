# CLAUDE.md

Instructions pour Claude Code dans ce dépôt.

## Contexte

Application **Melting Tribe** (menus de la semaine, en tribu), dépôt public sous AGPL. Le travail préparatoire (cadrage, architecture, découpage) est souvent fait en amont dans Claude (chat) puis déposé dans `docs/`.

## Avant de coder

- Si la demande fait référence à un plan, lire le fichier correspondant dans `docs/plans/` et suivre ses étapes dans l'ordre.
- Les critères d'acceptation sont dans `docs/specs/features/*.feature` (Gherkin, en français). Écrire les tests à partir de ces scénarios, de préférence avant le code, et citer l'identifiant de la story (ex. `C2`) dans les commits. Tout scénario nouveau ou modifié suit `docs/specs/conventions-gherkin.md`.
- Pour l'interface, suivre `docs/design/README.md` (tokens, composants, navigation) et les maquettes de `docs/design/maquettes/`.
- Consulter `docs/adr/` pour les décisions déjà prises ; ne pas les contredire sans le signaler.
- En cas d'ambiguïté ou d'écart par rapport au plan, le dire plutôt que d'improviser.

## Conventions

- Travailler sur une branche dédiée (`feature/<sujet>`), jamais directement sur `main`.
- Commits petits et explicites.
- Toute nouvelle décision d'architecture significative donne lieu à un ADR (modèle : `docs/adr/0000-template.md`).
- Mettre à jour le plan (cases cochées, notes) au fil de l'avancement.

## Stack

Décidée (voir `docs/adr/`).

- **Back-end** : Go, monolithe modulaire, bibliothèque standard d'abord, sqlc pour l'accès aux données (ADR 0002).
- **Données** : SQLite, un fichier par tribu plus un registre global (ADR 0003). Réplication hors site par Litestream reportée à une version ultérieure (ADR 0007).
- **Front-end** : SPA TypeScript avec Lit, esbuild, service worker écrit dans le projet, embarquée dans le binaire Go (ADR 0004).
- **Tests** : godog sur les `.feature` contre l'API, Playwright pour les scénarios `@ui`, `go test` et `node:test` pour les tests unitaires (ADR 0005).
- **Exposition** : Caddy en reverse proxy (TLS automatique) sur `meltingtribe.codingmatters.org` ; site public statique à `/`, application sous `/tribes/<identifiant>/` (ADR 0006, nom d'hôte ADR 0017).
- **Hébergement** : VPS OVHcloud (VPS-1), DNS `codingmatters.org` chez OVHcloud (ADR 0007).
- **Dépôt** : `tribe-menus`, module `github.com/nelt/tribe-menus`, un paquet Go par domaine dans `internal/`, front dans `web/`, site public dans `site/` (ADR 0008).
- **Nom** : marque « Melting Tribe » côté utilisateurs ; `tribe-menus` reste le nom technique (dépôt, module, service) (ADR 0017).
- **Langue** : code entièrement en anglais ; specs en français ; traduction des termes métier fixée par `docs/specs/glossaire.md` (ADR 0008).
- **Environnement de développement** : Dev Container de référence ; versions épinglées dans `go.mod` (Go 1.27.1 ; sqlc, staticcheck, govulncheck par `go tool`), `.nvmrc` (Node 24.21.0) et `web/package.json` (Lit, TypeScript 6, esbuild, Playwright) ; serveur local sur `http://localhost:8080` (`make dev`) ; VS Code avec une liste courte d'extensions épinglées (ADR 0009).
- **Claude Code** : s'exécute dans le Dev Container (version épinglée) ; peut commiter mais ne pousse jamais, le push se fait depuis l'hôte ; GitHub CLI en lecture seule pour suivre la CI (runs, journaux, artefacts, PR ; ADR 0019) ; autorisations partagées dans `.claude/settings.json`, réglages personnels dans `.claude/settings.local.json` (ADR 0018). Préparation du poste : `docs/poste-de-developpement.md`.
- **Commandes** : Makefile sommaire, toujours passer par ses cibles (ADR 0010), `make help` les liste. Disponibles : `make tools`, `dev`, `lint`, `test`, `e2e`, `vuln`, `build`, et `make ci` avant de pousser (c'est ce que lance la CI ; `lint` comprend actionlint). À venir avec la première story qui leur donne un contenu : `seed`, `generate`, `acceptance`.
- **Build et release** : build uniquement en CI ; archive (binaire `linux/amd64` avec front embarqué, `site/`, `deploy/`) ; versions sémantiques par étiquette manuelle `vX.Y.Z` ; chaque PR ajoute une ligne à `CHANGELOG.md` ; procédure dans `RELEASING.md` (ADR 0012).
- **E-mails** : code de connexion envoyé par le SMTP du MX Plan OVHcloud (`no-reply@codingmatters.org`) via `net/smtp`, derrière une interface `Mailer` ; SPF, DKIM, DMARC (ADR 0014).
- **CI** : GitHub Actions, runner `ubuntu-24.04` avec `setup-go` et `setup-node` ; workflow `ci.yml`, un job `ci` qui lance `make tools` puis `make ci`, obligatoire pour fusionner, et qui vérifie le DCO des commits d'une PR (`internal/tools/dcocheck`) ; `devcontainer.yml`, construction hebdomadaire du Dev Container ; `dependency-review.yml` et CodeQL sur les PR, actifs une fois le dépôt public ; Dependabot mensuel, groupé par écosystème (ADR 0013). `release.yml` viendra avec le déploiement.
- **Serveur** : Debian 13 provisionné par `deploy/provision.sh` ; service systemd confiné, activation de socket, secrets en credentials systemd ; alertes par e-mail via `msmtp` (ADR 0015).
- **Déploiement** : manuel par SSH (`sudo tribe-menus-deploy <version>`), instantané des bases et retour arrière automatique ; environnement de recette `recette.meltingtribe.codingmatters.org` alimenté par les archives de PR (ADR 0016).
- **Dépôt public, licence `AGPL-3.0-or-later` pour tout le dépôt** (documentation comprise, polices exceptées) ; contributions sous DCO (`git commit -s`) ; aucun secret ni valeur propre à l'instance dans le dépôt ; protections du dépôt listées dans `docs/securite-depot.md` (ADR 0011).

## Conventions Go

- Erreurs retournées, jamais ignorées, et enveloppées avec leur contexte (`fmt.Errorf("...: %w", err)`).
- Pas de `panic` hors de `main`.
- Tests en tableaux de cas (`go test`, `httptest`, `fstest.MapFS`).
- Logique métier pure, sans HTTP ni SQL ; handlers et accès aux données sont des adaptateurs minces (ADR 0002).
