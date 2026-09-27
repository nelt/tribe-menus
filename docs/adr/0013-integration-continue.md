# 0013. Intégration continue (GitHub Actions)

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

La CI exécute `make ci`, commande unique partagée avec le développeur et Claude Code (ADR 0010). Le dépôt est public : la CI est exposée aux PR externes et doit suivre les règles de durcissement de l'ADR 0011 et de `docs/securite-depot.md`. Les versions d'outils sont fixées par `go.mod`, `.nvmrc` et `web/package.json` (ADR 0009). La release est déclenchée par une étiquette (ADR 0012).

## Décision

1. **Workflows** :
   - **`ci.yml`** : sur chaque PR, chaque push sur `main`, à la demande, et une fois par semaine (pour détecter une vulnérabilité nouvellement publiée dans une dépendance même sans changement de code). Un seul job `ci` qui lance `make tools` puis `make ci` ; c'est la vérification obligatoire pour fusionner. Il vérifie aussi la signature DCO des commits d'une PR (ADR 0011) et, sur une PR, conserve quelques jours l'archive produite par `make build` comme artefact, pour l'environnement de recette (ADR 0016) ;
   - **`release.yml`** : sur une étiquette `v*` (ADR 0012) ;
   - **`dependency-review.yml`** : sur chaque PR, bloque l'ajout d'une dépendance vulnérable ou sous une licence incompatible avec l'AGPL ;
   - **`devcontainer.yml`** : une fois par semaine et sur les PR qui modifient `.devcontainer/`, construit le Dev Container et y lance `make tools`, pour vérifier qu'il fonctionne toujours ;
   - **CodeQL** en configuration par défaut (Go, JavaScript/TypeScript, workflows), activée dans les réglages du dépôt, sans fichier à maintenir.
2. **Environnement d'exécution** : runner hébergé par GitHub, image épinglée (`ubuntu-24.04`) ; Go et Node installés par les actions officielles `setup-go` (version lue dans `go.mod`) et `setup-node` (version lue dans `.nvmrc`). Mêmes versions que dans le Dev Container, sans le coût de sa construction à chaque PR.
3. **Caches** : modules et compilation Go, paquets npm, navigateurs Playwright (clé liée à la version de Playwright).
4. **Robustesse** :
   - annulation d'une exécution devenue obsolète quand une PR reçoit un nouveau commit (`concurrency`) ;
   - délai maximal par job (`timeout-minutes`) ;
   - en cas d'échec des scénarios `@ui`, traces, captures et vidéos Playwright conservées quelques jours comme artefacts.
5. **Sécurité** (en complément de `docs/securite-depot.md`) : `permissions` en lecture seule au niveau du workflow, élargies job par job ; checkout avec `persist-credentials: false` ; actions épinglées par empreinte de commit complète ; aucun secret accessible aux workflows de PR.
6. **Un seul job pour commencer.** Si la durée devient gênante, découpage en jobs parallèles appelant chacun une cible du Makefile, avec un job de synthèse comme vérification obligatoire.

## Alternatives envisagées

- **CI exécutée dans l'image du Dev Container** (action `devcontainers/ci`) : environnement strictement identique, mais plus lent (construction ou registre d'images) et une action de plus à surveiller. Remplacé par la construction hebdomadaire du Dev Container.
- **Jobs parallèles dès le départ** : retour plus rapide et échecs plus lisibles, mais un job de synthèse à maintenir ; reporté tant que la durée reste acceptable.
- **`ubuntu-latest`** : écarté, un changement d'image ne doit pas casser la CI à l'improviste. La montée de version du runner se fait par PR.

## Conséquences

- **Positif** : la CI lance exactement ce que le développeur lance (`make ci`) ; seules des actions officielles de GitHub sont nécessaires ; vulnérabilités détectées même en l'absence d'activité ; minutes illimitées sur un dépôt public.
- **Négatif** : durée estimée de 5 à 10 minutes par PR, essentiellement pour Playwright ; l'écart éventuel entre runner et Dev Container n'est détecté qu'à la construction hebdomadaire.
- **À faire à la montée de version** : l'image du runner (`ubuntu-24.04`) est mise à jour volontairement, par PR.
