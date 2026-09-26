# Gestion des membres et des sessions

Spécification fonctionnelle. S'appuie sur ENF-01 (`exigences-non-fonctionnelles.md`) et l'ADR 0001.

## Principes

- Une **famille** regroupe les personnes qui partagent les menus. Elle est créée par un script d'administration (EF-08).
- Une instance de l'application **héberge plusieurs familles**, strictement isolées : un membre ne voit que les membres, les sessions, les menus et le journal d'audit de sa propre famille.
- Un **membre** est une personne d'une famille autorisée à utiliser l'application.
- Le membre est **identifié par son adresse e-mail**, unique et normalisée (sans espaces, en minuscules). L'unicité vaut pour toute l'instance : une adresse appartient à une seule famille.
- Il n'y a **pas d'inscription libre** : on ne devient membre que par le script d'initialisation ou par l'ajout d'un membre existant.
- **Tous les membres ont les mêmes droits** en v1 (voir « Hors périmètre »).
- **Aucune notification par e-mail** n'est envoyée lors des opérations sur les membres. Le seul e-mail envoyé par l'application est le code de connexion (ENF-01).
- Toutes les opérations sur les membres et les sessions sont tracées dans un **journal d'audit** en base (EF-07).

## EF-01. Ajouter un membre

- Un membre connecté peut ajouter un nouveau membre en saisissant son adresse e-mail (et, facultativement, un nom d'affichage).
- Le nouveau membre peut ensuite se connecter avec le code par e-mail (ENF-01), sans autre démarche.
- L'ajout d'une adresse déjà membre active est refusé avec un message explicite.
- L'ajout d'une adresse de membre révoqué de la même famille est refusé ; l'application propose de le réactiver (EF-06).
- L'ajout d'une adresse appartenant à une autre famille est refusé, avec un message qui ne révèle pas l'existence de cette autre famille.

## EF-02. Consulter les membres

- Un membre connecté voit la liste des membres : nom d'affichage, e-mail, date d'ajout, ajouté par, statut (actif / révoqué).

## EF-03. Révoquer un membre

- Un membre connecté peut révoquer un autre membre, ou se révoquer lui-même (quitter la famille).
- La révocation est immédiate : toutes les sessions du membre révoqué sont fermées et ses éventuels codes de connexion en attente sont invalidés. Il ne peut plus demander de code.
- En cas d'auto-révocation, la session courante est fermée et l'utilisateur revient à l'écran de connexion.
- Le membre révoqué est désactivé, pas supprimé : ses données (menus, historique) sont conservées.
- Si le dernier membre actif se révoque, la famille n'a plus de membre actif ; seul le script d'administration (EF-09) permet alors d'en réactiver un.

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

## EF-06. Réactiver un membre

- Un membre connecté peut réactiver un membre révoqué depuis la liste des membres.
- Le membre réactivé retrouve ses données et peut de nouveau se connecter avec le code par e-mail. Aucune de ses anciennes sessions n'est restaurée.

## EF-07. Journal d'audit

- Sont enregistrées en base :
  - les opérations sur les membres : initialisation de la famille, ajout, révocation (y compris auto-révocation), réactivation ;
  - les opérations sur les sessions : ouverture (connexion réussie), révocation d'une session, « déconnecter tous les autres appareils », déconnexion, fermeture des sessions consécutive à la révocation d'un membre.
- Chaque entrée contient : la famille, la date et l'heure, l'opération, le membre concerné, l'auteur (un membre, ou « script d'administration »), et pour une opération sur une session, la session et l'appareil détecté.
- Le journal est en ajout seul : ses entrées ne sont ni modifiées ni supprimées par l'application.

## Administration

Scripts exécutés sur le serveur, hors de l'interface de l'application. Leurs opérations sont tracées dans le journal d'audit (EF-07) avec « script d'administration » comme auteur.

### EF-08. Initialiser une famille

- Le script crée une nouvelle famille et demande de manière interactive son nom, puis l'adresse e-mail du premier membre (et, facultativement, son nom d'affichage).
- Il peut être lancé autant de fois que nécessaire, chaque exécution créant une famille distincte.
- Il refuse une adresse déjà membre d'une famille.
- Le premier membre peut ensuite se connecter avec le code par e-mail et ajouter les autres membres (EF-01).

### EF-09. Réactiver un membre révoqué

- Le script réactive un membre révoqué à partir de son adresse e-mail (unique sur l'instance, elle désigne aussi la famille), avec le même effet qu'EF-06.
- C'est le moyen de récupération lorsque la famille n'a plus aucun membre actif.

## Hors périmètre (version ultérieure)

- Rôles et privilèges (administrateur, parent/enfant, lecture seule…).
- Consultation ou révocation des sessions d'un autre membre à l'unité (la révocation d'un membre, EF-03, ferme déjà toutes ses sessions).
- Consultation du journal d'audit dans l'interface.
