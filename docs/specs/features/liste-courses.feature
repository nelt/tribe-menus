# language: fr
Fonctionnalité: Liste de courses
  En tant que membre de la tribu,
  je veux obtenir des listes de courses correspondant aux repas prévus,
  afin de faire les courses sans rien oublier ni acheter en trop.

  Contexte:
    Étant donné que nous sommes le lundi 5 octobre 2026
    Et la bibliothèque contient le plat "Chili con carne" défini pour 4 parts :
      | ingrédient | quantité | unité |
      | bœuf haché | 500      | g     |
      | oignon     | 1        | pièce |

  @C1
  Scénario: Période par défaut
    Quand je crée une liste de courses sans modifier les dates proposées
    Alors la période retenue va du 6 au 12 octobre 2026 inclus

  @C1
  Scénario: Choisir les dates de début et de fin
    Quand je crée une liste de courses du 8 au 11 octobre 2026
    Alors la liste porte sur la période du 8 au 11 octobre 2026 inclus

  @C1
  Scénario: La date de fin ne peut pas précéder la date de début
    Quand je choisis une date de début au jeudi 8 octobre 2026 et une date de fin au mardi 6 octobre 2026
    Alors la création de la liste est impossible
    Et un message m'indique que la date de fin doit suivre la date de début

  @C1
  Scénario: Seuls les repas de la période sont pris en compte
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le mardi 13 octobre 2026 au soir
    Quand je crée une liste de courses du 6 au 12 octobre 2026
    Alors la liste contient 500 g de "bœuf haché"

  @C2
  Scénario: Les quantités sont ajustées au nombre de parts
    Étant donné le plat "Chili con carne" servi pour 6 parts le mardi 6 octobre 2026 au soir
    Quand je crée une liste de courses du 6 au 12 octobre 2026
    Alors la liste contient 750 g de "bœuf haché"

  @C2
  Scénario: Les quantités d'un même ingrédient sont additionnées
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Quand je crée une liste de courses du 6 au 12 octobre 2026
    Alors la liste contient une seule ligne "bœuf haché" de 875 g

  @C2
  Scénario: Les unités d'une même famille sont converties
    Étant donné le plat "Crème de courgettes" défini pour 4 parts avec 50 cl de "crème"
    Et le plat "Gratin dauphinois" défini pour 4 parts avec 1 l de "crème"
    Et chacun de ces plats servi pour 4 parts dans la période
    Quand je crée une liste de courses de la période
    Alors la liste contient une seule ligne "crème" de 1,5 l

  @C2
  Scénario: Les unités de familles différentes restent sur des lignes distinctes
    Étant donné le plat "Soupe à l'oignon" défini pour 4 parts avec 300 g de "oignon"
    Et le plat "Chili con carne" servi pour 4 parts dans la période
    Et le plat "Soupe à l'oignon" servi pour 4 parts dans la période
    Quand je crée une liste de courses de la période
    Alors la liste contient 1 pièce de "oignon"
    Et la liste contient 300 g de "oignon"

  @C2
  Scénario: Les quantités à la pièce sont arrondies à l'entier supérieur
    Étant donné le plat "Chili con carne" servi pour 6 parts dans la période
    Quand je crée une liste de courses de la période
    Alors la liste contient 2 pièces de "oignon"

  @C2
  Scénario: Les cuillères à soupe et à café ne sont pas converties entre elles
    Étant donné le plat "Vinaigrette" défini pour 4 parts avec 1 cuillère à soupe de "huile d'olive"
    Et le plat "Taboulé" défini pour 4 parts avec 2 cuillères à café de "huile d'olive"
    Et chacun de ces plats servi pour 4 parts dans la période
    Quand je crée une liste de courses de la période
    Alors la liste contient 1 cuillère à soupe de "huile d'olive"
    Et la liste contient 2 cuillères à café de "huile d'olive"

  @C2
  Scénario: Les quantités en cuillères sont arrondies à l'entier supérieur
    Étant donné le plat "Vinaigrette" défini pour 4 parts avec 1 cuillère à soupe de "huile d'olive"
    Et le plat "Vinaigrette" servi pour 6 parts dans la période
    Quand je crée une liste de courses de la période
    Alors la liste contient 2 cuillères à soupe de "huile d'olive"

  @C2
  Scénario: L'arrondi s'applique après l'addition
    Étant donné le plat "Chili con carne" servi pour 2 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 2 parts le jeudi 8 octobre 2026 à midi
    Quand je crée une liste de courses du 6 au 12 octobre 2026
    Alors la liste contient 1 pièce de "oignon"

  @C2
  Scénario: Un plat sans ingrédient n'ajoute rien à la liste
    Étant donné le plat "Restes" sans ingrédient servi pour 4 parts dans la période
    Quand je crée une liste de courses de la période
    Alors la liste est vide

  @C3
  Scénario: Cocher un article
    Étant donné une liste de courses contenant "bœuf haché"
    Quand je coche "bœuf haché"
    Alors "bœuf haché" apparaît comme coché

  @C4
  Scénario: Cocher un article sans réseau
    Étant donné une liste de courses affichée contenant "bœuf haché"
    Et le réseau est coupé
    Quand je coche "bœuf haché"
    Alors "bœuf haché" apparaît comme coché

  @C4
  Scénario: Les coches faites hors ligne sont synchronisées au retour du réseau
    Étant donné que j'ai coché "bœuf haché" sans réseau
    Quand le réseau revient
    Alors "bœuf haché" apparaît comme coché pour les autres membres de la tribu

  @C4
  Scénario: Ajouter un article sans réseau
    Étant donné une liste de courses affichée
    Et le réseau est coupé
    Quand j'ajoute l'article "pain"
    Alors la liste contient l'article ajouté "pain"
    Et quand le réseau revient, "pain" apparaît pour les autres membres de la tribu

  @C4
  Plan du scénario: Les modifications hors ligne d'une liste close entre-temps sont ignorées
    Étant donné une liste de courses du 6 au 12 octobre 2026 en cours
    Et j'ai coché "bœuf haché" et ajouté l'article "pain" sans réseau
    Et entre-temps un autre membre a <action> la liste
    Quand le réseau revient
    Alors un message m'indique que la liste a été <état> entre-temps et que mes modifications hors ligne n'ont pas été appliquées
    Et la liste est inchangée

    Exemples:
      | action                        | état             |
      | déclaré les courses faites de | déclarée faite   |
      | abandonné                     | abandonnée       |

  @C4
  Scénario: La coche d'un article supprimé par un recalcul est ignorée
    Étant donné le plat "Chili con carne" servi pour 4 parts le samedi 10 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026
    Et j'ai coché "bœuf haché" sans réseau
    Et entre-temps un autre membre a modifié la date de fin de la liste au jeudi 8 octobre 2026
    Quand le réseau revient
    Alors la liste ne contient pas "bœuf haché"
    Et un message m'indique que "bœuf haché" a été retiré par un recalcul et que ma coche n'a pas été appliquée

  @C4 @C8
  Scénario: Un ingrédient créé hors ligne n'est pas dupliqué
    Étant donné que le référentiel d'ingrédients ne contient pas "lessive"
    Et j'ai ajouté l'article "lessive" à une liste sans réseau
    Et entre-temps un autre membre a créé l'ingrédient "Lessive"
    Quand le réseau revient
    Alors le référentiel d'ingrédients contient un seul ingrédient "lessive"
    Et mon article ajouté est rattaché à cet ingrédient

  @C4
  Plan du scénario: Les actions sur la liste elle-même nécessitent le réseau
    Étant donné une liste de courses en cours affichée
    Et le réseau est coupé
    Quand je tente de <action>
    Alors un message m'indique que le réseau est nécessaire
    Et la liste est inchangée

    Exemples:
      | action                        |
      | créer une nouvelle liste      |
      | modifier les dates            |
      | recalculer la liste           |
      | déclarer les courses faites   |
      | abandonner la liste           |

  @C5
  Scénario: Modifier les dates recalcule la liste
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le mardi 13 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026 contenant 500 g de "bœuf haché"
    Quand je modifie la date de fin de la liste au mardi 13 octobre 2026
    Alors la liste porte sur la période du 6 au 13 octobre 2026
    Et la liste contient 1 kg de "bœuf haché"

  @C5
  Scénario: Les articles ajoutés à la main survivent au recalcul
    Étant donné une liste de courses du 6 au 12 octobre 2026
    Et j'ai ajouté à cette liste l'article "lessive"
    Quand je modifie la date de fin de la liste au samedi 10 octobre 2026
    Alors la liste contient toujours l'article ajouté "lessive"

  @C5
  Scénario: Une coche est conservée si la quantité n'augmente pas
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le samedi 10 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026 où "oignon" est coché
    Quand je modifie la date de fin de la liste au jeudi 8 octobre 2026
    Alors la liste contient 1 pièce de "oignon"
    Et "oignon" apparaît comme coché

  @C5
  Scénario: Une coche est retirée si la quantité augmente
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le mardi 13 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026 où "bœuf haché" est coché
    Quand je modifie la date de fin de la liste au mardi 13 octobre 2026
    Alors la liste contient 1 kg de "bœuf haché"
    Et "bœuf haché" apparaît comme non coché

  @C5
  Scénario: Un article qui n'est plus nécessaire disparaît, même coché
    Étant donné le plat "Chili con carne" servi pour 4 parts le samedi 10 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026 où "bœuf haché" est coché
    Quand je modifie la date de fin de la liste au jeudi 8 octobre 2026
    Alors la liste ne contient plus "bœuf haché"

  @C6
  Scénario: Un bandeau signale que les repas ont changé
    Étant donné une liste de courses du 6 au 12 octobre 2026 contenant 500 g de "bœuf haché"
    Quand un membre règle à 6 parts le "Chili con carne" du mardi 6 octobre 2026 au soir
    Alors un bandeau en haut de la liste indique que les repas ont changé
    Et le bandeau propose l'action "Recalculer"
    Et la liste contient toujours 500 g de "bœuf haché"

  @C6
  Scénario: Recalculer met la liste à jour et fait disparaître le bandeau
    Étant donné une liste de courses du 6 au 12 octobre 2026 signalée comme périmée
    Et le "Chili con carne" du mardi 6 octobre 2026 au soir réglé à 6 parts
    Quand je touche "Recalculer"
    Alors la liste contient 750 g de "bœuf haché"
    Et le bandeau n'est plus affiché

  @C6
  Plan du scénario: Les changements qui rendent une liste périmée
    Étant donné une liste de courses du 6 au 12 octobre 2026
    Quand <changement>
    Alors le bandeau de liste périmée est affiché

    Exemples:
      | changement                                                                        |
      | un plat est ajouté à un repas de la période                                        |
      | un plat est retiré d'un repas de la période                                        |
      | les parts d'un plat servi dans la période sont modifiées                           |
      | la quantité d'un ingrédient du plat "Chili con carne", servi dans la période, change |

  @C6
  Scénario: Un changement hors de la période ne rend pas la liste périmée
    Étant donné une liste de courses du 6 au 12 octobre 2026
    Quand un plat est ajouté au repas du mardi 13 octobre 2026 au soir
    Alors le bandeau de liste périmée n'est pas affiché

  @C6
  Scénario: Un changement sans effet sur les articles ne rend pas la liste périmée
    Étant donné le plat "Restes" sans ingrédient
    Et une liste de courses du 6 au 12 octobre 2026
    Quand le plat "Restes" est ajouté au repas du mercredi 7 octobre 2026 à midi
    Alors le bandeau de liste périmée n'est pas affiché

  @C7
  Scénario: Plusieurs listes en cours en même temps
    Étant donné une liste de courses du 6 au 8 octobre 2026
    Quand je crée une liste de courses du 9 au 12 octobre 2026
    Alors l'onglet Courses présente deux listes en cours : « 6 → 8 oct. » puis « 9 → 12 oct. »

  @C7
  Scénario: Des listes peuvent porter sur des périodes qui se chevauchent
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026
    Quand je crée une liste de courses du 6 au 8 octobre 2026
    Alors chacune des deux listes contient 500 g de "bœuf haché"

  @C7
  Scénario: Les listes sont indépendantes
    Étant donné deux listes de courses en cours contenant chacune "bœuf haché"
    Quand je coche "bœuf haché" dans la première
    Alors "bœuf haché" n'est pas coché dans la seconde

  @C8
  Scénario: Ajouter un article hors planning
    Étant donné une liste de courses du 6 au 12 octobre 2026
    Quand j'ajoute l'article "pain" pour 2 pièces
    Alors la liste contient l'article ajouté "pain" pour 2 pièces
    Et l'article "pain" n'est rattaché à aucun repas

  @C8
  Scénario: La quantité d'un article ajouté est facultative
    Étant donné une liste de courses
    Quand j'ajoute l'article "lessive" sans quantité
    Alors la liste contient l'article ajouté "lessive" sans quantité

  @C8
  Scénario: Un article ajouté n'est pas fusionné avec un article calculé
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026
    Quand j'ajoute l'article "oignon" pour 3 pièces
    Alors la liste contient 1 pièce de "oignon" rattachée à 1 repas
    Et la liste contient l'article ajouté "oignon" pour 3 pièces

  @C8
  Scénario: Un nouvel article enrichit le référentiel d'ingrédients
    Étant donné que le référentiel d'ingrédients ne contient pas "lessive"
    Quand j'ajoute l'article "lessive" à une liste de courses
    Alors "lessive" est proposé par l'autocomplétion lors de la saisie d'un ingrédient

  @C8
  Scénario: Retirer un article ajouté
    Étant donné une liste de courses contenant l'article ajouté "pain"
    Quand je retire "pain" de la liste
    Alors la liste ne contient plus "pain"

  @C9
  Scénario: Un article indique le nombre de repas dont il provient
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Quand je consulte la liste de courses du 6 au 12 octobre 2026
    Alors la ligne "bœuf haché" indique qu'elle provient de 2 repas

  @C9
  Scénario: Déplier un article pour voir les repas et plats concernés
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre 2026 à midi
    Et la liste de courses du 6 au 12 octobre 2026
    Quand je touche la ligne "bœuf haché"
    Alors elle se déplie et montre :
      | repas               | plat            | parts | quantité |
      | mardi 6 oct., soir  | Chili con carne | 3     | 375 g    |
      | jeudi 8 oct., midi  | Chili con carne | 4     | 500 g    |
    Et je la replie en la touchant de nouveau

  @C9
  Scénario: Un article ajouté ne se déplie pas
    Étant donné une liste de courses contenant l'article ajouté "pain"
    Alors la ligne "pain" est marquée comme ajoutée à la main
    Et elle ne propose pas de détail de repas

  @C10
  Scénario: Déclarer les courses faites
    Étant donné une liste de courses du 6 au 12 octobre 2026 en cours
    Quand je déclare les courses faites
    Alors la liste n'apparaît plus parmi les listes en cours
    Et elle apparaît en tête de l'historique

  @C10
  Scénario: Les courses peuvent être déclarées faites avec des articles non cochés
    Étant donné une liste de courses où "oignon" n'est pas coché
    Quand je déclare les courses faites
    Alors la liste apparaît dans l'historique avec "oignon" non coché

  @C10
  Scénario: Une liste faite est en lecture seule
    Étant donné une liste de courses faite, consultée depuis l'historique
    Alors je ne peux ni cocher, ni ajouter d'article, ni modifier les dates, ni recalculer
    Et aucun bandeau de liste périmée n'est affiché, même si les repas ont changé

  @C11
  Scénario: Abandonner une liste
    Étant donné une liste de courses du 6 au 12 octobre 2026 en cours
    Quand j'abandonne la liste
    Alors une confirmation m'est demandée
    Et après confirmation, la liste n'apparaît ni parmi les listes en cours ni dans l'historique

  @C11
  Scénario: Renoncer à abandonner une liste
    Étant donné une liste de courses du 6 au 12 octobre 2026 en cours
    Quand j'abandonne la liste puis j'annule la confirmation
    Alors la liste est toujours en cours et inchangée

  @C11
  Scénario: Abandonner une liste ne touche pas au planning
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et une liste de courses du 6 au 12 octobre 2026
    Quand j'abandonne la liste
    Alors le repas du mardi 6 octobre 2026 au soir contient toujours "Chili con carne" pour 4 parts
