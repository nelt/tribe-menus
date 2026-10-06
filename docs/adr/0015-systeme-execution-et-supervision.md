# 0015. Système, exécution et supervision du serveur

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Le serveur est un VPS OVHcloud (ADR 0007) qui fait tourner Caddy (ADR 0006) et le binaire Go de l'application (ADR 0002), avec ses bases SQLite (ADR 0003). Il n'y a qu'un seul serveur et pas de sauvegarde hors site en V1 : il doit pouvoir être reconstruit à l'identique, et chaque couche doit être durcie. Les e-mails sortants passent par le MX Plan (ADR 0014).

## Décision

### Système et durcissement

1. **Debian 13**, image minimale.
2. **Provisionnement par un script shell idempotent**, versionné dans `deploy/provision.sh`, relu en PR et lancé par SSH sur un serveur neuf ; il peut être relancé sans risque. Il installe et configure tout ce qui suit.
3. **SSH** : authentification par clé uniquement (ed25519), pas de mot de passe, pas de connexion directe en root ; un utilisateur d'administration personnel avec `sudo`. Les pénalités intégrées aux versions récentes d'OpenSSH limitent les tentatives répétées (fail2ban superflu).
4. **Pare-feu nftables** : seuls les ports 22, 80 et 443 sont ouverts en entrée.
5. **Mises à jour** : correctifs de sécurité installés automatiquement (`unattended-upgrades`) ; **redémarrage automatique la nuit (3 h 30) quand il est nécessaire**, précédé d'un rapport par e-mail signalant le redémarrage à venir, qu'on peut annuler (`shutdown -c`). Montées de version majeures de Debian manuelles.
6. **Relais d'envoi `msmtp`** vers le SMTP du MX Plan, avec un compte dédié (`server@codingmatters.org`), distinct de celui de l'application : rapports de mise à jour et alertes.
7. Pas de Docker sur le serveur ; journald persistant et plafonné, logs conservés 1 mois (PT-07) ; synchronisation de l'heure active.

### Exécution de l'application

8. **Utilisateur système dédié**, sans shell ni mot de passe, par environnement (ADR 0016).
9. **Emplacements** :
   - binaires : `/opt/tribe-menus/<environnement>/releases/vX.Y.Z/`, lien `current` vers la version active ;
   - données : `/var/lib/tribe-menus/<environnement>/` (registre et un fichier par tribu), accessibles au seul service ;
   - configuration non secrète : `/etc/tribe-menus/<environnement>/` ; précisé le 2026-10-06 (plan `production`, D1 et D2) : un fichier `config.json`, en JSON lu strictement (clé inconnue ou manquante refusée), avec les clés `data` (répertoire des bases), `listen` (`"systemd"` pour le socket transmis, ou une adresse TCP) et `baseURL` (adresse publique de l'instance), complétées par l'envoi SMTP et les alertes ; le service est lancé par `tribe-menus serve -config <fichier>`, mode exclusif du mode développement `-dev` ; exemple dans `deploy/config.example.json` ;
   - site public : `/var/www/tribe-menus/`, servi par Caddy.
10. **Secrets** (mot de passe SMTP…) : **credentials systemd**, chiffrés sur disque avec une clé propre à la machine (`systemd-creds`), déchiffrés au démarrage et présentés au seul service dans un répertoire privé, lus comme un fichier par le programme. Jamais en variables d'environnement. Précisé le 2026-10-06 (plan `production`, étape 11) : le mot de passe SMTP est le credential `smtp-password`, lu au démarrage dans le répertoire que désigne `CREDENTIALS_DIRECTORY` ; absent ou vide, le serveur ne démarre pas ; aucune option, variable d'environnement ni clé du fichier de configuration ne peut le porter.
11. **Activation de socket systemd** : systemd crée le socket Unix, accessible au seul groupe de Caddy, et le transmet au programme. Pendant un redémarrage, les requêtes attendent dans le socket au lieu d'échouer.
12. **Confinement systemd** : aucun privilège ni capacité ; système de fichiers en lecture seule sauf le répertoire de données ; `/home` invisible, `/tmp` privé ; appels système limités (`@system-service`) ; familles réseau limitées à Unix, IPv4 et IPv6 (envoi SMTP). Objectif : la meilleure note possible à `systemd-analyze security`.
13. **Commandes d'administration** (EF-08 à EF-11) : script `tribe-menus-admin` qui lance la sous-commande `admin` sous l'utilisateur du service ; aucune manipulation directe des fichiers de base.
14. **Logs** : logs structurés (`log/slog`, JSON) sur la sortie standard, recueillis par journald.

### Supervision

15. **Alertes par e-mail depuis le serveur** (via `msmtp`) :
    - panne ou redémarrages en boucle de l'application (`OnFailure=` dans systemd) ;
    - disque rempli à plus de 80 % (minuteur quotidien) ;
    - échecs répétés d'envoi SMTP, signalés par l'application ; précisé le 2026-10-06 (ADR 0023) : l'application signale une alerte par un enregistrement de son journal portant l'attribut `alert`, que le serveur relaie par `msmtp` ; l'alerte sur les limites de demandes de code part en plus par e-mail depuis l'application ;
    - rapports de mise à jour et redémarrages programmés.
16. **Pas de sonde externe en V1**, ni de centralisation des logs ni de métriques.

## Alternatives envisagées

- **Ubuntu 24.04 LTS** : écartée, plus de composants par défaut (dont snap).
- **Procédure manuelle** : écartée, sujette aux oublis et difficile à rejouer. **Ansible** : écarté, Python et ses modules sur le poste de pilotage, surdimensionné pour un seul serveur.
- **Secrets en variables d'environnement** : écartés, fuites faciles (processus enfants, `/proc`).
- **Socket créé par l'application** : plus simple, mais courte coupure à chaque redémarrage.
- **Docker** : écarté, une couche de plus sans bénéfice pour un binaire statique.
- **Sonde externe** (workflow GitHub planifié ou service dédié) : reportée.
- **SSH uniquement à travers un réseau privé (WireGuard, Tailscale)** : reporté.

## Conséquences

- **Positif** : serveur reconstructible par un script relu ; surface d'attaque réduite (trois ports, pas de mot de passe, service confiné sans privilège) ; secrets jamais exposés dans l'environnement ; redémarrages sans coupure visible ; aucune dépendance à un service de supervision tiers.
- **Négatif** : un script de provisionnement à maintenir ; un redémarrage nocturne occasionnel coupe le service environ une minute.
- **Risque accepté en V1** : sans sonde externe, une panne complète du serveur (qui empêche aussi l'envoi des alertes) n'est détectée que par les utilisateurs.
- **Reporté** : sonde externe, SSH derrière un réseau privé, centralisation des logs et métriques.
