# Plan : application prête pour la production

- **Nom** : `production`
- **Date** : 2026-10-06
- **Statut** : en cours (décisions D1 à D12 confirmées par le développeur le 2026-10-06 ; lots A et B faits, lot C en revue)

## Objectif

Donner au binaire tout ce qui lui manque pour tourner sur un serveur, **sans toucher au serveur** : le plan `recette` n'aura plus qu'à écrire `deploy/`, provisionner et déployer.

À la fin du plan :

- lancé par systemd avec un fichier de configuration, le binaire écoute sur le socket qu'on lui transmet, écrit ses logs en JSON et lit ses secrets dans ses credentials (ADR 0015) ;
- il envoie les codes de connexion par le SMTP du MX Plan (ADR 0014) et refuse les adresses que cet envoi ne saurait pas porter (`revue-securite`, constat 7) ;
- derrière Caddy, il compte les demandes de code par adresse IP réelle du client, une IPv6 comptant pour son préfixe /64 (ADR 0006, point 7) ;
- il signale à l'administrateur les limites atteintes de façon répétée et les échecs répétés d'envoi (ENF-01 ; `revue-securite`, D1 ; ADR 0015, point 15) ;
- le mode développement ne peut pas être activé par erreur sur un serveur (`revue-securite`, constat 8) ;
- les fichiers du front portent une empreinte de leur contenu, la CI produit l'archive de l'ADR 0012 pour chaque PR, et une étiquette `vX.Y.Z` publie une release selon `RELEASING.md`.

Ce plan est exécuté par **Claude Code dans le Dev Container**, en cinq PR successives (lots A à E), chacune fusionnable seule.

**Hors périmètre**, renvoyé au plan `recette` :

- tout ce qui vit dans `deploy/` au-delà de l'exemple de configuration : `provision.sh`, unité et socket systemd, configuration de Caddy, `tribe-menus-deploy`, `tribe-menus-admin` ;
- DNS et e-mail du domaine (SPF, DKIM, DMARC, boîtes `no-reply@` et `server@`) ;
- les essais sur un vrai serveur : activation de socket par systemd, en-têtes réellement posés par Caddy, envoi par le compte OVHcloud et réception sur les messageries des membres, première release ;
- le relais par e-mail des alertes que l'application écrit dans son journal (D6).

**Hors périmètre**, renvoyé plus loin : service worker et liste de pré-cache (plan du hors-ligne ; l'étape 23 lui prépare la liste des fichiers), attestation de provenance et SBOM (ADR 0012, reportés), sonde externe (ADR 0015).

## Contexte et contraintes

- À lire avant de commencer : ADR 0006 (points 7 et 8), 0011 (points 5 et 8), 0012, 0013, 0014, 0015 (points 9 à 15), 0016, 0020, 0021, 0022 ; `docs/specs/exigences-non-fonctionnelles.md` (ENF-01, ENF-02) ; `docs/plans/2026-10-04-revue-de-securite-avant-mise-en-ligne.md` (D1, constats 1, 7 et 8) ; `docs/traitement-des-vulnerabilites.md` ; `docs/securite-depot.md` (sections 2 et 4).
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, avec `git commit -s` ; une ligne dans `CHANGELOG.md` par PR, qui cite le lot (`production/B`) ; `make ci` avant de pousser.
- **Aucune dépendance ajoutée** (ADR 0002, point 5). Tout se fait avec la bibliothèque standard : `encoding/json`, `net/smtp`, `net/mail`, `mime`, `mime/quotedprintable`, `crypto/tls`, `net/netip`, `log/slog`, `archive/tar`, `compress/gzip`. Le protocole d'activation de socket de systemd se lit en quelques lignes, sans `go-systemd`.
- **Rien de propre à l'instance dans le dépôt** (ADR 0011, point 5) : l'adresse de l'administrateur et le mot de passe SMTP n'apparaissent ni dans le code, ni dans les tests, ni dans l'exemple de configuration, qui utilise `example.org` (réservé par la RFC 2606, à la différence d'`exemple.fr`, `revue-securite`, constat 9).
- **Rien n'est révélé avant connexion** (ENF-02) reste la contrainte du socle : aucune des nouveautés ne doit distinguer un membre d'une autre adresse, ni une tribu existante d'une autre, en réponse comme en durée. L'envoi reste hors de la requête (`mail.Outbox`).
- **Ce que la base de limitation n'apprend pas** (ADR 0021, point 4) : elle ne doit pas pouvoir dire quelles empreintes d'adresse sont celles de membres. Les compteurs d'alerte suivent la même règle (D7).
- **Les logs ne portent ni adresse e-mail, ni code, ni jeton**, hors du `Mailer` de développement : c'est l'état constaté par la revue de sécurité, à garder avec l'envoi SMTP, dont les erreurs citent volontiers le destinataire (étape 13).
- **Horloge injectée** : fenêtres d'alerte et délais se testent avec l'horloge du test, jamais avec `time.Sleep`.
- **Vocabulaire** : tout terme nouveau entre dans `docs/specs/glossaire.md` avant d'apparaître dans le code (alerte, demande refusée, code épuisé, fichier de configuration).
- **Workflows** : le jeton du Dev Container n'a pas la permission *Workflows* (ADR 0020). Le lot E, qui modifie `.github/workflows/`, est préparé par Claude Code et **poussé par le développeur depuis l'hôte** ; c'est pour cela qu'il est séparé du lot D.
- **Vulnérabilités** : rien n'est en production, tout suit le flux public (ADR 0022, point 6). Les lots A à C touchent à la sécurité : leur revue (`docs/revue-de-pr.md`) est faite par une session cloud qui n'a pas écrit le code, avec les points d'attention listés à chaque lot.
- **Sessions cloud** : elles n'atteignent ni `proxy.golang.org` ni le registre npm (vérifié le 2026-10-06), donc ni `make tools` ni `make ci`. Elles conviennent à ce plan écrit et aux revues, pas au code.

### État de départ

| Sujet | Aujourd'hui | Où |
| --- | --- | --- |
| Envoi des e-mails | hors `-dev`, `mail.Unconfigured` refuse tout envoi | `cmd/tribe-menus/main.go`, `migrateAndServe` |
| Écoute | `net.Listen("tcp", …)` seul | `listenAndServe` |
| Configuration | options de la ligne de commande seules | `serve`, `runAdmin` |
| Logs | texte, sur la sortie d'erreur | `serve` |
| Adresse IP du client | celle de la connexion TCP | `internal/server/api.go`, `clientIP` |
| Adresse e-mail | forme `local@domaine` avec un point dans le domaine, rien de plus | `internal/tribe/tribe.go`, `ParseEmail` |
| Limites atteintes | la demande est refusée (429), rien n'est compté ni signalé | `internal/tribe/auth.go`, `RequestCode` |
| Front | `main.js` et `app.css` à nom fixe, sans en-tête de cache | `web/scripts/build.mjs`, `internal/server/server.go`, `serveApp` |
| Build | binaire pour la plate-forme de la machine ; l'artefact de PR contient `bin/tribe-menus` et `site/` | `Makefile`, `.github/workflows/ci.yml` |
| Release | ni `release.yml`, ni `RELEASING.md`, ni `deploy/` | |

## Étapes

### Lot A : exécution derrière systemd et Caddy (`feature/execution-sous-systemd`)

Points d'attention de la revue : ce qui se passe quand la configuration est incomplète ou contradictoire ; qui peut faire croire au serveur qu'il est derrière Caddy ; contenu des logs de démarrage.

- [x] **1. Fichier de configuration** (D1), paquet `internal/config`.
  - JSON lu par `encoding/json`, clé inconnue refusée (`DisallowUnknownFields`), un seul document par fichier.
  - Clés du lot : `data` (répertoire des bases, chemin absolu), `listen` (`"systemd"` ou une adresse TCP, ADR 0006, point 8), `baseURL` (adresse publique de l'instance, `https://…` sans chemin).
  - Lecture et validation en fonction pure, testée en tableau de cas : clé manquante, chemin relatif, adresse mal formée, clé inconnue, fichier vide. Le message d'erreur nomme la clé fautive, jamais une valeur.
  - `deploy/config.example.json` et un `deploy/README.md` de quelques lignes (ce que le dossier contient aujourd'hui, ce qui arrive avec `recette`). Un test lit l'exemple avec le vrai lecteur, pour qu'il ne se périme pas.
- [x] **2. Modes de `serve`** (D2 ; `revue-securite`, constat 8).
  - `serve -config <fichier>` : mode serveur. `serve -dev` avec ses options actuelles : mode développement. **L'un des deux, jamais les deux, jamais aucun** ; `-addr`, `-data`, `-root` et `-mail-file` sont refusées avec `-config`.
  - `-dev` est aussi refusé quand un socket est transmis par systemd (variables `LISTEN_FDS` et `LISTEN_PID` désignant ce processus), au cas où une unité lancerait `serve -dev`.
  - `TestRun` complété : chaque combinaison refusée, avec son message et le code de sortie 2.
- [x] **3. Écoute sur le socket transmis par systemd** (ADR 0015, point 11).
  - `listen: "systemd"` : le serveur prend le descripteur 3 si `LISTEN_PID` est son propre identifiant et `LISTEN_FDS` vaut 1, puis retire ces variables de son environnement. Autre nombre de descripteurs, ou aucun : le serveur ne démarre pas et dit pourquoi. Un socket transmis alors que `listen` est une adresse TCP : refus aussi.
  - Le fichier du socket appartient à systemd : il ne doit pas être supprimé à l'arrêt (`SetUnlinkOnClose(false)`, à prouver par le test).
  - Tests : lecture des variables en fonction pure (environnement et identifiant de processus en paramètres) ; un test de `listenAndServe` sur un socket Unix créé par le test, qui appelle `/healthz`, arrête le serveur et vérifie que le fichier du socket existe encore.
- [x] **4. Logs en JSON sur la sortie standard** en mode serveur (ADR 0015, point 14) ; texte sur la sortie d'erreur en développement, comme aujourd'hui. Le log de démarrage donne la version, le commit, le mode d'écoute et le chemin du fichier de configuration.
- [x] **5. `admin init` et `admin seed` acceptent `-config`** : ils y lisent `data` et `baseURL`, à la place de `-data` et `-base-url`, refusées avec `-config`. Le script `tribe-menus-admin` du plan `recette` n'aura qu'à passer ce chemin (ADR 0015, point 13).
- [x] **6. Adresse IP du client** (D3, D4 ; ADR 0006, point 7). Specs et scénario d'abord.
  - ENF-01 précisée : la limite par IP porte sur l'adresse du client telle que Caddy la transmet, et une adresse IPv6 compte pour son préfixe /64. ADR 0006 (point 7) et ADR 0021 (point 3, empreinte de l'IP) reçoivent un renvoi daté.
  - `authentification.feature` : un scénario « Deux adresses IPv6 du même préfixe /64 partagent la limite par IP », selon `conventions-gherkin.md` ; le harnais fixe déjà l'adresse du client.
  - `internal/tribe` : fonction pure qui normalise l'adresse avant l'empreinte (IPv4 entière, IPv4 encapsulée dans une IPv6 ramenée à l'IPv4, IPv6 réduite à son /64, zone retirée), testée en tableau de cas.
  - `internal/server` : `Config` reçoit un indicateur « derrière le proxy », vrai seulement quand le serveur écoute sur le socket de systemd. Dans ce cas, l'adresse du client est la **dernière** de `X-Forwarded-For` ; sinon l'en-tête est ignoré et l'adresse reste celle de la connexion TCP.
  - En-tête absent ou illisible derrière le proxy : la demande est comptée sous une même adresse « inconnue », commune à toutes les demandes dans ce cas, et un avertissement est journalisé. La limite devient plus stricte, jamais plus lâche.
  - Tests : en-tête forgé ignoré en mode TCP ; plusieurs adresses dans l'en-tête, la dernière gagne ; en-tête absent.
- [x] **7. Documentation du lot** : ADR 0015 précisé (format et clés du fichier, points 9 et 10) ; `CLAUDE.md` (modes de `serve`, `deploy/`) ; `CHANGELOG.md` ; feuille de route (points reportés « adresse IP du client » et « rien ne refuse `-dev` » retirés) ; cases cochées.

### Lot B : envoi des e-mails par SMTP (`feature/envoi-smtp`)

Points d'attention de la revue : injection dans les en-têtes ou l'enveloppe par une adresse saisie ; ce que les logs gardent d'un échec ; où le mot de passe peut apparaître ; durée de réponse inchangée, que le code parte ou non.

- [x] **8. Règle d'adresse resserrée** (D5 ; `revue-securite`, constat 7). Specs et scénario d'abord.
  - `gestion-membres-et-sessions.md` (identification du membre) : partie locale faite de lettres sans accent, chiffres et `.!#$%&'*+/=?^_{|}~-`, sans point en tête, en fin ni doublé, 64 caractères au plus ; domaine fait de labels de lettres, chiffres et tirets, sans tiret en tête ni en fin ; 254 caractères au plus ; ASCII seul. Une saisie non conforme est refusée, pas corrigée.
  - `authentification.feature` : un `Plan du scénario` d'adresses refusées, avec celles du constat 7 (`a,b@c.d`, `a@c.d,e.f`, `<a>@c.d`) et une adresse accentuée.
  - `ParseEmail` : la règle ci-dessus, puis un contrôle croisé : l'adresse relue par `net/mail` doit redonner exactement la même adresse, sans nom. Tableau de cas étendu, octet nul compris. La règle vaut pour `admin init`, comme aujourd'hui.
- [x] **9. Composition du message**, fonction pure du paquet `mail`, testée octet par octet.
  - En-têtes : `From` (« Melting Tribe » et l'adresse d'envoi) et `To` écrits par `net/mail`, `Subject` encodé (RFC 2047), `Date`, `Message-ID` aléatoire sous le domaine de l'adresse d'envoi, `MIME-Version`, `Content-Type: text/plain; charset=utf-8`, `Content-Transfer-Encoding: quoted-printable`, `Auto-Submitted: auto-generated`.
  - Fins de ligne CRLF, corps encodé par `mime/quotedprintable`. Le texte du message ne change pas (ADR 0014, point 4).
- [x] **10. `SMTPMailer`** (ADR 0014, points 1 et 4).
  - Connexion TLS directe (port 465), TLS 1.2 au minimum, certificat vérifié pour le nom du serveur ; authentification `PLAIN` ; `MAIL FROM` et `RCPT TO` avec les adresses nues ; un message par connexion.
  - Le délai du contexte (30 secondes, `sendTimeout`) borne toute la conversation, connexion comprise.
  - Test avec un serveur SMTP factice dans le processus, en TLS, avec un certificat créé par le test : message reçu conforme à l'étape 9 ; authentification refusée ; destinataire refusé ; serveur muet jusqu'au délai ; certificat d'un autre nom refusé.
- [x] **11. Secret et configuration.**
  - Clés ajoutées : `smtp.host`, `smtp.port`, `smtp.username`, `smtp.from`. Elles sont obligatoires en mode serveur.
  - Mot de passe : fichier `smtp-password` du répertoire que désigne `CREDENTIALS_DIRECTORY`, lu au démarrage (ADR 0015, point 10). Variable absente, fichier absent ou vide : le serveur ne démarre pas. Aucune option, aucune variable d'environnement et aucune clé de configuration ne peut porter le mot de passe ; le message d'erreur et les logs ne le contiennent jamais.
  - `mail.Unconfigured` et `ErrNotConfigured` disparaissent : hors `-dev`, il y a toujours un envoi réel.
  - Clé écrite deux fois, ou dans une autre casse (`"LISTEN"`, `"baseurl"`), refusée comme une clé inconnue, clés imbriquées comprises : `encoding/json` garde la dernière valeur et ignore la casse (revue de la PR #45, point 3).
- [x] **12. Vérification au démarrage**, sans bloquer : une fois le serveur à l'écoute, il ouvre une connexion SMTP, s'authentifie et la referme. Un échec est journalisé comme un échec d'envoi (étape 13) ; le serveur continue, et `/healthz` ne dépend pas du SMTP, pour qu'un mot de passe périmé ne déclenche pas un retour arrière (ADR 0016).
- [x] **13. Ce qu'un échec laisse dans les logs.** Un enregistrement d'erreur par envoi échoué, avec l'étape (connexion, authentification, expéditeur, destinataire, contenu) et le code de réponse SMTP. Le texte de la réponse du serveur n'est gardé qu'après remplacement de l'adresse du destinataire. Test : après un `RCPT TO` refusé avec l'adresse dans la réponse, les logs ne la contiennent pas.
- [ ] **14. Essai réel, par le développeur, facultatif** (renvoyé à la recette le 2026-10-06) : `serve -config` sur le poste, avec un fichier de configuration et un mot de passe hors du dépôt, `listen` en TCP, et un code reçu dans sa propre boîte. Si le port 465 ne sort pas du Dev Container, l'essai attend la recette ; le dire dans les notes d'exécution.
- [x] **15. Documentation du lot** : exemple de configuration et son test ; `CLAUDE.md` ; `CHANGELOG.md` ; feuille de route (point reporté « `ParseEmail` » retiré) ; cases cochées.

### Lot C : alertes à l'administrateur (`feature/enf-01-alertes`)

Points d'attention de la revue : ce que les compteurs apprennent sur l'appartenance d'une adresse à une tribu ; contenu de l'e-mail d'alerte ; croissance des tables sous un flot de demandes.

- [x] **16. ADR 0023 : alertes de l'application** (D6 à D9), et specs.
  - L'ADR fixe : ce qu'est une alerte (un enregistrement du journal de niveau erreur portant un attribut `alert`, relayé par le serveur) ; lesquelles sont aussi envoyées par e-mail par l'application ; ce qui est compté et où ; ce que l'e-mail contient. Il précise l'ADR 0015 (point 15) et l'ADR 0021 (ce que contiennent les bases).
  - ENF-01 : la phrase « limites atteintes de façon répétée » reçoit sa définition et ses seuils (D8).
  - `authentification.feature` : scénarios « Des demandes refusées à répétition sont signalées à l'administrateur », « Des codes épuisés à répétition sont signalés à l'administrateur », « Des demandes répétées pour une même adresse sont signalées à l'administrateur » (D12), « L'alerte ne nomme ni adresse ni tribu », « Une alerte n'est pas répétée tant qu'elle dure ».
  - ADR 0021 (point 4) : renvoi daté vers D11, ce que la base de limitation savait des membres avant lui et ce qu'elle garde après.
- [x] **17. Un code fantôme pour toute demande** (D11). Test d'abord.
  - `Login.RequestCode` écrit le code fantôme de l'empreinte d'adresse pour **toute** demande acceptée par les limites, membre ou non, avant de chercher le membre. Celui d'un membre n'est jamais vérifié ; il expire et s'efface comme les autres.
  - Test : après une demande pour un membre et une demande pour une autre adresse, la base de limitation contient pour chacune une demande et un code fantôme de même forme ; rien de ce qu'elle garde ne les distingue.
  - Un membre révoqué dans les dix minutes qui suivent sa demande voit ses saisies vérifiées contre ce code fantôme, exactement comme une adresse inconnue : scénarios ENF-02 existants à relancer.
- [x] **18. Compteurs** (D7).
  - Table de compteurs par tranche de dix minutes et par nature d'événement, sans empreinte ni identifiant : une ligne par tranche et par nature, incrémentée. Sa taille ne dépend pas du nombre de demandes.
  - Base de limitation (`internal/storage/migrations/ratelimit/`) : demandes refusées par la limite par adresse, par la limite par IP, codes fantômes épuisés ; et la date de la dernière alerte envoyée, par nature d'alerte.
  - Base de la tribu (`internal/storage/migrations/tribe/`) : demandes refusées par la limite de la tribu, codes réels épuisés.
  - Requêtes sqlc, `make generate`. Effacement automatique des tranches sorties de la fenêtre, avec le reste (PT-07).
- [x] **19. Comptage**, dans la transaction qui constate l'événement.
  - `RateLimitStore.RecordCodeRequest` dit quelle limite a refusé, et compte le refus.
  - `Store.IssueLoginCode` compte le refus par la limite de la tribu ; `Store.OpenSession` et `CheckDecoyCode` comptent le code invalidé par son dernier essai.
  - Ni la réponse de l'API ni sa durée ne changent : les scénarios ENF-02 existants le prouvent déjà, à relancer.
- [x] **20. Évaluation et envoi.**
  - Avec l'effacement périodique, toutes les dix minutes : somme des compteurs de la dernière heure, base de limitation et toutes les tribus, et nombre d'empreintes d'adresse à neuf demandes ou plus dans l'heure, lu dans les demandes que la base de limitation garde déjà (D12) ; décision par une fonction pure (compteurs, dernière alerte, heure), testée en tableau de cas.
  - Seuil franchi et aucune alerte de cette nature depuis six heures : un enregistrement `alert=code_request_limits` dans le journal, et un e-mail à l'adresse `alerts.to` de la configuration (clé ajoutée, obligatoire en mode serveur), par le même `Mailer`.
  - E-mail en français, en texte brut : nom de l'instance (`baseURL`), fenêtre, compteurs par nature, renvoi vers les journaux d'accès de Caddy. **Ni adresse e-mail, ni adresse IP, ni tribu.**
  - En développement, l'e-mail d'alerte part dans les logs comme les autres.
- [x] **21. Échecs répétés d'envoi** (ADR 0015, point 15). Trois envois échoués en une heure, vérification du démarrage comprise : un enregistrement `alert=smtp_failures`, sans e-mail (D6), au plus un par six heures. Compteur en mémoire (D9).
- [x] **22. Scénarios et documentation du lot** : définitions d'étapes godog des scénarios de l'étape 16 (l'évaluation est appelée par l'étape, avec l'horloge du test) ; `TestLimitsSurviveRestart` étendu aux compteurs ; glossaire ; `CLAUDE.md` ; `CHANGELOG.md` ; feuille de route (point reporté « alerte de D1 » retiré, relais des alertes inscrit au plan `recette`) ; cases cochées.

### Lot D : front à empreinte et archive (`feature/front-a-empreinte-et-archive`)

- [ ] **23. Noms de fichiers avec empreinte** (ADR 0012, point 2 ; D10).
  - `web/scripts/build.mjs` : `main.ts` et `app.css` deviennent deux points d'entrée d'esbuild, nommés d'après leur contenu ; polices et fichiers d'identité passent par le chargeur de fichiers, et le CSS comme le code y font référence par le nom produit. Sourcemaps liées, publiées (ADR 0011, point 8). Le texte de l'OFL reste copié à côté des polices (ADR 0011, point 3).
  - `index.html` est écrit par le build à partir de `web/src/index.html`, avec les noms produits ; les marques du gabarit Go (`{{.Base}}`, `{{.SourceURL}}`) sont conservées.
  - Le build écrit la liste des fichiers produits (`metafile` d'esbuild), qui servira au pré-cache du service worker.
  - En mode surveillance (`make dev`), les noms restent fixes : on recharge la page, rien de plus.
  - Si un nom de police avec crochets (`Figtree[wght].woff2`) gêne esbuild, renommer le fichier produit, pas la source ; le signaler dans les notes.
- [ ] **24. En-têtes de cache**, `internal/server`.
  - Fichier de la liste : `Cache-Control: public, max-age=31536000, immutable`. Page d'entrée et tout autre fichier : `Cache-Control: no-cache`. L'API garde `no-store`.
  - La liste elle-même n'est pas servie.
  - Tests en tableau de cas sur un front factice (`fstest.MapFS`) ; le parcours Playwright, qui construit le front, prouve que la page charge ses fichiers sous leurs nouveaux noms, sans erreur de CSP.
- [ ] **25. Archive** (ADR 0012, point 3).
  - `make build` construit pour `linux/amd64`, quelle que soit la machine.
  - Outil `internal/tools/dist` (ADR 0010 : rien de plus long que deux lignes dans le Makefile) : archive `tribe-menus-<version>-linux-amd64.tar.gz` avec le binaire, `site/` et `deploy/`, et son empreinte SHA-256 dans un fichier voisin. Entrées triées, dates, propriétaires et droits fixés : deux exécutions sur le même commit donnent la même empreinte.
  - Cible `make dist`, ajoutée à la fin de `make ci`.
  - Test : contenu et droits des entrées, empreinte stable d'une exécution à l'autre.
- [ ] **26. Notes de version** : outil du projet qui extrait de `CHANGELOG.md` la section d'une version, en fonction pure testée en tableau de cas (section absente, dernière section, « Non publié » refusé).
- [ ] **27. Documentation du lot** : `CLAUDE.md` (`make dist`, noms à empreinte) ; `CHANGELOG.md` ; cases cochées.

### Lot E : workflows et procédure de release (`feature/chaine-de-release`), poussé depuis l'hôte

- [ ] **28. `ci.yml`** : l'artefact d'une PR devient l'archive et son empreinte (ADR 0016, points 6 et 7), à la place de `bin/tribe-menus` et `site/`.
- [ ] **29. `release.yml`** (ADR 0012, point 5 ; ADR 0013).
  - Déclenché par une étiquette `v*`, et à la demande pour un essai à blanc qui ne publie rien.
  - Contrôles avant tout : l'étiquette a la forme `vX.Y.Z`, son commit appartient à `main`, `CHANGELOG.md` a une section pour cette version.
  - Deux jobs. Le premier, jeton en lecture seule, lance `make tools` puis `make ci` avec la version de l'étiquette et dépose l'archive. Le second, seul à avoir `contents: write`, crée la release avec l'archive, son empreinte et les notes extraites : il n'exécute aucun code du dépôt ni des dépendances.
  - Actions de GitHub seules, épinglées par empreinte ; `gh` pour créer la release ; aucun secret (`docs/securite-depot.md`, section 4). `make lint` (actionlint) passe.
- [ ] **30. `RELEASING.md`** (ADR 0012, point 5 ; ADR 0022).
  - Release ordinaire : PR `release/vX.Y.Z`, recette de son archive, fusion, étiquette annotée, release publiée, vérification de l'empreinte. Le déploiement est une action distincte, renvoyée à la documentation de `recette`.
  - Version corrective préparée dans la PR du correctif (`docs/traitement-des-vulnerabilites.md`, niveaux accéléré et urgent) : ce que la branche contient, ce que la version emporte de `main`, termes neutres.
  - Migrations signalées dans les notes de version (ADR 0012, point 6).
  - Workflows planifiés désactivés par GitHub après 60 jours sans activité : comment les réactiver (plan `ci`, notes d'exécution).
- [ ] **31. Essai à blanc**, par le développeur : `release.yml` lancé à la demande sur `main` ; l'archive déposée est téléchargée, son empreinte vérifiée, `tribe-menus version` affiche la version attendue. Résultat dans les notes d'exécution.
- [ ] **32. Clôture** : `CLAUDE.md` (« `release.yml` viendra avec le déploiement » retiré), `docs/securite-depot.md` si un réglage a changé, statut « terminé », feuille de route, `CHANGELOG.md`.

## Critères de validation

- `make ci` passe à la fin de chaque lot ; aucun scénario nouveau ne reste dans `acceptance/pending.txt`.
- **Modes** : `serve` sans mode, avec les deux, ou `-dev` avec un socket transmis, refuse de démarrer. Un fichier de configuration avec une clé inconnue ou une clé manquante est refusé, et le message nomme la clé.
- **Socket** : sur un socket Unix transmis, `/healthz` répond, les logs sont du JSON sur la sortie standard, et le fichier du socket existe encore après l'arrêt.
- **Adresse IP** : `X-Forwarded-For` n'a d'effet que sur le socket de systemd ; deux adresses IPv6 du même /64 partagent une limite, deux /64 voisins non.
- **E-mail** : le serveur factice reçoit un message conforme ; les adresses du constat 7 sont refusées avant tout envoi ; aucune valeur saisie ne peut ajouter un en-tête ni un destinataire.
- **Secret** : le mot de passe SMTP ne se trouve ni dans une option, ni dans l'environnement, ni dans le fichier de configuration, ni dans un log, y compris après un échec d'authentification.
- **Logs** : après des envois réussis et échoués, les logs du mode serveur ne contiennent ni adresse e-mail, ni code, ni jeton.
- **Alertes** : les seuils de D8 et de D12 déclenchent un e-mail à l'administrateur et un enregistrement `alert` ; rien en dessous ; une seule alerte par nature en six heures ; compteurs inchangés après un redémarrage ; l'e-mail ne nomme ni adresse ni tribu.
- **Base de limitation** : elle ne contient toujours ni adresse ni IP en clair, et rien de ce qu'elle garde ne dépend de l'appartenance d'une adresse à une tribu, codes fantômes compris (D11).
- **Front** : modifier une source change le nom du fichier produit ; un fichier à empreinte est servi avec `immutable`, la page d'entrée avec `no-cache`.
- **Archive** : deux `make dist` sur le même commit donnent la même empreinte ; l'archive contient le binaire `linux/amd64`, `site/` et `deploy/`.
- **Release** : l'essai à blanc de `release.yml` dépose une archive vérifiable ; le job qui lance `make ci` n'a aucun droit d'écriture.
- **Feuille de route** : les quatre points reportés à `production` en sont retirés ; ce que ce plan remet à `recette` y est inscrit.

### Ce que ce plan ne prouve pas

À vérifier en recette, et à inscrire dans le plan `recette` :

- l'activation de socket par un vrai systemd, et les credentials déchiffrés par `systemd-creds` ;
- ce que Caddy écrit réellement dans `X-Forwarded-For` (D3 suppose que son adresse vient en dernier) ;
- l'envoi par le compte `no-reply@` du MX Plan, et la réception hors des indésirables (ADR 0014, point 6) ;
- le relais des enregistrements `alert` du journal par `msmtp` ;
- la première release réelle, par une étiquette.

### Ce que ce plan demande à `recette`

- **Unité systemd** : `ExecStart=… serve -config /etc/tribe-menus/<environnement>/config.json` ; credential `smtp-password` ; un seul socket transmis ; `TimeoutStopSec` d'au moins 45 secondes (10 d'attente des requêtes, 30 d'un envoi en cours).
- **Caddy** : son adresse de client en dernière position de `X-Forwarded-For`, `Host` transmis tel quel.
- **Alertes** : un relais qui envoie par `msmtp` tout enregistrement du journal de l'application portant l'attribut `alert` (ADR 0023) ; l'adresse de l'administrateur dans la clé `alerts.to` du fichier de configuration.
- **Archive** : `deploy/` au complet.
- **E-mail** (revue de la PR #46, points 2 et 3) : lire les réponses réelles du MX Plan à un destinataire refusé, et ne garder que les codes si elles citent la partie locale ; vérifier que `EHLO localhost` ne pénalise pas la réception hors des indésirables.

## Questions ouvertes

- **Blocage ciblé au rythme exact de la limite** et **ce que la base de limitation sait déjà des membres** : questions du brouillon, tranchées le 2026-10-06 par le développeur (D12 et D11).
- **Seuils de D8** : valeurs proposées sans mesure. À revoir après les premières semaines de recette, où tout faux positif sera visible.
- **Nouvel essai d'envoi** : aucun en V1 ; le membre dispose de « Je n'ai rien reçu : renvoyer un code ». À reprendre si les logs de recette montrent des échecs passagers.
- **Adresses hors ASCII** (D5) : refusées. À rouvrir si un membre en a une, avec la prise en charge de SMTPUTF8 par le MX Plan à vérifier d'abord.
- **Essais sur un code fantôme de membre** (lot C, étape 16) : avec D11, le code fantôme d'un membre n'est jamais vérifié, celui d'une autre adresse l'est à chaque essai depuis le navigateur de la demande. Après un tel essai, ses essais restants le distinguent, pour qui lit la base de limitation ; avant, rien. Inscrit comme limite dans l'ADR 0023. Pour la lever : reporter aussi sur le code fantôme chaque essai d'un membre, un code accepté comptant comme un essai erroné, au prix d'une transaction de plus par essai et d'un compte des codes épuisés à prendre dans une seule base. À trancher par le développeur.
- **Nombre de PR** : cinq lots, là où `socle` en avait trois. A et B peuvent se fondre si le développeur préfère moins de PR ; D et E restent séparés à cause des workflows.

## Décisions

D1 à D10 proposées par la session qui a écrit le plan, confirmées telles quelles par le développeur le 2026-10-06 ; D11 et D12 tranchées par lui le même jour, à partir des deux premières questions ouvertes du brouillon.

- **D1. Fichier de configuration en JSON, lu strictement.** La bibliothèque standard le lit sans dépendance, et une clé mal orthographiée arrête le démarrage au lieu d'être ignorée. Le fichier vit sur le serveur (`/etc/tribe-menus/<environnement>/`, ADR 0015, point 9), ce qui garde hors du dépôt les valeurs propres à l'instance (ADR 0011, point 5). Écartés : TOML ou YAML (une dépendance) ; des options dans l'unité systemd, versionnée dans `deploy/` (les valeurs de l'instance entreraient dans le dépôt) ; des variables d'environnement (l'ADR 0015 les écarte pour les secrets, et deux canaux de configuration valent moins qu'un).
- **D2. Deux modes exclusifs, `-dev` ou `-config`.** L'écoute sur le socket et l'envoi SMTP ne s'expriment que dans le fichier de configuration : le constat 8 est tenu par construction, et le mode à moitié configuré d'aujourd'hui (ni développement, ni envoi) disparaît. Écarté : garder les options et ajouter un contrôle par combinaison dangereuse, liste qu'il faudrait tenir à jour.
- **D3. Derrière le proxy, l'adresse du client est la dernière de `X-Forwarded-For`.** C'est celle qu'écrit Caddy ; ce qui la précède vient du client et ne vaut rien. La règle tient quelle que soit la configuration de confiance de Caddy, tant qu'aucun autre proxy n'est placé devant lui. « Derrière le proxy » se déduit de l'écoute sur le socket de systemd, que seul le groupe de Caddy peut ouvrir (ADR 0015, point 11) : pas de liste de proxys de confiance à configurer. Écartées : la première adresse de l'en-tête (forgeable) ; une clé de configuration dédiée (un réglage de sécurité de plus, qu'on peut activer en TCP par erreur).
- **D4. Une adresse IPv6 compte pour son préfixe /64**, taille de ce qu'un fournisseur attribue au moins à un abonné. Sans cela, la limite par IP se contourne en changeant d'adresse dans son propre préfixe. Limite assumée : des abonnés d'un même /64 partagent la limite, comme ceux d'une même IPv4 derrière un NAT.
- **D5. Adresse e-mail : ASCII, sans guillemets ni commentaires**, vérifiée par une règle du projet puis recoupée par `net/mail`. Tout ce que la règle accepte s'écrit tel quel dans un en-tête et dans l'enveloppe. Écartés : s'en remettre à `net/mail` seul (il accepte les noms, les groupes et les parties locales entre guillemets) ; accepter les adresses internationalisées (prise en charge par le MX Plan inconnue).
- **D6. Une alerte est un enregistrement du journal ; celle des limites part aussi par e-mail depuis l'application.**
  - Toute alerte est écrite dans le journal avec un attribut `alert`, et le serveur relaie ces enregistrements par `msmtp`, avec le compte `server@` (ADR 0015, points 6 et 15 ; plan `recette`). Un seul mécanisme, qui fonctionne encore quand le compte d'envoi de l'application est en panne.
  - L'alerte sur les limites est en plus envoyée par l'application à l'administrateur, comme le demandent ENF-01 et `revue-securite`, D1 : elle n'attend pas le relais du serveur.
  - L'alerte sur les échecs d'envoi ne part pas par l'application : son envoi est justement ce qui échoue.
  - Écartés : tout envoyer par l'application (muette quand le SMTP tombe) ; tout laisser au serveur (ENF-01 lie l'alerte à l'envoi réel, apporté par ce plan) ; lancer `msmtp` depuis l'application (un programme externe exécuté par un service confiné).
- **D7. Des compteurs par tranche de dix minutes, sans empreinte, chacun dans la base qui constate l'événement.**
  - Une ligne par tranche et par nature : la taille ne dépend pas du nombre de demandes refusées. Le socle n'enregistre pas les demandes refusées pour borner la base de limitation ; ce choix est gardé.
  - Les événements qui supposent un membre (limite de la tribu, code réel épuisé) sont comptés dans la base de la tribu. Comptés dans la base de limitation, à côté de demandes horodatées, ils laisseraient deviner quelles empreintes sont celles de membres, ce que l'ADR 0021 (point 4) veut éviter (voir la deuxième question ouverte).
  - Les compteurs survivent au redémarrage, comme les limites (`socle`, D3).
  - Écarté : déduire l'alerte des seules demandes déjà enregistrées. C'est sans table de plus, mais les codes épuisés n'y laissent aucune trace, et ce sont eux qui signalent la force brute (D8).
  - L'ADR 0021 (point 4) n'était pas tenu avant ce plan : voir D11.
- **D8. « Limites atteintes de façon répétée » : sur une heure et pour toute l'instance, dix demandes refusées, ou cinq codes invalidés par essais épuisés.** Une alerte par nature et par six heures au plus, tant que la situation dure. Constantes du code, pas de la configuration (`socle`, D15 : pas de paramètre de sécurité réglable au lancement).
  - Les trois attaques du plan `revue-securite` ne laissent pas la même trace. Une force brute menée au rythme des limites n'est jamais refusée : elle épuise des codes, douze par heure et par adresse visée. Une inondation est refusée. Un blocage ciblé au rythme de la limite ne laisse que les refus subis par la victime (première question ouverte).
  - Un membre qui se trompe trois fois épuise un code ; cinq en une heure sur une instance de quelques tribus n'arrivent pas par maladresse.
  - L'e-mail ne nomme ni adresse, ni IP, ni tribu : les empreintes ne se relisent pas, et les journaux d'accès de Caddy donnent l'heure, les IP et les URL. Écarté : nommer la tribu visée, ce qui écrirait dans une boîte aux lettres qu'elle existe et qu'on s'en prend à ses membres.
- **D9. Les échecs d'envoi se comptent en mémoire.** C'est un état de supervision, pas de sécurité : un redémarrage retarde l'alerte, et chaque échec reste dans le journal. Les compter dans la base de limitation y daterait des envois, donc des demandes faites pour des membres (ADR 0021). Seuil : trois échecs en une heure.
- **D10. Noms fixes en développement, noms à empreinte dans tout build.** `make dev` recharge la page sans rien d'autre ; `make e2e`, qui construit le front, exerce le chemin de la production. Écarté : des noms à empreinte partout, qui obligeraient le serveur de développement à relire la liste à chaque requête.
- **D11. Un code fantôme pour toute demande, membre ou non** (étape 17). Avant ce plan, la base de limitation n'écrit un code fantôme que pour une adresse qui n'est pas membre active : une demande enregistrée sans code fantôme au même instant désigne, pendant les dix minutes de validité du code, l'empreinte d'un membre, à qui lit le fichier. Écrire le code fantôme pour toute demande rend les deux cas identiques dans cette base, comme le veut l'ADR 0021 (point 4), pour le prix d'une ligne par adresse demandée, effacée avec les autres. Gravité faible selon la grille de `docs/traitement-des-vulnerabilites.md` (accès au serveur requis, rien en production) : flux public. Écarté : l'écrire comme une limite assumée.
- **D12. Troisième signal d'alerte : une même empreinte d'adresse à neuf demandes ou plus dans l'heure** (étape 20). Il couvre le blocage ciblé mené au rythme exact de la limite (trois demandes par quart d'heure, douze par heure), qui ne produit ni refus ni code épuisé. Il se calcule sur les demandes que la base de limitation garde déjà, sans donnée de plus, et ne dépend pas de l'appartenance de l'adresse une fois D11 en place. Mêmes règles que D8 : constante du code, une alerte par six heures au plus, ni adresse, ni IP, ni tribu dans l'e-mail. Neuf demandes en une heure pour une même adresse n'arrivent pas par maladresse.

## Notes d'exécution

- **2026-10-06, mise à jour du plan** : décisions D1 à D10 confirmées par le développeur ; les deux premières questions ouvertes deviennent D11 (code fantôme pour toute demande) et D12 (troisième signal d'alerte), au lot C, qui gagne une étape : les étapes 17 à 31 du brouillon deviennent 18 à 32. Chemin des migrations corrigé (`internal/storage/migrations/`). Lots exécutés un à un : le suivant attend la fusion du précédent.
- **Lot A** (`feature/execution-sous-systemd`), 2026-10-06, Claude Code dans le Dev Container :
  - Étapes 2 à 4 en un seul commit : elles réécrivent la même fonction `serve`.
  - Port `0` accepté dans `listen` (port choisi par le système) : utile aux tests, sans effet sur un serveur.
  - Erreurs du fichier de configuration écrites sur la sortie d'erreur avec le code de sortie 2, avant la création du journal JSON : journald les recueille aussi.
  - Le front embarqué est vide quand `make test` tourne en CI : les tests du mode serveur remplacent `web.Dist` par un `fstest.MapFS` (variable `embeddedWeb`).
  - Le socket transmis par systemd est simulé en plaçant le descripteur d'un socket Unix créé par le test (`systemdFirstFD`, 3 hors des tests). Le test prouve aussi, de bout en bout, que la limite par IP suit la dernière adresse de `X-Forwarded-For` sur ce socket.
  - Hors du socket de systemd, `serve -config` en TCP ignore `X-Forwarded-For`, comme `-dev` (D3).
  - ADR 0015 : point 9 précisé ici ; le point 10 (mot de passe SMTP en credential) le sera au lot B, avec sa lecture.
  - Sans en-tête lisible derrière le proxy, l'avertissement ne cite pas la valeur reçue.
  - Revue de la PR #45 : un socket transmis par systemd qui n'est pas un socket Unix (unité écrite avec `ListenStream=8080`) est refusé, sans quoi n'importe qui atteignant le port écrirait `X-Forwarded-For` (point 1) ; le test du socket ne ferme plus deux fois le même descripteur (point 2). Point 3, à noter : une clé en double ou dans une autre casse est acceptée par `encoding/json` (`{"listen":"systemd","listen":"0.0.0.0:80"}` est lu avec la seconde valeur) ; à refuser au lot B, étape 11, quand le fichier reçoit les clés `smtp.*`.
- **Lot B** (`feature/envoi-smtp`), 2026-10-06, Claude Code dans le Dev Container :
  - Étape 8 : la règle refuse aussi l'accent grave (`` ` ``) de la RFC 5322, absent de la liste de D5, et l'antislash ; une adresse refusée garde le message du front « Cette adresse ne semble pas complète ». Scénario « Une adresse que l'e-mail ne saurait pas porter est refusée ». `chloé@exemple.fr`, acceptée jusqu'ici, est refusée (D5).
  - Étape 9 : `compose` refuse en plus toute adresse qui n'est pas nue, et un sujet ou un identifiant contenant une fin de ligne : défense en profondeur derrière `ParseEmail`.
  - Étape 10 : erreurs typées `SendError` (étape, code SMTP). L'échec de `QUIT` après un `DATA` accepté n'est pas une erreur.
  - Étape 11 : `smtp.from` doit être une adresse acceptée par `ParseEmail`, en minuscules ; `smtp.username` en ASCII imprimable. Le contrôle des clés (revue de la PR #45, point 3) parcourt le document une seconde fois avec `json.Decoder.Token` : clé en double ou d'une autre casse refusée, clés de `smtp` comprises. Fin de ligne finale du credential retirée (`\n` ou `\r\n`), espaces gardés. Erreur de credential sur la sortie d'erreur avec le code 2, comme celles du fichier de configuration.
  - Étape 13 : la réponse du serveur est gardée après remplacement de l'adresse du destinataire et du mot de passe, quelle que soit la casse ; `mail.ErrorAttrs` donne l'étape et le code aux enregistrements « login code not sent » et « SMTP check failed », que le lot C comptera.
  - Revue de la PR #46 : à l'étape d'authentification, la réponse du serveur ne garde que son code SMTP, son texte pouvant citer la commande `AUTH PLAIN`, dont le base64 porte le mot de passe (point 1). À noter pour la recette : le remplacement ne reconnaît que l'adresse entière du destinataire, une réponse qui cite la partie locale seule (`user alice.martin unknown`) passe telle quelle ; si le MX Plan répond ainsi, ne garder que le code SMTP et le code d'état étendu (`5.1.1`) (point 2). Le client se présente par `EHLO localhost`, valeur par défaut de `net/smtp`, qui apparaît dans l'en-tête `Received` ; `Client.Hello` avec l'hôte de `baseURL` si un filtre s'en formalise (point 3).
  - Étape 14 : non faite par Claude Code, faute de compte. Vérifié depuis le Dev Container le 2026-10-06 : le port 465 de `ssl0.ovh.net` est joignable, son certificat (TLS 1.2) porte le nom `ssl0.ovh.net` et passe la vérification de Go. Le 2026-10-06, le développeur renvoie l'essai d'un envoi réel à la recette, avec la réception sur les messageries des membres (« Ce que ce plan ne prouve pas »).
- **Lot C** (`feature/enf-01-alertes`), 2026-10-06, Claude Code dans le Dev Container :
  - Étape 16 : un signal d'alerte par seuil (demandes refusées, codes épuisés, demandes répétées), chacun avec sa dernière alerte : « une alerte par nature et par six heures » s'entend par signal ; un même e-mail nomme tous les signaux dus à une évaluation. Glossaire complété dès cette étape, avant le code. Une limite de D11 relevée et inscrite dans l'ADR 0023 et les questions ouvertes : les essais restants d'un code fantôme le distinguent, après un essai, de celui d'un membre, jamais vérifié.
  - Étape 18 : la valeur des natures et des signaux est contrainte par un `CHECK` dans chaque table. La dernière alerte par signal n'est pas effacée : trois lignes au plus.
  - Étape 19 : `CodeRequestCounts.Allowed` devient `Refusal`, qui nomme la limite ; une demande refusée par les deux limites compte une fois, par adresse.
  - Étape 20 : l'alerte est notée dans la base de limitation avant d'être écrite et envoyée, pour qu'un envoi échoué ne la répète pas toutes les dix minutes ; l'enregistrement du journal reste. Une tribu illisible n'empêche pas l'évaluation des autres. Nouveau paquet `internal/alert` (notificateur, texte de l'e-mail, échecs d'envoi), `Outbox.SendAlert`. En développement, l'administrateur est `admin@example.org` et l'e-mail part dans les logs.
  - Étape 21 : chaque échec d'un envoi, code ou alerte, est compté par un `Mailer` qui enveloppe celui du SMTP ; le test du mode serveur en TCP provoque l'alerte (vérification du démarrage et deux codes non envoyés) et vérifie qu'aucune adresse n'est dans les logs.
  - Le 2026-10-06, le développeur confirme deux interprétations de l'étape 16 et de l'étape 20 : une alerte par signal et par six heures, un même e-mail nommant tous les signaux dus ; l'e-mail distingue les codes de membres épuisés des codes fantômes épuisés.
  - Étape 22 : les étapes godog appellent `CheckAlerts` avec l'horloge du scénario ; elles détectent un seuil faussé (essai fait en passant le seuil des refus à 11).
