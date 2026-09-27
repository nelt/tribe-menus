# CLAUDE.md

Instructions pour Claude Code dans ce dépôt.

## Contexte

Dépôt bac à sable personnel. Le travail préparatoire (cadrage, architecture, découpage) est souvent fait en amont dans Claude (chat) puis déposé dans `docs/`.

## Avant de coder

- Si la demande fait référence à un plan, lire le fichier correspondant dans `docs/plans/` et suivre ses étapes dans l'ordre.
- Les critères d'acceptation sont dans `docs/specs/features/*.feature` (Gherkin, en français). Écrire les tests à partir de ces scénarios, de préférence avant le code, et citer l'identifiant de la story (ex. `C2`) dans les commits.
- Pour l'interface, suivre `docs/design/README.md` (tokens, composants, navigation) et les maquettes de `docs/design/maquettes/`.
- Consulter `docs/adr/` pour les décisions déjà prises ; ne pas les contredire sans le signaler.
- En cas d'ambiguïté ou d'écart par rapport au plan, le dire plutôt que d'improviser.

## Conventions

- Travailler sur une branche dédiée (`feature/<sujet>`), jamais directement sur `main`.
- Commits petits et explicites.
- Toute nouvelle décision d'architecture significative donne lieu à un ADR (modèle : `docs/adr/0000-template.md`).
- Mettre à jour le plan (cases cochées, notes) au fil de l'avancement.

## Stack

Décidée (voir `docs/adr/`) ; le projet n'est pas encore initialisé.

- **Back-end** : Go, monolithe modulaire, bibliothèque standard d'abord, sqlc pour l'accès aux données (ADR 0002).
- **Données** : SQLite, un fichier par tribu plus un registre global (ADR 0003). Réplication hors site par Litestream reportée à une version ultérieure (ADR 0007).
- **Front-end** : SPA TypeScript avec Lit, esbuild, service worker écrit dans le projet, embarquée dans le binaire Go (ADR 0004).
- **Tests** : godog sur les `.feature` contre l'API, Playwright pour les scénarios `@ui`, `go test` et `node:test` pour les tests unitaires (ADR 0005).
- **Exposition** : Caddy en reverse proxy (TLS automatique) sur `tribe-menus.codingmatters.org` ; site public statique à `/`, application sous `/tribes/<identifiant>/` (ADR 0006).
- **Hébergement** : VPS OVHcloud (VPS-1), DNS `codingmatters.org` chez OVHcloud (ADR 0007).

Organisation du dépôt, commandes de build et de test, et reste de l'hébergement : à définir (prochains ADR). Mettre à jour cette section à l'initialisation du projet.
