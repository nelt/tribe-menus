# `deploy/`

Ce qui sert à faire tourner l'application sur un serveur (ADR 0015, 0016). Le dossier est livré dans l'archive de chaque version (ADR 0012) : le serveur se provisionne depuis l'archive qu'il fait tourner.

Rien de propre à l'instance ici (ADR 0011, point 5) : les exemples utilisent `example.org`, les vraies valeurs vivent dans un fichier du serveur (plan `recette`, D4), les secrets dans des credentials chiffrés du serveur (D5).

## Contenu

| Fichier | Rôle | Installé dans |
| --- | --- | --- |
| `provision.sh` | Provisionnement du serveur, idempotent (ADR 0015, point 2). | lancé depuis l'archive |
| `instance.example.conf` | Modèle des valeurs de l'instance. | `/etc/tribe-menus/instance.conf`, à la main, une fois |
| `bin/tribe-menus-credential` | Saisie d'un secret au clavier, chiffré par `systemd-creds`. | `/usr/local/sbin/` |
| `bin/tribe-menus-admin` | Sous-commande `admin` sous l'utilisateur du service (ADR 0015, point 13). | `/usr/local/sbin/` |
| `bin/tribe-menus-deploy` | Déploiement d'une version ou d'une PR, instantané, retour arrière (ADR 0016). | `/usr/local/sbin/` |
| `bin/tribe-menus-maintenance` | Mise en attente d'un environnement, et sa levée. | `/usr/local/sbin/` |
| `bin/tribe-menus-alert` | Alertes du serveur : relais de celles de l'application, panne, disque (ADR 0015, point 15). | `/usr/local/sbin/` |
| `bin/tribe-menus-check` | Contrôle du serveur, rapport de l'heure, exercice des alertes (D10). | `/usr/local/sbin/` |
| `systemd/tribe-menus-alert-relay.{service,timer}`, `systemd/tribe-menus-disk.{service,timer}`, `systemd/tribe-menus-failure@.service` | Minuteurs du relais (toutes les cinq minutes) et du disque (chaque jour), alerte de panne (`OnFailure=`). | `/etc/systemd/system/` |
| `systemd/tribe-menus@.socket`, `systemd/tribe-menus@.service` | Socket et service de chaque environnement (`tribe-menus@production`, `tribe-menus@recette`). | `/etc/systemd/system/` |
| `caddy/attente.html` | Page d'attente, servie en 503 quand l'application ne répond pas. | `/usr/share/tribe-menus/` |
| `apt/caddy-stable.asc` | Clé de signature du dépôt apt du projet Caddy (D15). | `/etc/apt/keyrings/` |
| `config.example.json` | Exemple du fichier de configuration de `tribe-menus serve -config` (plan `production`, D1), lu par un test avec le vrai lecteur. Sur le serveur, `provision.sh` produit ce fichier. | — |

Le contrôle externe, lancé du Dev Container, est un outil du projet : `make probe` (`internal/tools/probe`, voir « Contrôles »).

## Valeurs de l'instance

`/etc/tribe-menus/instance.conf` se copie depuis `instance.example.conf` et se remplit avec les vraies valeurs : compte d'administration, adresse de l'administrateur, noms d'hôte, serveur et comptes SMTP de l'application et du serveur. Une ligne `CLÉ=valeur`, sans guillemets ni espaces ; toutes les clés sont obligatoires.

`provision.sh` lit ce fichier comme une donnée, jamais comme du shell : chaque valeur doit correspondre au motif de sa clé (nom d'hôte, adresse e-mail en minuscules, nom de compte, port), ce qui exclut guillemets, espaces, `$` et barres obliques inverses ; une clé inconnue, manquante ou écrite deux fois est refusée, l'erreur nomme la ligne et la clé, jamais la valeur. Il en tire, à chaque lancement :

| Fichier produit | Contenu |
| --- | --- |
| `/etc/tribe-menus/<environnement>/config.json` | Configuration de `serve -config` : données, socket, `baseURL`, compte SMTP de l'application, `alerts.to`. |
| `/etc/caddy/Caddyfile` | Un bloc par nom d'hôte. |
| `/etc/msmtprc`, `/etc/tribe-menus/aliases` | Relais d'envoi du serveur ; le courrier de `root` va à l'administrateur. |

Ces fichiers ne se retouchent jamais à la main : on corrige `instance.conf` ou le script, puis on relance `provision.sh`. `provision.sh --render <dossier> --instance <fichier> production recette` les écrit dans un dossier, sans `root` ni rien toucher, pour les relire ; un test (`internal/config/provision_test.go`) fait lire les `config.json` produits par le vrai lecteur.

## Première installation

C'est la séance S2 du plan `recette`. Depuis l'hôte (pas le Dev Container), avec l'archive d'une PR ou d'une version :

```sh
gh run download <identifiant de l'exécution> -n tribe-menus-pr-<numéro>
sha256sum -c tribe-menus-<version>-linux-amd64.tar.gz.sha256
scp tribe-menus-<version>-linux-amd64.tar.gz root@<serveur>:
```

Sur le serveur, connecté en `root` (seul compte de l'image PulseHeberg, ADR 0025) :

```sh
tar -xzf tribe-menus-<version>-linux-amd64.tar.gz
install -d -m 0755 /etc/tribe-menus
cp tribe-menus-<version>-linux-amd64/deploy/instance.example.conf /etc/tribe-menus/instance.conf
nano /etc/tribe-menus/instance.conf
./tribe-menus-<version>-linux-amd64/deploy/provision.sh production recette
```

Le premier lancement crée le compte d'administration (`ADMIN_USER`), lui copie la clé SSH de `root` et demande son mot de passe, que `sudo` exigera (D14) : à ranger dans le gestionnaire de mots de passe. `root` peut encore se connecter, par clé seulement.

Ensuite, **dans un second terminal, sans fermer le premier** :

```sh
ssh <ADMIN_USER>@<serveur>
sudo /root/tribe-menus-<version>-linux-amd64/deploy/provision.sh production recette
```

Lancé par `sudo` depuis le compte d'administration, le script ferme SSH à `root` : la connexion de ce compte est prouvée par ce lancement même. Rien d'autre ne change. La fermeture ne se défait pas en relançant le script depuis `root`.

Le script se termine par ce qui manque encore : secrets, version installée.

## Secrets

Saisis au clavier sur le serveur, jamais collés dans une commande ni copiés par `scp` (D5) :

```sh
sudo tribe-menus-credential production smtp-password   # compte no-reply@ de l'application
sudo tribe-menus-credential recette smtp-password
sudo tribe-menus-credential server smtp-password       # compte server@ du relais msmtp
sudo tribe-menus-credential server github-token        # artefacts des PR, pour tribe-menus-deploy --pr
```

Le secret est lu deux fois sans écho, chiffré avec la clé de la machine (`systemd-creds --with-key=host`), vérifié par un déchiffrement, puis écrit en 0600 dans `/etc/tribe-menus/<environnement>/smtp-password.cred` ou `/etc/tribe-menus/server/<nom>.cred` (dossier en 0700). Il passe de `read` à `systemd-creds` par un tube, jamais par les arguments d'une commande. Le service de l'environnement est redémarré s'il tournait.

- Le service reçoit son mot de passe déchiffré par systemd, dans son seul `$CREDENTIALS_DIRECTORY` (`LoadCredentialEncrypted=`) : l'utilisateur du service ne peut pas lire le fichier chiffré.
- `msmtp` déchiffre le mot de passe de `server@` à chaque envoi (`passwordeval`) : seul `root` peut envoyer, ce que font `unattended-upgrades` et les alertes du serveur.
- Un serveur reconstruit a une autre clé de machine : les secrets se saisissent de nouveau.

## Tribu de recette

Une fois une version installée en recette (`tribe-menus-deploy --env recette`, voir « Déployer »), le fichier des membres (D3) s'écrit sur le serveur, une ligne par membre, adresse puis nom d'affichage, le premier étant le premier membre :

```sh
sudo tribe-menus-admin recette seed -members recette-membres.txt
```

Le script affiche les membres lus et demande confirmation : une tribu qui existe déjà ne change plus (revue de la PR #52). Le fichier peut rester dans le répertoire du compte d'administration : le script en remet une copie privée à l'utilisateur du service.

**Recréer la tribu de recette** (adresse mal saisie) : la recette ne contient que des données de démonstration (ADR 0016, point 5), on efface donc ses bases.

```sh
sudo systemctl stop tribe-menus@recette.socket tribe-menus@recette.service
sudo find /var/lib/tribe-menus/recette -mindepth 1 -delete
sudo systemctl start tribe-menus@recette.socket
sudo tribe-menus-admin recette seed -members recette-membres.txt
```

Jamais en production. `sudo tribe-menus-admin production init` crée une tribu, de façon interactive (EF-08).

## Ce que fait `provision.sh`

Dans l'ordre ; chaque fichier n'est réécrit, et chaque service redémarré, que s'il a changé.

1. **Paquets** : de Debian 13 (`sudo`, `tzdata`, `nftables`, `unattended-upgrades`, `msmtp-mta`, `sqlite3`, `jq`, `curl`, `unzip`, `ca-certificates`, et `systemd-timesyncd` si l'image n'a pas chrony) ; Caddy depuis le dépôt du projet Caddy (D15), signé par la clé de `apt/caddy-stable.asc`, d'empreinte `6576 0C51 EDEA 2017 CEA2  CA15 155B 6D79 CA56 EA34` (celle de la documentation de Caddy, vérifiée le 2026-10-17 sur la signature du dépôt). Une préférence apt interdit à ce dépôt tout autre paquet que `caddy`, et le préfère au paquet `caddy` de Debian, plus ancien.
2. **Heure et journal** : fuseau `Europe/Paris` (les redémarrages de 3 h 30 sont en heure locale), synchronisation de l'heure, journal persistant, plafonné à 2 Go et conservé un mois (PT-07).
3. **Mises à jour** : correctifs de sécurité et mises à jour de la version stable de Debian (motifs de `50unattended-upgrades` de Debian, auxquels s'ajoutent ceux du script), paquets du dépôt de Caddy, chaque jour. Les versions intermédiaires de Debian portent aussi les failles mineures que l'équipe de sécurité ne publie pas en avis (« no-DSA ») ; elles ne changent pas de version amont. Un paquet dont un fichier de configuration modifié sur le serveur change chez Debian (`/etc/nftables.conf`) n'est pas mis à jour seul : il attend une mise à jour manuelle, sans que rien ne soit écrasé ; rapport par e-mail à `root` quand quelque chose a changé, qui annonce aussi un redémarrage nécessaire, fait à 3 h 30 et qu'on annule par `shutdown -c` (ADR 0015, point 5).
4. **Relais `msmtp`** : compte `server@`, TLS dès la connexion, expéditeur imposé (`allow_from_override off`), courrier de `root` et de tout compte local vers l'administrateur. Clé de chiffrement des credentials créée (`systemd-creds setup`).
5. **Compte d'administration et SSH** : voir « Première installation ». Clé seule (`AuthenticationMethods publickey`), ni mot de passe ni clavier interactif, seul le compte d'administration admis (`AllowUsers`), pas de redirection de ports ni d'agent. Le fichier `/etc/ssh/sshd_config.d/00-tribe-menus.conf` est lu le premier, et la première valeur lue l'emporte : il prévaut sur ceux de l'image. Il n'est gardé que si `sshd -t` l'accepte, sinon l'ancien est remis ; ce que `sshd -T` applique est relu. Le script signale une règle `sudo` sans mot de passe (`NOPASSWD`) laissée par l'image, sans la retirer.
6. **Pare-feu** (`/etc/nftables.conf`) : en entrée, TCP 22, 80 et 443, en IPv4 comme en IPv6 ; ICMP utile (inaccessible, délai dépassé, problème de paramètre, écho) et ICMPv6 (de même, plus paquet trop gros, découverte des voisins, limitée au lien par `hoplimit 255`, et requêtes MLD du routeur, venant d'une adresse de lien, RFC 4890) ; réponses par le suivi de connexion ; aucun port UDP (D16), les réponses DNS et NTP passant par le suivi de connexion ; transfert refusé. Le jeu de règles est vérifié (`nft --check`) puis remplacé d'un bloc par `systemctl reload nftables` (`nft -f`, une seule transaction), jamais par `restart`, dont l'arrêt vide les règles avant de les recharger : la session SSH en cours est gardée.
7. **Scripts et unités** : les scripts de `bin/`, page d'attente, socket et service, unités et minuteurs de la supervision (« Supervision »), lancés.
8. **Chaque environnement** (D8 ; ADR 0015, points 8 et 9) :

   | Emplacement | Droits | Contenu |
   | --- | --- | --- |
   | utilisateur `tribe-menus-<environnement>` | système, sans shell ni répertoire | exécute le service |
   | `/opt/tribe-menus/<environnement>/releases/` | `root`, 0755 | versions installées ; liens `current` et `previous` à côté (`tribe-menus-deploy`) |
   | `/etc/tribe-menus/<environnement>/` | `root:tribe-menus-<environnement>`, 0750 | `config.json` en 0640, credential en 0600 à `root` |
   | `/var/lib/tribe-menus/<environnement>/` | `tribe-menus-<environnement>`, 0700 | registre et bases des tribus, en 0600 (`UMask=0077`) |
   | `/var/www/tribe-menus/<environnement>` | lien | vers `/opt/tribe-menus/<environnement>/current/site` : le site est celui de la version en service, et change avec elle, retour arrière compris |

   Une instance ne lit rien de l'autre : données et configuration n'appartiennent qu'à son utilisateur.
9. **Caddy** : configuration produite, validée (`caddy validate`, sous l'utilisateur `caddy`), puis Caddy redémarré (avec `admin off`, `caddy reload` n'a pas d'API à qui parler).

## Socket et service

Le socket `/run/tribe-menus/<environnement>.sock` appartient à `root:caddy`, en 0660 : seuls `root` et Caddy l'ouvrent, ce qui permet à l'application de croire le `X-Forwarded-For` des requêtes qui en viennent (ADR 0006, point 7 ; plan `production`, D3). Il **n'écoute qu'une fois une version, `config.json` et le credential en place** (conditions de l'unité) : sinon Caddy ne trouve aucun socket et sert la page d'attente, au lieu de garder la requête pendant que systemd échoue à lancer le service. Le service, lancé par le socket, vérifie les mêmes fichiers et nomme celui qui manque ; il redémarre sur échec, cinq fois en cinq minutes au plus. `TimeoutStopSec=45s` laisse au serveur ses dix secondes d'arrêt et aux envois de code en cours leur propre délai.

### Confinement du service

Note de `systemd-analyze security --offline=true` sur l'unité : **1,1** (« OK », sur une échelle où 0 est le mieux confiné), relevée le 2026-10-17 avec systemd 257 ; la note sur le serveur se relève en S2. Le service n'a aucun privilège ni capacité (`NoNewPrivileges`, `CapabilityBoundingSet=`), voit le système de fichiers en lecture seule hors de son répertoire de données (`ProtectSystem=strict`), ni `/home` ni les `/tmp` des autres, ni périphériques, ni processus des autres, ni réglages du noyau ; ses appels système sont limités à `@system-service`, moins `@privileged` et `@resources`, en architecture native ; il ne peut ni créer d'espace de noms, ni de mémoire écrivable et exécutable ; ses utilisateurs sont isolés (`PrivateUsers`).

Réglages laissés ouverts, et pourquoi :

| Point relevé | Raison |
| --- | --- |
| `PrivateNetwork=`, `RestrictAddressFamilies=` (IPv4, IPv6) | L'envoi des codes par SMTP et la résolution DNS du serveur SMTP demandent le réseau. |
| `IPAddressDeny=` | Les adresses du serveur SMTP du MX Plan ne sont pas fixes ; pas de filtrage des sorties en V1 (plan `recette`, questions ouvertes). |
| `RestrictAddressFamilies=` (Unix) | Le socket transmis par systemd et le journal. |
| `RootDirectory=` | Le binaire est statique, mais un système de fichiers racine propre au service n'apporte rien de plus que `ProtectSystem=strict` et les droits des dossiers. |
| `DeviceAllow=` | Accès ajouté par systemd pour `LoadCredentialEncrypted=` (TPM, horloge) ; le service ne voit aucun périphérique (`PrivateDevices`). |

## Caddy

Configuration produite par `provision.sh` (ADR 0006 ; plan `recette`, étape 12, constat 11 de `revue-securite`) :

- **Certificats** : Let's Encrypt seul, l'autorité que le CAA du domaine autorise, sans le repli de Caddy vers ZeroSSL ; adresse de contact ACME : celle de l'administrateur.
- **Serveur** : API d'administration coupée, HTTP/1.1 et HTTP/2 seuls (D16), délais de lecture des en-têtes (10 s) et du corps (20 s), d'écriture (30 s) et d'inactivité (2 min), en-têtes plafonnés à 32 Ko, corps des requêtes à 64 Ko.
- **Un bloc par nom d'hôte**, chacun vers le socket de son environnement :
  - `/tribes/*` seul est transmis à l'application, `Host` tel quel ; Caddy ne fait confiance à aucun proxy et remplace `X-Forwarded-For` par l'adresse du client (vérifié le 2026-10-17 avec Caddy 2.11 : onze demandes de code portant chacune une adresse inventée, la onzième est refusée) ;
  - `/healthz` n'est jamais transmis : il tombe sur le site, en 404 ;
  - le reste est le site public de l'environnement (D8), avec sa CSP (aucun script, styles, images et polices du site seulement), `nosniff`, `Referrer-Policy` et `Cross-Origin-Opener-Policy` ; sans site installé, la page d'attente.
- **HSTS** sur les deux noms d'hôte, une journée pour commencer (à allonger après quelques semaines sans incident) ; en-tête `Server` retiré.
- **Recette** : `X-Robots-Tag: noindex, nofollow` sur tout le nom d'hôte, `robots.txt` qui interdit tout.
- **Page d'attente** en 503, avec `Retry-After`, quand le socket ne répond pas (aucune version, mise en attente) ou qu'aucun site n'est installé.
- **Journaux d'accès** dans le journal de `caddy.service` : heure, adresse IP, URL, ce que l'e-mail d'alerte promet (ADR 0023, point 6), sans les en-têtes `Cookie`, `Authorization` ni `Set-Cookie`.

## Déployer

`tribe-menus-deploy` installe l'application et son site, rien d'autre (D6). Une seule exécution à la fois sur le serveur.

```sh
sudo tribe-menus-deploy v1.2.3                        # production : une version publiée
sudo tribe-menus-deploy --env recette v1.2.3          # recette : une version publiée
sudo tribe-menus-deploy --env recette --pr 57         # recette : l'archive de la PR 57
```

- **Une version** (`vX.Y.Z`) est l'archive de la release publiée, lue sans jeton.
- **Une PR** n'est déployée qu'en recette (refusée en production, ADR 0016, point 4), et seulement si elle est ouverte, écrite par le propriétaire du dépôt, depuis une branche du dépôt et non d'un fork ; l'archive est l'artefact `tribe-menus-pr-<numéro>` de l'exécution réussie de `ci.yml` sur son **dernier commit** : une CI pas encore finie, en échec ou expirée (trois jours) est refusée, et le dit. Le jeton du serveur passe à `curl` par un tube, jamais par une ligne de commande, l'environnement ou un fichier en clair ; l'adresse de téléchargement, une redirection vers une URL signée, est suivie sans lui.
- **Vérifications** avant toute installation : l'empreinte SHA-256 du fichier `.sha256` (pour une PR, elle ne prouve que le transfert, les deux venant du même artefact) ; le contenu de l'archive, qui ne doit avoir que des fichiers et des dossiers sous son dossier de tête, sans chemin absolu, `..`, lien ni caractère inattendu (test : `internal/deploy`) ; le binaire, lancé sous l'utilisateur du service, doit dire la version attendue.
- **Installation** dans `releases/<version>` (pour une PR, `pr-<numéro>-<commit>`), puis **instantané** des bases par `VACUUM INTO`, sous l'utilisateur du service, dans `/var/lib/tribe-menus/<environnement>/backups/avant-<version>/` : registre et chaque tribu, avec la version de schéma de chaque base (fichier `schema`) ; la base de limitation n'est pas copiée (ADR 0021, point 6), sa version est notée. L'instantané est pris sur la version en service : ce qu'elle écrit dans les secondes qui suivent, avant la bascule, ne s'y trouve pas.
- **Bascule** atomique du lien `current`, qui emmène le site avec lui, redémarrage, puis `/healthz` appelé par le socket, soixante secondes au plus.
- **Échec** (`/healthz` ne répond pas) : retour automatique à la version précédente, avec restauration de l'instantané si la version de schéma du registre ou d'une tribu a changé (aucune requête n'a été servie par la nouvelle version, rien n'est perdu, D7) ; le script le dit, montre les dernières lignes du journal et sort en erreur. Au premier déploiement, sans version précédente, le service reste arrêté et la page d'attente servie.
- **Interruption** : dès que l'environnement commence à changer (levée de la mise en attente, instantané, bascule), `Ctrl-C` et une session SSH perdue n'arrêtent plus le script, et toute erreur remet l'environnement comme il était : version précédente si le lien a bougé, mise en attente remise s'il y en avait une (revue de la PR #56). Il en va de même pour un retour arrière ou une restauration qui échoue ; une restauration en échec met l'environnement en attente, même s'il ne l'était pas.
- **Conservation** : cinq versions (dont celle en service et la précédente) et leurs instantanés.
- Chaque déploiement est écrit au journal (`journalctl -t tribe-menus-deploy`).

Un socket mis en échec par une version qui ne démarrait pas (`StartLimitBurst`, revue de la PR #55) est remis en route par chaque déploiement, retour arrière et levée d'une mise en attente.

## Revenir en arrière et restaurer

```sh
sudo tribe-menus-deploy --rollback                         # production
sudo tribe-menus-deploy --env recette --rollback
sudo tribe-menus-deploy --rollback --restore-data          # si le schéma a changé
```

`--rollback` remet la version précédente (`previous`). Si une migration a changé le schéma du registre ou d'une tribu depuis le déploiement de la version en service, la précédente refuserait les bases (D7) : le script refuse, dit la date de l'instantané et ce qui serait perdu, et ne fait rien. Avec `--restore-data`, il restaure l'instantané pris avant ce déploiement : tout ce qui a été écrit depuis est perdu. Rien n'est effacé : les bases remplacées, et les tribus créées depuis, sont mises de côté dans `backups/remplacees-<date>/`. Sans migration, les données sont gardées, et `--restore-data` est ignoré.

Une base de limitation dont le schéma a changé est mise de côté sans rien demander : la perdre remet les compteurs à zéro, rien d'autre.

Un seul retour arrière de suite : ensuite, `previous` est retiré ; pour revenir à la version quittée, on la déploie de nouveau.

**Instantané à la demande, et restauration d'un instantané** :

```sh
sudo tribe-menus-deploy --snapshot                          # backups/manuel-<date>/, la version en service tournant
sudo tribe-menus-deploy --restore avant-v1.2.3              # ou manuel-<date>
```

`--restore` remet le registre et les tribus d'un instantané de `backups/` sous la version en service, qui doit connaître son schéma (celle avant laquelle il a été pris, ou une plus récente : sinon elle ne répond pas, le script le dit et met l'environnement en attente). Les bases présentes sont mises de côté, comme pour un retour arrière. Les instantanés `manuel-*` ne sont jamais effacés par le script.

## Mettre en attente

Mesure d'attente du traitement des vulnérabilités (`docs/traitement-des-vulnerabilites.md`) :

```sh
sudo tribe-menus-maintenance production on       # page d'attente, en 503
sudo tribe-menus-maintenance production status
sudo tribe-menus-maintenance production off      # l'application répond de nouveau
```

`on` arrête le socket **et** le service, puis les masque : arrêter le service seul ne suffirait pas, la première requête le relancerait par le socket, et une unité masquée le reste après un redémarrage du serveur (celui de 3 h 30 compris). Caddy ne trouve plus de socket et sert la page d'attente ; le site public reste en ligne. `provision.sh` et `tribe-menus-credential` laissent un environnement en attente tel quel.

Un déploiement ou un retour arrière **lève** la mise en attente s'il réussit (c'est le geste qui livre le correctif), et la **remet** s'il échoue : la version laissée en service est alors la précédente, que la mise en attente protégeait.

## Supervision

Alertes par e-mail à `root`, que `msmtp` envoie à l'administrateur avec le compte `server@` (ADR 0015, point 15) :

| Alerte | Déclenchée par | Unité |
| --- | --- | --- |
| Alerte de l'application (`code_request_limits`, `smtp_failures`) | un enregistrement du journal de l'application portant l'attribut `alert` (ADR 0023, point 1) | `tribe-menus-alert-relay.timer`, toutes les cinq minutes (D9) |
| Panne | le service en échec, une fois que systemd a cessé de le relancer (cinq fois en cinq minutes) | `OnFailure=` vers `tribe-menus-failure@<environnement>.service` |
| Disque | un système de fichiers rempli à plus de 80 % | `tribe-menus-disk.timer`, chaque jour vers 8 h, au démarrage suivant si le serveur était éteint |
| Mises à jour | `unattended-upgrades`, quand quelque chose a changé ; redémarrage annoncé | (paquet de Debian) |

- **Relais** : le minuteur lit le journal de chaque service depuis son dernier passage (curseur dans `/var/lib/tribe-menus/alert/`, la dernière heure au premier passage) et ne garde que les enregistrements JSON portant l'attribut `alert` (test : `internal/deploy`). Le journal est une donnée : l'enregistrement va tel quel dans le corps de l'e-mail, jamais dans ses en-têtes, et une nature d'alerte qui n'est pas un nom devient `unknown`. Au plus un e-mail par environnement et par nature en une heure ; l'application, elle, n'écrit chaque signal qu'une fois en six heures (ADR 0023, point 4). L'enregistrement ne porte ni adresse, ni IP, ni tribu.
- **Panne** : l'état du service et ses trente dernières lignes de journal, et les commandes de remise en route.
- `sudo tribe-menus-check drill` exerce la panne (sur la recette) et le disque.

## Contrôles

**Sur le serveur** (D10), une ligne par point, `ok`, `FAIL` ou `info`, sans valeur de l'instance : le rapport se copie dans une PR.

```sh
sudo tribe-menus-check           # état du serveur, et e-mail d'essai à l'administrateur
sudo tribe-menus-check report    # l'heure écoulée, par environnement
sudo tribe-menus-check drill     # alerte de panne de la recette, essai de l'alerte du disque
```

- Sans argument : SSH effectif (`sshd -T`), `sudo` sans règle `NOPASSWD`, règles du pare-feu et ports à l'écoute hors boucle locale (TCP 22, 80 et 443 seulement ; un port UDP à l'écoute est signalé, le pare-feu le refuse), unités actives et minuteurs, aucune unité en échec, `/healthz` de chaque environnement, droits des répertoires de données, des bases et des credentials, credentials déchiffrables, lien du site, note de `systemd-analyze security` (2,0 au plus), configuration de Caddy, simulation d'`unattended-upgrades`, synchronisation de l'heure, journal persistant, e-mail d'essai.
- `report` : sur l'heure écoulée, par environnement, les envois en échec par étape et code SMTP (un envoi réussi n'écrit rien), le contrôle SMTP au démarrage, les alertes, les erreurs et les avertissements par message (dont « client IP unknown », plan `production`, étape 6), les démarrages par version ; pour le serveur, les envois de `msmtp` par code de retour (sans les adresses, que `msmtp` écrit aussi), et les relais, déploiements et mises en attente.
- `drill` : coupe le redémarrage automatique de la recette le temps de l'exercice (fichier dans `/run`), tue son service pour que `OnFailure=` envoie l'alerte de panne, remet tout en route, puis envoie l'essai de l'alerte du disque. Deux e-mails sont attendus.

**De l'extérieur**, depuis le Dev Container (`internal/tools/probe`) :

```sh
make probe URL=https://recette.meltingtribe.codingmatters.org ARGS="-dkim-selector <sélecteur>"
make probe URL=https://recette.meltingtribe.codingmatters.org ARGS="-timing -tribe demo -members <adresse d'un membre> -rounds 3"
```

- Par défaut : certificat (Let's Encrypt, TLS 1.2 au moins), redirection depuis HTTP, HSTS, pas d'HTTP/3 annoncé ni d'en-tête `Server` ; en-têtes du site (CSP sans script, aucun cookie) et de l'application ; `noindex` et `robots.txt` en recette, site indexable en production ; `/healthz` introuvable ; corps de 128 Kio refusé, requête dont les en-têtes ou le corps n'arrivent jamais fermée ; seuls les ports 22, 80 et 443 répondent, parmi vingt-sept courants, en IPv4 et en IPv6 ; un `X-Forwarded-For` forgé sans effet ; DNS par un résolveur public qui valide DNSSEC (`dns.google`) : A et AAAA signés, CAA, SPF, DKIM (avec `-dkim-selector`, le `s=` de l'en-tête `DKIM-Signature` d'un e-mail reçu), DMARC.
- **Budget** : l'essai de `X-Forwarded-For` envoie onze demandes de code pour des adresses inventées, et consomme les dix de l'heure de la connexion d'où il part (ENF-01) ; `-skip-xff` le saute.
- `-timing` mesure les temps de réponse de la connexion (`revue-securite`, « Ce qui n'a pas pu être vérifié »), en recette : requêtes alternées pour un membre et une autre adresse, une tribu existante et une autre ; d'abord `POST sessions` sans demande de code (non limité : 60 requêtes par cas, toutes les 200 ms), puis `POST login-codes` au rythme que les limites laissent à un attaquant depuis une connexion, neuf par heure (`-rounds` heures ; le membre reçoit trois e-mails par heure). Pour chaque cas : médiane, quartiles, codes de réponse, qui doivent être les mêmes ; pour chaque paire, la probabilité qu'un temps du premier cas dépasse un temps du second (0,5 quand les temps ne les distinguent pas) : écart à examiner au-delà de 0,35 à 0,65, pas de verdict sous vingt mesures par cas.

## Relancer le provisionnement

Quand `deploy/` a changé, ce que disent les notes de version (D6) : depuis la version installée en production, avec les mêmes environnements.

```sh
sudo /opt/tribe-menus/production/current/deploy/provision.sh production recette
```

**Avant la première version en production** (de la séance S2 à la séance S4 du plan `recette`), la seule version installée est l'archive d'une PR en recette ; on relance depuis elle, `/opt/tribe-menus/recette/current/deploy/provision.sh`. C'est sans risque à ce moment-là : la production n'a ni version ni données, et l'archive est celle d'une PR du propriétaire, relue. Ensuite, toujours depuis la production.

`tribe-menus-deploy` n'installe que l'application et son site ; il ne relance jamais le provisionnement.

## Reconstruire le serveur

Quand le serveur est perdu ou compromis, ou qu'une correction manuelle a été nécessaire (plan `recette`, « Règles communes »).

1. **S'il y a des données en production**, et que le serveur répond encore : un instantané, copié hors du serveur. Sinon, la sauvegarde de l'hébergeur, s'il y en a une (ADR 0025, point 4).

   ```sh
   sudo tribe-menus-maintenance production on                 # plus rien ne s'écrit
   sudo tribe-menus-deploy --snapshot                         # l'attente reste : le service n'est pas relancé
   sudo tar -czf /root/reprise.tar.gz -C /var/lib/tribe-menus/production/backups manuel-<date>
   ```

   Depuis l'hôte, copier `/root/reprise.tar.gz` (rendu lisible au compte d'administration, puis `scp`). Le fichier contient les adresses des membres : il reste sur l'hôte, chiffré, et s'efface après la reprise.
2. **Réinstaller le VPS** depuis l'espace client de l'hébergeur (Debian 13, même clé SSH), puis suivre « Première installation » et « Secrets » : la clé de la machine a changé, les secrets se saisissent de nouveau. L'empreinte de la clé d'hôte SSH change aussi : la remplacer dans `known_hosts` et dans le gestionnaire.
3. **Déployer** la version qui tournait : `sudo tribe-menus-deploy <version>`.
4. **Restaurer les données** : copier l'archive sur le serveur, puis

   ```sh
   sudo tar -xzf reprise.tar.gz --no-same-owner -C /var/lib/tribe-menus/production/backups
   sudo tribe-menus-deploy --restore manuel-<date>
   ```

5. `sudo tribe-menus-check`, `make probe` sur les deux noms d'hôte.

Sans données en production, les étapes 1 et 4 tombent : c'est la séance S2 rejouée.
