# `deploy/`

Ce qui sert à faire tourner l'application sur un serveur (ADR 0015, 0016). Le dossier est livré dans l'archive de chaque version (ADR 0012).

- `config.example.json` : exemple du fichier de configuration de `tribe-menus serve -config` et `tribe-menus admin … -config` (plan `production`, D1). Le vrai fichier vit sur le serveur, dans `/etc/tribe-menus/<environnement>/config.json`, hors du dépôt : ses valeurs sont propres à l'instance (ADR 0011, point 5). La clé `alerts.to` donne l'adresse de l'administrateur, qui reçoit l'alerte sur les limites de demandes de code (ADR 0023). Un test le lit avec le vrai lecteur, pour qu'il ne se périme pas.

À venir avec le plan `recette` : `provision.sh`, unité et socket systemd, configuration de Caddy, `tribe-menus-deploy` et `tribe-menus-admin`.
