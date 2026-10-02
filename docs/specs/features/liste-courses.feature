# language: fr
Fonctionnalité: Listes de courses
  En tant que membre de la tribu,
  je veux tenir des listes de courses, alimentées à la main ou par les repas prévus,
  afin de faire les courses sans rien oublier ni acheter en trop.

  Contexte:
    Étant donné que nous sommes le lundi 5 octobre 2026
    Et la liste principale de la tribu est la liste "Courses"
    Et la bibliothèque contient le plat "Chili con carne" défini pour 4 parts :
      | ingrédient | quantité | unité |
      | bœuf haché | 500      | g     |
      | oignon     | 1        | pièce |

  @C1
  Scénario: Une tribu naît avec une liste principale
    Quand une nouvelle tribu est créée
    Alors elle a une seule liste de courses, "Courses", vide
    Et "Courses" est sa liste principale

  @C1
  Scénario: L'onglet Courses ouvre la liste principale
    Étant donné les listes "Courses" et "Marché"
    Quand j'ouvre l'onglet Courses
    Alors la liste "Courses" est affichée

  @C1
  Scénario: Changer de liste principale
    Étant donné les listes "Courses" et "Marché"
    Quand je désigne la liste "Marché" comme principale
    Alors la liste principale de la tribu est la liste "Marché"
    Et la liste "Courses" n'est plus principale

  @C1
  Scénario: Il y a toujours exactement une liste principale
    Étant donné les listes "Courses" et "Marché"
    Alors aucune action ne permet de retirer son rôle à la liste principale sans en désigner une autre

  @C2
  Scénario: Créer une liste vide
    Quand je crée la liste "Marché"
    Alors la liste "Marché" existe et est vide
    Et la liste principale de la tribu est toujours la liste "Courses"

  @C2
  Scénario: Le nom d'une liste est obligatoire
    Quand je crée une liste sans nom
    Alors la création de la liste est impossible
    Et un message m'indique que la liste doit avoir un nom

  @C2
  Scénario: Renommer une liste
    Étant donné la liste "Marché"
    Quand je renomme la liste "Marché" en "Marché du samedi"
    Alors la liste "Marché du samedi" existe

  @C2
  Scénario: Ordre des listes
    Étant donné que j'ai créé la liste "Marché" puis la liste "Drive"
    Et la liste principale de la tribu est la liste "Drive"
    Quand j'ouvre le choix des listes
    Alors les listes sont présentées dans l'ordre : "Drive", "Courses", "Marché"

  @C3
  Scénario: Ajouter un article à la main
    Quand j'ajoute l'article "pain" pour 2 pièces à la liste "Courses"
    Alors la liste "Courses" contient l'article ajouté "pain" pour 2 pièces
    Et l'article "pain" n'est rattaché à aucun plat

  @C3
  Scénario: La quantité d'un article ajouté est facultative
    Quand j'ajoute l'article "lessive" sans quantité à la liste "Courses"
    Alors la liste "Courses" contient l'article ajouté "lessive" sans quantité

  @C3
  Scénario: Un nouvel article enrichit le référentiel d'ingrédients
    Étant donné que le référentiel d'ingrédients ne contient pas "lessive"
    Quand j'ajoute l'article "lessive" à la liste "Courses"
    Alors "lessive" est proposé par l'autocomplétion lors de la saisie d'un ingrédient

  @C3
  Scénario: Retirer un article ajouté
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Quand je retire "pain" de la liste "Courses"
    Alors la liste "Courses" ne contient plus "pain"

  @C4
  Scénario: Période et liste par défaut
    Quand j'ajoute les courses des repas sans modifier la période ni la liste proposées
    Alors la période retenue va du 6 au 12 octobre 2026 inclus
    Et les courses sont ajoutées à la liste "Courses"

  @C4
  Scénario: Choisir la liste qui reçoit les courses
    Étant donné la liste "Marché"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Marché"
    Alors la liste "Marché" contient 500 g de "bœuf haché"
    Et la liste "Courses" est vide

  @C4
  Scénario: La date de fin ne peut pas précéder la date de début
    Quand je choisis une date de début au jeudi 8 octobre 2026 et une date de fin au mardi 6 octobre 2026
    Alors l'ajout des courses est impossible
    Et un message m'indique que la date de fin doit suivre la date de début

  @C4
  Scénario: Seuls les repas de la période sont pris en compte
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le mardi 13 octobre 2026 au soir
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient 500 g de "bœuf haché"

  @C4
  Scénario: Les quantités sont ajustées au nombre de parts
    Étant donné le plat "Chili con carne" servi pour 6 parts le mardi 6 octobre 2026 au soir
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient 750 g de "bœuf haché"

  @C4
  Scénario: Les quantités d'un même ingrédient sont additionnées
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient une seule ligne "bœuf haché" de 875 g

  @C4
  Scénario: Les unités d'une même famille sont converties
    Étant donné le plat "Crème de courgettes" défini pour 4 parts avec 50 cl de "crème"
    Et le plat "Gratin dauphinois" défini pour 4 parts avec 1 l de "crème"
    Et chacun de ces plats servi pour 4 parts dans la période
    Quand j'ajoute les courses des repas de la période à la liste "Courses"
    Alors la liste "Courses" contient une seule ligne "crème" de 1,5 l

  @C4
  Scénario: Les unités de familles différentes restent sur des lignes distinctes
    Étant donné le plat "Soupe à l'oignon" défini pour 4 parts avec 300 g de "oignon"
    Et le plat "Chili con carne" servi pour 4 parts dans la période
    Et le plat "Soupe à l'oignon" servi pour 4 parts dans la période
    Quand j'ajoute les courses des repas de la période à la liste "Courses"
    Alors la liste "Courses" contient 1 pièce de "oignon"
    Et la liste "Courses" contient 300 g de "oignon"

  @C4
  Scénario: Les quantités à la pièce sont arrondies à l'entier supérieur
    Étant donné le plat "Chili con carne" servi pour 6 parts dans la période
    Quand j'ajoute les courses des repas de la période à la liste "Courses"
    Alors la liste "Courses" contient 2 pièces de "oignon"

  @C4
  Scénario: Les cuillères à soupe et à café ne sont pas converties entre elles
    Étant donné le plat "Vinaigrette" défini pour 4 parts avec 1 cuillère à soupe de "huile d'olive"
    Et le plat "Taboulé" défini pour 4 parts avec 2 cuillères à café de "huile d'olive"
    Et chacun de ces plats servi pour 4 parts dans la période
    Quand j'ajoute les courses des repas de la période à la liste "Courses"
    Alors la liste "Courses" contient 1 cuillère à soupe de "huile d'olive"
    Et la liste "Courses" contient 2 cuillères à café de "huile d'olive"

  @C4
  Scénario: Les quantités en cuillères sont arrondies à l'entier supérieur
    Étant donné le plat "Vinaigrette" défini pour 4 parts avec 1 cuillère à soupe de "huile d'olive"
    Et le plat "Vinaigrette" servi pour 6 parts dans la période
    Quand j'ajoute les courses des repas de la période à la liste "Courses"
    Alors la liste "Courses" contient 2 cuillères à soupe de "huile d'olive"

  @C4
  Scénario: L'arrondi s'applique après l'addition
    Étant donné le plat "Chili con carne" servi pour 2 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 2 parts le jeudi 8 octobre 2026 à midi
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient 1 pièce de "oignon"

  @C4
  Scénario: Un plat sans ingrédient n'ajoute rien à la liste
    Étant donné le plat "Restes" sans ingrédient servi pour 4 parts dans la période
    Quand j'ajoute les courses des repas de la période à la liste "Courses"
    Alors la liste "Courses" est vide

  @C4
  Scénario: Les courses s'ajoutent aux articles déjà présents
    Étant donné la liste "Courses" contenant 250 g de "bœuf haché" provenant du "Chili con carne" du lundi 5 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient une seule ligne "bœuf haché" de 750 g

  @C4
  Scénario: Un article coché dont la quantité augmente est décoché
    Étant donné la liste "Courses" contenant 250 g de "bœuf haché" provenant du "Chili con carne" du lundi 5 octobre 2026 au soir
    Et "bœuf haché" est coché dans la liste "Courses"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors "bœuf haché" apparaît comme non coché dans la liste "Courses"

  @C4
  Scénario: Un article ajouté n'est pas fusionné avec un article calculé
    Étant donné la liste "Courses" contenant l'article ajouté "oignon" pour 3 pièces
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient 1 pièce de "oignon" provenant de 1 plat servi
    Et la liste "Courses" contient l'article ajouté "oignon" pour 3 pièces

  @C4
  Scénario: Un plat servi déjà reçu par la liste n'est pas ajouté une seconde fois
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient une seule ligne "bœuf haché" de 1 kg
    Et un message m'indique que 1 plat servi déjà reçu par la liste a été ignoré

  @C4
  Scénario: Un plat servi dont les articles ont été achetés n'est pas ajouté de nouveau
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et j'ai coché tous les articles de la liste "Courses" et déclaré les courses faites
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" est vide
    Et un message m'indique que 1 plat servi déjà reçu par la liste a été ignoré

  @C4
  Scénario: Un même plat servi peut être reçu par deux listes
    Étant donné la liste "Marché"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Marché"
    Alors chacune des listes "Courses" et "Marché" contient 500 g de "bœuf haché"

  @C5
  Scénario: Un article indique le nombre de plats servis dont il provient
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la ligne "bœuf haché" indique qu'elle provient de 2 plats servis

  @C5
  Scénario: Déplier un article pour voir les plats et repas concernés
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand je touche la ligne "bœuf haché"
    Alors elle se déplie et montre :
      | repas               | plat            | parts | quantité |
      | mardi 6 oct., soir  | Chili con carne | 3     | 375 g    |
      | jeudi 8 oct., midi  | Chili con carne | 4     | 500 g    |
    Et je la replie en la touchant de nouveau

  @C5
  Scénario: Un article ajouté ne se déplie pas
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Alors la ligne "pain" est marquée comme ajoutée à la main
    Et elle ne propose pas de détail de repas

  @C6
  Scénario: Cocher un article
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Quand je coche "pain" dans la liste "Courses"
    Alors "pain" apparaît comme coché dans la liste "Courses"

  @C7
  Scénario: Cocher un article sans réseau
    Étant donné la liste "Courses" affichée contenant l'article ajouté "pain"
    Et le réseau est coupé
    Quand je coche "pain" dans la liste "Courses"
    Alors "pain" apparaît comme coché dans la liste "Courses"

  @C7
  Scénario: Les coches faites hors ligne sont synchronisées au retour du réseau
    Étant donné que j'ai coché "pain" dans la liste "Courses" sans réseau
    Quand le réseau revient
    Alors "pain" apparaît comme coché dans la liste "Courses" pour les autres membres de la tribu

  @C7
  Scénario: Ajouter un article sans réseau
    Étant donné la liste "Courses" affichée
    Et le réseau est coupé
    Quand j'ajoute l'article "pain" à la liste "Courses"
    Alors la liste "Courses" contient l'article ajouté "pain"
    Et quand le réseau revient, "pain" apparaît pour les autres membres de la tribu

  @C7
  Scénario: Les articles ajoutés hors ligne à une liste supprimée entre-temps sont ignorés
    Étant donné la liste "Marché" affichée et vide
    Et j'ai ajouté l'article "pain" à la liste "Marché" sans réseau
    Et entre-temps un autre membre a supprimé la liste "Marché"
    Quand le réseau revient
    Alors un message m'indique que la liste "Marché" a été supprimée entre-temps et que "pain" n'a pas été ajouté

  @C7 @C9
  Scénario: La coche d'un article déplacé entre-temps s'applique dans sa nouvelle liste
    Étant donné la liste "Marché"
    Et la liste "Courses" contenant l'article ajouté "pain"
    Et j'ai coché "pain" dans la liste "Courses" sans réseau
    Et entre-temps un autre membre a déplacé "pain" vers la liste "Marché"
    Quand le réseau revient
    Alors "pain" apparaît comme coché dans la liste "Marché"

  @C7 @C8
  Scénario: La coche d'un article supprimé par un recalcul est ignorée
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et j'ai coché "bœuf haché" dans la liste "Courses" sans réseau
    Et entre-temps un autre membre a retiré le "Chili con carne" du repas du mardi 6 octobre 2026 au soir et recalculé la liste "Courses"
    Quand le réseau revient
    Alors la liste "Courses" ne contient pas "bœuf haché"
    Et un message m'indique que "bœuf haché" a été retiré par un recalcul et que ma coche n'a pas été appliquée

  @C7 @C3
  Scénario: Un ingrédient créé hors ligne n'est pas dupliqué
    Étant donné que le référentiel d'ingrédients ne contient pas "lessive"
    Et j'ai ajouté l'article "lessive" à la liste "Courses" sans réseau
    Et entre-temps un autre membre a créé l'ingrédient "Lessive"
    Quand le réseau revient
    Alors le référentiel d'ingrédients contient un seul ingrédient "lessive"
    Et mon article ajouté est rattaché à cet ingrédient

  @C7
  Plan du scénario: Les actions sur les listes elles-mêmes nécessitent le réseau
    Étant donné la liste "Courses" affichée
    Et le réseau est coupé
    Quand je tente de <action>
    Alors un message m'indique que le réseau est nécessaire
    Et rien n'est modifié

    Exemples:
      | action                                  |
      | ajouter les courses des repas           |
      | recalculer la liste                     |
      | déplacer des articles sélectionnés      |
      | créer une liste                         |
      | créer une liste à partir d'une sélection |
      | renommer la liste                       |
      | supprimer une liste                     |
      | changer la liste principale             |
      | déclarer les courses faites             |

  @C8
  Scénario: Un bandeau signale que des plats servis ont changé
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand un membre règle à 6 parts le "Chili con carne" du mardi 6 octobre 2026 au soir
    Alors un bandeau en haut de la liste "Courses" indique que des repas ont changé
    Et le bandeau propose l'action "Recalculer"
    Et la liste "Courses" contient toujours 500 g de "bœuf haché"

  @C8
  Scénario: Recalculer met la liste à jour et fait disparaître le bandeau
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et le "Chili con carne" du mardi 6 octobre 2026 au soir réglé à 6 parts
    Quand je touche "Recalculer"
    Alors la liste "Courses" contient 750 g de "bœuf haché"
    Et le bandeau n'est plus affiché

  @C8
  Plan du scénario: Les changements qui rendent une liste périmée
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand <changement>
    Alors le bandeau de liste périmée est affiché

    Exemples:
      | changement                                                                  |
      | le "Chili con carne" est retiré du repas du mardi 6 octobre 2026 au soir      |
      | les parts du "Chili con carne" du mardi 6 octobre 2026 au soir sont modifiées |
      | la quantité de "bœuf haché" du plat "Chili con carne" change                  |

  @C8
  Scénario: Un plat ajouté au planning ne rend pas la liste périmée
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand le plat "Chili con carne" est ajouté au repas du jeudi 8 octobre 2026 à midi
    Alors le bandeau de liste périmée n'est pas affiché

  @C8
  Scénario: Un changement sans effet sur les articles ne rend pas la liste périmée
    Étant donné le plat "Restes" sans ingrédient servi pour 4 parts le mercredi 7 octobre 2026 à midi
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand les parts des "Restes" du mercredi 7 octobre 2026 à midi sont modifiées
    Alors le bandeau de liste périmée n'est pas affiché

  @C8
  Scénario: Les articles ajoutés à la main survivent au recalcul
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et la liste "Courses" contenant l'article ajouté "lessive"
    Et le "Chili con carne" du mardi 6 octobre 2026 au soir réglé à 6 parts
    Quand je touche "Recalculer"
    Alors la liste "Courses" contient toujours l'article ajouté "lessive"

  @C8
  Scénario: Une coche est conservée si la quantité n'augmente pas
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et "bœuf haché" est coché dans la liste "Courses"
    Et le "Chili con carne" du mardi 6 octobre 2026 au soir réglé à 2 parts
    Quand je touche "Recalculer"
    Alors la liste "Courses" contient 250 g de "bœuf haché"
    Et "bœuf haché" apparaît comme coché dans la liste "Courses"

  @C8
  Scénario: Une coche est retirée si la quantité augmente
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et "bœuf haché" est coché dans la liste "Courses"
    Et le "Chili con carne" du mardi 6 octobre 2026 au soir réglé à 6 parts
    Quand je touche "Recalculer"
    Alors la liste "Courses" contient 750 g de "bœuf haché"
    Et "bœuf haché" apparaît comme non coché dans la liste "Courses"

  @C8
  Scénario: Un article dont tous les plats servis ont été retirés disparaît, même coché
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et "bœuf haché" est coché dans la liste "Courses"
    Et le "Chili con carne" retiré du repas du mardi 6 octobre 2026 au soir
    Quand je touche "Recalculer"
    Alors la liste "Courses" ne contient plus "bœuf haché"

  @C8
  Scénario: Le recalcul ne recrée pas les articles déjà achetés
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et j'ai coché "bœuf haché" dans la liste "Courses" et déclaré les courses faites
    Et le "Chili con carne" du mardi 6 octobre 2026 au soir réglé à 6 parts
    Quand je touche "Recalculer"
    Alors la liste "Courses" ne contient pas "bœuf haché"
    Et la liste "Courses" contient 2 pièces de "oignon"

  @C9
  Scénario: Déplacer des articles sélectionnés vers une autre liste
    Étant donné la liste "Marché"
    Et la liste "Courses" contenant les articles ajoutés "pain" et "lessive"
    Quand je sélectionne "pain" dans la liste "Courses"
    Et je déplace la sélection vers la liste "Marché"
    Alors la liste "Marché" contient l'article ajouté "pain"
    Et la liste "Courses" ne contient plus "pain"
    Et la liste "Courses" contient toujours l'article ajouté "lessive"

  @C9
  Scénario: Un article déplacé garde sa coche et ses provenances
    Étant donné la liste "Marché"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et "bœuf haché" est coché dans la liste "Courses"
    Quand je déplace "bœuf haché" de la liste "Courses" vers la liste "Marché"
    Alors la liste "Marché" contient 500 g de "bœuf haché" provenant de 1 plat servi
    Et "bœuf haché" apparaît comme coché dans la liste "Marché"

  @C9
  Scénario: Un article calculé déplacé est fusionné avec le même ingrédient
    Étant donné la liste "Marché" contenant 500 g de "bœuf haché" provenant du "Chili con carne" du mardi 6 octobre 2026 au soir
    Et la liste "Courses" contenant 500 g de "bœuf haché" provenant du "Chili con carne" du jeudi 8 octobre 2026 à midi
    Quand je déplace "bœuf haché" de la liste "Courses" vers la liste "Marché"
    Alors la liste "Marché" contient une seule ligne "bœuf haché" de 1 kg
    Et la ligne "bœuf haché" de la liste "Marché" indique qu'elle provient de 2 plats servis

  @C9
  Scénario: Un plat servi déjà présent dans la liste d'arrivée n'est pas compté deux fois
    Étant donné la liste "Marché"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Marché"
    Quand je déplace "bœuf haché" de la liste "Courses" vers la liste "Marché"
    Alors la liste "Marché" contient une seule ligne "bœuf haché" de 500 g

  @C9
  Scénario: L'article fusionné n'est coché que si les deux l'étaient
    Étant donné la liste "Marché" contenant 500 g de "bœuf haché" provenant du "Chili con carne" du mardi 6 octobre 2026 au soir
    Et la liste "Courses" contenant 500 g de "bœuf haché" provenant du "Chili con carne" du jeudi 8 octobre 2026 à midi
    Et "bœuf haché" est coché dans la liste "Courses"
    Quand je déplace "bœuf haché" de la liste "Courses" vers la liste "Marché"
    Alors "bœuf haché" apparaît comme non coché dans la liste "Marché"

  @C9
  Scénario: Un article ajouté déplacé reste une ligne distincte
    Étant donné la liste "Marché" contenant l'article ajouté "pain" pour 1 pièce
    Et la liste "Courses" contenant l'article ajouté "pain" pour 2 pièces
    Quand je déplace "pain" de la liste "Courses" vers la liste "Marché"
    Alors la liste "Marché" contient l'article ajouté "pain" pour 1 pièce
    Et la liste "Marché" contient l'article ajouté "pain" pour 2 pièces

  @C9 @C4
  Scénario: Les plats servis des articles déplacés restent reçus par la liste de départ
    Étant donné la liste "Marché"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et j'ai déplacé "bœuf haché" de la liste "Courses" vers la liste "Marché"
    Quand j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" ne contient pas "bœuf haché"

  @C10
  Scénario: Créer une liste à partir d'une sélection
    Étant donné la liste "Courses" contenant les articles ajoutés "pain" et "lessive"
    Quand je sélectionne "lessive" dans la liste "Courses"
    Et je crée la liste "Drive" à partir de la sélection
    Alors la liste "Drive" contient l'article ajouté "lessive"
    Et la liste "Courses" ne contient plus "lessive"
    Et la liste principale de la tribu est toujours la liste "Courses"

  @C10
  Scénario: Une liste créée à partir d'une sélection garde les provenances
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Quand je sélectionne "bœuf haché" dans la liste "Courses"
    Et je crée la liste "Boucherie" à partir de la sélection
    Alors la liste "Boucherie" contient 500 g de "bœuf haché" provenant de 1 plat servi

  @C9 @C10
  Scénario: La sélection est impossible sans article sélectionné
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Quand aucun article n'est sélectionné
    Alors les actions de déplacement et de création à partir de la sélection sont indisponibles

  @C11
  Scénario: Déclarer les courses faites retire les articles cochés
    Étant donné la liste "Courses" contenant les articles ajoutés "pain" et "lessive"
    Et "pain" est coché dans la liste "Courses"
    Quand je déclare les courses faites pour la liste "Courses"
    Alors la liste "Courses" ne contient plus "pain"
    Et la liste "Courses" contient toujours l'article ajouté "lessive", non coché

  @C11
  Scénario: La liste reste utilisable après les courses faites
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Et "pain" est coché dans la liste "Courses"
    Quand je déclare les courses faites pour la liste "Courses"
    Alors la liste "Courses" existe et est vide
    Et la liste principale de la tribu est toujours la liste "Courses"

  @C11
  Scénario: Les courses faites ne touchent pas au planning
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ajouté les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"
    Et j'ai coché tous les articles de la liste "Courses"
    Quand je déclare les courses faites pour la liste "Courses"
    Alors le repas du mardi 6 octobre 2026 au soir contient toujours "Chili con carne" pour 4 parts

  @C12
  Scénario: Supprimer une liste vide
    Étant donné la liste "Marché" vide
    Quand je supprime la liste "Marché"
    Alors la liste "Marché" n'existe plus

  @C12
  Scénario: Une liste qui contient des articles ne peut pas être supprimée
    Étant donné la liste "Marché" contenant l'article ajouté "pain"
    Quand je tente de supprimer la liste "Marché"
    Alors la suppression est impossible
    Et un message m'indique que la liste doit être vide pour être supprimée

  @C12 @C1
  Scénario: La liste principale ne peut pas être supprimée
    Étant donné la liste "Courses" vide
    Quand je tente de supprimer la liste "Courses"
    Alors la suppression est impossible
    Et un message m'indique qu'il faut d'abord désigner une autre liste comme principale
