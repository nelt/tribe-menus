# language: fr
Fonctionnalité: Administration des tribus
  En tant qu'administrateur de l'instance,
  je veux créer, débloquer et supprimer des tribus, et anonymiser des membres depuis le serveur,
  afin d'amorcer chaque tribu, de la débloquer si elle n'a plus de membre actif et de répondre aux demandes d'effacement.

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
    Quand "alice@exemple.fr" se connecte à la tribu "martin" avec un code reçu par e-mail
    Alors elle peut ajouter le membre "bruno@exemple.fr"

  @EF-08
  Scénario: L'identifiant d'URL doit être libre
    Étant donné la tribu d'identifiant "martin" existe
    Quand je lance le script d'initialisation
    Et je réponds "martin" comme identifiant d'URL
    Alors le script refuse cet identifiant et m'en demande un autre

  @EF-08
  Plan du scénario: L'identifiant d'URL doit respecter le format
    Quand je lance le script d'initialisation
    Et je réponds "<identifiant>" comme identifiant d'URL
    Alors le script refuse cet identifiant et m'en demande un autre

    Exemples:
      | identifiant | raison                               |
      | Martin      | majuscule                            |
      | les_durand  | tiret bas                            |
      | é-nous      | lettre accentuée                     |
      | 42          | ne commence pas par une lettre       |
      | -martin     | commence par un tiret                |
      | ab          | moins de 3 caractères                |

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
  Plan du scénario: Refus de réactivation
    Étant donné <situation>
    Quand je réactive "alice@exemple.fr" dans la tribu "<tribu>" avec le script
    Alors le script refuse avec le message "<message>"

    Exemples:
      | situation                                                | tribu   | message                               |
      | la tribu d'identifiant "inconnue" n'existe pas           | inconnue | tribu inconnue                        |
      | "alice@exemple.fr" n'est pas membre de la tribu "martin" | martin  | ce membre n'existe pas dans la tribu  |
      | "alice@exemple.fr" est membre actif de la tribu "martin" | martin  | ce membre est déjà actif              |

  @EF-10
  Scénario: Supprimer une tribu
    Étant donné la tribu "Les Martin" d'identifiant "martin" existe
    Quand je lance le script de suppression
    Et je réponds "martin" comme tribu
    Alors le script affiche "Les Martin" et le nombre de membres
    Quand je retape "martin" pour confirmer
    Alors la tribu "martin" n'existe plus
    Et l'URL de la tribu "martin" répond comme celle d'une tribu qui n'existe pas

  @EF-10
  Scénario: Une confirmation erronée annule la suppression
    Étant donné la tribu d'identifiant "martin" existe
    Quand je lance le script de suppression pour la tribu "martin"
    Et je retape "martine" pour confirmer
    Alors le script abandonne
    Et la tribu "martin" existe toujours

  @EF-10
  Scénario: La suppression ne touche pas les autres tribus
    Étant donné les tribus "martin" et "durand" existent
    Quand je supprime la tribu "martin" avec le script
    Alors la tribu "durand" et ses données sont inchangées

  @EF-11
  Scénario: Anonymiser un membre révoqué
    Étant donné "bruno@exemple.fr" est membre révoqué de la tribu "martin"
    Et "bruno@exemple.fr" a créé le plat "Ratatouille"
    Quand j'anonymise "bruno@exemple.fr" dans la tribu "martin" avec le script
    Alors "bruno@exemple.fr" n'apparaît plus ni dans la liste des membres ni dans le journal d'audit de la tribu "martin"
    Et la liste des membres contient un « membre anonymisé » au statut "révoqué"
    Et le plat "Ratatouille" figure toujours dans la bibliothèque
    Et le journal d'audit contient une entrée "anonymisation" avec "script d'administration" comme auteur

  @EF-11
  Scénario: Un membre actif ne peut pas être anonymisé
    Étant donné "bruno@exemple.fr" est membre actif de la tribu "martin"
    Quand j'anonymise "bruno@exemple.fr" dans la tribu "martin" avec le script
    Alors le script refuse avec le message "révoquez d'abord ce membre"

  @EF-11
  Scénario: Une adresse anonymisée peut être ajoutée de nouveau
    Étant donné "bruno@exemple.fr" a été anonymisé dans la tribu "martin"
    Quand un membre ajoute "bruno@exemple.fr" à la tribu "martin"
    Alors "bruno@exemple.fr" figure dans la liste des membres avec le statut "actif"
