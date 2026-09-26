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

À définir. Mettre à jour cette section dès que le premier projet est initialisé (langage, build, commandes de test).
