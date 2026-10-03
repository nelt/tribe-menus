# Plan : socle de données et connexion (première tranche verticale)

- **Date** : 2026-10-03
- **Statut** : en cours (lot A fusionné ; décisions D1 à D7 prises le 2026-10-03, D8 à D13 ajoutées le même jour à la relecture du lot B)

## Objectif

Première tranche qui traverse tout le système : une tribu créée par la commande d'administration, un membre qui ouvre l'URL de sa tribu, reçoit un code, se connecte et voit le nom de sa tribu, sur des bases SQLite compartimentées, avec les scénarios Gherkin exécutés par godog dans `make ci`.

À la fin du plan :

- `make seed`, `make generate` et `make acceptance` existent et ont un contenu (ADR 0010) ;
- `make dev` permet de se connecter à la tribu de démonstration, le code étant lu dans les logs (ADR 0009) ;
- 25 scénarios passent : 16 de `authentification.feature`, 5 de `compartimentage-tribus.feature`, 4 de `administration.feature` (détail plus bas).

Ce plan est exécuté par **Claude Code dans le Dev Container**, en trois PR successives (lots A à C), chacune fusionnable seule (D1).

**Hors périmètre**, renvoyé à un plan ultérieur :

- **Membres et sessions dans l'interface** (EF-01 à EF-07, sauf l'ouverture de session et la déconnexion) : plan suivant. Le scénario EF-08 « Le premier membre peut se connecter et ajouter des membres » attend donc EF-01.
- **Administration** : EF-09, EF-10, EF-11.
- **Hors-ligne et PWA installable** : service worker, IndexedDB, manifeste par tribu, icônes ; les 5 scénarios hors-ligne d'ENF-01 et les 2 scénarios `@manuel`.
- **Compartimentage des données métier** : les 7 scénarios d'ENF-02 qui supposent des plats, un planning, des listes ou la liste des membres ; chacun arrive avec sa story.
- **Déploiement** : envoi SMTP réel (ADR 0014), fichier de configuration et credentials systemd, activation de socket, lecture de `X-Forwarded-For` (ADR 0006, point 7), `deploy/`, `release.yml`.

## Contexte et contraintes

- À lire avant de commencer : ADR 0001, 0002, 0003, 0004, 0005, 0006, 0008, 0010, 0014 ; `docs/specs/exigences-non-fonctionnelles.md`, `gestion-membres-et-sessions.md`, `glossaire.md`, `conventions-gherkin.md` ; `docs/design/README.md` (« Connexion et chargement ») et les maquettes `identite/Connexion`, `ConnexionBureau`, `Code`, `Chargement`.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, avec `git commit -s` ; une ligne dans `CHANGELOG.md` par PR ; `make ci` avant de pousser.
- **Dépendances ajoutées** (ADR 0002, point 5) : `modernc.org/sqlite` à l'exécution (ADR 0003) ; `github.com/cucumber/godog` pour les tests (ADR 0005). Aucune autre.
- **Vocabulaire** : le code suit `glossaire.md` à la lettre (`tribe`, `slug`, `registry`, `member`, `login code`, `attempt`, `rate limit`, `session`, `audit log`…). Tout terme nouveau y est ajouté d'abord.
- **Rien n'est révélé avant connexion** (ENF-02, ADR 0006, point 9) : c'est la contrainte qui structure le lot B. Pour une tribu qui n'existe pas, une adresse inconnue, révoquée ou d'une autre tribu, les réponses de l'API (corps, code HTTP, essais restants, limitation) sont identiques à celles d'un membre actif.
- **Rien d'essentiel dans la mémoire du processus** (D3) : compteurs de limitation et codes fantômes vivent dans une base SQLite dédiée, derrière une interface, pour garder ouverte la porte du multi-instance.
- **Horloge injectée** : validité de 10 minutes, fenêtres de limitation et expiration à 90 jours se testent avec une horloge fournie par le test, jamais avec `time.Sleep`.
- **Mode sans proxy** (ADR 0006, point 8) : seul mode de ce plan. L'adresse IP du client est celle de la connexion TCP, lue en un seul endroit du serveur, pour que la lecture de `X-Forwarded-For` s'y ajoute plus tard.

### Scénarios couverts

| Fichier | Scénarios | Lot |
| --- | --- | --- |
| `administration.feature` | EF-08 : initialiser une tribu ; l'identifiant d'URL doit être libre ; plusieurs tribus ; le premier membre appartient déjà à une autre tribu | A |
| `authentification.feature` | ENF-01 : les 13 scénarios de la demande et de la saisie du code (de « Se connecter avec un code reçu par e-mail » à « Un code ne sert qu'une fois ») ; expiration glissante ; session inutilisée ; attributs du cookie | B |
| `compartimentage-tribus.feature` | ENF-02 : la session d'une tribu ne donne pas accès à une autre ; sessions distinctes pour une même adresse ; l'écran de connexion ne révèle pas la tribu ; le nom de la tribu apparaît une fois connecté ; le journal d'audit est propre à la tribu | B |

## Étapes

### Lot A : stockage, harnais d'acceptation et initialisation d'une tribu (`feature/stockage-et-ef-08`)

- [x] **1. `internal/storage` : ouverture et migrations.**
  - Dépendance `modernc.org/sqlite`. À chaque connexion : WAL, `foreign_keys = ON`, `busy_timeout` ; tables `STRICT` (ADR 0003, point 5).
  - Migrations en fichiers SQL embarqués, une série par sorte de base (`migrations/registry/`, `migrations/tribe/`), version suivie par `PRAGMA user_version`, chaque fichier appliqué dans une transaction.
  - Registre : identifiant d'URL, nom, fichier de la base. Le nom du fichier est un identifiant aléatoire, jamais dérivé de l'identifiant d'URL.
  - Un type qui migre le registre puis toutes les bases au démarrage, ouvre la base d'une tribu par son identifiant d'URL et fait passer ses écritures par une seule connexion.
  - Tests en tableaux sur `t.TempDir()` : migration rejouée sans effet, migration en échec qui remonte l'erreur et laisse la version inchangée, identifiant inconnu.
- [x] **2. sqlc.**
  - `sqlc.yaml` avec un paquet généré par sorte de base ; code généré versionné.
  - Cible `make generate` ; `go tool sqlc diff` dans `make lint`, pour refuser un code généré périmé.
- [x] **3. Serveur : répertoire de données et `/healthz`.**
  - Option `-data` de `serve` (par défaut `data/`, déjà ignoré par Git) ; migrations appliquées avant d'écouter ; échec de migration : le serveur ne démarre pas (PT-12).
  - `GET /healthz` à la racine, qui ne répond qu'une fois les bases migrées (ADR 0016, point 1.6).
- [x] **4. `acceptance/` : harnais godog.**
  - Dépendance `github.com/cucumber/godog`. Lecture directe de `docs/specs/features/` (`# language: fr`), serveur dans le processus (`httptest`), bases dans un dossier temporaire par scénario, horloge et `Mailer` de test.
  - Scénarios `@manuel` et `@ui` exclus (ADR 0005).
  - `acceptance/pending.txt` (D2) : un scénario attendu comme non implémenté par ligne (fichier et titre). L'exécution échoue si un scénario hors liste échoue ou a une étape non définie, et si un scénario de la liste passe. La comparaison est une fonction pure, testée en tableau de cas.
  - Cible `make acceptance`, ajoutée à `make ci` entre `test` et `e2e`.
- [x] **5. Format de l'identifiant d'URL dans les specs** (D4).
  - `gestion-membres-et-sessions.md` (EF-08) : 3 à 40 caractères ; lettres minuscules sans accent, chiffres et tirets ; une lettre en premier, pas de tiret final ni de tirets consécutifs ; une saisie non conforme est refusée, pas convertie ; aucun mot réservé (ADR 0006) ; non modifiable en V1.
  - `administration.feature` : un `Plan du scénario` d'identifiants refusés (`Martin`, `les_durand`, `é-nous`, `42`, `-martin`, `ab`), selon `conventions-gherkin.md`.
- [x] **6. Schéma de la tribu, première migration** : `members` (adresse normalisée unique, nom d'affichage, statut, dates et auteurs d'ajout et de révocation), `audit_log` (EF-07 : date, opération, membre concerné, auteur, session et appareil détecté le cas échéant), nom de la tribu.
- [x] **7. `internal/tribe` : membres et audit.**
  - Logique pure : normalisation de l'adresse (sans espaces, en minuscules), validation de l'identifiant d'URL, opérations d'audit nommées selon le glossaire.
  - Accès aux données par sqlc, en adaptateur mince.
- [x] **8. Sous-commande `tribe-menus admin init`** (EF-08).
  - Questions sur l'entrée standard : nom, identifiant d'URL (redemandé s'il est invalide ou pris), adresse du premier membre, nom d'affichage facultatif.
  - Crée la base, la migre, inscrit la tribu au registre, ajoute le premier membre, trace « initialisation de la tribu » avec « script d'administration » comme auteur, affiche l'URL de la tribu (option `-base-url`, par défaut `http://localhost:8080`). Aucun e-mail.
  - En cas d'échec en cours de route, ni fichier ni entrée de registre ne restent.
- [x] **9. Scénarios EF-08** (4, plus celui de l'étape 5) : définitions d'étapes godog qui pilotent la sous-commande avec une entrée et une sortie simulées ; lignes retirées de `pending.txt`.
- [x] **10. `make seed`** : sous-commande `admin seed`, qui crée la tribu de démonstration `demo` (« Les Démo ») avec les membres des conventions Gherkin (Alice, Bruno, Chloé, David) ; sans effet si elle existe déjà. Elle s'enrichira avec les plats et le planning.
- [x] **11. Documentation du lot** : `CLAUDE.md` (cibles `generate`, `acceptance` et `seed` disponibles), `CHANGELOG.md`, cases cochées.

### Lot B : connexion et session, côté API (`feature/enf-01-connexion-et-session`)

- [x] **12. ADR 0021 : base de limitation des demandes** (D3). Troisième sorte de base, à côté du registre et des bases de tribu ; précise les ADR 0001 et 0003. Il consigne aussi la forme des empreintes (D10) et la source de vérité du nom de la tribu (D9). Terme « code fantôme » ajouté au glossaire avant d'apparaître dans le code.
- [x] **13. Schémas.**
  - Base de la tribu : `login_codes` (membre, empreinte du code, échéance, essais restants ; un seul code valable par membre) et `sessions` (empreinte SHA-256 du jeton, membre, dates d'ouverture, de dernière activité et d'expiration, appareil détecté, app installée ou onglet, nom de session).
  - Base de limitation (`migrations/ratelimit/`, migrée au démarrage comme les autres) : demandes de code horodatées et codes fantômes (échéance, essais restants). Adresses et IP stockées en empreinte SHA-256, jamais en clair ; l'empreinte d'une adresse est calculée avec l'identifiant d'URL de la tribu, pour qu'une même adresse ne soit pas reconnaissable d'une tribu à l'autre (D10). Troisième paquet généré dans `sqlc.yaml`.
- [x] **14. Logique pure de `internal/tribe`**, testée en tableaux de cas :
  - code à 6 chiffres tiré avec `crypto/rand` ; validité de 10 minutes ; 3 essais, invalidé au troisième échec ; usage unique ; une nouvelle demande invalide le précédent ; comparaison en temps constant ;
  - jeton de session opaque de 32 octets aléatoires ; expiration glissante de 90 jours ;
  - limitation des demandes : 3 par quart d'heure pour une adresse dans une tribu, 10 par heure pour une adresse IP, 30 par heure pour une tribu (ENF-01) ; la décision se calcule à partir des demandes passées, fournies par une interface de stockage ;
  - appareil détecté (D12) : type, système et navigateur déduits de l'en-tête `User-Agent` par une fonction pure, sans dépendance ; « app installée » ou « onglet » vient du client à l'ouverture de la session. Les User-Agent Client Hints attendent EF-04 ;
  - révocation d'un membre, dans l'adaptateur de la tribu, en avance sur EF-02 : le scénario « Un membre révoqué ne reçoit pas de code » en a besoin comme état de départ (même cas que l'ajout de membre du lot A).
- [x] **15. `Mailer`** (ADR 0014, point 4) : interface, implémentation qui écrit dans les logs pour le développement, implémentation de test qui enregistre les envois. Message en texte brut, en français, avec le code, sa durée de validité et « Melting Tribe » ; ni lien ni nom de tribu. L'envoi est fait hors de la requête, pour que le temps de réponse ne distingue pas une adresse membre d'une autre. Le serveur sait attendre la fin des envois en cours (D11) : à l'arrêt, avant de fermer les bases, et dans les tests, avant de vérifier qu'un e-mail est parti ou qu'aucun ne l'est.
- [x] **16. API sous `/tribes/<identifiant>/api/`.**
  - `POST login-codes` (adresse) : toujours la même réponse, ou « trop de demandes » (429).
  - `POST sessions` (adresse, code, app installée ou non) : ouvre la session, pose le cookie, renvoie le nom de la tribu et le membre ; sinon code incorrect avec essais restants, ou code à redemander (expiré, essais épuisés).
  - `GET session` : nom de la tribu et membre, ou 401. `DELETE session` : déconnexion.
  - Le nom de la tribu renvoyé est lu dans la base de la tribu, jamais dans le registre (D9).
  - Cookie `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/tribes/<identifiant>/`, 90 jours, renouvelé avec l'expiration glissante.
  - Intergiciels : résolution de la tribu puis de la session dans la base de cette tribu (ADR 0003, point 3) ; vérification de l'en-tête `Origin` sur les méthodes autres que GET (ADR 0001) ; `Cache-Control: no-store` sur l'API.
  - Tribu inexistante et adresse non membre : un code fantôme, jamais envoyé et qui ne peut pas réussir, avec la même échéance et les mêmes 3 essais ; réponses identiques à celles d'un membre actif.
  - Audit : « ouverture de session » et « déconnexion ».
- [x] **17. Effacement automatique** (PT-07), au démarrage puis périodiquement : codes expirés ou utilisés et sessions expirées dans les bases de tribu ; dans la base de limitation, toute ligne sortie de sa fenêtre (une heure au plus).
- [x] **18. Scénarios ENF-01 (16) et ENF-02 (5)** : définitions d'étapes godog ; lignes retirées de `pending.txt`. Les états de départ (membre révoqué, session vieille de 80 jours, demandes de code déjà faites) sont posés par le code du domaine et l'horloge de test, les actions et les vérifications passent par l'API.
  - **`Contexte` de `compartimentage-tribus.feature`** (D8) : l'étape « la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon" » quitte le `Contexte` et passe en première étape des cinq scénarios qui s'en servent (bibliothèque de plats, référentiel d'ingrédients, même nom d'ingrédient, accès croisé refusé, toutes les données compartimentées). Sans cela, les 5 scénarios du lot dépendent des plats, qui n'existent pas encore.
  - **Harnais** (D13) : les étapes appellent le handler du serveur directement (`httptest.NewRequest` et `httptest.NewRecorder`), sans connexion réseau. Le test fixe ainsi l'adresse IP du client (`RemoteAddr`), ce qu'exige le scénario des limites par IP et par tribu, et reporte lui-même le cookie de session d'une requête à l'autre, un « appareil » du scénario étant un porte-cookie distinct.
- [ ] **19. Documentation du lot.**
  - Conservation d'une heure des empreintes d'adresse et d'IP : à reporter dans `gestion-membres-et-sessions.md` (conservation, PT-07) et dans la page Confidentialité (`docs/design/README.md`, maquette `Confidentialite`).
  - `CHANGELOG.md`, cases cochées.

### Lot C : écrans de connexion (`feature/enf-01-ecrans-de-connexion`)

- [ ] **20. Polices et tokens** (D7).
  - Bricolage Grotesque et Figtree en `woff2` variables, pris dans les dépôts officiels de leurs auteurs à une version précise, dans `web/src/fonts/` avec le texte de l'OFL et un `README` (source, version, empreinte SHA-256). Aucun paquet npm. Si seul le `ttf` est publié, le signaler avant de convertir.
  - `@font-face` avec `font-display: swap` ; tokens de `docs/design/README.md` au complet dans `app.css`.
- [ ] **21. Routeur et client d'API** (ADR 0004, point 6) : routeur fondé sur l'API History avec le préfixe de tribu, client `fetch` qui ramène à la connexion sur un 401. Modules purs, testés avec `node:test`.
- [ ] **22. Écrans**, en DOM classique pour les formulaires (ADR 0004, point 3) :
  - saisie de l'e-mail, téléphone et ordinateur : états saisie, adresse mal formée, trop de demandes ;
  - saisie du code : états saisie, code erroné avec essais restants, essais épuisés, code expiré ; champ `autocomplete="one-time-code"` ;
  - chargement, affiché seulement au-delà de 300 ms, animations coupées avec `prefers-reduced-motion` ;
  - accueil provisoire après connexion : nom de la tribu et « Se déconnecter », en attendant le planning ;
  - erreurs avec icône et `role="alert"` ; pied de page avec le lien vers le code source de la version.
- [ ] **23. `Mailer` de développement lisible par les tests** (D6) : option `-mail-file` de `serve`, qui écrit aussi chaque message dans un fichier (une ligne JSON par message) ; refusée sans `-dev`. `make dev` ne s'en sert pas.
- [ ] **24. Playwright** : parcours de connexion et de déconnexion, états d'erreur, écran identique pour une tribu inexistante ; sur Chromium et WebKit. Tests d'écran ordinaires, sans tag `@ui` ni lien avec les `.feature` (D5). Le scénario de fumée actuel est remplacé.
- [ ] **25. Documentation du lot** : `docs/design/README.md` si un écart avec les maquettes apparaît, `CLAUDE.md`, `CHANGELOG.md`, cases cochées.
- [ ] **26. Validation par le développeur** : `make seed` puis `make dev`, connexion à `http://localhost:8080/tribes/demo/` avec le code lu dans les logs, sur ordinateur et en mode téléphone du navigateur.

## Notes d'exécution

- **Étape 1** : une seule connexion par base, pour les lectures comme pour les écritures, plutôt qu'une connexion d'écriture à côté d'un pool de lecture : plus simple, et suffisant à cette échelle. Une tribu créée par un autre processus (`admin init` pendant que le serveur tourne) est ouverte et migrée à sa première requête.
- **Étape 4** : le harnais donne à chaque scénario son propre dossier de données. Le serveur `httptest`, l'horloge et le `Mailer` de test arrivent avec le lot B, qui les crée ; aucun scénario du lot A n'en a besoin. Le test `TestAcceptance` ne s'exécute qu'avec l'option `-acceptance` (cible `make acceptance`), pour que `make test` ne le lance pas une seconde fois, sans étiquette de build qui cacherait les étapes à `go vet` et `staticcheck`. `pending.txt` contient au départ les 173 scénarios exécutables (175 moins les 2 `@manuel`).
- **Étape 6** : `members.email` peut être NULL, pour l'anonymisation (EF-11), sans recréer la table plus tard ; l'auteur « script d'administration » est un auteur NULL.
- **Étape 9** : les scénarios EF-08 pilotent `admin init` question par question ; les questions que le scénario ne mentionne pas reçoivent une réponse par défaut. L'étape « aucun e-mail n'est envoyé » consulte les envois enregistrés, qu'aucun code n'alimente avant le lot B.
- **Étape 10** : la tribu `demo` est initialisée avec Alice, qui ajoute ensuite Bruno, Chloé et David ; d'où une méthode d'ajout de membre dans l'adaptateur de la tribu, en avance sur EF-01.
- **Revue de la PR #26, points à reprendre plus tard** :
  - **Fichier de base orphelin** : si le processus meurt entre la création de la base et son inscription au registre (`CreateTribe`), le fichier reste dans `tribes/` avec l'adresse du premier membre, et rien ne le nettoie. À traiter avec EF-10 (suppression d'une tribu), par exemple par un balayage au démarrage des fichiers absents du registre.
  - **Nom de la tribu stocké deux fois** : dans le registre (ADR 0003, point 2) et dans la table `tribe` de sa base. Aucune story ne renomme une tribu ; désigner la source de vérité au plus tard quand le lot B lira le nom après connexion. *Tranché : D9, consigné dans l'ADR 0021.*
  - **Une seule connexion par base, lectures comprises** (étape 1) : une requête lancée sur la base pendant qu'une transaction ou un curseur est ouvert attend indéfiniment. Règle pour le lot B : dans une transaction, tout passe par elle ; les curseurs sont fermés avant toute autre requête.
- **Relecture du lot B après la fusion du lot A** (2026-10-03) : le code du lot A correspond à ce que le lot B suppose (colonnes `session_id` et `detected_device` du journal d'audit, sans clé étrangère, donc compatibles avec l'effacement des sessions ; opérations d'audit du glossaire ; format des dates). Deux points de l'étape 18 ne pouvaient pas passer tels qu'écrits (D8, D13) et quatre restaient implicites (D9 à D12). Le serveur ne reçoit pas encore le stockage, l'horloge ni le `Mailer` : câblage attendu de l'étape 16.

## Critères de validation

- `make ci` passe à la fin de chaque lot, `make acceptance` compris.
- Les 25 scénarios listés passent, ainsi que celui des identifiants refusés ; `pending.txt` ne contient plus aucun d'eux.
- Deux tribus créées par `admin init` ont deux fichiers distincts ; la session de l'une n'ouvre rien dans l'autre.
- L'URL d'une tribu inexistante et celle d'une tribu existante sont indiscernables sans session : page, réponses de l'API, limitation, essais.
- Aucun code ni jeton de session n'est stocké en clair, ni écrit dans les logs hors du `Mailer` de développement ; aucune adresse ni IP en clair dans la base de limitation.
- Après un redémarrage du serveur, les limites de demandes et les essais restants sont inchangés.
- Un échec de migration empêche le démarrage et `/healthz` ne répond pas.

## Décisions prises

Le 2026-10-03, avec le développeur.

- **D1. Trois PR** : stockage et harnais livrés avec EF-08 (lot A), pour que le harnais arrive avec ses premiers scénarios qui passent ; puis l'API de connexion (lot B) et les écrans (lot C). Proposition initiale de quatre PR écartée.
- **D2. Scénarios pas encore implémentés** : liste `acceptance/pending.txt`, avec échec dans les deux sens ; elle ne peut que se vider. Écartés : le mode non strict de godog (une faute de frappe dans une étape passe inaperçue), un tag d'avancement dans les `.feature`, la sélection par tag de story (trop grossière).
- **D3. Limitation des demandes dans une base SQLite dédiée**, pour garder au maximum la porte ouverte au multi-instance.
  - Elle contient les demandes de code horodatées et les codes fantômes ; les codes des membres réels restent dans la base de leur tribu (ADR 0003).
  - Adresses et IP en empreinte SHA-256, purgées dès la sortie de leur fenêtre. Une empreinte reste une donnée personnelle : conservation à mentionner dans la page Confidentialité.
  - Le code métier ne voit qu'une interface ; SQLite suppose toujours une seule machine, le multi-instance réel demandera de revoir aussi le stockage des tribus.
  - Pas d'instantané avant déploiement pour ce fichier : le perdre remet les compteurs à zéro.
  - Écartés : la mémoire du processus (proposition initiale, perdue au redémarrage et propre à une instance), le registre (aucune donnée personnelle, ADR 0003).
  - ADR 0021 à rédiger dans le lot B.
- **D4. Format de l'identifiant d'URL** : voir l'étape 5. Il n'a pas à être difficile à deviner ; la protection contre l'énumération vient de l'écran de connexion identique pour toute URL (ADR 0006, point 9).
- **D5. Lien entre Playwright et les `.feature`** (ADR 0005, « à préciser ») : non tranché ici, à décider avec la première story dont un scénario ne se prouve que dans le navigateur (C7 ou C5). Écart assumé : le parcours de connexion, cité par l'ADR 0005 parmi les candidats au tag `@ui`, est couvert par Playwright sans être relié à un scénario Gherkin.
- **D6. `Mailer` de développement lisible par Playwright** : option `-mail-file`, refusée sans `-dev`. Écartés : une route de test dans le serveur, un code fixe en développement.
- **D7. Polices** : `woff2` variables des dépôts officiels, versionnés dans `web/src/fonts/`, sans paquet npm.

Le 2026-10-03, à la relecture du lot B après la fusion du lot A.

- **D8. `Contexte` de `compartimentage-tribus.feature`** : l'étape qui crée un plat et un ingrédient dans la tribu "durand" est déplacée dans les scénarios qui s'en servent (étape 18). Écartés : une définition d'étape vide en attendant les plats (un test qui ne prouve rien), le report des 5 scénarios ENF-02 à la story des plats (le compartimentage des sessions ne serait pas prouvé avec la connexion).
- **D9. Nom de la tribu : la base de la tribu fait foi.** Une fois la session vérifiée, tout se lit dans la base de la tribu (ADR 0003, point 3). Le nom du registre ne sert qu'aux commandes d'administration, qui écriront les deux le jour où une story renommera une tribu.
- **D10. Empreinte d'une adresse calculée avec l'identifiant d'URL de la tribu** dans la base de limitation. Avec l'adresse seule, une personne membre de deux tribus y aurait la même empreinte, alors que rien ne doit relier ses appartenances (ENF-02). L'empreinte d'une IP reste commune : la limite par IP vaut pour toute l'instance.
- **D11. Fin des envois d'e-mail attendue** à l'arrêt du serveur et dans les tests. Écarté : un envoi dans la requête pendant les tests seulement (le test n'exercerait pas le code de production).
- **D12. Appareil détecté dès le lot B**, par une fonction pure sur l'en-tête `User-Agent`, parce que l'entrée d'audit « ouverture de session » le porte (EF-07). Écarté : une colonne laissée vide jusqu'à EF-04, qui laisserait des entrées d'audit incomplètes. À confirmer par le développeur.
- **D13. Harnais godog sans connexion réseau** : appel direct du handler. Un serveur `httptest.NewServer` ne voit que `127.0.0.1`, et le client HTTP de Go ne renvoie pas un cookie `Secure` sur `http://` (vérifié le 2026-10-03). Précise l'étape 4 et l'ADR 0005, point 2 : le serveur tourne toujours dans le processus de test. Écartés : `X-Forwarded-For` (hors périmètre, ADR 0006 point 7), un serveur de test en TLS (ne règle pas l'adresse IP).

## Questions ouvertes

- **Cookie `Secure` sur `http://localhost` avec WebKit** : Chromium et Firefox l'acceptent ; à vérifier pour WebKit dès l'étape 16, avant d'écrire les écrans. Si WebKit le refuse, il faudra soit du TLS local pour les tests Playwright, soit limiter le parcours de connexion à Chromium et couvrir WebKit en recette.
- **Écriture de la dernière activité à chaque requête** : négligeable à cette échelle (ADR 0001) ; à espacer seulement si la mesure le justifie.
- **Sessions cloud de Claude** : aujourd'hui elles n'atteignent pas `proxy.golang.org`, donc ni `make tools` ni `make ci`. Elles conviennent à la documentation ; le code de ce plan se fait dans le Dev Container, sauf à ouvrir cet accès dans les réglages réseau de l'environnement cloud.
