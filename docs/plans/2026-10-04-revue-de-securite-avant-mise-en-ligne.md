# Plan : revue de sécurité avant mise en ligne

- **Nom** : `revue-securite`
- **Date** : 2026-10-04
- **Statut** : en cours (étapes 1 à 6 faites ; corrections à relire par la session de revue ; restent les protections du dépôt, étape 7, et la clôture)

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

- [x] **1. Décisions de specs**, avec le développeur, avant la revue.
  - **Force brute** (tableau ci-dessus). Pistes à comparer : accepter le risque et alerter l'administrateur sur les limites atteintes de façon répétée (l'alerte par e-mail arrive avec le déploiement) ; allonger le code (8 chiffres divisent le risque par 100) ; plafonner les demandes par adresse et par jour. Un plafond plus strict facilite le blocage ciblé d'un membre : les deux points se tranchent ensemble.
  - **Blocage ciblé d'une tribu** par la limite de 30 demandes par heure : l'accepter en V1, ou revoir ce que compte la limite par tribu, sans rien révéler de l'existence de la tribu ni de ses membres (ENF-02).
  - **Appareil détecté dans le journal d'audit** (D12 à confirmer) : chaîne d'affichage en anglais aujourd'hui (`audit_log.detected_device`), colonnes structurées dans `sessions` ; fixer la forme avant que des données réelles existent.
  - Résultat : ENF-01 et `gestion-membres-et-sessions.md` mis à jour, décisions consignées dans ce plan ; un ADR si le mécanisme d'authentification change (ADR 0001).
  - Fait le 2026-10-05 : D1 à D4. Pas d'ADR : le mécanisme ne change pas, seuls la longueur du code et ce que compte une limite changent ; l'ADR 0001 porte un renvoi. Scénarios Gherkin, code et front suivent à l'étape 6.
- [x] **2. Revue : authentification et sessions** (`internal/tribe`, `internal/server/api.go`, `internal/mail`).
  - Codes : tirage, empreinte, comparaison en temps constant, usage unique, remplacement, expiration, essais ; codes fantômes indiscernables des vrais, en réponse comme en durée.
  - Sessions : tirage et empreinte du jeton, expiration glissante, déconnexion, attributs et portée du cookie, cookie d'une tribu présenté à une autre.
  - Limitation : contournements (casse, espaces, variantes de l'adresse ou de l'identifiant d'URL), demandes concurrentes, taille de la base de limitation.
  - Protection contre les requêtes d'une autre origine sur les méthodes autres que GET ; taille et forme des corps de requête.
  - Ce que les logs contiennent : aucun code ni jeton hors du `Mailer` de développement ; adresses e-mail dans les logs.
- [x] **3. Revue : stockage et compartimentage** (`internal/storage`, `internal/admin`, `cmd/tribe-menus`).
  - Chemin du fichier d'une tribu : jamais construit à partir d'une valeur venue de la requête ; identifiant d'URL hors format.
  - Requêtes SQL : toutes paramétrées par sqlc ; aucune requête construite par concaténation.
  - Droits des fichiers et des dossiers de données créés par le programme, fichier de `-mail-file` compris.
  - Options réservées au développement (`-dev`, `-mail-file`) : ce qu'elles ouvrent si elles sont activées par erreur en production, et ce qui l'empêche.
  - Commandes d'administration : entrées non validées, états partiels après un échec.
- [x] **4. Revue : serveur HTTP et front** (`internal/server/server.go`, `web/`).
  - En-têtes : CSP au regard de ce que le front charge réellement, `nosniff`, `Referrer-Policy`, `noindex` ; ce qui reviendra à Caddy (HSTS) est noté pour le plan `recette`.
  - Service des fichiers du front : traversée de chemin, fichiers servis par erreur (sourcemaps, sources), mise en cache.
  - Gabarit `index.html` : valeurs injectées (`Base`, adresse du code source) et leur échappement.
  - Front : aucune échappatoire au rendu échappé (`webcheck`), rien de sensible gardé côté client, nom de la tribu jamais affiché avant connexion, comportement sur un 401.
  - Délais et limites du serveur (`ReadHeaderTimeout` seul aujourd'hui) : lecture du corps, écriture, connexions inactives.
  - Dépendances : `make vuln`, licences, et ce que `go.mod` et `package.json` embarquent réellement dans le binaire.
- [x] **5. Constats et arbitrage.**
  - Le relecteur ouvre une PR qui ajoute à ce plan la section « Constats » : « à corriger » et « à noter », numérotés, avec ce qu'il n'a pas pu vérifier ; chaque constat de sécurité porte ses trois évaluations (gravité, risque de livrer sans recette, risque d'exploitation une fois publié) et le niveau de traitement qu'il aurait en production.
  - Le développeur arbitre (retirer un point, le changer de catégorie, corriger une évaluation), puis fusionne la PR.
  - Section « Constats » ajoutée le 2026-10-05. Arbitrage du même jour : catégories et évaluations gardées telles quelles, point 1 tranché (D5), lecture de la grille confirmée pour le point 4 (D6).
- [x] **6. Corrections**, par Claude Code dans le Dev Container.
  - Les décisions qui touchent au code (D1 à D3 et D5, voir « Ce que les décisions demandent à l'étape 6 »), et chaque point « à corriger » : un commit par point, test compris ; `make ci`. Une PR, relue selon `docs/revue-de-pr.md` par la session qui a fait la revue.
  - Les points « à noter » rejoignent `docs/feuille-de-route.md` (points reportés), avec le plan qui les reprendra.
  - Fait le 2026-10-05 : six commits (D1 avec le constat 2, D2, D3, D5 avec le constat 1, constats 3 et 4), points 5 à 11 et suites des constats 1 et 4 dans la feuille de route. Choix faits en route dans les notes d'exécution.
- [ ] **7. Protections du dépôt**, par le développeur avec l'aide d'une session : parcourir `docs/securite-depot.md`, cocher ce qui est en place depuis le passage en public (CodeQL, détection de secrets et blocage des pushes, Dependabot, revue des dépendances, ruleset de `main`), et lire les alertes ouvertes. Les sessions Claude n'ont pas accès aux alertes d'analyse de code : c'est au développeur de les consulter.
- [ ] **8. Clôture** : statut « terminé », `docs/feuille-de-route.md` mis à jour, `CHANGELOG.md`.

## Critères de validation

- Chaque constat « à corriger » a un commit et un test qui échouait avant la correction.
- Chaque constat « à noter » a une destination écrite dans la feuille de route.
- Les décisions de l'étape 1 sont dans les specs, et les scénarios Gherkin concernés passent (`make acceptance`).
- `make ci` passe ; aucune alerte ouverte de CodeQL, de Dependabot ou de détection de secrets qui ne soit expliquée.
- La revue dit ce qu'elle n'a pas couvert.

## Questions ouvertes

- **Outil d'analyse supplémentaire** (`gosec`, par exemple) : à n'ajouter que si CodeQL et `staticcheck` laissent un manque constaté, pour ne pas multiplier l'outillage (ADR 0009).

## Décisions

Le 2026-10-05, avec le développeur (étape 1).

- **D1. Code à 8 chiffres, et alerte à l'administrateur.** Avec les limites d'ENF-01 inchangées, la force brute continue passe d'environ 1 sur 4 à 0,3 % par an pour une adresse, et de 1 sur 2 à 0,8 % pour une tribu. Les limites atteintes de façon répétée sont signalées à l'administrateur par e-mail ; l'alerte rejoint le plan `production`, qui apporte l'envoi réel. Écartés : un plafond par jour et par adresse (environ 1 % par an, mais un attaquant qui l'épuise empêche le membre visé de se connecter une journée entière) ; accepter le risque avec la seule alerte (le risque reste celui du tableau tant que personne ne réagit).
- **D2. La limite par tribu ne compte que les demandes adressées à ses membres actifs.** Des adresses au hasard ne bloquent plus la tribu : il faut connaître des adresses de membres. Le quota d'envoi reste protégé, puisque seuls les membres actifs reçoivent un e-mail. Limite assumée, écrite dans ENF-01 : la limite d'une tribu atteinte, une adresse de membre reçoit « Réessayez dans quelques minutes » et une autre adresse non, ce qui ne se produit qu'après 30 demandes pour des membres dans l'heure. **Contrainte d'implémentation** : la base de limitation ne doit pas apprendre quelles empreintes d'adresse sont celles de membres, ce que l'ADR 0021 évite aujourd'hui ; où tenir ce décompte (base de la tribu, par exemple) se choisit à l'étape 6, et l'ADR 0021 est précisé en conséquence. Écartés : accepter le blocage en V1 ; supprimer la limite par tribu (la force brute répartie sur plusieurs IP et plusieurs membres ne serait plus plafonnée).
- **D3. Appareil détecté enregistré sous forme structurée dans le journal d'audit.** `audit_log` reçoit les colonnes de `sessions` (type, système, navigateur, app installée) à la place de la chaîne `detected_device` ; le libellé, en français, est composé à l'affichage par l'interface d'EF-07. Confirme `socle`, D12, sous cette forme. Rien n'est déployé : la forme de la migration (modifier la 0001 ou en ajouter une) se choisit à l'étape 6. L'iPad vu comme un ordinateur reste à EF-04. Écartés : une chaîne en français (le serveur fixerait le libellé et le journal resterait figé s'il change) ; garder la forme actuelle.
- **D4. Une seule session cloud relit les étapes 2 à 4**, puis relit les corrections de l'étape 6. Le code est petit et rien n'est en production. Tranche la question ouverte « une ou deux sessions ».

Le 2026-10-05, avec le développeur, à l'arbitrage des constats (étape 5).

- **D5. Le code ne se saisit que depuis le navigateur qui l'a demandé** (constat 1). `POST login-codes` pose un cookie aléatoire, `HttpOnly`, `Secure`, `SameSite=Lax`, au `Path` de la tribu, valable 10 minutes ; son empreinte est gardée avec le code, réel comme fantôme. Un essai sans ce cookie reçoit `new_code_needed` et ne consomme rien. Pour bloquer un membre, il faut alors redemander un code à sa place : c'est limité, visible de la victime, et vu par l'alerte de D1. ENF-01, l'ADR 0001 (renvoi) et le texte de la page Confidentialité (`docs/design/README.md` : deux cookies au lieu d'un) sont mis à jour ; pas de nouvel ADR, le mécanisme reste un code par e-mail et une session par cookie. Limite assumée : un code demandé sur un appareil ne se saisit pas sur un autre. Écartée : une limite des essais par adresse IP, que quelques adresses suffisent à contourner.
- **D6. La gravité se juge comme la grille l'écrit, sur ce qu'il faut à l'attaquant** (constat 4). « Accès au serveur requis » reste une gravité faible, donc un traitement normal, même quand la conséquence est la session d'un membre : l'attaquant a déjà un compte sur le serveur, et un constat écrit en public ne l'aide pas à en obtenir un. `docs/traitement-des-vulnerabilites.md` n'est pas modifié.

### Ce que les décisions demandent à l'étape 6

Un commit par décision, les scénarios écrits ou modifiés d'abord (`docs/specs/conventions-gherkin.md`) :

- D1 : `authentification.feature` (« un code à 8 chiffres »), `internal/tribe/login.go` (`loginCodeDigits`), écrans et tests du front (champ en huit cases, textes) ;
- D2 : l'exemple « 30 demandes de code pour la tribu » de `authentification.feature` précisé, et un scénario « des demandes pour des adresses qui ne sont pas membres ne bloquent pas la tribu » ; décompte et ADR 0021 ;
- D3 : migration, requêtes sqlc, `sessionAudit` et `Device.String` dans `internal/tribe` ;
- D5 : `authentification.feature` (un scénario « des essais venus d'un autre navigateur ne consomment pas ceux du code », et le cas d'une adresse non membre) ; colonne d'empreinte dans `login_codes` et `decoy_codes`, requêtes sqlc ; `requestCode` et `openSession` dans `internal/server/api.go`, `Login.RequestCode` et `Login.OpenSession` ; nom du cookie à choisir, distinct de `session` ; ADR 0021 précisé (ce que garde un code fantôme) ; maquette `identite/Confidentialite` accordée au texte ; harnais godog et tests Playwright, qui reportent déjà les cookies.

## Constats

Revue des étapes 2 à 4, faite le 2026-10-05 par une session cloud qui n'a pas écrit le code, sur `main` au commit `4bc8c86`. Aucune version n'est publiée ni en production (aucune étiquette `v*`) : tout suit le flux public.

**Avis d'ensemble.** Le socle tient ce qu'il annonce : réponses identiques pour un membre, une adresse inconnue et une tribu inexistante, limites tenues sous des demandes simultanées, aucune requête SQL construite à la main, fichiers du front servis sans traversée de chemin, rien de sensible dans les logs hors `-dev`. Quatre points sont à corriger avant le déploiement, dont un demandait d'abord une décision de specs (point 1, tranché par D5) ; sept sont à noter pour les plans qui suivent.

Chaque constat de sécurité porte les trois évaluations de `docs/traitement-des-vulnerabilites.md`, dans l'ordre **gravité / risque de livrer sans recette / risque d'exploitation une fois publié**, puis le niveau qu'il aurait si le code était en production.

Déjà connus et non repris ici : adresse IP du client derrière Caddy et IPv6 par préfixe /64 (plan `production`), fichier de base orphelin (EF-10), décisions D1 à D3 (étape 6).

### À corriger

**1. Les essais d'un code se consomment depuis n'importe quel client, sans limite** (`internal/tribe/store_login.go`, `Store.OpenSession` ; `internal/server/api.go`, `openSession`)

`POST sessions` ne connaît ni l'adresse IP ni le navigateur qui a demandé le code, et n'est soumis à aucune limite : seuls comptent les 3 essais du code.

- Scénario : qui connaît l'identifiant d'URL d'une tribu et l'adresse d'un membre envoie en boucle `POST sessions` avec un code faux ; dès que le membre demande un code, ses 3 essais sont consommés avant qu'il l'ait saisi.
- Conséquence : le membre ne peut plus se connecter tant que la boucle tourne. Il ne reçoit aucun e-mail de plus, aucune limite d'ENF-01 n'est atteinte, et l'alerte de D1 ne voit rien : c'est le blocage ciblé que D1 et D2 ont voulu éviter, en silencieux.
- Constaté par l'exécution : après une demande de code pour `alice@exemple.fr`, trois `POST sessions` erronés sans cookie ni demande préalable répondent `incorrect_code` (2, 1, 0), puis le bon code reçoit `new_code_needed`. 300 `POST sessions` de suite depuis la même adresse IP reçoivent tous 400, jamais 429.
- Effet de bord : le nombre d'appels étant illimité, c'est aussi la route qui donnerait le plus d'échantillons à une mesure de temps de réponse (voir « Ce qui n'a pas pu être vérifié »).

Évaluations : moyenne (blocage d'un membre à distance et sans compte, aucune donnée exposée) / moyen (authentification touchée, couverte par les scénarios ; une colonne ajoutée à deux tables) / élevé (se déduit d'ENF-01, sans même lire un correctif). Niveau en production : **accéléré**, sans autre mesure d'attente que l'arrêt.

Attendu : une décision dans ENF-01, puis le scénario Gherkin et le code. **Tranché le 2026-10-05 : la première des deux pistes (D5).**

- **lier le code au navigateur qui l'a demandé** : `POST login-codes` pose un cookie aléatoire (`HttpOnly`, même `Path` que la session, 10 minutes), dont l'empreinte est gardée avec le code, réel comme fantôme ; un essai sans ce cookie reçoit `new_code_needed` et ne consomme rien. Pour bloquer un membre, il faut alors redemander un code à sa place, ce qui est limité, visible de la victime et vu par l'alerte de D1. L'écran du code ne s'atteint déjà qu'après une demande faite depuis la même page ;
- **limiter les essais par adresse IP** dans la base de limitation : plus simple, mais quelques adresses IP suffisent à la contourner.

Test : après trois essais erronés venus d'un autre client, le code du membre ouvre encore sa session. Dans les deux cas, l'alerte de D1 (plan `production`) gagnerait à compter aussi les codes invalidés par essais épuisés.

**2. Le tirage du code ne suit pas `loginCodeDigits`** (`internal/tribe/login.go`, `newLoginCode`)

Le code est tiré dans `big.NewInt(1_000_000)`, constante écrite à part de `loginCodeDigits`, qui ne sert qu'au remplissage par des zéros. « Ce que les décisions demandent à l'étape 6 » ne cite que `loginCodeDigits` pour D1.

- Scénario : D1 est implémenté en passant `loginCodeDigits` à 8 ; les codes ont bien 8 chiffres, mais commencent tous par `00`.
- Conséquence : la force brute reste celle du tableau de ce plan (environ 1 chance sur 4 par an pour une adresse) alors qu'ENF-01 annonce 0,3 %, et un test qui ne vérifie que la longueur passe.
- Constaté par lecture. Sans objet tant que le code a 6 chiffres.

Évaluations, si D1 était livré ainsi : moyenne (force brute facilitée, mais toujours longue) / faible (une ligne et un test) / élevé (lisible dans `login.go`). Niveau en production : **accéléré**.

Attendu, dans le commit de D1 : la borne du tirage calculée à partir de `loginCodeDigits`, et un test qui échoue si les codes tirés restent tous sous 1 000 000.

**3. Le serveur HTTP n'a qu'un délai de lecture des en-têtes** (`cmd/tribe-menus/main.go`, `listenAndServe`)

`http.Server` ne fixe que `ReadHeaderTimeout` : ni `ReadTimeout`, ni `WriteTimeout`, ni `IdleTimeout`.

- Scénario : un client ouvre des connexions, envoie les en-têtes d'un `POST` avec un `Content-Length`, puis rien.
- Conséquence : chaque connexion garde une goroutine et un descripteur de fichier sans limite de durée, jusqu'à épuiser ceux du processus. Une seule suffit à faire échouer l'arrêt du serveur, ce qui pèsera sur le déploiement et le retour arrière (ADR 0016).
- Constaté par l'exécution : après 75 secondes, une connexion au corps jamais envoyé et une connexion inactive après une requête servie sont toujours ouvertes ; celle dont les en-têtes sont incomplets est fermée à 10 secondes. Avec une connexion au corps jamais envoyé, `SIGTERM` arrête `serve` au bout de 10 secondes, en erreur (`shutdown: context deadline exceeded`, code de sortie 1). La part que Caddy absorbera n'a pas pu être vérifiée (configuration à écrire, plan `recette`).

Évaluations : faible (déni de service, sans donnée) / faible (quelques lignes, un test) / moyen (attaque classique, sans rien à lire). Niveau en production : **normal**.

Attendu : `ReadTimeout`, `WriteTimeout` et `IdleTimeout` sur `http.Server`, et un test où une connexion au corps incomplet est fermée au délai.

**4. Les fichiers de base sont créés lisibles par tous** (`internal/storage/storage.go`, `Open` et `openDB`)

SQLite crée ses fichiers en 0644 (moins l'umask), `-wal` et `-shm` compris. Seuls les dossiers les protègent : `Open` crée `<data>` et `<data>/tribes` en 0700, mais ne touche pas à un dossier qui existe déjà.

- Scénario : le dossier de données est préparé par le provisionnement en 0755 (défaut de `mkdir` comme de `StateDirectory=`), et un autre compte du serveur, par exemple celui d'une autre application (ADR 0006), lit les fichiers.
- Conséquence : `registry.db` (identifiants et noms des tribus) et `ratelimit.db` (empreintes d'IP, qui se renversent en énumérant les adresses IPv4) sont lisibles ; les bases des tribus aussi si `tribes/` préexiste. Qui lit la base d'une tribu y trouve les adresses des membres, demande un code pour l'une d'elles, lit `login_codes.code_hash`, retrouve le code et ouvre une session.
- Constaté par l'exécution, avec le SQLite de Python et non `modernc.org/sqlite` (voir plus bas) : `registry.db`, `ratelimit.db` et la base d'une tribu en `-rw-r--r--` ; dans un dossier préexistant en 0755, les deux premiers sont lisibles par tous. Un code à 6 chiffres est retrouvé depuis son empreinte en moins d'une seconde.

Évaluations : faible (accès au serveur requis, selon la grille ; la conséquence, elle, est la session d'un membre) / faible / faible. Niveau en production : **normal**.

Attendu : des fichiers en 0600 quel que soit le dossier, pour `serve` comme pour `admin` (umask 0077 au démarrage du programme, par exemple), et un test sur le mode des fichiers créés par `Open` et `CreateTribe` dans un dossier préexistant en 0755. Pour le plan `recette` : dossier de données en 0700 et `UMask=0077` dans l'unité systemd.

### À noter

**5. Aucun moyen de fermer une session à distance ni de révoquer un membre** (`internal/tribe/store_login.go`, `Store.RevokeMember`, appelé par les seuls tests ; `cmd/tribe-menus/main.go`, `runAdmin`). Un téléphone perdu garde une session de 90 jours, prolongée à chaque usage ; le seul recours est de modifier la base à la main sur le serveur, ce que l'ADR 0015 (point 13) exclut. Évaluations : moyenne / élevé (une fonction à écrire, pas un correctif) / faible ; niveau : accéléré, avec pour mesure d'attente la suppression de la ligne de `sessions`. À reprendre dans la feuille de route : EF-03 et EF-05, ou à défaut une commande d'administration, avant « Ouverture de la production ».

**6. Le cookie de session n'a pas le préfixe `__Host-`, et la recette est un sous-domaine de la production** (`internal/server/api.go`, `setSessionCookie`). Une page servie par `recette.meltingtribe.codingmatters.org` (archives de PR) ou par un autre hôte de `codingmatters.org` peut poser un cookie `session` avec `Domain=meltingtribe.codingmatters.org` : le navigateur de la victime présente alors en production la session de l'attaquant, ou perd la sienne. Le cookie de production, lui, n'est jamais envoyé à la recette. Évaluations : faible / moyen (renommer le cookie déconnecte tout le monde) / faible ; niveau : normal. À trancher dans le plan `recette`, avant la première session réelle : cookie `__Host-` nommé par tribu avec `Path=/` (option laissée ouverte par l'ADR 0001), ou nom d'hôte de recette hors de celui de la production.

**7. `ParseEmail` accepte des adresses que l'envoi SMTP devra refuser** (`internal/tribe/tribe.go`). Constaté par l'exécution : `a,b@c.d`, `a@c.d,e.f`, `<a>@c.d` et une adresse contenant un octet nul sont acceptées ; un retour à la ligne ne passe pas (les espaces sont retirés). Aujourd'hui l'adresse ne va que dans les logs et le fichier de `-mail-file`. Évaluations : faible / faible / faible ; niveau : normal. Plan `production`, avec l'envoi SMTP : en-tête `To` et enveloppe construits par `net/mail`, et règle de `ParseEmail` resserrée.

**8. Rien ne refuse `-dev` hors du développement** (`cmd/tribe-menus/main.go`, `migrateAndServe`). Avec `-dev`, chaque code part dans les logs avec l'adresse du membre, que journald garde un mois (ADR 0015). L'erreur se voit aujourd'hui, parce que les pages répondent 503 faute de `web/dist` sur le disque, mais l'API fonctionne. Évaluations : faible (erreur d'administration et accès aux logs requis) / faible / faible ; niveau : normal. Plan `production` : refuser `-dev` avec l'écoute sur le socket de systemd ou avec une configuration SMTP.

**9. La tribu de démonstration ne doit pas arriver telle quelle sur une instance joignable** (`internal/admin/seed.go`, `demoMembers`). Ses membres sont en `@exemple.fr`, domaine que le projet ne contrôle pas et que la RFC 2606 ne réserve pas : avec l'envoi réel, leurs codes partiraient vers ce domaine, et qui en reçoit le courrier entrerait dans la tribu `demo`. Évaluations : faible (données de démonstration) / faible / faible ; niveau : normal. Plan `recette` : des adresses du développeur, ou un domaine réservé, pour la tribu de recette.

**10. `webcheck` ne connaît pas toutes les échappatoires au rendu échappé** (`internal/tools/webcheck/main.go`, `forbidden`). `unsafeStatic` (`lit/static-html.js`), `unsafeMathML`, `setHTMLUnsafe`, `srcdoc` et `createContextualFragment` passeraient. Aucun n'est utilisé dans `web/src`, et la CSP en limiterait l'effet. Évaluations : faible / faible / faible ; niveau : normal. À compléter avec le premier écran qui affiche un texte saisi par un membre (EF-01, EF-02 ou la bibliothèque de plats).

**11. Ce qui revient à Caddy**, pour le plan `recette` ; ce n'est pas un défaut du code :

- HSTS ;
- en-têtes du site public : `withCommonHeaders` ne couvre que ce que sert le binaire, et le site est servi par Caddy en production ;
- en-tête `Host` transmis tel quel : la protection contre les requêtes d'une autre origine compare `Origin` à `Host` ;
- `/healthz` non transmis ;
- délais et taille des requêtes, en complément du point 3 ;
- journaux d'accès : ils contiennent les adresses IP et les identifiants d'URL des tribus, à accorder avec la page Confidentialité.

### Vérifié, sans constat

- **Codes et sessions** (lecture) : tirage par `crypto/rand`, empreinte SHA-256, comparaison en temps constant, usage unique et remplacement dans la transaction qui ouvre la session, expiration, code fantôme sans empreinte que rien n'égale ; jeton de 32 octets, empreinte seule en base, expiration glissante, session supprimée à la déconnexion et à la révocation du membre.
- **Rien n'est révélé** (exécution) : pour un membre actif, une adresse inconnue, une tribu inexistante et un identifiant hors format, `POST login-codes`, `POST sessions` et la page de connexion renvoient les mêmes en-têtes et le même corps ; `GET session` ne diffère que par le `Path` du cookie effacé.
- **Limitation** (exécution) : 20 demandes simultanées pour une adresse, 3 acceptées ; casse, espaces et tabulation dans l'adresse, identifiant d'URL encodé (`%64emo`) : même compteur ; 30 demandes simultanées pour des adresses distinctes : la limite par IP tient ; un identifiant hors format n'est gardé qu'en empreinte ; un corps de 5 Ko est refusé.
- **Cookie** (exécution et lecture) : `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/tribes/<identifiant>/` ; l'identifiant de l'URL y passe par `url.PathEscape`, sans injection d'attribut possible ; le jeton d'une session de `demo`, présenté à une autre tribu dont la même adresse est membre, reçoit 401 ; un code rejoué reçoit `new_code_needed`, un jeton réutilisé après la déconnexion, 401.
- **SQL et chemins** (lecture) : requêtes sqlc paramétrées, seul `PRAGMA user_version` est formaté, avec un entier ; le fichier d'une tribu porte un nom aléatoire lu au registre, jamais une valeur de la requête.
- **Options de développement** (lecture) : `-mail-file` refusée sans `-dev`, fichier créé en 0600 ; hors `-dev`, aucun code n'est envoyé ni journalisé.
- **Commandes d'administration** (lecture) : identifiant et adresse validés, échec en cours de création sans fichier ni entrée de registre restants ; nom de la tribu et nom d'affichage libres, affichés par Lit avec échappement.
- **Front servi** (exécution) : `..%2f`, `%2e%2e`, octet nul et `index.html` demandé directement ne sortent pas de `web/dist` ni ne livrent le gabarit brut ; `Base` est échappé pour un identifiant contenant `">`, `javascript:` ou un retour à la ligne encodé ; la redirection de `/tribes/<identifiant>` reste sous `/tribes/`.
- **Front** (lecture) : ni `style` en ligne ni shadow DOM stylé, donc rien que la CSP bloque ; aucun stockage côté client ; nom de la tribu lu dans la seule réponse de session ; un 401 ramène à la connexion.
- **Logs** (exécution et lecture) : hors du `Mailer` de développement, sept appels au journal, dont aucun ne porte d'adresse, de code ni de jeton ; aucune requête n'est journalisée.
- **Dépendances** : `make ci`, `make vuln` compris, est passé sur `main` au commit relu (CI du 2026-10-05). D'après les imports, le binaire n'embarque hors bibliothèque standard que `modernc.org/sqlite` et ses dépendances ; le front, `lit` et ses cinq dépendances (BSD-3-Clause, MIT), d'après `package-lock.json`.

### Ce qui n'a pas pu être vérifié

- **`make ci` n'a pas tourné dans la session** : `proxy.golang.org`, le registre npm et le stockage des artefacts et des journaux de la CI y sont inaccessibles. La revue s'appuie sur la CI de `main`.
- **Les exécutions viennent d'une copie de travail jetable**, compilée avec Go 1.24.7, où trois pièces sont remplacées : `modernc.org/sqlite` par le SQLite de Python derrière un pilote d'essai, `http.CrossOriginProtection` par un intergiciel neutre, `sync.WaitGroup.Go` par son équivalent. Restent donc vérifiés par la seule lecture : le mode des fichiers créés par `modernc.org/sqlite` (point 4 ; 0644 est le défaut documenté de SQLite) et le refus des requêtes d'une autre origine (couvert par `TestCrossOriginRefused` en CI).
- **Temps de réponse** : aucune mesure entre un membre et une autre adresse, ni entre une tribu existante et une autre. À la lecture, les chemins diffèrent d'une requête `SELECT` et de la base qui reçoit l'écriture (celle de la tribu ou la base de limitation). À mesurer en recette, sur `POST sessions` d'abord (point 1).
- **Front construit** : `web/dist` n'a pas pu être produit ; la CSP est confrontée aux sources, pas au paquet d'esbuild ni à sa sourcemap. Playwright n'a pas tourné.
- **Contenu exact du binaire** (`go version -m`) et licences des dépendances de `modernc.org/sqlite`, une à une.
- **Derrière Caddy et le socket Unix**, sous Safari, et l'état des alertes CodeQL, Dependabot et de détection de secrets (étape 7).
- **`exemple.fr`** : son enregistrement n'a pas été vérifié (point 9).

### Retour sur la grille

Premier rodage de `docs/traitement-des-vulnerabilites.md` :

- **« Accès au serveur requis » classe le point 4 en gravité faible**, donc en traitement normal, alors que sa conséquence est la session de n'importe quel membre. *Confirmé par le développeur le 2026-10-05 : la grille reste telle quelle (D6).*
- **Une faille lisible dans les specs** (point 1) est exposée avant tout correctif : le traitement accéléré ne raccourcit que ce qui suit le constat.
- **Une fonction manquante** (point 5) se range mal sur l'axe 2, pensé pour un correctif.

## Notes d'exécution

- **Étape 1** (2026-10-05) : décisions D1 à D4 prises avec le développeur sur les recommandations proposées ; ENF-01, EF-07 (`gestion-membres-et-sessions.md`), `docs/design/README.md` (texte de l'écran du code) et l'ADR 0001 (renvoi) mis à jour. L'alerte rejoint le plan `production` dans la feuille de route.
- **Étapes 2 à 4** (2026-10-05) : revue faite par une session cloud, en lecture et par l'exécution d'une copie de travail jetable ; rien n'a été corrigé. Résultat dans « Constats », conditions de vérification dans « Ce qui n'a pas pu être vérifié ».
- **Étape 5** (2026-10-05) : arbitrage dans la conversation avec la session de revue, consigné dans la même PR que les constats ; décisions D5 et D6 ; ENF-01, ADR 0001 et `docs/design/README.md` mis à jour pour D5.
- **Étape 6** (2026-10-05), choix faits en route :
  - **Migrations ajoutées**, pas de modification des 0001 (D3 laissait le choix) : rien n'est déployé, mais les bases de développement existantes migrent sans être recréées. Tribu : 0003 (demandes pour les membres), 0004 (appareil du journal d'audit ; l'ancien texte n'est pas converti), 0005 (`login_codes` recréée avec l'empreinte du jeton de la demande, codes en attente abandonnés). Limitation : 0002 (plus d'identifiant d'URL), 0003 (`decoy_codes` recréée de même).
  - **D2, où compter** : dans la base de la tribu, table `code_requests` réduite à l'heure de la demande. Une demande est d'abord comptée par la base de limitation, comme toute autre, puis, pour un membre actif, par la base de la tribu, dans la transaction qui remplace le code (`Store.IssueLoginCode`). Ce que la base de limitation enregistre ne dépend donc pas de l'appartenance, même quand la limite de la tribu refuse ; contrepartie, une demande refusée par la limite de la tribu compte pour les limites par adresse et par IP, comme une demande pour une autre adresse. La base de limitation ne garde plus l'identifiant d'URL, devenu inutile hors de l'empreinte d'adresse. ADR 0021 précisé (points 2 à 4).
  - **D5** : cookie `code_request`, 32 octets aléatoires, effacé à l'ouverture de la session. Le jeton est tiré pour toute demande acceptée, membre ou non. Un jeton absent ou d'une autre demande reçoit `new_code_needed` sans consommer d'essai ; un code expiré est supprimé quel que soit le jeton. Rien à changer dans le front, le navigateur renvoie le cookie.
  - **D1, front** : à huit cases, les chiffres dérivaient d'une case à l'autre : `1ch`, largeur d'un zéro proportionnel, dépasse celle d'un chiffre tabulaire (17 px contre 16 sous Chromium). L'écran de connexion mesure la largeur d'un chiffre et la passe au CSS (`--code-digit`), `1ch` restant le repli. Maquettes de connexion et `docs/design/README.md` accordés.
  - **Constat 3** : lecture 20 s, écriture 30 s, inactivité 60 s, en-têtes 10 s comme avant.
  - **Constat 4** : le fichier de base est créé en 0600 avant que SQLite l'ouvre, et SQLite donne ce mode à `-wal` et `-shm` ; un fichier plus ouvert est ramené à 0600. Pas d'umask du processus : la correction tient sans elle, et l'unité systemd posera `UMask=0077` (plan `recette`).
