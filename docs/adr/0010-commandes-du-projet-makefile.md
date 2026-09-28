# 0010. Commandes du projet : Makefile sommaire

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Le projet a besoin d'un point d'entrée unique pour ses actions courantes : installer les outils, développer, générer, analyser, tester, construire. Le développeur, Claude Code et la CI doivent appeler exactement les mêmes commandes, pour qu'un contrôle qui passe en local passe aussi en CI. Le souci de limiter les adhérences s'applique (ADR 0009).

## Décision

1. **Un Makefile à la racine, réduit au rôle de sommaire** : chaque cible appelle en une ou deux lignes `go`, `go tool` ou `npm` ; aucune logique ne vit dans le Makefile. Toute logique qui dépasse quelques lignes va dans un outil du projet (programme Go, script npm), appelé par la cible.
2. **Cibles** :

   | Cible | Rôle |
   | --- | --- |
   | `make help` | liste les cibles (cible par défaut) |
   | `make tools` | installation idempotente des outils (ADR 0009) |
   | `make dev` | esbuild en surveillance et serveur local |
   | `make seed` | données de démonstration |
   | `make generate` | génération du code sqlc |
   | `make lint` | `gofmt`, `go vet`, `staticcheck`, `tsc --noEmit`, `actionlint` |
   | `make test` | tests unitaires Go et TypeScript |
   | `make acceptance` | scénarios godog contre l'API |
   | `make e2e` | scénarios Playwright `@ui` |
   | `make vuln` | `govulncheck` et audit des dépendances npm |
   | `make build` | front, puis binaire statique |
   | `make ci` | l'ensemble des contrôles exigés par la CI, dans l'ordre |

3. **`make ci` est l'unique commande lancée par la CI**, et peut être lancée en local avant de pousser.
4. **Les cibles sont documentées dans `CLAUDE.md`**, pour que Claude Code les utilise plutôt que des commandes improvisées.
5. **Conventions d'écriture** : cibles déclarées `.PHONY`, commentaire `##` sur chaque cible pour alimenter `make help`, GNU make supposé.

## Alternatives envisagées

- **Task (Taskfile)** : écrit en Go, épinglable dans `go.mod` et lançable par `go tool task` ; gère bien les dépendances entre tâches. Écarté : description des tâches en YAML, moins lisible, et gain faible pour un sommaire de commandes.
- **just** : syntaxe proche du Makefile sans ses pièges, mais un binaire de plus (écrit en Rust, donc hors de `go tool`) à installer et épingler.
- **Mage** : tâches écrites en Go, typées ; verbeux et peu connu.
- **Scripts shell dans `scripts/`** : simples, mais sans vue d'ensemble des commandes ni enchaînement.
- **Scripts npm** : ne couvrent que le front.

## Conséquences

- **Positif** : aucun outil à installer (make est présent sur tout système Linux et dans le Dev Container) ; commandes identiques pour le développeur, Claude Code et la CI ; découverte immédiate par `make help`.
- **Négatif** : syntaxe datée (tabulations, pièges du shell) ; tenue à distance en gardant les cibles triviales.
- **Discipline** : une cible qui grossit est le signal qu'il faut déplacer sa logique dans un outil du projet.
