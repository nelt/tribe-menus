# docs

- `specs/` : spécifications fonctionnelles (le *quoi*, indépendamment de la technique) et exigences non fonctionnelles. `specs/glossaire.md` fixe la traduction anglaise des termes métier utilisée dans le code. `specs/points-a-trancher.md` trace les décisions prises avant de coder ; `specs/conventions-gherkin.md` fixe la façon d'écrire les scénarios.
- `design/` : `README.md` est la référence unique du design, dans son état actuel (identité, site public, connexion, application : tokens, écrans ↔ stories, navigation, composants) ; `maquettes/` contient les sources des maquettes, `identite/` les fichiers du symbole et du logotype.
- `plans/` : plans de travail préparés en amont, nommés `AAAA-MM-JJ-sujet.md`. Modèle : `plans/_template.md`.
- `adr/` : Architecture Decision Records, numérotés `NNNN-titre.md`. Modèle : `adr/0000-template.md`.
- `securite-depot.md` : liste des protections du dépôt GitHub (ADR 0011), à appliquer avant le passage en public.
- `poste-de-developpement.md` : préparation d'un poste de développement (Docker, VS Code, Git, Dev Container, Claude Code).
