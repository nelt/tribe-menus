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
| `systemd/tribe-menus@.socket`, `systemd/tribe-menus@.service` | Socket et service de chaque environnement (`tribe-menus@production`, `tribe-menus@recette`). | `/etc/systemd/system/` |
| `caddy/attente.html` | Page d'attente, servie en 503 quand l'application ne répond pas. | `/usr/share/tribe-menus/` |
| `apt/caddy-stable.asc` | Clé de signature du dépôt apt du projet Caddy (D15). | `/etc/apt/keyrings/` |
| `config.example.json` | Exemple du fichier de configuration de `tribe-menus serve -config` (plan `production`, D1), lu par un test avec le vrai lecteur. Sur le serveur, `provision.sh` produit ce fichier. | — |

À venir avec le lot C du plan `recette` : `tribe-menus-deploy`, la supervision, `tribe-menus-maintenance` et `tribe-menus-check`.

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
sudo tribe-menus-credential server github-token        # artefacts des PR (lot C)
```

Le secret est lu deux fois sans écho, chiffré avec la clé de la machine (`systemd-creds --with-key=host`), vérifié par un déchiffrement, puis écrit en 0600 dans `/etc/tribe-menus/<environnement>/smtp-password.cred` ou `/etc/tribe-menus/server/<nom>.cred` (dossier en 0700). Il passe de `read` à `systemd-creds` par un tube, jamais par les arguments d'une commande. Le service de l'environnement est redémarré s'il tournait.

- Le service reçoit son mot de passe déchiffré par systemd, dans son seul `$CREDENTIALS_DIRECTORY` (`LoadCredentialEncrypted=`) : l'utilisateur du service ne peut pas lire le fichier chiffré.
- `msmtp` déchiffre le mot de passe de `server@` à chaque envoi (`passwordeval`) : seul `root` peut envoyer, ce que font `unattended-upgrades` et les alertes du serveur.
- Un serveur reconstruit a une autre clé de machine : les secrets se saisissent de nouveau.

## Tribu de recette

Une fois une version installée en recette (lot C), le fichier des membres (D3) s'écrit sur le serveur, une ligne par membre, adresse puis nom d'affichage, le premier étant le premier membre :

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
7. **Scripts et unités** : `tribe-menus-credential`, `tribe-menus-admin`, page d'attente, socket et service.
8. **Chaque environnement** (D8 ; ADR 0015, points 8 et 9) :

   | Emplacement | Droits | Contenu |
   | --- | --- | --- |
   | utilisateur `tribe-menus-<environnement>` | système, sans shell ni répertoire | exécute le service |
   | `/opt/tribe-menus/<environnement>/releases/` | `root`, 0755 | versions installées, lien `current` (lot C) |
   | `/etc/tribe-menus/<environnement>/` | `root:tribe-menus-<environnement>`, 0750 | `config.json` en 0640, credential en 0600 à `root` |
   | `/var/lib/tribe-menus/<environnement>/` | `tribe-menus-<environnement>`, 0700 | registre et bases des tribus, en 0600 (`UMask=0077`) |
   | `/var/www/tribe-menus/<environnement>/` | `root`, 0755 | site public de l'environnement (lot C) |

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

## Relancer le provisionnement

Quand `deploy/` a changé, ce que disent les notes de version (D6) : depuis la version installée, avec les mêmes environnements.

```sh
sudo /opt/tribe-menus/production/current/deploy/provision.sh production recette
```

`tribe-menus-deploy` (lot C) n'installe que l'application et son site ; il ne relance jamais le provisionnement.
