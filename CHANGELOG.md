# Journal des modifications

Format : une ligne par PR, sous la version à venir (ADR 0012). Versions sémantiques, étiquettes `vX.Y.Z`.

## Non publié

- Suivi de la CI : Dependabot limité aux outils Go et aux dépendances directes, sans version majeure de `@types/node` ni de TypeScript ; suivi de la CI avec `gh` documenté pour un jeton à portée fine.
- Intégration continue : workflows `ci` (`make tools` puis `make ci`, contrôle DCO, artefact de PR), `devcontainer` (construction hebdomadaire) et `dependency-review` ; Dependabot mensuel ; actionlint dans `make lint` ; `CONTRIBUTING.md` (ADR 0011, 0013).
- Environnement de développement : Dev Container avec Claude Code, préparation du poste et initialisation du projet (module Go, front Lit construit par esbuild, serveur squelette, contrôle `webcheck`, scénario Playwright de fumée, Makefile) (ADR 0009, 0010, 0018).
