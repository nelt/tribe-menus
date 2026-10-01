# language: fr
Fonctionnalité: Bibliothèque de plats
  En tant que membre de la tribu,
  je veux constituer une bibliothèque de plats avec leurs ingrédients,
  afin de pouvoir les choisir rapidement pour les repas.

  @P1
  Scénario: Créer un plat avec ses ingrédients
    Quand je crée le plat "Chili con carne" avec les ingrédients :
      | ingrédient      | quantité | unité |
      | bœuf haché      | 500      | g     |
      | haricots rouges | 400      | g     |
      | oignon          | 2        | pièce |
    Alors le plat "Chili con carne" figure dans la bibliothèque
    Et il est défini pour 4 parts

  @P1
  Scénario: Définir un plat pour un autre nombre de parts
    Quand je crée le plat "Gratin dauphinois" pour 6 parts
    Alors le plat "Gratin dauphinois" est défini pour 6 parts

  @P1
  Scénario: Un nouveau plat reçoit la teinte la moins utilisée
    Étant donné la bibliothèque contient un plat de chaque teinte de la palette, sauf la teinte "#6E6680"
    Quand je crée le plat "Curry de lentilles"
    Alors le plat "Curry de lentilles" a la teinte "#6E6680"

  @P1
  Scénario: La couleur d'un plat ne change pas quand on le modifie
    Étant donné le plat "Chili con carne" a la teinte "#C2502E"
    Quand je modifie le plat "Chili con carne"
    Alors le plat "Chili con carne" a toujours la teinte "#C2502E"

  @P1
  Scénario: Le nom d'un plat est obligatoire
    Quand je crée un plat sans nom
    Alors le plat n'est pas enregistré
    Et un message m'indique que le nom est obligatoire

  @P1
  Scénario: Un plat peut n'avoir aucun ingrédient
    Quand je crée le plat "Restes" sans ingrédient
    Alors le plat "Restes" figure dans la bibliothèque

  @P2
  Scénario: La modification d'un plat s'applique aux repas où il est servi
    Étant donné le plat "Chili con carne" défini pour 4 parts avec 500 g de "bœuf haché"
    Et le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand je modifie la quantité de "bœuf haché" du plat "Chili con carne" à 600 g
    Et j'ajoute les courses des repas du 6 au 6 octobre 2026 à la liste "Courses"
    Alors la liste "Courses" contient 600 g de "bœuf haché"

  @P2
  Scénario: Un plat ne peut pas être supprimé de la bibliothèque
    Étant donné le plat "Chili con carne" dans la bibliothèque
    Alors aucune action ne permet de supprimer le plat "Chili con carne"

  @P3
  Plan du scénario: Rechercher un plat par les mots de son nom ou par ses ingrédients
    Étant donné la bibliothèque contient les plats :
      | plat               | ingrédients                   |
      | Chili con carne    | bœuf haché, haricots rouges   |
      | Gratin dauphinois  | pomme de terre, crème         |
      | Gratin de courgettes | courgette, crème, gruyère   |
      | Ratatouille        | courgette, aubergine, tomate  |
    Quand je recherche "<recherche>"
    Alors les résultats sont "<résultats>"

    Exemples:
      | recherche        | résultats                                   |
      | chili            | Chili con carne                             |
      | courgette        | Gratin de courgettes, Ratatouille           |
      | GRATIN           | Gratin dauphinois, Gratin de courgettes     |
      | creme            | Gratin dauphinois, Gratin de courgettes     |
      | gratin courgette | Gratin de courgettes                        |
      | courg            | Gratin de courgettes, Ratatouille           |
      | poulet           |                                             |

  @P4
  Scénario: L'autocomplétion propose les ingrédients existants
    Étant donné l'ingrédient "tomate" existe dans le référentiel
    Quand je saisis "tom" comme ingrédient d'un plat
    Alors l'ingrédient "tomate" m'est proposé

  @P4
  Scénario: L'autocomplétion ignore la casse et les accents
    Étant donné l'ingrédient "crème fraîche" existe dans le référentiel
    Quand je saisis "CREME" comme ingrédient d'un plat
    Alors l'ingrédient "crème fraîche" m'est proposé

  @P4
  Scénario: Un nom qui ne diffère que par la casse, les accents ou les espaces désigne le même ingrédient
    Étant donné l'ingrédient "crème fraîche" existe dans le référentiel
    Quand j'ajoute l'ingrédient " Creme  Fraiche " à un plat
    Alors le référentiel contient un seul ingrédient "crème fraîche"
    Et le plat utilise l'ingrédient "crème fraîche"

  @P4
  Scénario: L'unité par défaut est proposée à la saisie
    Étant donné l'ingrédient "riz" a été saisi pour la première fois en "g"
    Quand je saisis "riz" comme ingrédient d'un plat
    Alors l'unité "g" est proposée
    Et je peux choisir une autre unité pour ce plat

  @P4
  Scénario: Créer un nouvel ingrédient pendant la saisie d'un plat
    Étant donné l'ingrédient "gingembre" n'existe pas dans le référentiel
    Quand j'ajoute l'ingrédient "gingembre" à un plat
    Alors l'ingrédient "gingembre" est ajouté au référentiel
    Et il m'est proposé lors des saisies suivantes
