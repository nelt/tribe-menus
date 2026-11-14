# language: fr
Fonctionnalité: Compartimentage des données entre tribus
  En tant que membre d'une tribu,
  je veux que les données de ma tribu restent inaccessibles aux autres tribus,
  afin que nos menus, nos plats et nos membres restent privés.

  Contexte:
    Étant donné les tribus "Les Martin" d'identifiant "martin" et "Les Durand" d'identifiant "durand"
    Et "alice@exemple.fr" est membre actif de la tribu "martin" uniquement
    Et "david@exemple.fr" est membre actif de la tribu "durand" uniquement

  @ENF-02
  Scénario: La bibliothèque de plats est propre à la tribu
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je consulte la bibliothèque de plats
    Alors je ne vois pas le plat "Tartiflette"
    Et une recherche sur "tartiflette" ne renvoie aucun résultat

  @ENF-02
  Scénario: Le référentiel d'ingrédients est propre à la tribu
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je saisis "reb" comme ingrédient d'un plat
    Alors "reblochon" ne fait pas partie des propositions

  @ENF-02
  Scénario: Un même nom d'ingrédient est indépendant d'une tribu à l'autre
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et l'ingrédient "reblochon" de la tribu "durand" a l'unité par défaut "g"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand j'ajoute 1 pièce de "reblochon" au plat "Raclette"
    Alors l'ingrédient "reblochon" est créé dans la tribu "martin" avec l'unité par défaut "pièce"
    Et l'ingrédient "reblochon" de la tribu "durand" a toujours l'unité par défaut "g"

  @ENF-02
  Plan du scénario: Accès croisé refusé
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je tente de <action> une donnée de la tribu "durand" en utilisant son identifiant
    Alors la requête est refusée
    Et la réponse est la même que pour un identifiant qui n'existe pas
    Et la donnée de la tribu "durand" est inchangée

    Exemples:
      | action   |
      | lire     |
      | modifier |
      | supprimer |

  @ENF-02
  Plan du scénario: Toutes les données sont compartimentées
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je consulte <donnée>
    Alors je ne vois aucun élément de la tribu "durand"

    Exemples:
      | donnée                  |
      | la bibliothèque de plats |
      | la liste des membres    |
      | mes sessions            |

  @ENF-02 @T1
  Scénario: Le planning est propre à la tribu
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et le plat "Tartiflette" servi pour 4 parts le mardi 6 octobre 2026 au soir dans la tribu "durand"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je consulte le planning
    Alors je ne vois aucun élément de la tribu "durand"
    Et le repas du mardi 6 octobre 2026 au soir ne contient aucun plat

  @ENF-02 @T1
  Scénario: Les listes de courses sont propres à la tribu
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et la liste "Courses" de la tribu "durand" contient l'article ajouté "reblochon"
    Et la tribu "durand" a la liste "Marché" vide
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je consulte les listes de courses
    Alors je ne vois aucun élément de la tribu "durand"
    Et la tribu "martin" a une seule liste de courses, "Courses", vide

  @ENF-02
  Plan du scénario: Une liste d'une autre tribu ne peut être ni lue, ni modifiée, ni supprimée
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et la tribu "durand" a la liste "Marché" vide
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je tente de <action> la liste "Marché" de la tribu "durand" en utilisant son identifiant
    Alors la requête est refusée
    Et la réponse est la même que pour un identifiant qui n'existe pas
    Et la liste "Marché" de la tribu "durand" est inchangée

    Exemples:
      | action                                  |
      | renommer                                |
      | désigner comme principale               |
      | ajouter un article à                    |
      | ajouter les courses des repas à         |
      | déclarer les courses faites de          |
      | supprimer                               |

  @ENF-02
  Plan du scénario: Une liste d'une autre tribu ne peut être ni recalculée ni vidée par un déplacement
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et la liste "Courses" de la tribu "durand" contient l'article ajouté "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je tente de <action> la liste "Courses" de la tribu "durand" en utilisant son identifiant
    Alors la requête est refusée
    Et la réponse est la même que pour un identifiant qui n'existe pas
    Et l'article "reblochon" de la tribu "durand" est inchangé

    Exemples:
      | action                                 |
      | recalculer                             |
      | déplacer l'article "reblochon" depuis  |

  @ENF-02
  Scénario: Des articles ne peuvent pas être déplacés vers une liste d'une autre tribu
    Étant donné la tribu "durand" a la liste "Marché" vide
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Et la liste "Courses" contenant l'article ajouté "pain"
    Quand je tente de déplacer "pain" vers la liste "Marché" de la tribu "durand" en utilisant son identifiant
    Alors la requête est refusée
    Et la réponse est la même que pour une liste d'arrivée qui n'existe pas
    Et la liste "Courses" contient toujours l'article ajouté "pain"

  @ENF-02
  Scénario: Les listes périmées d'une autre tribu ne sont jamais visibles
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et la liste "Courses" de la tribu "durand" est périmée
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je demande quelles listes sont périmées
    Alors je ne vois aucun élément de la tribu "durand"

  @ENF-02
  Scénario: Un article d'une autre tribu ne peut être ni coché ni retiré
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et la liste "Courses" de la tribu "durand" contient l'article ajouté "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je tente de cocher puis de retirer cet article de la tribu "durand" en utilisant son identifiant et celui de sa liste
    Alors chaque requête est refusée
    Et la réponse est la même que pour un identifiant qui n'existe pas
    Et l'article "reblochon" de la tribu "durand" est inchangé

  @ENF-02
  Plan du scénario: Un plat servi d'une autre tribu ne peut être ni modifié ni retiré
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et le plat "Tartiflette" servi pour 4 parts le mardi 6 octobre 2026 au soir dans la tribu "durand"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand je tente de <action> ce plat servi de la tribu "durand" en utilisant son identifiant
    Alors la requête est refusée
    Et la réponse est la même que pour un identifiant qui n'existe pas
    Et le plat servi de la tribu "durand" est inchangé

    Exemples:
      | action              |
      | régler les parts de |
      | retirer             |

  @ENF-02
  Scénario: Un plat d'une autre tribu ne peut pas être servi
    Étant donné la tribu "durand" a le plat "Tartiflette" et l'ingrédient "reblochon"
    Et je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand j'ajoute au repas du mardi 6 octobre 2026 au soir le plat "Tartiflette" de la tribu "durand" en utilisant son identifiant
    Alors l'ajout est refusé
    Et la réponse est la même que pour un identifiant qui n'existe pas
    Et le repas du mardi 6 octobre 2026 au soir ne contient aucun plat

  @ENF-02
  Scénario: La session d'une tribu ne donne pas accès à une autre tribu
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand j'ouvre l'URL de la tribu "durand"
    Alors je vois l'écran de connexion de la tribu "durand"
    Et aucune donnée de la tribu "durand" n'est transmise

  @ENF-02
  Scénario: La tribu d'une requête est celle de la session
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand j'envoie une requête sur l'URL de la tribu "martin" en indiquant la tribu "durand" dans son contenu
    Alors la requête est traitée pour la tribu "martin" ou refusée
    Et aucune donnée de la tribu "durand" n'est lue ni modifiée

  @ENF-02
  Scénario: Une même adresse a des sessions distinctes dans chaque tribu
    Étant donné "bruno@exemple.fr" est membre actif des tribus "martin" et "durand"
    Et "bruno@exemple.fr" est connecté à la tribu "martin" dans son navigateur
    Quand Bruno ouvre l'URL de la tribu "durand" dans le même navigateur
    Alors il doit se connecter à la tribu "durand" avec un code
    Et une fois connecté, il reste connecté à la tribu "martin"

  @ENF-02
  Scénario: L'écran de connexion ne révèle pas la tribu
    Étant donné je ne suis connecté à aucune tribu
    Quand j'ouvre l'URL de la tribu "martin"
    Alors je vois l'écran de connexion
    Et le nom "Les Martin" n'apparaît nulle part dans la réponse
    Et la réponse est la même que pour l'URL d'une tribu qui n'existe pas

  @ENF-02
  Scénario: Le nom de la tribu apparaît une fois connecté
    Étant donné "alice@exemple.fr" a reçu un code de connexion pour la tribu "martin"
    Quand elle saisit ce code
    Alors elle voit le nom "Les Martin"

  @ENF-02
  Scénario: Les messages ne révèlent pas l'existence d'une autre tribu
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Quand j'ajoute le membre "david@exemple.fr"
    Alors le message affiché est le même que pour une adresse inconnue

  @ENF-02
  Scénario: Le journal d'audit est propre à la tribu
    Étant donné "david@exemple.fr" s'est connecté à la tribu "durand"
    Alors cette connexion figure dans le journal d'audit de la tribu "durand"
    Et elle ne figure pas dans le journal d'audit de la tribu "martin"
