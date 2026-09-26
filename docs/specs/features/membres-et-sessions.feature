# language: fr
Fonctionnalité: Membres et sessions
  En tant que membre de la tribu,
  je veux gérer les membres de ma tribu et mes appareils connectés,
  afin de contrôler qui accède à nos menus et depuis où.

  Contexte:
    Étant donné la tribu "Les Martin" d'identifiant "martin"
    Et "alice@exemple.fr" est membre actif de la tribu "martin"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"

  @EF-01
  Scénario: Ajouter un membre
    Quand j'ajoute le membre "bruno@exemple.fr" avec le nom d'affichage "Bruno"
    Alors "bruno@exemple.fr" figure dans la liste des membres avec le statut "actif"
    Et il est indiqué comme ajouté par "alice@exemple.fr"
    Et "bruno@exemple.fr" peut demander un code de connexion pour la tribu "martin"
    Et aucun e-mail n'est envoyé à "bruno@exemple.fr"

  @EF-01
  Scénario: L'adresse est normalisée
    Quand j'ajoute le membre " Bruno@Exemple.FR "
    Alors "bruno@exemple.fr" figure dans la liste des membres

  @EF-01
  Scénario: Refuser une adresse déjà membre active
    Étant donné "bruno@exemple.fr" est membre actif de la tribu "martin"
    Quand j'ajoute le membre "bruno@exemple.fr"
    Alors l'ajout est refusé
    Et un message m'indique que cette adresse est déjà membre

  @EF-01 @EF-06
  Scénario: L'ajout d'un membre révoqué propose sa réactivation
    Étant donné "bruno@exemple.fr" est membre révoqué de la tribu "martin"
    Quand j'ajoute le membre "bruno@exemple.fr"
    Alors l'ajout est refusé
    Et l'application me propose de réactiver "bruno@exemple.fr"

  @EF-01
  Scénario: Une adresse déjà membre d'une autre tribu peut être ajoutée
    Étant donné la tribu "Les Durand" d'identifiant "durand"
    Et "bruno@exemple.fr" est membre actif de la tribu "durand"
    Quand j'ajoute le membre "bruno@exemple.fr"
    Alors "bruno@exemple.fr" figure dans la liste des membres de la tribu "martin"
    Et le message de confirmation est le même que pour une adresse inconnue

  @EF-02
  Scénario: Consulter les membres de sa tribu
    Étant donné "bruno@exemple.fr" est membre actif de la tribu "martin"
    Et "chloe@exemple.fr" est membre révoqué de la tribu "martin"
    Et "david@exemple.fr" est membre actif de la tribu "durand"
    Quand je consulte la liste des membres
    Alors je vois "alice@exemple.fr" et "bruno@exemple.fr" avec le statut "actif"
    Et je vois "chloe@exemple.fr" avec le statut "révoqué"
    Et je ne vois pas "david@exemple.fr"
    Et chaque membre affiche son nom d'affichage, sa date d'ajout et l'auteur de l'ajout

  @EF-03
  Scénario: Révoquer un autre membre
    Étant donné "bruno@exemple.fr" est membre actif de la tribu "martin"
    Et "bruno@exemple.fr" a deux sessions ouvertes dans la tribu "martin"
    Et "bruno@exemple.fr" a demandé un code de connexion non encore utilisé
    Quand je révoque "bruno@exemple.fr"
    Alors "bruno@exemple.fr" a le statut "révoqué"
    Et ses deux sessions sont fermées
    Et son code de connexion en attente n'est plus accepté
    Et une nouvelle demande de code pour "bruno@exemple.fr" n'envoie aucun e-mail

  @EF-03
  Scénario: Un membre révoqué est déconnecté à sa requête suivante
    Étant donné "bruno@exemple.fr" est connecté à la tribu "martin" sur son téléphone
    Quand je révoque "bruno@exemple.fr"
    Et Bruno rafraîchit l'application sur son téléphone
    Alors Bruno voit l'écran de connexion

  @EF-03
  Scénario: Les données d'un membre révoqué sont conservées
    Étant donné "bruno@exemple.fr" est membre actif de la tribu "martin"
    Et "bruno@exemple.fr" a créé le plat "Ratatouille"
    Quand je révoque "bruno@exemple.fr"
    Alors le plat "Ratatouille" figure toujours dans la bibliothèque

  @EF-03
  Scénario: Se révoquer soi-même
    Étant donné "bruno@exemple.fr" est membre actif de la tribu "martin"
    Quand je me révoque
    Alors "alice@exemple.fr" a le statut "révoqué"
    Et je vois l'écran de connexion
    Et toutes mes sessions dans la tribu "martin" sont fermées

  @EF-03
  Scénario: Le dernier membre actif peut se révoquer
    Étant donné je suis le seul membre actif de la tribu "martin"
    Quand je me révoque
    Alors la tribu "martin" n'a plus aucun membre actif

  @EF-03
  Scénario: La révocation ne concerne que la tribu courante
    Étant donné "bruno@exemple.fr" est membre actif des tribus "martin" et "durand"
    Et "bruno@exemple.fr" est connecté à la tribu "durand"
    Quand je révoque "bruno@exemple.fr"
    Alors "bruno@exemple.fr" est toujours membre actif de la tribu "durand"
    Et sa session dans la tribu "durand" reste ouverte

  @EF-04
  Scénario: Consulter ses sessions
    Étant donné j'ai une session ouverte le 1er octobre 2026 depuis l'app installée sur un iPhone avec Safari
    Et j'ai une session ouverte depuis un onglet Firefox sur un ordinateur Windows
    Et "bruno@exemple.fr" a une session ouverte dans la tribu "martin"
    Quand je consulte mes sessions
    Alors je vois deux sessions
    Et l'une est décrite comme "iPhone · Safari · app installée", ouverte le 1er octobre 2026, avec sa date de dernière activité
    Et l'autre est décrite comme "Ordinateur · Windows · Firefox"
    Et la session que j'utilise porte le repère "cet appareil"
    Et je ne vois pas la session de "bruno@exemple.fr"

  @EF-04
  Scénario: Les sessions d'une autre tribu n'apparaissent pas
    Étant donné "alice@exemple.fr" est aussi membre actif de la tribu "durand"
    Et j'ai une session ouverte dans la tribu "durand"
    Quand je consulte mes sessions dans la tribu "martin"
    Alors je ne vois pas ma session de la tribu "durand"

  @EF-04
  Scénario: Nommer une session
    Quand je nomme la session "iPhone · Safari · app installée" "Téléphone de cuisine"
    Alors la session s'affiche sous le nom "Téléphone de cuisine"
    Et l'appareil détecté reste visible

  @EF-05
  Scénario: Révoquer une autre de ses sessions
    Étant donné j'ai une session ouverte sur mon ordinateur et une sur mon téléphone
    Et j'utilise mon ordinateur
    Quand je révoque la session de mon téléphone
    Alors elle n'apparaît plus dans mes sessions
    Et mon téléphone affiche l'écran de connexion à sa requête suivante
    Et je reste connecté sur mon ordinateur

  @EF-05
  Scénario: Déconnecter tous les autres appareils
    Étant donné j'ai trois sessions ouvertes dans la tribu "martin"
    Quand je choisis "déconnecter tous les autres appareils"
    Alors seule la session que j'utilise reste ouverte

  @EF-05
  Scénario: Se déconnecter
    Quand je me déconnecte
    Alors ma session courante est fermée
    Et je vois l'écran de connexion

  @EF-06
  Scénario: Réactiver un membre révoqué
    Étant donné "bruno@exemple.fr" est membre révoqué de la tribu "martin"
    Et "bruno@exemple.fr" avait une session ouverte avant sa révocation
    Quand je réactive "bruno@exemple.fr"
    Alors "bruno@exemple.fr" a le statut "actif"
    Et il peut demander un code de connexion pour la tribu "martin"
    Et son ancienne session n'est pas restaurée

  @EF-07
  Plan du Scénario: Les opérations sur les membres sont tracées
    Étant donné "bruno@exemple.fr" est membre <statut initial> de la tribu "martin"
    Quand j'effectue l'opération "<opération>" sur "bruno@exemple.fr"
    Alors le journal d'audit de la tribu "martin" contient une entrée "<opération>"
    Et cette entrée indique la date et l'heure, "bruno@exemple.fr" comme membre concerné et "alice@exemple.fr" comme auteur

    Exemples:
      | statut initial | opération   |
      | inexistant     | ajout       |
      | actif          | révocation  |
      | révoqué        | réactivation |

  @EF-07
  Plan du Scénario: Les opérations sur les sessions sont tracées
    Quand <action>
    Alors le journal d'audit de la tribu "martin" contient une entrée "<opération>"
    Et cette entrée indique la session et l'appareil détecté

    Exemples:
      | action                                                   | opération                         |
      | je me connecte depuis un nouvel appareil                 | ouverture de session              |
      | je révoque une autre de mes sessions                     | révocation de session             |
      | je choisis "déconnecter tous les autres appareils"       | déconnexion des autres appareils  |
      | je me déconnecte                                         | déconnexion                       |

  @EF-07
  Scénario: La révocation d'un membre trace la fermeture de ses sessions
    Étant donné "bruno@exemple.fr" a deux sessions ouvertes dans la tribu "martin"
    Quand je révoque "bruno@exemple.fr"
    Alors le journal d'audit contient une entrée "révocation" pour "bruno@exemple.fr"
    Et une entrée de fermeture pour chacune de ses deux sessions

  @EF-07
  Scénario: Le journal d'audit est en ajout seul
    Étant donné le journal d'audit de la tribu "martin" contient des entrées
    Alors aucune fonction de l'application ne permet de modifier ou de supprimer une entrée
