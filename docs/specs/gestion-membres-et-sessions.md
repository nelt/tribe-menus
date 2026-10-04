# Gestion des membres et des sessions

Spécification fonctionnelle. S'appuie sur ENF-01 et ENF-02 (`exigences-non-fonctionnelles.md`) et l'ADR 0001.

Critères d'acceptation : `features/membres-et-sessions.feature` (EF-01 à EF-07) et `features/administration.feature` (EF-08 à EF-11).

## Principes

- Une **tribu** regroupe les personnes qui partagent les menus. Elle est créée par un script d'administration (EF-08).
- Une instance de l'application **héberge plusieurs tribus**, strictement compartimentées (ENF-02).
- Chaque tribu a un **identifiant d'URL** (par exemple `/tribes/<identifiant>/…`), unique sur l'instance, qui sert de discriminant : toutes les pages et tous les appels de l'application sont faits dans le contexte d'une tribu.
- Un **membre** est une personne autorisée à utiliser l'application au sein d'une tribu.
- Le membre est **identifié par son adresse e-mail** dans sa tribu, normalisée (sans espaces, en minuscules). Une adresse est de la forme `partie-locale@domaine`, avec un point dans le domaine, ni en tête ni en fin (`alice@exemple` est refusée) ; la règle est portée par le serveur seul, à la connexion comme dans le script d'administration (2026-10-04). L'adresse est unique au sein d'une tribu, mais **une même adresse peut être membre de plusieurs tribus** ; chaque appartenance est un membre distinct, avec son propre statut.
- **Les sessions sont propres à une tribu** : se connecter à une tribu n'ouvre pas de session dans une autre, même avec la même adresse. La connexion (ENF-01) se fait depuis l'URL de la tribu ; le code n'est envoyé que si l'adresse est membre actif de cette tribu.
- Il n'y a **pas d'inscription libre** : on ne devient membre que par le script d'initialisation ou par l'ajout d'un membre existant.
- **Tous les membres ont les mêmes droits** en v1 (voir « Hors périmètre »).
- **Aucune notification par e-mail** n'est envoyée lors des opérations sur les membres. Le seul e-mail envoyé par l'application est le code de connexion (ENF-01).
- Toutes les opérations sur les membres et les sessions sont tracées dans un **journal d'audit** en base (EF-07).
- **Conservation** (PT-07, 2026-09-28) : les codes de connexion expirés ou utilisés et les sessions expirées ou révoquées sont effacés automatiquement ; les entrées du journal d'audit sont conservées 12 mois, puis effacées automatiquement ; les données d'une tribu sont conservées tant qu'elle existe. Les demandes de code et les codes fantômes, qui ne portent que des empreintes d'adresse e-mail et d'IP, sont effacés au plus tard une heure et dix minutes après la demande : une fenêtre d'une heure, puis l'effacement automatique, qui passe toutes les dix minutes (ENF-01, ADR 0021, 2026-10-03 ; durée précisée le 2026-10-04).

## EF-01. Ajouter un membre

- Un membre connecté peut ajouter un nouveau membre en saisissant son adresse e-mail (et, facultativement, un nom d'affichage).
- Le nouveau membre peut ensuite se connecter avec le code par e-mail (ENF-01), sans autre démarche.
- L'ajout d'une adresse déjà membre active est refusé avec un message explicite.
- L'ajout d'une adresse de membre révoqué de la tribu est refusé ; l'application propose de le réactiver (EF-06).
- Que l'adresse soit déjà membre d'autres tribus n'a aucune incidence, et n'est pas révélé.

## EF-02. Consulter les membres

- Un membre connecté voit la liste des membres de sa tribu : nom d'affichage, e-mail, date d'ajout, ajouté par, statut (actif / révoqué), et pour un membre révoqué, la date de révocation et son auteur (« vous », un membre, lui-même s'il a quitté la tribu, ou « script d'administration »).

## EF-03. Révoquer un membre

- Un membre connecté peut révoquer un autre membre, ou se révoquer lui-même (quitter la tribu).
- La révocation est immédiate : toutes les sessions du membre révoqué sont fermées et ses éventuels codes de connexion en attente sont invalidés. Il ne peut plus demander de code.
- En cas d'auto-révocation, la session courante est fermée et l'utilisateur revient à l'écran de connexion.
- Le membre révoqué est désactivé, pas supprimé : ses données (menus, historique) sont conservées.
- Si le dernier membre actif se révoque, la tribu n'a plus de membre actif ; seul le script d'administration (EF-09) permet alors d'en réactiver un.

## EF-04. Consulter ses sessions

- Un membre connecté voit la liste de ses sessions actives dans la tribu courante. Pour chacune :
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
  - les opérations sur les membres : initialisation de la tribu, ajout, révocation (y compris auto-révocation), réactivation ;
  - les opérations sur les sessions : ouverture (connexion réussie), révocation d'une session, « déconnecter tous les autres appareils », déconnexion, fermeture des sessions consécutive à la révocation d'un membre.
- Chaque entrée contient : la tribu, la date et l'heure, l'opération, le membre concerné, l'auteur (un membre, ou « script d'administration »), et pour une opération sur une session, la session et l'appareil détecté.
- Le journal est en ajout seul : aucune fonction de l'interface ne modifie ni ne supprime ses entrées. Seules exceptions, hors interface : l'effacement automatique des entrées de plus de 12 mois, et l'anonymisation d'un membre (EF-11).

## Administration

Scripts exécutés sur le serveur, hors de l'interface de l'application. Leurs opérations sont tracées dans le journal d'audit (EF-07) avec « script d'administration » comme auteur.

### EF-08. Initialiser une tribu

- Le script crée une nouvelle tribu et demande de manière interactive son nom et son identifiant d'URL (refusé s'il est déjà pris), puis l'adresse e-mail du premier membre (et, facultativement, son nom d'affichage).
- **Format de l'identifiant d'URL** (2026-10-03) : 3 à 40 caractères ; lettres minuscules sans accent, chiffres et tirets ; une lettre en premier, pas de tiret final ni de tirets consécutifs. Une saisie non conforme est refusée, pas convertie (`Martin` n'est pas changé en `martin`). Aucun identifiant n'est réservé (ADR 0006). L'identifiant n'est pas modifiable en v1. Il n'a pas à être difficile à deviner : l'écran de connexion identique pour toute URL protège de l'énumération (ADR 0006, point 9).
- Il peut être lancé autant de fois que nécessaire, chaque exécution créant une tribu distincte.
- Il affiche l'URL de la tribu à transmettre au premier membre.
- Le premier membre peut ensuite se connecter avec le code par e-mail et ajouter les autres membres (EF-01).

### EF-09. Réactiver un membre révoqué

- Le script demande la tribu (par son identifiant d'URL), puis l'adresse e-mail du membre révoqué, et le réactive avec le même effet qu'EF-06.
- C'est le moyen de récupération lorsque la tribu n'a plus aucun membre actif.

### EF-10. Supprimer une tribu

- Sur demande de la tribu, le script demande l'identifiant d'URL de la tribu, affiche son nom et son nombre de membres, puis exige de retaper l'identifiant pour confirmer.
- Il supprime la base de la tribu et son entrée au registre. L'URL de la tribu se comporte ensuite comme celle d'une tribu qui n'existe pas (ENF-02).
- Les instantanés pris avant les déploiements (ADR 0016) qui contiennent encore la tribu disparaissent au plus tard après cinq déploiements ; le script le rappelle.
- L'opération est tracée dans les journaux du serveur (le journal d'audit de la tribu disparaît avec elle).

### EF-11. Anonymiser un membre révoqué

- Sur demande de la personne ou de la tribu, le script demande la tribu, puis l'adresse e-mail d'un membre **révoqué** (un membre actif doit d'abord être révoqué).
- Il remplace son adresse et son nom d'affichage par « membre anonymisé », dans la liste des membres comme dans le journal d'audit. Ses plats, menus et listes restent dans la tribu.
- L'adresse est ensuite libre : elle peut être ajoutée de nouveau comme un nouveau membre (EF-01), sans lien avec l'ancien.
- L'opération est tracée dans le journal d'audit, sans l'adresse effacée.

## Hors périmètre (version ultérieure)

- Rôles et privilèges (administrateur, parent/enfant, lecture seule…).
- Consultation ou révocation des sessions d'un autre membre à l'unité (la révocation d'un membre, EF-03, ferme déjà toutes ses sessions).
- Consultation du journal d'audit dans l'interface.
