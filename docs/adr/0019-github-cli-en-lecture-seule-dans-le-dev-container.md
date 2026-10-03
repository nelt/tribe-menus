# 0019. GitHub CLI en lecture seule dans le Dev Container

- **Date** : 2026-10-01
- **Statut** : accepté ; amendé par 0020 (jeton en écriture sur *Contents* et *Pull requests*)

## Contexte

L'ADR 0018 ne laisse aucun identifiant GitHub dans le Dev Container : le push et tout accès à GitHub se font depuis l'hôte. Il a écarté le jeton « devcontainer » en écriture, en le gardant possible « si le besoin apparaît ».

Les premières PR de la CI (ADR 0013) ont fait apparaître ce besoin, surtout en lecture. Pour comprendre l'échec d'une PR, il a fallu chaque fois passer par l'hôte : copier un journal de run, télécharger un artefact, relever l'état des runs et des alertes Dependabot, puis le transmettre à Claude Code.

Un jeton à portée fine ne peut pas être limité à certaines branches : un jeton qui peut pousser peut aussi pousser sur `main`. Sur un dépôt privé avec un compte GitHub Free, aucun ruleset ne l'empêche (`docs/securite-depot.md`).

## Décision

1. **Jeton `tribe-menus-devcontainer` en lecture seule**, à portée fine, limité au dépôt `tribe-menus`, avec une date d'expiration. Permissions : *Actions*, *Contents*, *Pull requests* et *Metadata* en lecture. Rien en écriture : le push et la création de PR restent sur l'hôte (ADR 0018, point 1, inchangé sur ce point).
2. **GitHub CLI (`gh`) dans l'image** du Dev Container : archive officielle, version épinglée dans le `Dockerfile`, vérifiée par l'empreinte publiée avec la version. Télémétrie et notifications de mise à jour désactivées (`GH_TELEMETRY=0`, `GH_NO_UPDATE_NOTIFIER=1`).
3. **Stockage du jeton dans un volume Docker dédié**, `tribe-menus-gh`, monté sur `~/.config/gh`, sur le modèle du volume de Claude Code. Le jeton est enregistré une fois par `gh auth login --with-token` dans un terminal du conteneur. Le conteneur n'ayant pas de trousseau, il est écrit en clair dans `hosts.yml`, dans le volume (sur l'hôte, sous `/var/lib/docker/volumes/`, lisible par root seulement) ; il n'est ni dans le dépôt, ni dans l'image, ni dans le dossier personnel de l'hôte, et survit à la reconstruction du conteneur.
4. **Autorisations de Claude Code** (`.claude/settings.json`) : commandes `gh` de lecture sans confirmation (`gh run list|view|download`, `gh pr list|view|diff`, `gh api`, `gh auth status`) ; refus de `gh auth token` et de la lecture de `~/.config/gh/`, pour que le jeton n'apparaisse pas dans une conversation.

## Alternatives envisagées

- **Jeton en écriture** (*Contents*, *Pull requests*) pour pousser et créer les PR depuis le conteneur : le plus pratique, mais tant qu'aucun ruleset ne protège `main`, une extension compromise ou une erreur de Claude Code pourrait pousser ou forcer `main`. Reporté au passage en public, quand les rulesets interdiront le push direct et le force push sur `main` ; à rouvrir alors par un nouvel ADR.
- **Statu quo** : aucun risque supplémentaire, mais un aller-retour par l'hôte pour chaque information de la CI. Écarté.
- **Variable d'environnement transmise par l'hôte** (`remoteEnv` avec `${localEnv:…}`) : rien d'écrit dans le conteneur, mais le jeton doit être dans l'environnement de VS Code à son lancement, en pratique en clair dans `~/.profile`, ou lu depuis le trousseau au démarrage de la session, ce qui dépend de la façon dont le bureau lance VS Code. Écartée, pour une exposition équivalente.

## Conséquences

- **Positif** : Claude Code suit lui-même la CI (runs, journaux d'échec, artefacts, PR, alertes) ; plus d'aller-retour par l'hôte pour lire.
- **Négatif** : un jeton lisible par toute extension et par Claude Code dans le conteneur ; en cas de fuite, il donne accès en lecture au code et aux journaux d'un dépôt destiné à devenir public. Une version d'outil de plus à monter à la main (`GH_VERSION`). Un jeton de plus à renouveler à son expiration.
- **Limite connue** : `gh pr checks` et `gh run view` sans option échouent avec un jeton à portée fine (pas de permission *Checks*) ; on passe par l'API d'Actions (`docs/poste-de-developpement.md`).
