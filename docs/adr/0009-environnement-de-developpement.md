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
5. **Éditeur libre.** Le dépôt ne dépend d'aucun éditeur : seuls sont partagés un `.editorconfig` et, dans le Dev Container, une liste facultative d'extensions recommandées.
6. **Tests sur un vrai téléphone** : en première itération, en déployant sur le VPS. Un tunnel depuis le poste de développement est à étudier plus tard (plusieurs solutions existent).

## Alternatives envisagées

- **Outils installés sur le poste uniquement** : plus simple et plus rapide, mais reproductibilité dépendante de chaque machine. Reste possible, le Dev Container n'étant qu'une enveloppe.
- **Nix, devbox, mise** : reproductibilité excellente, mais un outil de plus à apprendre et à maintenir ; écartés au nom de la limitation des adhérences.
- **HTTPS local (Caddy avec autorité interne, mkcert)** : inutile tant que les tests se font sur `localhost` ; l'installation d'une autorité racine sur iOS est en outre fastidieuse.

## Conséquences

- **Positif** : mêmes versions partout ; mise en route d'un poste ou d'une session Claude Code par une seule commande ; outils Go épinglés sans installation globale.
- **Négatif** : Docker requis pour le Dev Container ; l'image avec les navigateurs Playwright est lourde. sqlc exige un compilateur C pour se compiler (présent dans l'image).
- **Limite de la première itération** : chaque vérification sur téléphone passe par un déploiement ; l'ADR sur la chaîne de build et le déploiement doit rendre celui-ci rapide.
- **À étudier dans une version ultérieure** : tunnel pour tester sur téléphone depuis le poste de développement.
