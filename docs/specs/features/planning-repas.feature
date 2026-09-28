# language: fr
Fonctionnalité: Planning des repas
  En tant que membre de la tribu,
  je veux placer des plats de la bibliothèque dans les repas de la semaine,
  afin d'organiser nos menus.

  Contexte:
    Étant donné que nous sommes le lundi 5 octobre 2026
    Et la bibliothèque contient les plats "Chili con carne" et "Chili végétarien"

  @R1
  Scénario: Afficher les repas de 7 jours à partir d'aujourd'hui
    Quand j'ouvre le planning
    Alors je vois les jours du lundi 5 au dimanche 11 octobre 2026
    Et chaque jour comporte deux repas : midi et soir

  @R2
  Scénario: Ajouter un plat à un repas
    Quand j'ajoute le plat "Chili con carne" au repas du mardi 6 octobre au soir
    Alors le repas du mardi 6 octobre au soir contient "Chili con carne" pour 4 parts

  @R2
  Scénario: Servir plusieurs plats dans un même repas
    Quand j'ajoute le plat "Chili con carne" au repas du mardi 6 octobre au soir
    Et j'ajoute le plat "Chili végétarien" au même repas
    Alors le repas du mardi 6 octobre au soir contient :
      | plat             | parts |
      | Chili con carne  | 4     |
      | Chili végétarien | 4     |

  @R2
  Scénario: Un même plat ne peut pas être ajouté deux fois à un repas
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre au soir
    Quand je choisis un plat à ajouter au repas du mardi 6 octobre au soir
    Alors "Chili con carne" ne m'est pas proposé
    Et l'API refuse l'ajout de "Chili con carne" à ce repas

  @R3
  Scénario: Modifier le nombre de parts d'un plat servi
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre au soir
    Et le plat "Chili végétarien" servi pour 4 parts le même repas
    Quand je règle "Chili con carne" à 3 parts et "Chili végétarien" à 1 part
    Alors le repas du mardi 6 octobre au soir contient :
      | plat             | parts |
      | Chili con carne  | 3     |
      | Chili végétarien | 1     |

  @R4
  Scénario: Retirer un plat d'un repas
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre au soir
    Quand je retire "Chili con carne" de ce repas
    Alors le repas du mardi 6 octobre au soir ne contient plus "Chili con carne"
    Mais le plat "Chili con carne" figure toujours dans la bibliothèque

  @R4
  Scénario: Un plat retiré puis ajouté de nouveau repart de 4 parts
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre au soir
    Quand je retire "Chili con carne" de ce repas
    Et j'ajoute de nouveau "Chili con carne" à ce repas
    Alors le repas du mardi 6 octobre au soir contient "Chili con carne" pour 4 parts

  @R5
  Scénario: Naviguer vers la semaine suivante
    Quand j'ouvre le planning
    Et je passe à la semaine suivante
    Alors je vois les jours du lundi 12 au dimanche 18 octobre 2026

  @R5
  Scénario: Naviguer vers la semaine précédente
    Quand j'ouvre le planning
    Et je passe à la semaine précédente
    Alors je vois les jours du lundi 28 septembre au dimanche 4 octobre 2026

  @R5
  Scénario: Les repas passés restent modifiables
    Quand j'ajoute le plat "Chili con carne" au repas du samedi 3 octobre au soir
    Alors le repas du samedi 3 octobre 2026 au soir contient "Chili con carne" pour 4 parts
