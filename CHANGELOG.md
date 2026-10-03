# Journal des modifications

Format : une ligne par PR, sous la version à venir (ADR 0012). Versions sémantiques, étiquettes `vX.Y.Z`.

## Non publié

- Plan de la première tranche verticale : stockage SQLite par tribu, harnais godog, initialisation d'une tribu (EF-08), connexion et session (ENF-01, ENF-02), écrans de connexion ; en brouillon, avec sept décisions à valider.
- Autorisations de Claude Code : la suppression d'une branche distante n'est plus refusée mais soumise à confirmation, comme la fermeture d'une PR (ADR 0020).
- Claude pousse les branches `feature/…` et ouvre les PR, depuis le Dev Container (jeton en écriture sans *Workflows*, Git authentifié par `gh`) comme depuis une session cloud ; Remote Control connecté par défaut dans le Dev Container, où seules les mises à jour automatiques de Claude Code restent coupées ; avertissement hebdomadaire quand une version plus récente est publiée ; dépôt public, ruleset `main` actif (ADR 0020).
- Design : `docs/design/README.md` devient la référence unique, à l'état actuel (identité, site public, connexion, application) ; `identite.md` y est fusionné ; anciennes maquettes d'accueil retirées.
- Maquettes des listes de courses reprises pour le nouveau modèle : liste principale, sélection d'articles, choix des listes, courses des repas ; écrans d'historique retirés (PT-17).
- Listes de courses refondues : listes nommées indépendantes du planning, liste principale, ajout des courses d'une période vers une liste, déplacement d'articles et nouvelle liste à partir d'une sélection, courses faites sans historique (stories C1 à C12 renumérotées).
- GitHub CLI en lecture seule dans le Dev Container : `gh` dans l'image, jeton dans le volume `tribe-menus-gh` (ADR 0019).
- `LICENSE` (texte de l'AGPL-3.0) et `SECURITY.md` (signalement privé des vulnérabilités) (ADR 0011).
- Suivi de la CI : Dependabot limité aux outils Go et aux dépendances directes, sans version majeure de `@types/node` ni de TypeScript ; suivi de la CI avec `gh` documenté pour un jeton à portée fine.
- Intégration continue : workflows `ci` (`make tools` puis `make ci`, contrôle DCO, artefact de PR), `devcontainer` (construction hebdomadaire) et `dependency-review` ; Dependabot mensuel ; actionlint dans `make lint` ; `CONTRIBUTING.md` (ADR 0011, 0013).
- Environnement de développement : Dev Container avec Claude Code, préparation du poste et initialisation du projet (module Go, front Lit construit par esbuild, serveur squelette, contrôle `webcheck`, scénario Playwright de fumée, Makefile) (ADR 0009, 0010, 0018).
