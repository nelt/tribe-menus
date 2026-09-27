# 0009. Environnement de développement

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Le développement se fait dans trois contextes qui doivent utiliser exactement les mêmes versions d'outils : le poste du développeur, Claude Code (y compris dans le cloud) et la CI. Les outils nécessaires sont :

- Go, sqlc, staticcheck, govulncheck ; godog est une bibliothèque du module (ADR 0005) ;
- Node, TypeScript, esbuild, Playwright et ses navigateurs (ADR 0004, 0005) ;
- le client `sqlite3` pour inspecter les bases.

Le souci de limiter les adhérences (ADR 0002, 0004) s'applique aussi à l'outillage.

## Décision

1. **Une seule source de vérité pour les versions, sans gestionnaire d'outils supplémentaire** :
   - **Go** : directive `toolchain` de `go.mod` (téléchargement automatique de la bonne version) ; outils de développement (sqlc, staticcheck, govulncheck) épinglés par la directive `tool` de `go.mod` et lancés par `go tool <outil>` ;
   - **Node** : version dans `.nvmrc` ; TypeScript, esbuild et Playwright dans les `devDependencies` de `web/package.json`, avec le fichier de verrouillage.
2. **Une commande unique et idempotente installe tout le reste** (dépendances npm, navigateurs Playwright et leurs bibliothèques système). C'est la même commande pour le Dev Container, pour le script d'installation de l'environnement Claude Code dans le cloud et pour la CI. Sa forme (cible de Makefile ou autre) relève de la décision sur les commandes du projet.
3. **Dev Container comme environnement de référence** (`.devcontainer/devcontainer.json`) : image de base Go, Node ajouté, puis appel de la commande d'installation. Il reste mince : aucune version n'y est dupliquée. Travailler hors conteneur reste possible avec Go et Node installés sur le poste.
4. **Boucle de développement** :
   - esbuild en mode surveillance reconstruit le front à chaque modification ;
   - le serveur Go en mode développement écoute sur `http://localhost:8080`, lit le front sur le disque plutôt que dans le binaire, sert aussi `site/` à la racine (ce que fait Caddy en production) et écrit les codes de connexion dans les logs au lieu de les envoyer par e-mail ;
   - pas de TLS en local : les navigateurs traitent `localhost` comme un contexte sécurisé, ce qui suffit au service worker et à l'installation de la PWA ;
   - une commande crée des données de démonstration (tribu, membres, plats, planning) via la sous-commande `admin`.
5. **Éditeur de référence : VS Code** (distribution officielle, seule à disposer de l'extension Dev Containers). Le dépôt n'en dépend pas pour autant : seuls sont partagés un `.editorconfig` et la configuration du Dev Container.
6. **Politique des extensions VS Code.** Une extension s'exécute sans bac à sable, avec les droits de l'utilisateur, et peut être compromise par une mise à jour automatique. Règles :
   - **liste courte, déclarée dans `devcontainer.json` avec des versions épinglées** : Go (équipe Go), Playwright (Microsoft), et Claude Code (Anthropic) au besoin. Aucune extension communautaire au départ ; lit-plugin (vérification des gabarits Lit), maintenu essentiellement par une personne, n'est pas retenu, `tsc` vérifiant déjà les types ;
   - toute extension ajoutée ou mise à jour passe par une PR qui modifie `devcontainer.json` ;
   - mises à jour automatiques désactivées (`extensions.autoUpdate: false`) ;
   - un profil VS Code dédié au projet, pour que les extensions installées pour d'autres usages ne s'y invitent pas ;
   - authentification GitHub par un jeton limité à ce dépôt plutôt que par une clé SSH personnelle : VS Code transmet par défaut l'agent SSH et les identifiants Git au conteneur, où une extension pourrait s'en servir ;
   - télémétrie désactivée (`telemetry.telemetryLevel: off`).
7. **Tests sur un vrai téléphone** : en première itération, en déployant sur le VPS. Un tunnel depuis le poste de développement est à étudier plus tard (plusieurs solutions existent).

## Alternatives envisagées

- **Outils installés sur le poste uniquement** : plus simple et plus rapide, mais reproductibilité dépendante de chaque machine. Reste possible, le Dev Container n'étant qu'une enveloppe.
- **Nix, devbox, mise** : reproductibilité excellente, mais un outil de plus à apprendre et à maintenir ; écartés au nom de la limitation des adhérences.
- **VSCodium et autres forks de VS Code** : sans télémétrie, mais l'extension Dev Containers n'y est pas disponible officiellement et leur registre d'extensions (Open VSX) est moins contrôlé.
- **IntelliJ IDEA (plugin Go)** : familier, refactorings puissants, mais support des Dev Containers moins mûr et plugin Go réservé à la version payante.
- **HTTPS local (Caddy avec autorité interne, mkcert)** : inutile tant que les tests se font sur `localhost` ; l'installation d'une autorité racine sur iOS est en outre fastidieuse.

## Conséquences

- **Positif** : mêmes versions partout ; mise en route d'un poste ou d'une session Claude Code par une seule commande ; outils Go épinglés sans installation globale.
- **Isolation partielle** : dans le Dev Container, les extensions liées au code s'exécutent dans le conteneur et ne voient pas le dossier personnel de l'hôte ; elles voient en revanche le projet et les identifiants transmis au conteneur.
- **Négatif** : Docker requis pour le Dev Container ; l'image avec les navigateurs Playwright est lourde. sqlc exige un compilateur C pour se compiler (présent dans l'image).
- **Limite de la première itération** : chaque vérification sur téléphone passe par un déploiement ; l'ADR sur la chaîne de build et le déploiement doit rendre celui-ci rapide.
- **À étudier dans une version ultérieure** : tunnel pour tester sur téléphone depuis le poste de développement.
