# Sécurité du dépôt GitHub

Liste des protections du dépôt public `nelt/tribe-menus` (ADR 0011). Elles sont appliquées **avant** le passage en public, puis revues à chaque release majeure. Chaque case cochée correspond à un réglage vérifié.

## 1. Compte et accès

- [ ] Double authentification sur le compte GitHub, par clé de sécurité ou passkey (pas de SMS).
- [ ] Aucun jeton « classique » : uniquement des jetons à portée fine (*fine-grained*), limités au dépôt `tribe-menus`, avec les seuls droits nécessaires et une date d'expiration.
- [ ] Jeton dédié pour le Dev Container (ADR 0009, 0019), distinct de tout autre usage, **en lecture seule** : *Actions*, *Contents*, *Pull requests* et *Metadata*. Rangé dans le volume Docker `tribe-menus-gh`.
- [ ] Jeton dédié au poste de développement (commits et pushes depuis l'hôte, en HTTPS), distinct du jeton du Dev Container ; conservé par le gestionnaire d'identifiants du système, jamais en clair (`credential.helper store` proscrit).
- [ ] Git configuré avec l'adresse « noreply » de GitHub, et option « Block command line pushes that expose my email » activée.
- [ ] Revue périodique des applications GitHub et OAuth autorisées sur le compte.

## 2. Branches et étiquettes (rulesets)

**Branche `main`** :

- [ ] modification uniquement par PR ; aucun push direct, y compris pour l'administrateur ;
- [ ] vérification `ci` (le job qui lance `make ci`) obligatoire et à jour avec `main` avant fusion ;
- [ ] fusion par *squash* uniquement, historique linéaire ;
- [ ] commits signés exigés : satisfait par la fusion *squash*, dont le commit est créé et signé par GitHub, les commits des branches pouvant rester non signés ;
- [ ] conversations de revue résolues avant fusion ;
- [ ] ni force push, ni suppression.

Projet mené seul : aucune approbation n'est exigée (on ne peut pas approuver sa propre PR), mais la PR reste obligatoire, avec la CI au vert.

**Étiquettes `v*`** :

- [ ] création réservée au propriétaire du dépôt ;
- [ ] ni mise à jour, ni suppression (ADR 0012).

## 3. Contributions externes

- [ ] Workflows des PR venant de forks : approbation exigée pour **tous** les contributeurs externes (pas seulement les nouveaux).
- [ ] Vérification DCO en CI : chaque commit d'une PR porte une ligne `Signed-off-by` (étape écrite dans le projet, sans action tierce).
- [ ] `CONTRIBUTING.md` (DCO, conventions, lien vers les ADR) et `SECURITY.md`.
- [ ] Wiki et Projects désactivés ; Discussions désactivées ; tickets activés avec modèles.
- [ ] Limites d'interaction temporaires disponibles en cas d'abus (réglage « Interaction limits »).

## 4. GitHub Actions

- [ ] Actions autorisées : celles de GitHub et une liste explicite d'actions tierces ; épinglage par empreinte de commit complète exigé (réglage de dépôt, s'il est disponible ; sinon vérifié par l'analyse des workflows).
- [ ] Jeton `GITHUB_TOKEN` en lecture seule par défaut ; droits élargis explicitement, job par job (`permissions:`).
- [ ] « Allow GitHub Actions to create and approve pull requests » désactivé.
- [ ] Jamais de déclencheur `pull_request_target` ; jamais de runner auto-hébergé.
- [ ] Aucune donnée venant d'une PR (titre, branche, corps) interpolée directement dans un script de workflow (injection de commande).
- [ ] Aucun secret ni clé de déploiement chez GitHub : le déploiement est manuel, par SSH depuis le poste de l'administrateur (ADR 0016).
- [ ] Workflows analysés en CI par **actionlint** (écrit en Go, épinglé par `go tool`).

## 5. Sécurité du code et des dépendances

- [ ] Détection de secrets (*secret scanning*) et blocage des pushes qui en contiennent (*push protection*).
- [ ] Analyse de code **CodeQL** (Go, JavaScript/TypeScript, workflows Actions) sur chaque PR, en configuration par défaut. Disponible une fois le dépôt public seulement (GitHub Code Security sinon).
- [ ] **Dependabot** : alertes, mises à jour de sécurité, et mises à jour de versions groupées (Go, npm, actions, Dev Container), mensuelles (`.github/dependabot.yml`, 2026-09-30).
- [ ] **Dependency review** sur chaque PR : bloque l'ajout d'une dépendance vulnérable ou sous une licence incompatible avec l'AGPL (`.github/workflows/dependency-review.yml`). Le job ne s'exécute qu'une fois le dépôt public ; vérifier alors qu'il passe.
- [ ] `govulncheck` et audit npm dans `make ci` (ADR 0010).
- [ ] npm : `npm ci --ignore-scripts`, versions exactes, fichier de verrouillage versionné (ADR 0004).
- [ ] Signalement privé des vulnérabilités activé (réservé aux dépôts publics ; `SECURITY.md` y renvoie).

## 6. Présentation du dépôt

Réglages sans enjeu de sécurité, faits au même moment parce que certains n'existent que pour un dépôt public.

- [ ] Description « Melting Tribe : les menus de la semaine, en tribu » et site `https://meltingtribe.codingmatters.org` (ADR 0017).
- [ ] Aperçu social (Settings › General › Social preview) : `docs/design/identite/partage.png` (`docs/design/README.md`). Réservé aux dépôts publics.

## 7. Plus tard

- Signature des commits locaux par clé SSH (clé dédiée à la signature, protégée par une phrase de passe) et « vigilant mode » activé. Reportée le 2026-09-29 : la signature par GitHub des commits de fusion *squash* suffit pour `main` au démarrage. Le vigilant mode ne s'active qu'avec la signature, sinon tous les commits du compte apparaissent non vérifiés.
- Attestation de provenance des archives de release (ADR 0012).
- Score OpenSSF Scorecard publié.
- Clé de déploiement restreinte côté serveur (commande forcée), si le déploiement tiré par le serveur (alternative reportée de l'ADR 0016) est retenu.
