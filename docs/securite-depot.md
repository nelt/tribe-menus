# Sécurité du dépôt GitHub

Liste des protections du dépôt public `nelt/tribe-menus` (ADR 0011). Le dépôt est public depuis le 2026-10-03. Les protections sont appliquées au passage en public (certaines, comme les rulesets, n'existent qu'ensuite sur un compte Free), puis revues à chaque release majeure. Chaque case cochée correspond à un réglage vérifié. Dernière revue complète : 2026-10-05 (plan `revue-securite`, étape 7), par le développeur dans GitHub, avec une session Claude pour ce que l'API laisse lire.

## 1. Compte et accès

- [x] Double authentification sur le compte GitHub, par clé de sécurité ou passkey (pas de SMS).
- [ ] Aucun jeton « classique » : uniquement des jetons à portée fine (*fine-grained*), limités au dépôt `tribe-menus`, avec les seuls droits nécessaires et une date d'expiration.
- [x] Jeton dédié pour le Dev Container (ADR 0019, 0020), distinct de tout autre usage : *Contents* et *Pull requests* en lecture et écriture, *Actions* et *Metadata* en lecture, **sans *Workflows***. À ne passer en écriture qu'une fois le ruleset de `main` actif (section 2). Rangé dans le volume Docker `tribe-menus-gh`.
- [x] Jeton dédié au poste de développement (commits et pushes depuis l'hôte, en HTTPS), distinct du jeton du Dev Container ; conservé par le gestionnaire d'identifiants du système, jamais en clair (`credential.helper store` proscrit).
- [x] Git configuré avec l'adresse « noreply » de GitHub, et option « Block command line pushes that expose my email » activée.
- [x] Revue périodique des applications GitHub et OAuth autorisées sur le compte. Dernière revue le 2026-10-05 : rien d'inconnu ni d'abandonné.
- [x] Double authentification sur le compte claude.ai : il a accès au dépôt en écriture (sessions cloud) et, avec Remote Control, aux sessions du Dev Container (ADR 0020). La connexion passe par un compte Google : c'est sa double authentification qui protège l'accès.

**Application GitHub de Claude** (sessions cloud), constatée le 2026-10-05 : installée sur le seul dépôt `tribe-menus`. Ses permissions sont fixées par son éditeur et ne se réduisent pas : lecture de l'administration, des statuts de commit, des files de fusion et des métadonnées ; lecture et écriture des actions, des vérifications, du code, des discussions, des tickets, des PR, **des hooks du dépôt et des workflows**. La restriction « sans *Workflows* » ne vaut donc que pour le jeton du Dev Container. Ce qui limite la portée de cet écart :

- le ruleset de `main` (section 2) : un workflow modifié n'atteint `main` que par une PR fusionnée par le développeur ;
- aucun secret chez GitHub et un `GITHUB_TOKEN` en lecture seule (section 4) : un workflow modifié sur une branche s'exécute sur sa PR, sans rien à lire ni droit d'écrire ;
- l'intermédiaire par lequel passent les sessions cloud refuse la lecture des hooks, des réglages d'Actions et des collaborateurs (constaté le 2026-10-05 ; l'écriture n'a pas été essayée) ;
- la liste des webhooks du dépôt, vide à cette date, est à relire à chaque revue.

L'application ne lit ni les alertes d'analyse de code, de Dependabot et de détection de secrets, ni les avis de sécurité (ADR 0022).

## 2. Branches et étiquettes (rulesets)

**Branche `main`** (ruleset `main`, actif depuis le 2026-10-03, liste de contournement vide) :

- [x] modification uniquement par PR ; aucun push direct, y compris pour l'administrateur ;
- [x] vérification `ci` (le job qui lance `make ci`) obligatoire et à jour avec `main` avant fusion ;
- [x] fusion par *squash* uniquement, historique linéaire ;
- [ ] commits signés exigés : satisfait par la fusion *squash*, dont le commit est créé et signé par GitHub, les commits des branches pouvant rester non signés ;
- [x] conversations de revue résolues avant fusion ;
- [x] ni force push, ni suppression.

Projet mené seul : aucune approbation n'est exigée (on ne peut pas approuver sa propre PR), mais la PR reste obligatoire, avec la CI au vert.

**Étiquettes `v*`** :

- [x] création réservée au propriétaire du dépôt ; ruleset `versions-creation`, que seul le rôle d'administrateur contourne ;
- [x] ni mise à jour, ni suppression (ADR 0012) : ruleset `versions-immuables`, sans contournement.

## 3. Contributions externes

- [x] Workflows des PR venant de forks : approbation exigée pour **tous** les contributeurs externes (pas seulement les nouveaux).
- [x] Vérification DCO en CI : chaque commit d'une PR porte une ligne `Signed-off-by` (étape écrite dans le projet, sans action tierce).
- [x] `CONTRIBUTING.md` (DCO, conventions, lien vers les ADR) et `SECURITY.md`.
- [x] Wiki et Projects désactivés ; Discussions désactivées ; tickets activés avec modèles. Modèles « Bogue » et « Proposition » dans `.github/ISSUE_TEMPLATE/`, tickets vierges désactivés, lien vers le signalement privé (2026-10-05).
- [x] Limites d'interaction temporaires disponibles en cas d'abus (réglage « Interaction limits »).

## 4. GitHub Actions

- [x] Actions autorisées : celles de GitHub et une liste explicite d'actions tierces ; épinglage par empreinte de commit complète exigé (réglage de dépôt « Require actions to be pinned to a full-length commit SHA », activé le 2026-10-05 ; la liste explicite est vide, les workflows n'utilisent que des actions de GitHub).
- [x] Jeton `GITHUB_TOKEN` en lecture seule par défaut ; droits élargis explicitement, job par job (`permissions:`).
- [x] « Allow GitHub Actions to create and approve pull requests » désactivé.
- [x] Jamais de déclencheur `pull_request_target` ; jamais de runner auto-hébergé.
- [x] Aucune donnée venant d'une PR (titre, branche, corps) interpolée directement dans un script de workflow (injection de commande).
- [x] Aucun secret ni clé de déploiement chez GitHub : le déploiement est manuel, par SSH depuis le poste de l'administrateur (ADR 0016).
- [x] Workflows analysés en CI par **actionlint** (écrit en Go, épinglé par `go tool`).

## 5. Sécurité du code et des dépendances

- [x] Détection de secrets (*secret scanning*) et blocage des pushes qui en contiennent (*push protection*). Activés le 2026-10-05 ; le balayage de l'historique n'a rien trouvé.
- [x] Analyse de code **CodeQL** (Go, JavaScript/TypeScript, workflows Actions) sur chaque PR, en configuration par défaut. Disponible une fois le dépôt public seulement (GitHub Code Security sinon). Activée le 2026-10-05 ; première analyse sans alerte.
- [x] **Dependabot** : alertes, mises à jour de sécurité, et mises à jour de versions groupées (Go, npm, actions, Dev Container), mensuelles (`.github/dependabot.yml`, 2026-09-30).
- [x] **Dependency review** sur chaque PR : bloque l'ajout d'une dépendance vulnérable ou sous une licence incompatible avec l'AGPL (`.github/workflows/dependency-review.yml`). Le job ne s'exécute qu'une fois le dépôt public ; vérifier alors qu'il passe.
- [x] `govulncheck` et audit npm dans `make ci` (ADR 0010).
- [x] npm : `npm ci --ignore-scripts`, versions exactes, fichier de verrouillage versionné (ADR 0004).
- [x] Signalement privé des vulnérabilités activé (réservé aux dépôts publics ; `SECURITY.md` y renvoie). Activé le 2026-10-05 ; lu par le seul développeur, aucun jeton de session n'a de droit sur les avis de sécurité (ADR 0022).

## 6. Présentation du dépôt

Réglages sans enjeu de sécurité, faits au même moment parce que certains n'existent que pour un dépôt public.

- [x] Description « Melting Tribe : les menus de la semaine, en tribu » et site `https://meltingtribe.codingmatters.org` (ADR 0017).
- [x] Aperçu social (Settings › General › Social preview) : `docs/design/identite/partage.png` (`docs/design/README.md`). Réservé aux dépôts publics.

## 7. Plus tard

- Signature des commits locaux par clé SSH (clé dédiée à la signature, protégée par une phrase de passe) et « vigilant mode » activé. Reportée le 2026-09-29 : la signature par GitHub des commits de fusion *squash* suffit pour `main` au démarrage. Le vigilant mode ne s'active qu'avec la signature, sinon tous les commits du compte apparaissent non vérifiés.
- Attestation de provenance des archives de release (ADR 0012).
- Score OpenSSF Scorecard publié.
- Clé de déploiement restreinte côté serveur (commande forcée), si le déploiement tiré par le serveur (alternative reportée de l'ADR 0016) est retenu.
