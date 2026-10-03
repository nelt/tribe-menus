# Plan : socle de données et connexion (première tranche verticale)

- **Date** : 2026-10-03
- **Statut** : brouillon (les « Décisions à valider » sont à trancher avant de passer à « prêt »)

## Objectif

Première tranche qui traverse tout le système : une tribu créée par la commande d'administration, un membre qui ouvre l'URL de sa tribu, reçoit un code, se connecte et voit le nom de sa tribu, sur des bases SQLite compartimentées, avec les scénarios Gherkin exécutés par godog dans `make ci`.

À la fin du plan :

- `make seed`, `make generate` et `make acceptance` existent et ont un contenu (ADR 0010) ;
- `make dev` permet de se connecter à la tribu de démonstration, le code étant lu dans les logs (ADR 0009) ;
- 25 scénarios passent : 16 de `authentification.feature`, 5 de `compartimentage-tribus.feature`, 4 de `administration.feature` (détail plus bas).

Ce plan est exécuté par **Claude Code dans le Dev Container**, en quatre PR successives (lots A à D), chacune fusionnable seule.

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
- **Rien n'est révélé avant connexion** (ENF-02, ADR 0006, point 9) : c'est la contrainte qui structure le lot C. Pour une tribu qui n'existe pas, une adresse inconnue, révoquée ou d'une autre tribu, les réponses de l'API (corps, code HTTP, essais restants, limitation) sont identiques à celles d'un membre actif.
- **Horloge injectée** : validité de 10 minutes, fenêtres de limitation et expiration à 90 jours se testent avec une horloge fournie par le test, jamais avec `time.Sleep`.
- **Mode sans proxy** (ADR 0006, point 8) : seul mode de ce plan. L'adresse IP du client est celle de la connexion TCP.

### Scénarios couverts

| Fichier | Scénarios | Lot |
| --- | --- | --- |
| `administration.feature` | EF-08 : initialiser une tribu ; l'identifiant d'URL doit être libre ; plusieurs tribus ; le premier membre appartient déjà à une autre tribu | B |
| `authentification.feature` | ENF-01 : les 13 scénarios de la demande et de la saisie du code (de « Se connecter avec un code reçu par e-mail » à « Un code ne sert qu'une fois ») ; expiration glissante ; session inutilisée ; attributs du cookie | C |
| `compartimentage-tribus.feature` | ENF-02 : la session d'une tribu ne donne pas accès à une autre ; sessions distinctes pour une même adresse ; l'écran de connexion ne révèle pas la tribu ; le nom de la tribu apparaît une fois connecté ; le journal d'audit est propre à la tribu | C |

## Étapes

### Lot A : stockage et harnais d'acceptation (`feature/stockage-et-acceptation`)

- [ ] **1. `internal/storage` : ouverture et migrations.**
  - Dépendance `modernc.org/sqlite`. À chaque connexion : WAL, `foreign_keys = ON`, `busy_timeout` ; tables `STRICT` (ADR 0003, point 5).
  - Migrations en fichiers SQL embarqués, deux séries (`migrations/registry/`, `migrations/tribe/`), version suivie par `PRAGMA user_version`, chaque fichier appliqué dans une transaction.
  - Registre : identifiant d'URL, nom, fichier de la base. Le nom du fichier est un identifiant aléatoire, jamais dérivé de l'identifiant d'URL.
  - Un type qui migre le registre puis toutes les bases au démarrage, ouvre la base d'une tribu par son identifiant d'URL et fait passer ses écritures par une seule connexion.
  - Tests en tableaux sur `t.TempDir()` : migration rejouée sans effet, migration en échec qui remonte l'erreur et laisse la version inchangée, identifiant inconnu.
- [ ] **2. sqlc.**
  - `sqlc.yaml` avec deux paquets générés (registre, tribu) ; code généré versionné.
  - Cible `make generate` ; `go tool sqlc diff` dans `make lint`, pour refuser un code généré périmé.
- [ ] **3. Serveur : répertoire de données et `/healthz`.**
  - Option `-data` de `serve` (par défaut `data/`, déjà ignoré par Git) ; migrations appliquées avant d'écouter ; échec de migration : le serveur ne démarre pas (PT-12).
  - `GET /healthz` à la racine, qui ne répond qu'une fois les bases migrées (ADR 0016, point 1.6).
- [ ] **4. `acceptance/` : harnais godog.**
  - Dépendance `github.com/cucumber/godog`. Lecture directe de `docs/specs/features/` (`# language: fr`), serveur dans le processus (`httptest`), bases dans un dossier temporaire par scénario, horloge et `Mailer` de test.
  - Scénarios `@manuel` et `@ui` exclus (ADR 0005).
  - Scénarios pas encore implémentés : voir la décision D2.
  - Cible `make acceptance`, ajoutée à `make ci` entre `test` et `e2e`.
- [ ] **5. Documentation du lot** : `CLAUDE.md` (cibles `generate` et `acceptance` disponibles), `CHANGELOG.md`, cases cochées.

### Lot B : initialiser une tribu (`feature/ef-08-initialiser-une-tribu`)

- [ ] **6. Schéma de la tribu, première migration** : `members` (adresse normalisée unique, nom d'affichage, statut, dates et auteurs d'ajout et de révocation), `audit_log` (EF-07 : date, opération, membre concerné, auteur, session et appareil détecté le cas échéant), nom de la tribu.
- [ ] **7. `internal/tribe` : membres et audit.**
  - Logique pure : normalisation de l'adresse (sans espaces, en minuscules), validation de l'identifiant d'URL (décision D4), opérations d'audit nommées selon le glossaire.
  - Accès aux données par sqlc, en adaptateur mince.
- [ ] **8. Sous-commande `tribe-menus admin init`** (EF-08).
  - Questions sur l'entrée standard : nom, identifiant d'URL (redemandé s'il est invalide ou pris), adresse du premier membre, nom d'affichage facultatif.
  - Crée la base, la migre, inscrit la tribu au registre, ajoute le premier membre, trace « initialisation de la tribu » avec « script d'administration » comme auteur, affiche l'URL de la tribu (option `-base-url`, par défaut `http://localhost:8080`). Aucun e-mail.
  - En cas d'échec en cours de route, ni fichier ni entrée de registre ne restent.
- [ ] **9. Scénarios EF-08** (4) : définitions d'étapes godog qui pilotent la sous-commande avec une entrée et une sortie simulées.
- [ ] **10. `make seed`** : sous-commande `admin seed`, qui crée la tribu de démonstration `demo` (« Les Démo ») avec les membres des conventions Gherkin (Alice, Bruno, Chloé, David) ; sans effet si elle existe déjà. Elle s'enrichira avec les plats et le planning.
- [ ] **11. Documentation du lot** : `CLAUDE.md` (`seed`), `CHANGELOG.md`, cases cochées.

### Lot C : connexion et session, côté API (`feature/enf-01-connexion-et-session`)

- [ ] **12. Schéma : `login_codes` et `sessions`.**
  - Code : membre, empreinte du code, échéance, essais restants. Un seul code valable par membre.
  - Session : empreinte SHA-256 du jeton, membre, dates d'ouverture, de dernière activité et d'expiration, appareil détecté (`User-Agent`, app installée ou onglet), nom de session.
- [ ] **13. Logique pure de `internal/tribe`**, testée en tableaux de cas :
  - code à 6 chiffres tiré avec `crypto/rand` ; validité de 10 minutes ; 3 essais, invalidé au troisième échec ; usage unique ; une nouvelle demande invalide le précédent ; comparaison en temps constant ;
  - jeton de session opaque de 32 octets aléatoires ; expiration glissante de 90 jours ;
  - limitation des demandes : 3 par quart d'heure pour une adresse dans une tribu, 10 par heure pour une adresse IP, 30 par heure pour une tribu (ENF-01).
- [ ] **14. `Mailer`** (ADR 0014, point 4) : interface, implémentation qui écrit dans les logs pour le développement, implémentation de test qui enregistre les envois. Message en texte brut, en français, avec le code, sa durée de validité et « Melting Tribe » ; ni lien ni nom de tribu. L'envoi est fait hors de la requête, pour que le temps de réponse ne distingue pas une adresse membre d'une autre.
- [ ] **15. API sous `/tribes/<identifiant>/api/`.**
  - `POST login-codes` (adresse) : toujours la même réponse, ou « trop de demandes » (429).
  - `POST sessions` (adresse, code, app installée ou non) : ouvre la session, pose le cookie, renvoie le nom de la tribu et le membre ; sinon code incorrect avec essais restants, ou code à redemander (expiré, essais épuisés).
  - `GET session` : nom de la tribu et membre, ou 401. `DELETE session` : déconnexion.
  - Cookie `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/tribes/<identifiant>/`, 90 jours, renouvelé avec l'expiration glissante.
  - Intergiciels : résolution de la tribu puis de la session dans la base de cette tribu (ADR 0003, point 3) ; vérification de l'en-tête `Origin` sur les méthodes autres que GET (ADR 0001) ; `Cache-Control: no-store` sur l'API.
  - Tribu inexistante et adresse non membre : mêmes réponses qu'un membre actif, essais compris (décision D3).
  - Audit : « ouverture de session » et « déconnexion ».
- [ ] **16. Effacement automatique** (PT-07) : codes expirés ou utilisés, sessions expirées, au démarrage puis une fois par jour.
- [ ] **17. Scénarios ENF-01 (16) et ENF-02 (5)** : définitions d'étapes godog. Les états de départ (membre révoqué, session vieille de 80 jours) sont posés par le code du domaine et l'horloge de test, les actions et les vérifications passent par l'API.
- [ ] **18. Documentation du lot** : ADR si D3 est retenue, glossaire si besoin, `CHANGELOG.md`, cases cochées.

### Lot D : écrans de connexion (`feature/enf-01-ecrans-de-connexion`)

- [ ] **19. Polices et tokens** : Bricolage Grotesque et Figtree auto-hébergées (fichiers `woff2` et texte de l'OFL, décision D7) ; tokens de `docs/design/README.md` au complet dans `app.css`.
- [ ] **20. Routeur et client d'API** (ADR 0004, point 6) : routeur fondé sur l'API History avec le préfixe de tribu, client `fetch` qui ramène à la connexion sur un 401. Modules purs, testés avec `node:test`.
- [ ] **21. Écrans**, en DOM classique pour les formulaires (ADR 0004, point 3) :
  - saisie de l'e-mail, téléphone et ordinateur : états saisie, adresse mal formée, trop de demandes ;
  - saisie du code : états saisie, code erroné avec essais restants, essais épuisés, code expiré ; champ `autocomplete="one-time-code"` ;
  - chargement, affiché seulement au-delà de 300 ms, animations coupées avec `prefers-reduced-motion` ;
  - accueil provisoire après connexion : nom de la tribu et « Se déconnecter », en attendant le planning ;
  - erreurs avec icône et `role="alert"` ; pied de page avec le lien vers le code source de la version.
- [ ] **22. Playwright** : parcours de connexion et de déconnexion, états d'erreur, écran identique pour une tribu inexistante ; sur Chromium et WebKit. Le code est lu dans le fichier du `Mailer` de développement (décision D6). Le scénario de fumée actuel est remplacé.
- [ ] **23. Documentation du lot** : `docs/design/README.md` si un écart avec les maquettes apparaît, `CLAUDE.md`, `CHANGELOG.md`, cases cochées.
- [ ] **24. Validation par le développeur** : `make seed` puis `make dev`, connexion à `http://localhost:8080/tribes/demo/` avec le code lu dans les logs, sur ordinateur et en mode téléphone du navigateur.

## Notes d'exécution

- (à remplir au fil de l'avancement)

## Critères de validation

- `make ci` passe à la fin de chaque lot, `make acceptance` compris.
- Les 25 scénarios listés passent ; aucun autre n'est compté comme passant par erreur (D2).
- Deux tribus créées par `admin init` ont deux fichiers distincts ; la session de l'une n'ouvre rien dans l'autre.
- L'URL d'une tribu inexistante et celle d'une tribu existante sont indiscernables sans session : page, réponses de l'API, limitation, essais.
- Aucun code ni jeton de session n'est stocké en clair, ni écrit dans les logs hors du `Mailer` de développement.
- Un échec de migration empêche le démarrage et `/healthz` ne répond pas.

## Décisions à valider

Chacune vient avec une proposition ; sans objection, c'est elle qui s'applique.

- **D1. Quatre PR.** Proposition : un lot par PR, dans l'ordre. Variante : fusionner A et B, le lot A seul n'apportant aucun scénario qui passe.
- **D2. Scénarios pas encore implémentés.** godog trouvera 175 scénarios, dont la plupart sans définition d'étape pendant des mois. Proposition : un fichier `acceptance/pending.txt` liste les scénarios attendus comme non implémentés ; l'exécution échoue si un scénario hors liste a une étape non définie ou échoue, et aussi si un scénario de la liste passe. La liste ne peut que se vider, et elle donne l'avancement de la V1. Variante écartée : le mode non strict de godog, où une faute de frappe dans une étape passe inaperçue.
- **D3. Limitation et « codes fantômes » en mémoire.** Les compteurs de limitation ne peuvent pas vivre dans la base d'une tribu, puisqu'ils doivent se comporter de la même façon pour une tribu qui n'existe pas ; ni dans le registre, qui ne contient aucune donnée personnelle (ADR 0003, point 2). Proposition : compteurs en mémoire du processus, perdus au redémarrage ; pour une adresse non membre ou une tribu inexistante, un code fantôme en mémoire, jamais envoyé, avec la même échéance et les mêmes 3 essais. Les codes des membres restent en base (ADR 0003). Taille bornée par la limite par IP et par un plafond global. Si elle est retenue : ADR 0021.
- **D4. Format de l'identifiant d'URL.** Les specs ne le fixent pas. Proposition : 3 à 40 caractères, lettres minuscules sans accent, chiffres et tirets, une lettre en premier, pas de tiret final. À reporter dans `gestion-membres-et-sessions.md` (EF-08).
- **D5. Lien entre Playwright et les `.feature`** (ADR 0005, « à préciser »). Proposition : ne pas trancher ici. Les scénarios de connexion sont prouvés contre l'API ; les tests Playwright du lot D vérifient les écrans sans tag `@ui`. La question se pose vraiment avec le hors-ligne (C7).
- **D6. `Mailer` de développement lisible par Playwright.** Proposition : option `-mail-file` de `serve`, réservée au mode `-dev`, qui écrit aussi chaque message dans un fichier ; Playwright y lit le code. Aucune route de test dans le serveur.
- **D7. Fichiers de polices.** Proposition : fichiers `woff2` variables pris dans les dépôts officiels des deux polices, déposés dans `web/src/fonts/` avec le texte de l'OFL, sans paquet npm.

## Questions ouvertes

- **Cookie `Secure` sur `http://localhost` avec WebKit** : Chromium et Firefox l'acceptent ; à vérifier pour WebKit dès l'étape 15, avant d'écrire les écrans. Si WebKit le refuse, il faudra soit du TLS local pour les tests Playwright, soit limiter le parcours de connexion à Chromium et couvrir WebKit en recette.
- **Écriture de la dernière activité à chaque requête** : négligeable à cette échelle (ADR 0001) ; à espacer seulement si la mesure le justifie.
- **Sessions cloud de Claude** : aujourd'hui elles n'atteignent pas `proxy.golang.org`, donc ni `make tools` ni `make ci`. Elles conviennent à la documentation ; le code de ce plan se fait dans le Dev Container, sauf à ouvrir cet accès dans les réglages réseau de l'environnement cloud.
