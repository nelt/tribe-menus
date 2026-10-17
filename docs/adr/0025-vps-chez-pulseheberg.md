# 0025. VPS chez PulseHeberg

- **Date** : 2026-10-17
- **Statut** : accepté

## Contexte

L'ADR 0007 prévoyait un VPS OVHcloud de la gamme VPS-1. À la séance S1 du plan `recette`, le développeur a commandé le VPS chez PulseHeberg, un hébergeur que l'ADR 0007 citait parmi les alternatives crédibles. Le domaine `codingmatters.org`, sa zone DNS et le MX Plan restent chez OVHcloud.

Les critères de l'ADR 0007 tiennent : hébergement dans l'UE, coût, simplicité. La charge reste minime : un binaire Go, Caddy et des bases SQLite, sans build sur le serveur (ADR 0012).

## Décision

1. **Serveur** : VPS PulseHeberg, offre CLASSIC-2 (2 vCPU, 2 Go de RAM, 60 Go NVMe en RAID 10 au moment de la commande), dans un datacenter de Paris, sous Debian 13. Le bloc d'adresses est enregistré au RIPE sous le nom `PULSEHEBERG-PARIS-2`. Une adresse IPv4 et une adresse IPv6 publiques sont fournies avec le VPS ; elles ne sont écrites que dans le gestionnaire de mots de passe du développeur (ADR 0011, point 5).
2. **Remplace le point 1 de l'ADR 0007.** Le point 2 (zone DNS chez OVHcloud, CAA, DNSSEC) et l'ADR 0014 (envoi par le MX Plan OVHcloud) sont inchangés : le serveur et le domaine ne sont plus chez le même fournisseur.
3. **Compte de l'image** : l'image PulseHeberg n'ouvre que `root`. `provision.sh` crée le compte d'administration personnel depuis `root` ; le « retrait du compte de l'image » prévu par le plan `recette` (étape 8, S2) revient à fermer la connexion SSH de `root`.
4. **Sauvegarde** : les sauvegardes de PulseHeberg sont « non contractuelles », la sauvegarde des données reste de la responsabilité exclusive du client, et une restauration se demande par ticket (conditions particulières VPS, articles 8.1, 8.2, 8.6 et 8.7). Des instantanés sont disponibles depuis l'espace client. La sauvegarde de V1 que l'ADR 0007 (point 3) attribuait à l'hébergeur n'est donc plus garantie. **Décision à prendre avant l'ouverture de la production** (séance S4 du plan `recette`) : accepter ce niveau de sauvegarde, ou avancer la sauvegarde hors site que l'ADR 0007 reportait.

## Alternatives envisagées

- **VPS-1 OVHcloud** (ADR 0007) : non retenu à la commande. Il gardait le serveur et le domaine sous le même compte, avec une sauvegarde quotidienne incluse.

## Conséquences

- **Positif** : hébergeur français, prix comparable ; une seule machine reste largement suffisante malgré 2 Go de RAM au lieu de 4.
- **Négatif** : deux fournisseurs à suivre (PulseHeberg pour le serveur, OVHcloud pour le domaine et le courrier) ; la réinstallation du VPS se fait depuis l'espace client PulseHeberg.
- **Risque ouvert** : sans sauvegarde contractuelle ni réplication hors site, une perte du VPS fait perdre les données ; c'est l'objet du point 4.
- **Mentions légales** : l'hébergeur à nommer est SAS PulseHeberg, 9 boulevard de Strasbourg, 83000 Toulon, RCS Toulon 824 070 619 (conditions générales de PulseHeberg).
- **Décisions liées** : ADR 0007 (point 1 remplacé, point 3 à revoir), 0015 (contexte), 0016 (réinstallation et recette sur ce même VPS).
