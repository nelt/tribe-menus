# language: fr
Fonctionnalité: Administration des tribus
  En tant qu'administrateur de l'instance,
  je veux créer des tribus et réactiver des membres depuis le serveur,
  afin d'amorcer chaque tribu et de la débloquer si elle n'a plus de membre actif.

  @EF-08
  Scénario: Initialiser une tribu
    Quand je lance le script d'initialisation
    Et je réponds "Les Martin" comme nom de la tribu
    Et je réponds "martin" comme identifiant d'URL
    Et je réponds "alice@exemple.fr" comme e-mail du premier membre
    Alors la tribu "Les Martin" d'identifiant "martin" est créée
    Et "alice@exemple.fr" en est membre actif
    Et le script affiche l'URL de la tribu "martin"
    Et aucun e-mail n'est envoyé à "alice@exemple.fr"
    Et le journal d'audit de la tribu "martin" contient une entrée "initialisation de la tribu" avec "script d'administration" comme auteur

  @EF-08
  Scénario: Le premier membre peut se connecter et ajouter des membres
    Étant donné la tribu "martin" initialisée avec le premier membre "alice@exemple.fr"
    Quand Alice se connecte à la tribu "martin" avec un code reçu par e-mail
    Alors elle peut ajouter le membre "bruno@exemple.fr"

  @EF-08
  Scénario: L'identifiant d'URL doit être libre
    Étant donné la tribu d'identifiant "martin" existe
    Quand je lance le script d'initialisation
    Et je réponds "martin" comme identifiant d'URL
    Alors le script refuse cet identifiant et m'en demande un autre

  @EF-08
  Scénario: Plusieurs tribus peuvent être créées
    Étant donné la tribu d'identifiant "martin" existe
    Quand j'initialise la tribu "Les Durand" d'identifiant "durand"
    Alors les tribus "martin" et "durand" existent toutes les deux

  @EF-08
  Scénario: Le premier membre peut déjà appartenir à une autre tribu
    Étant donné "alice@exemple.fr" est membre actif de la tribu "martin"
    Quand j'initialise la tribu "Les Durand" d'identifiant "durand" avec le premier membre "alice@exemple.fr"
    Alors "alice@exemple.fr" est membre actif des tribus "martin" et "durand"

  @EF-09
  Scénario: Réactiver un membre révoqué
    Étant donné "alice@exemple.fr" est membre révoqué de la tribu "martin"
    Quand je lance le script de réactivation
    Et je réponds "martin" comme tribu
    Et je réponds "alice@exemple.fr" comme e-mail du membre
    Alors "alice@exemple.fr" est membre actif de la tribu "martin"
    Et le journal d'audit de la tribu "martin" contient une entrée "réactivation" avec "script d'administration" comme auteur

  @EF-09
  Scénario: Débloquer une tribu sans membre actif
    Étant donné la tribu "martin" n'a plus aucun membre actif
    Et "alice@exemple.fr" est membre révoqué de la tribu "martin"
    Quand je réactive "alice@exemple.fr" dans la tribu "martin" avec le script
    Alors Alice peut se connecter à la tribu "martin"

  @EF-09
  Scénario: La réactivation ne touche que la tribu indiquée
    Étant donné "alice@exemple.fr" est membre révoqué des tribus "martin" et "durand"
    Quand je réactive "alice@exemple.fr" dans la tribu "martin" avec le script
    Alors "alice@exemple.fr" est membre actif de la tribu "martin"
    Et "alice@exemple.fr" est toujours membre révoqué de la tribu "durand"

  @EF-09
  Plan du Scénario: Refus de réactivation
    Étant donné <situation>
    Quand je réactive "alice@exemple.fr" dans la tribu "<tribu>" avec le script
    Alors le script refuse avec le message "<message>"

    Exemples:
      | situation                                                | tribu   | message                               |
      | la tribu d'identifiant "inconnue" n'existe pas           | inconnue | tribu inconnue                        |
      | "alice@exemple.fr" n'est pas membre de la tribu "martin" | martin  | ce membre n'existe pas dans la tribu  |
      | "alice@exemple.fr" est membre actif de la tribu "martin" | martin  | ce membre est déjà actif              |
