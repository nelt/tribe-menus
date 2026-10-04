# Journal des modifications

Format : une ligne par PR, sous la version à venir (ADR 0012). Versions sémantiques, étiquettes `vX.Y.Z`.

## Non publié

- Revue du lot B : un identifiant d'URL hors format n'est plus écrit en clair dans la base de limitation (ADR 0021) ; durée de conservation des empreintes annoncée telle qu'elle est, une heure et dix minutes au plus.
- Connexion et session côté API (ENF-01, ENF-02) : code à 6 chiffres envoyé par e-mail (écrit dans les logs en développement), session par cookie à expiration glissante de 90 jours, limitation des demandes et codes fantômes dans une base dédiée (ADR 0021), effacement automatique, déconnexion ; 21 scénarios Gherkin passent.
- Plan de la première tranche verticale, lot B précisé après la fusion du lot A : `Contexte` de `compartimentage-tribus.feature`, harnais godog sans connexion réseau, source du nom de la tribu, empreintes par tribu, fin des envois d'e-mail, appareil détecté (D8 à D13).
- Protocole de revue de PR entre sessions Claude (`docs/revue-de-pr.md`) : un commentaire de revue structuré, corrections par l'auteur, clôture par le relecteur ; `gh pr comment` autorisé sans confirmation (ADR 0020).
- Stockage et initialisation d'une tribu : bases SQLite (registre et une base par tribu, migrations au démarrage, `/healthz`), code sqlc, sous-commandes `admin init` (EF-08) et `admin seed`, harnais godog avec liste des scénarios en attente ; cibles `make generate`, `acceptance` et `seed` ; format de l'identifiant d'URL dans les specs.
- Plan de la première tranche verticale : stockage SQLite par tribu, harnais godog, initialisation d'une tribu (EF-08), connexion et session (ENF-01, ENF-02), écrans de connexion ; trois PR à venir.
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
