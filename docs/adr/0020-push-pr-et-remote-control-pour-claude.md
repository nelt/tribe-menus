# 0020. Push, PR et Remote Control pour Claude

- **Date** : 2026-10-03
- **Statut** : accepté ; amende 0018 (point 1 et autorisations du point 2) et 0019 (point 1 et autorisations du point 4)

## Contexte

Les ADR 0018 et 0019 ne laissaient aucun identifiant en écriture à Claude : depuis le Dev Container, il commitait mais chaque push et chaque PR se faisaient à la main depuis l'hôte ; depuis une session cloud, les fichiers produits étaient téléchargés puis intégrés à la main. L'objectif du projet, un travail fluide entre le développeur et Claude, n'était plus tenu.

Le verrou tenait à une seule raison : un jeton à portée fine ne se limite pas à une branche, et aucun ruleset ne pouvait protéger `main` sur un dépôt privé avec un compte GitHub Free. L'ADR 0019 reportait l'écriture « au passage en public, quand les rulesets interdiront le push direct et le force push sur `main` ».

Le dépôt est public depuis le 2026-10-03 et le ruleset `main` est actif, sans contournement possible, y compris pour l'administrateur : modification par PR seulement, vérification `ci` obligatoire, fusion *squash*, ni force push ni suppression (`docs/securite-depot.md`, section 2).

Par ailleurs, les sessions de Claude Code dans le Dev Container ne pouvaient pas être suivies depuis un autre appareil : la variable `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` (ADR 0018) coupe aussi Remote Control.

## Décision

1. **Claude pousse des branches `feature/…` et ouvre les PR**, depuis le Dev Container comme depuis une session cloud. La fusion reste un geste du développeur.
2. **Jeton `tribe-menus-devcontainer` en écriture** : *Contents* et *Pull requests* en lecture et écriture, *Actions* et *Metadata* en lecture. **Pas de permission *Workflows*** : un push qui modifie `.github/workflows/` est refusé par GitHub et se fait depuis l'hôte. Le reste de l'ADR 0019 est inchangé (jeton à portée fine, limité au dépôt, avec expiration, rangé dans le volume `tribe-menus-gh`).
3. **Git du conteneur authentifié par `gh`** : `devcontainer.json` passe par l'environnement (`GIT_CONFIG_COUNT`…) deux réglages `credential.helper` qui vident la liste héritée de `.git/config`, écrite pour le trousseau de l'hôte, puis désignent `gh auth git-credential`. Rien n'est écrit dans `.git/config` ni dans le dossier personnel du conteneur. Le conteneur ne reçoit toujours ni le jeton du poste, ni l'agent SSH, ni la configuration Git de l'hôte.
4. **Autorisations de Claude Code** (`.claude/settings.json`) :
   - sans confirmation : `git fetch`, `git push` vers `origin feature/…`, `gh pr create`, `gh pr edit`, `gh pr comment` (ajouté le 2026-10-03 pour les revues entre sessions, `docs/revue-de-pr.md`) ; `gh api` limité aux chemins `actions/` du dépôt (lecture de la CI) ;
   - refusés : push forcé, tout push qui nomme `main`, `gh pr merge` ;
   - sur confirmation : la fermeture d'une PR, la suppression d'une branche distante, tout autre push ou appel `gh api`. La suppression d'une branche était d'abord refusée ; le refus ne couvrait qu'une des façons de la faire (`git push --delete`, mais ni `gh pr close --delete-branch` ni `git push origin :branche`) et empêchait le ménage des branches fusionnées. Le ruleset interdit de toute façon la suppression de `main` (modifié le 2026-10-03, après le premier essai dans le conteneur).
   Ces motifs préviennent une erreur de Claude ; la garantie est le ruleset, qui s'applique quel que soit le client.
5. **Sessions cloud** : le dépôt y est rattaché en écriture à la demande du développeur. Les commits portent l'identité « noreply » du développeur et sa ligne `Signed-off-by` (DCO, ADR 0011), comme ceux du poste.
6. **Remote Control connecté par défaut dans le Dev Container** :
   - `remoteControlAtStartup: true` dans `/etc/claude-code/managed-settings.json`, copié dans l'image depuis `.devcontainer/claude-managed-settings.json`. Ce réglage est ignoré dans `.claude/settings.json` : un fichier versionné du projet ne peut pas l'activer pour qui ouvre le dépôt ; le fichier système de l'image le peut ;
   - `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` est remplacée par le seul `DISABLE_AUTOUPDATER`, qui préserve l'épinglage de la version. Télémétrie, rapports d'erreur et commande `/feedback` de Claude Code ne sont plus coupés : ils n'étaient désactivés que par effet de la variable globale, et `DISABLE_TELEMETRY` gêne Remote Control dans certains cas ;
   - le workflow hebdomadaire `devcontainer` signale par un avertissement une version de Claude Code plus récente que la version épinglée, que Dependabot ne suit pas ;
   - pour s'en passer sur un poste : `"remoteControlAtStartup": false` dans `.claude/settings.local.json`.

## Alternatives envisagées

- **Statu quo** (ADR 0018, 0019) : aucun identifiant en écriture, mais un aller-retour manuel par push et par PR. Écarté : c'est la friction à supprimer.
- **GitHub Pro pour garder le dépôt privé** : donne les mêmes rulesets. Écarté : le passage en public était déjà décidé (ADR 0011).
- **Écriture sans ruleset, avec les seuls refus de `.claude/settings.json` et un hook `pre-push`** : couvre une erreur de Claude, pas une extension compromise. Écarté.
- **Permission *Workflows* dans le jeton du conteneur** : permettrait à Claude de pousser des changements de CI. Écarté : la CI est la vérification qui garde `main`, sa modification reste un geste fait depuis l'hôte.
- **`gh auth setup-git`** : écrit l'assistant d'identifiants dans le `~/.gitconfig` du conteneur, perdu à chaque reconstruction, et sans effet ici puisque `.git/config` vide la liste des assistants. Écarté au profit de l'environnement.
- **Remote Control activé par `/config`** (réglage utilisateur, dans le volume `tribe-menus-claude`) : fonctionne, mais n'est ni versionné ni reproduit sur un autre poste. Écarté.
- **Conserver `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` et renoncer à Remote Control** : écarté, c'est l'un des deux besoins.
- **Reproduire son périmètre par des réglages unitaires** (`DISABLE_TELEMETRY`, `DISABLE_ERROR_REPORTING`, `DISABLE_FEEDBACK_COMMAND`) : écarté. Ces données ne contiennent pas le code d'un dépôt de toute façon public, et `DISABLE_TELEMETRY` rend Remote Control indisponible si l'option « Require trusted devices » du compte est activée.

## Conséquences

- **Positif** : plus d'aller-retour par l'hôte pour pousser et ouvrir une PR ; plus de fichiers à intégrer à la main depuis une session cloud ; une session du Dev Container se pilote depuis claude.ai ou l'application mobile.
- **Négatif, jeton** : un jeton en écriture est lisible par toute extension et par Claude Code dans le conteneur. En cas de fuite ou d'erreur, il permet de créer, modifier et supprimer des branches autres que `main`, et d'ouvrir ou de fusionner une PR dont la CI est verte. `main` reste protégée par le ruleset et par la CI, que ce jeton ne peut pas modifier.
- **Négatif, fusion** : projet mené seul, aucune approbation ne peut être exigée ; rien, côté GitHub, n'empêche le jeton de fusionner une PR verte. Seul le refus de `gh pr merge` dans `.claude/settings.json` s'y oppose.
- **Négatif, Remote Control** : pour le dépôt, aucun risque nouveau, le compte claude.ai y ayant déjà accès en écriture par les sessions cloud. Ce qui s'ajoute : qui accède à ce compte peut lancer des commandes dans le conteneur, sur le poste (dossier du dépôt, réseau local), ce qu'une session cloud ne permet pas. Tant que Remote Control est connecté, la transcription de la session est conservée sur les serveurs d'Anthropic. Le conteneur n'ouvre aucun port : seules des requêtes HTTPS sortantes.
- **Négatif, trafic** : Claude Code envoie de nouveau ses métriques d'usage (sans code, prompts ni chemins) et ses rapports d'erreur internes à Anthropic, et interroge ses réglages de fonctionnalités ; seules les mises à jour automatiques restent coupées. La télémétrie de VS Code et de `gh` reste désactivée.
- **Limites connues** :
  - Remote Control exige une connexion par compte claude.ai (pas de clé d'API) ;
  - les changements de `.github/workflows/` se poussent depuis l'hôte.
- **À faire sur le poste** : modifier les permissions du jeton `tribe-menus-devcontainer` (sa valeur ne change pas, pas de nouvelle connexion de `gh`), puis reconstruire le conteneur (`docs/poste-de-developpement.md`).
