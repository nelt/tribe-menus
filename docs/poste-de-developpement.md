# Préparer un poste de développement

Mise en route d'un poste Linux pour travailler sur Melting Tribe (ADR 0009, 0018). Sur l'hôte ne tournent que Docker, l'interface de VS Code, Git et GitHub CLI avec identifiants ; tout le reste (Go, Node, outils, Claude Code) tourne dans le Dev Container.

```
Hôte                              Dev Container
────                              ─────────────
VS Code (interface)  ◄──────────► extensions, terminaux, Claude Code
navigateur :8080     ◄── port ──► make dev (esbuild + serveur Go)
tribe-menus/         ◄─ partagé ─► /workspaces/tribe-menus
git push, gh (jeton)              git commit, make ci
```

## 1. Compte GitHub

Réglages du compte, décrits dans `docs/securite-depot.md` (section 1) :

- double authentification par passkey ou clé de sécurité ;
- adresse e-mail privée et blocage des pushes qui l'exposent (Settings › Emails) ; noter l'adresse `ID+nelt@users.noreply.github.com` ;
- un jeton à portée fine **dédié au poste**, limité au dépôt `tribe-menus`, avec une date d'expiration. Permissions : *Contents*, *Pull requests* et *Workflows* en lecture et écriture.

## 2. Logiciels de l'hôte

- **git** ;
- **Docker Engine** (paquets de la distribution ou dépôt officiel Docker) ; ajouter son utilisateur au groupe `docker`, puis se reconnecter ;
- **VS Code**, distribution officielle, avec la seule extension **Dev Containers** (Microsoft) ;
- **GitHub CLI** (`gh`), pour ouvrir les PR et suivre la CI depuis le terminal ;
- un **trousseau** pour les identifiants Git et `gh` : `git-credential-libsecret` (souvent dans le paquet `git` ou `libsecret`, selon la distribution).

## 3. Profil VS Code

Créer un profil « Melting Tribe » vide (Profiles › New Profile), y installer Dev Containers, et régler dans ses paramètres (`settings.json` du profil) :

```json
{
  "telemetry.telemetryLevel": "off",
  "extensions.autoUpdate": false,
  "dev.containers.copyGitConfig": false,
  "dev.containers.gitCredentialHelperConfigLocation": "none"
}
```

Les deux derniers réglages empêchent VS Code de copier la configuration Git de l'hôte et de transmettre son assistant d'identifiants au conteneur (ADR 0018). L'agent SSH est coupé par `devcontainer.json`.

## 4. Clone et configuration Git

```bash
git clone https://github.com/nelt/tribe-menus.git
cd tribe-menus

# Identité (propre à ce dépôt, lue aussi dans le conteneur)
git config user.name  "Nel Taurisson"
git config user.email "ID+nelt@users.noreply.github.com"

# Jeton du poste : un identifiant pour ce seul dépôt, rangé dans le trousseau
git config credential.https://github.com.useHttpPath true
git config credential.https://github.com.username nelt
git config --add credential.helper ""
git config --add credential.helper /usr/lib/git-core/git-credential-libsecret

git ls-remote origin      # coller le jeton comme mot de passe, une seule fois
```

`gh` ne lit pas les identifiants de Git : il a sa propre entrée dans le trousseau. On lui donne le même jeton, repris du trousseau sans passer par le presse-papiers :

```bash
printf "protocol=https\nhost=github.com\npath=nelt/tribe-menus.git\n\n" | git credential fill | sed -n 's/^password=//p' | gh auth login --with-token
gh auth status            # doit indiquer le compte nelt et un jeton rangé dans le trousseau (keyring)
```

Si `gh auth status` ne mentionne pas le trousseau, le jeton a été écrit en clair dans `~/.config/gh/hosts.yml` (aucun trousseau disponible) : le retirer par `gh auth logout` et installer le trousseau d'abord.

Ne pas utiliser `gh auth login` par le navigateur : il crée un jeton OAuth valable sur tout le compte, contraire à `docs/securite-depot.md` (section 1).

Pour remplacer le jeton (expiration, nouvelles permissions), mettre à jour les deux copies :

```bash
printf "protocol=https\nhost=github.com\npath=nelt/tribe-menus.git\n\n" | git credential reject
git ls-remote origin      # coller le nouveau jeton
gh auth logout --hostname github.com
printf "protocol=https\nhost=github.com\npath=nelt/tribe-menus.git\n\n" | git credential fill | sed -n 's/^password=//p' | gh auth login --with-token
```

Les commits portent la ligne DCO : `git commit -s` (ADR 0011).

## 5. Ouvrir le Dev Container

1. Ouvrir le dossier `tribe-menus` dans VS Code, avec le profil « Melting Tribe ».
2. « Reopen in Container ». La première construction télécharge l'image Go, Node et Claude Code : compter quelques minutes.
3. Dans un terminal du conteneur, lancer `claude` une première fois pour se connecter à son compte. La connexion est conservée dans le volume `tribe-menus-claude`.

Tant que le projet n'est pas initialisé, suivre `docs/plans/2026-09-29-environnement-de-developpement.md`, par exemple en demandant à Claude Code : « lis `docs/plans/2026-09-29-environnement-de-developpement.md` et exécute-le étape par étape ».

## 6. Au quotidien

- Dans le conteneur : `make dev` (application sur `http://localhost:8080`), `make test`, `make ci` avant de pousser. `make help` liste les cibles.
- Claude Code travaille dans le conteneur et peut commiter ; il ne pousse pas.
- Depuis un terminal **de l'hôte** : `git push`, puis `gh pr create` (ou l'interface de GitHub) ; `gh pr checks` suit la CI.
