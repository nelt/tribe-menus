# Gestion des membres et des sessions

Spécification fonctionnelle. S'appuie sur ENF-01 (`exigences-non-fonctionnelles.md`) et l'ADR 0001.

## Principes

- Un **membre** est une personne du foyer autorisée à utiliser l'application.
- Le membre est **identifié par son adresse e-mail**, unique et normalisée (sans espaces, en minuscules).
- Il n'y a **pas d'inscription libre** : on ne devient membre que si un membre existant nous ajoute.
- **Tous les membres ont les mêmes droits** en v1 (voir « Hors périmètre »).

## EF-01. Ajouter un membre

- Un membre connecté peut ajouter un nouveau membre en saisissant son adresse e-mail (et, facultativement, un nom d'affichage).
- Le nouveau membre peut ensuite se connecter avec le code par e-mail (ENF-01), sans autre démarche.
- L'ajout d'une adresse déjà membre active est refusé avec un message explicite.
- Le système enregistre qui a ajouté le membre et quand.

## EF-02. Consulter les membres

- Un membre connecté voit la liste des membres : nom d'affichage, e-mail, date d'ajout, ajouté par, statut (actif / révoqué).

## EF-03. Révoquer un membre

- Un membre connecté peut révoquer un autre membre.
- La révocation est immédiate : toutes les sessions du membre révoqué sont fermées et ses éventuels codes de connexion en attente sont invalidés. Il ne peut plus demander de code.
- Le membre révoqué est désactivé, pas supprimé : ses données (menus, historique) sont conservées. Il peut être réactivé par un nouvel ajout de la même adresse.
- Le système enregistre qui a révoqué le membre et quand.

## EF-04. Consulter ses sessions

- Un membre connecté voit la liste de ses sessions actives. Pour chacune :
  - l'appareil détecté : type (téléphone, tablette, ordinateur), système, navigateur, et le fait qu'il s'agisse de la PWA installée ou d'un onglet de navigateur (par exemple « iPhone · Safari · app installée ») ;
  - la date d'ouverture et la date de dernière activité ;
  - un repère « cet appareil » pour la session courante.
- **Détection de l'appareil** : à partir des en-têtes `User-Agent` et, quand ils sont disponibles, des User-Agent Client Hints ; le mode PWA est signalé par le client (`display-mode: standalone`) à l'ouverture de la session. Les informations sont enregistrées à la création de la session et mises à jour si elles changent.
- Le membre peut donner un nom à une session (« Téléphone de cuisine ») pour la reconnaître plus facilement.

## EF-05. Révoquer une session

- Un membre peut fermer n'importe laquelle de ses sessions ; l'appareil concerné est déconnecté à sa prochaine requête.
- Il peut aussi « déconnecter tous les autres appareils » en une action.
- Fermer la session courante équivaut à une déconnexion.

## Hors périmètre (version ultérieure)

- Rôles et privilèges (administrateur, parent/enfant, lecture seule…).
- Consultation ou révocation des sessions d'un autre membre à l'unité (la révocation d'un membre, EF-03, ferme déjà toutes ses sessions).

## Questions ouvertes

- **Premier membre** : comment est créé le tout premier membre du foyer (configuration au déploiement, commande d'administration, premier arrivant) ?
- **Notification** : faut-il envoyer un e-mail au nouveau membre lors de son ajout, et au membre révoqué ?
- **Auto-révocation** : un membre peut-il se révoquer lui-même (quitter le foyer) ? Faut-il interdire de laisser le foyer sans aucun membre actif ?
