# Plan : revue de sécurité avant mise en ligne

- **Nom** : `revue-securite`
- **Date** : 2026-10-04
- **Statut** : prêt (commence quand `socle/C` est fusionné et que le plan `canal-prive` a livré son document, étape 3)

## Objectif

S'assurer, avant d'écrire le déploiement, que le code existant ne livre pas de vulnérabilité : relire les lots A à C avec un regard neuf, trancher les points de sécurité laissés ouverts par les revues, et corriger ce qui doit l'être.

À la fin du plan :

- les constats de la revue sont écrits dans ce plan, chacun corrigé, reporté avec sa raison, ou écarté ;
- les points de specs sur la force brute et le blocage ciblé sont tranchés et, s'il y a lieu, implémentés ;
- l'état des protections du dépôt (`docs/securite-depot.md`) est vérifié.

**Hors périmètre** : ce qui n'existe qu'avec le déploiement, renvoyé aux plans `production` et `recette` de `docs/feuille-de-route.md` (adresse IP du client derrière Caddy, secrets, confinement systemd, configuration de Caddy, scripts de `deploy/`).

## Contexte et contraintes

- À lire avant de commencer : ADR 0001, 0003, 0004 (point 9), 0006, 0011, 0021 ; `docs/specs/exigences-non-fonctionnelles.md` (ENF-01, ENF-02) ; `gestion-membres-et-sessions.md` ; `docs/securite-depot.md` ; les notes d'exécution du plan `2026-10-03-socle-donnees-et-connexion.md` (revues des PR #26 et #29).
- **Qui fait quoi.** La revue (étapes 2 à 4) est faite par une **session cloud**, qui n'a pas écrit le code. Les corrections (étape 6) sont faites par **Claude Code dans le Dev Container**, qui peut lancer `make ci`. Le développeur arbitre entre les deux (étape 5).
- **Pas de PR à relire** : le code est déjà fusionné, `docs/revue-de-pr.md` ne s'applique donc pas tel quel. Les constats gardent les deux catégories du protocole, « à corriger » et « à noter ».
- **Les constats suivent le flux public**, par une PR : rien n'est en production, donc toute vulnérabilité relève du traitement normal (plan `canal-prive`, D8). Chaque constat de sécurité porte quand même les trois évaluations de `docs/traitement-des-vulnerabilites.md` : c'est le rodage de la grille avant qu'une version soit en ligne.
- **Un constat est concret** : fichier et fonction, scénario d'attaque en une phrase, conséquence. Pas de recommandation générale sans cas d'usage dans ce code.
- **Git** : une branche `feature/…` et une PR par étape qui change le dépôt ; `git commit -s` ; une ligne dans `CHANGELOG.md` par PR ; `make ci` avant de pousser pour les PR de code.

### Ordre de grandeur de la force brute

Avec les limites d'ENF-01, pour qui connaît l'identifiant d'URL d'une tribu et l'adresse d'un membre :

| Cible | Demandes de code par heure | Essais par heure | Chance de réussite sur un an d'attaque continue |
| --- | --- | --- | --- |
| Une adresse | 12 (3 par quart d'heure) | 36 | environ 1 sur 4 |
| Une tribu, plusieurs adresses | 30 | 90 | environ 1 sur 2 |

Chaque essai a une chance sur un million. L'attaque demande au moins trois adresses IP (10 demandes par heure et par IP) et envoie 12 e-mails par heure à chaque membre visé : elle est visible de la victime, mais rien ne la signale à l'administrateur ni ne l'arrête. En sens inverse, 30 demandes par heure suffisent à empêcher toute connexion à une tribu.

## Étapes

- [ ] **1. Décisions de specs**, avec le développeur, avant la revue.
  - **Force brute** (tableau ci-dessus). Pistes à comparer : accepter le risque et alerter l'administrateur sur les limites atteintes de façon répétée (l'alerte par e-mail arrive avec le déploiement) ; allonger le code (8 chiffres divisent le risque par 100) ; plafonner les demandes par adresse et par jour. Un plafond plus strict facilite le blocage ciblé d'un membre : les deux points se tranchent ensemble.
  - **Blocage ciblé d'une tribu** par la limite de 30 demandes par heure : l'accepter en V1, ou revoir ce que compte la limite par tribu, sans rien révéler de l'existence de la tribu ni de ses membres (ENF-02).
  - **Appareil détecté dans le journal d'audit** (D12 à confirmer) : chaîne d'affichage en anglais aujourd'hui (`audit_log.detected_device`), colonnes structurées dans `sessions` ; fixer la forme avant que des données réelles existent.
  - Résultat : ENF-01 et `gestion-membres-et-sessions.md` mis à jour, décisions consignées dans ce plan ; un ADR si le mécanisme d'authentification change (ADR 0001).
- [ ] **2. Revue : authentification et sessions** (`internal/tribe`, `internal/server/api.go`, `internal/mail`).
  - Codes : tirage, empreinte, comparaison en temps constant, usage unique, remplacement, expiration, essais ; codes fantômes indiscernables des vrais, en réponse comme en durée.
  - Sessions : tirage et empreinte du jeton, expiration glissante, déconnexion, attributs et portée du cookie, cookie d'une tribu présenté à une autre.
  - Limitation : contournements (casse, espaces, variantes de l'adresse ou de l'identifiant d'URL), demandes concurrentes, taille de la base de limitation.
  - Protection contre les requêtes d'une autre origine sur les méthodes autres que GET ; taille et forme des corps de requête.
  - Ce que les logs contiennent : aucun code ni jeton hors du `Mailer` de développement ; adresses e-mail dans les logs.
- [ ] **3. Revue : stockage et compartimentage** (`internal/storage`, `internal/admin`, `cmd/tribe-menus`).
  - Chemin du fichier d'une tribu : jamais construit à partir d'une valeur venue de la requête ; identifiant d'URL hors format.
  - Requêtes SQL : toutes paramétrées par sqlc ; aucune requête construite par concaténation.
  - Droits des fichiers et des dossiers de données créés par le programme, fichier de `-mail-file` compris.
  - Options réservées au développement (`-dev`, `-mail-file`) : ce qu'elles ouvrent si elles sont activées par erreur en production, et ce qui l'empêche.
  - Commandes d'administration : entrées non validées, états partiels après un échec.
- [ ] **4. Revue : serveur HTTP et front** (`internal/server/server.go`, `web/`).
  - En-têtes : CSP au regard de ce que le front charge réellement, `nosniff`, `Referrer-Policy`, `noindex` ; ce qui reviendra à Caddy (HSTS) est noté pour le plan `recette`.
  - Service des fichiers du front : traversée de chemin, fichiers servis par erreur (sourcemaps, sources), mise en cache.
  - Gabarit `index.html` : valeurs injectées (`Base`, adresse du code source) et leur échappement.
  - Front : aucune échappatoire au rendu échappé (`webcheck`), rien de sensible gardé côté client, nom de la tribu jamais affiché avant connexion, comportement sur un 401.
  - Délais et limites du serveur (`ReadHeaderTimeout` seul aujourd'hui) : lecture du corps, écriture, connexions inactives.
  - Dépendances : `make vuln`, licences, et ce que `go.mod` et `package.json` embarquent réellement dans le binaire.
- [ ] **5. Constats et arbitrage.**
  - Le relecteur ouvre une PR qui ajoute à ce plan la section « Constats » : « à corriger » et « à noter », numérotés, avec ce qu'il n'a pas pu vérifier ; chaque constat de sécurité porte ses trois évaluations (gravité, risque de livrer sans recette, risque d'exploitation une fois publié) et le niveau de traitement qu'il aurait en production.
  - Le développeur arbitre (retirer un point, le changer de catégorie, corriger une évaluation), puis fusionne la PR.
- [ ] **6. Corrections**, par Claude Code dans le Dev Container.
  - Les décisions de l'étape 1 qui touchent au code, et chaque point « à corriger » : un commit par point, test compris ; `make ci`. Une PR, relue selon `docs/revue-de-pr.md` par la session qui a fait la revue.
  - Les points « à noter » rejoignent `docs/feuille-de-route.md` (points reportés), avec le plan qui les reprendra.
- [ ] **7. Protections du dépôt**, par le développeur avec l'aide d'une session : parcourir `docs/securite-depot.md`, cocher ce qui est en place depuis le passage en public (CodeQL, détection de secrets et blocage des pushes, Dependabot, revue des dépendances, ruleset de `main`), et lire les alertes ouvertes. Les sessions Claude n'ont pas accès aux alertes d'analyse de code : c'est au développeur de les consulter.
- [ ] **8. Clôture** : statut « terminé », `docs/feuille-de-route.md` mis à jour, `CHANGELOG.md`.

## Critères de validation

- Chaque constat « à corriger » a un commit et un test qui échouait avant la correction.
- Chaque constat « à noter » a une destination écrite dans la feuille de route.
- Les décisions de l'étape 1 sont dans les specs, et les scénarios Gherkin concernés passent (`make acceptance`).
- `make ci` passe ; aucune alerte ouverte de CodeQL, de Dependabot ou de détection de secrets qui ne soit expliquée.
- La revue dit ce qu'elle n'a pas couvert.

## Questions ouvertes

- **Revue par une seule session, ou par deux sessions indépendantes** sur l'authentification (étape 2), la partie la plus exposée ? Deux revues coûtent plus cher, mais leurs constats se recoupent.
- **Outil d'analyse supplémentaire** (`gosec`, par exemple) : à n'ajouter que si CodeQL et `staticcheck` laissent un manque constaté, pour ne pas multiplier l'outillage (ADR 0009).
