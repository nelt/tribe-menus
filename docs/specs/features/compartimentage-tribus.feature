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
      | le planning             |
      | les listes de courses   |
      | la liste des membres    |
      | mes sessions            |

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
