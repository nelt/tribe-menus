# Sécurité

## Signaler une vulnérabilité

**Pas d'issue publique.** Utilisez le signalement privé de GitHub : onglet *Security* du dépôt, puis *Report a vulnerability*. Le signalement n'est visible que du mainteneur, et la correction peut être préparée dans un espace privé avant d'être publiée.

Indiquez si possible :

- la version concernée (`tribe-menus version`, ou le lien vers le code source affiché par l'application) ;
- les étapes pour reproduire, et ce que la faille permet d'obtenir ;
- une proposition de correction, si vous en avez une.

Le projet est maintenu par une seule personne, sur son temps libre : l'accusé de réception et la correction se font au mieux, sans délai garanti. Vous serez tenu informé, et cité dans l'avis publié si vous le souhaitez.

## Versions prises en charge

Seule la dernière version publiée (étiquette `vX.Y.Z`, voir `CHANGELOG.md`) reçoit des correctifs de sécurité.

## Périmètre

Sont concernés le code de ce dépôt et l'instance `meltingtribe.codingmatters.org`.

Sur cette instance, merci de ne tester qu'avec votre propre tribu et de vous abstenir de tout ce qui pourrait gêner les autres utilisateurs : déni de service, envoi massif de codes de connexion, accès aux données d'une autre tribu au-delà de ce qui suffit à démontrer la faille.
