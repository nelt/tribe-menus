# language: fr
Fonctionnalité: Planning des repas
  En tant que membre de la tribu,
  je veux placer des plats de la bibliothèque dans les repas de la semaine,
  afin d'organiser nos menus.

  Contexte:
    Étant donné que nous sommes le lundi 5 octobre 2026
    Et la bibliothèque contient les plats "Chili con carne" et "Chili végétarien"

  @R1 @ui
  Scénario: Afficher les repas de 7 jours à partir d'aujourd'hui
    Quand j'ouvre le planning
    Alors je vois les jours du lundi 5 au dimanche 11 octobre 2026
    Et chaque jour comporte deux repas : midi et soir

  @R2
  Scénario: Ajouter un plat à un repas
    Quand j'ajoute le plat "Chili con carne" au repas du mardi 6 octobre 2026 au soir
    Alors le repas du mardi 6 octobre 2026 au soir contient "Chili con carne" pour 4 parts

  @R2
  Scénario: Servir plusieurs plats dans un même repas
    Quand j'ajoute le plat "Chili con carne" au repas du mardi 6 octobre 2026 au soir
    Et j'ajoute le plat "Chili végétarien" au même repas
    Alors le repas du mardi 6 octobre 2026 au soir contient :
      | plat             | parts |
      | Chili con carne  | 4     |
      | Chili végétarien | 4     |

  @R2
  Scénario: Un même plat ne peut pas être ajouté deux fois à un repas
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand j'ajoute le plat "Chili con carne" au repas du mardi 6 octobre 2026 au soir
    Alors l'ajout est refusé
    Et un message m'indique que "Chili con carne" est déjà servi dans ce repas
    Et le repas du mardi 6 octobre 2026 au soir contient :
      | plat            | parts |
      | Chili con carne | 4     |

  @R2 @ui
  Scénario: Un plat déjà servi dans un repas ne m'est pas proposé
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand je choisis un plat à ajouter au repas du mardi 6 octobre 2026 au soir
    Alors "Chili con carne" ne m'est pas proposé
    Mais "Chili végétarien" m'est proposé

  @R2
  Scénario: Un plat absent de la bibliothèque ne peut pas être servi
    Quand j'ajoute au repas du mardi 6 octobre 2026 au soir un plat dont l'identifiant est inconnu de la bibliothèque
    Alors l'ajout est refusé
    Et le repas du mardi 6 octobre 2026 au soir ne contient aucun plat

  @R2
  Scénario: Un plat servi entre-temps par un autre membre n'est pas ajouté une seconde fois
    Étant donné j'ai ouvert le repas du mardi 6 octobre 2026 au soir
    Et Bruno a ajouté entre-temps le plat "Chili con carne" à ce repas
    Quand j'ajoute le plat "Chili con carne" à ce repas
    Alors l'ajout est refusé
    Et un message m'indique que "Chili con carne" est déjà servi dans ce repas
    Et le repas à jour m'est présenté, avec "Chili con carne" pour 4 parts

  @R2 @R4
  Scénario: Les plats d'un repas sont dans l'ordre où ils ont été ajoutés
    Étant donné le plat "Chili végétarien" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili con carne" servi pour 4 parts le même repas
    Quand je retire "Chili végétarien" de ce repas
    Et j'ajoute de nouveau "Chili végétarien" à ce repas
    Alors le repas du mardi 6 octobre 2026 au soir contient, dans cet ordre :
      | plat             | parts |
      | Chili con carne  | 4     |
      | Chili végétarien | 4     |

  @R3
  Scénario: Modifier le nombre de parts d'un plat servi
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et le plat "Chili végétarien" servi pour 4 parts le même repas
    Quand je règle "Chili con carne" à 3 parts et "Chili végétarien" à 1 part
    Alors le repas du mardi 6 octobre 2026 au soir contient :
      | plat             | parts |
      | Chili con carne  | 3     |
      | Chili végétarien | 1     |

  @R3
  Plan du scénario: Le nombre de parts d'un plat servi va de 1 à 50
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand je règle "Chili con carne" à <parts>
    Alors le repas du mardi 6 octobre 2026 au soir contient "Chili con carne" pour <parts>

    Exemples:
      | parts    |
      | 1 part   |
      | 50 parts |

  @R3
  Plan du scénario: Un nombre de parts hors de 1 à 50 est refusé
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand je règle "Chili con carne" à <parts>
    Alors le réglage est refusé
    Et le repas du mardi 6 octobre 2026 au soir contient toujours "Chili con carne" pour 4 parts

    Exemples:
      | parts    |
      | 0 part   |
      | 51 parts |

  @R3
  Scénario: Le dernier réglage des parts l'emporte
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ouvert le repas du mardi 6 octobre 2026 au soir
    Et Bruno a réglé entre-temps "Chili con carne" à 6 parts dans ce repas
    Quand je règle "Chili con carne" à 3 parts
    Alors le repas du mardi 6 octobre 2026 au soir contient "Chili con carne" pour 3 parts

  @R3 @R4
  Scénario: Les parts d'un plat retiré entre-temps par un autre membre ne sont pas enregistrées
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Et j'ai ouvert le repas du mardi 6 octobre 2026 au soir
    Et Bruno a retiré entre-temps "Chili con carne" de ce repas
    Quand je règle "Chili con carne" à 3 parts
    Alors le réglage est refusé
    Et un message m'indique que le plat a été retiré du repas entre-temps
    Et le repas du mardi 6 octobre 2026 au soir ne contient plus "Chili con carne"

  @R4
  Scénario: Retirer un plat d'un repas
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Quand je retire "Chili con carne" de ce repas
    Alors le repas du mardi 6 octobre 2026 au soir ne contient plus "Chili con carne"
    Mais le plat "Chili con carne" figure toujours dans la bibliothèque

  @R4
  Scénario: Un plat retiré puis ajouté de nouveau repart de 4 parts
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre 2026 au soir
    Quand je retire "Chili con carne" de ce repas
    Et j'ajoute de nouveau "Chili con carne" à ce repas
    Alors le repas du mardi 6 octobre 2026 au soir contient "Chili con carne" pour 4 parts

  @R5 @ui
  Scénario: Naviguer vers la semaine suivante
    Quand j'ouvre le planning
    Et je passe à la semaine suivante
    Alors je vois les jours du lundi 12 au dimanche 18 octobre 2026

  @R5 @ui
  Scénario: Naviguer vers la semaine précédente
    Quand j'ouvre le planning
    Et je passe à la semaine précédente
    Alors je vois les jours du lundi 28 septembre 2026 au dimanche 4 octobre 2026

  @R5
  Scénario: Les repas passés restent modifiables
    Quand j'ajoute le plat "Chili con carne" au repas du samedi 3 octobre 2026 au soir
    Alors le repas du samedi 3 octobre 2026 au soir contient "Chili con carne" pour 4 parts

  @R5
  Plan du scénario: Un repas se place entre 2000 et 2099
    Quand j'ajoute le plat "Chili con carne" au repas du <jour> au soir
    Alors le repas du <jour> au soir contient "Chili con carne" pour 4 parts

    Exemples:
      | jour                      |
      | samedi 1er janvier 2000   |
      | jeudi 31 décembre 2099    |

  @R5
  Plan du scénario: Un repas hors de 2000 à 2099 est refusé
    Quand j'ajoute le plat "Chili con carne" au repas du <jour> au soir
    Alors l'ajout est refusé
    Et le repas du <jour> au soir ne contient aucun plat

    Exemples:
      | jour                      |
      | vendredi 31 décembre 1999 |
      | vendredi 1er janvier 2100 |
