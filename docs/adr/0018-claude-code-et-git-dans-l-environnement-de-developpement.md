# 0018. Claude Code et Git dans l'environnement de développement

- **Date** : 2026-09-29
- **Statut** : accepté ; complété par 0019 (GitHub CLI en lecture seule dans le conteneur) ; amendé par 0020 (push et PR depuis le conteneur, Remote Control)

## Contexte

L'ADR 0009 fait du Dev Container l'environnement de référence et demande de ne pas exposer au conteneur les identifiants de l'hôte, qu'une extension pourrait utiliser. Deux points restaient ouverts :

- **où s'exécutent les opérations Git** qui demandent des identifiants (push) ;
- **comment Claude Code s'insère** dans cet environnement : le projet est développé avec lui, sur le poste comme dans le cloud.

Claude Code lit et écrit des fichiers et lance des commandes : le même raisonnement d'isolement que pour les extensions s'applique à lui. La signature des commits est reportée (`docs/securite-depot.md`, section « Plus tard »).

## Décision

1. **Git avec identifiants depuis l'hôte uniquement.** Le push se fait depuis le poste, avec le jeton dédié au poste (`docs/securite-depot.md`). Le conteneur ne reçoit ni ce jeton, ni l'agent SSH (`SSH_AUTH_SOCK` vidé dans `devcontainer.json`), ni la configuration Git de l'hôte ni son assistant d'identifiants (réglages du profil VS Code, `docs/poste-de-developpement.md`). Les commits restent possibles dans le conteneur : l'identité est lue dans `.git/config`, qui fait partie du dossier partagé.
2. **Claude Code s'exécute dans le Dev Container**, jamais directement sur l'hôte pour ce projet : il n'y voit que le dépôt, pas le dossier personnel.
   - le socle du conteneur (`.devcontainer/`, `.nvmrc`, `.claude/settings.json`) est versionné avant l'initialisation du projet, pour que Claude Code puisse l'initialiser depuis le conteneur (plan `docs/plans/2026-09-29-environnement-de-developpement.md`) ;
   - la CLI est installée dans l'image par l'installateur officiel, **version épinglée** dans le `Dockerfile` ; l'extension VS Code est épinglée à la même version dans `devcontainer.json` ;
   - mises à jour automatiques, télémétrie et rapports d'erreur désactivés (`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`) ; une montée de version passe par une PR ;
   - sa configuration et sa connexion au compte sont conservées dans un volume Docker dédié (`CLAUDE_CONFIG_DIR`), pour survivre à la reconstruction du conteneur ;
   - `.claude/settings.json` (versionné) autorise sans confirmation les commandes courantes du projet (`make`, `go`, `npm`, Git en lecture et commit), refuse `git push` et la lecture des fichiers de secrets ; les réglages personnels vont dans `.claude/settings.local.json`, ignoré par Git.
3. **Le client `sqlite3`** est installé dans l'image (ADR 0009), qui part de l'image officielle `golang` : Node y est ajouté à la version de `.nvmrc`, avec vérification de l'empreinte publiée, sans nvm ni fonctionnalité Dev Container tierce.

## Alternatives envisagées

- **Push depuis le conteneur avec le jeton « devcontainer »** : pratique depuis le terminal de VS Code, mais met un jeton en écriture à portée de toute extension et de Claude Code. Écarté ; le jeton reste possible plus tard si le besoin apparaît. *Le besoin est apparu en lecture : jeton en lecture seule dans le conteneur, ADR 0019 ; l'écriture reste écartée.*
- **Claude Code sur l'hôte** : installation plus simple, mais accès à tout le dossier personnel, et versions d'outils différentes de celles du conteneur. Écarté.
- **Fonctionnalité Dev Container de Claude Code** (`ghcr.io/anthropics/devcontainer-features/claude-code`) : installe par npm une version non épinglée par défaut. L'installateur officiel avec numéro de version suffit.
- **Image `mcr.microsoft.com/devcontainers/go` et fonctionnalité Node** : fonctionne, mais la version de Node serait dupliquée hors de `.nvmrc`.
- **Pare-feu sortant dans le conteneur** (Dev Container de référence de Claude Code) : utile pour lancer Claude Code sans demande d'autorisation ; non nécessaire tant que les autorisations restent actives. Reporté.

## Conséquences

- **Positif** : aucun identifiant en écriture dans le conteneur ; Claude Code confiné au dépôt et aux mêmes versions d'outils que le développeur et la CI ; la connexion à Claude survit aux reconstructions.
- **Négatif** : un aller-retour vers un terminal de l'hôte pour chaque push ; la version de Claude Code est à monter à la main, par PR (deux lignes : `Dockerfile` et `devcontainer.json`).
- **Signature des commits** : si elle est activée plus tard, elle se fera également depuis l'hôte.
