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

  @P1
  Scénario: Deux plats ne peuvent pas porter le même nom
    Étant donné le plat "Soupe à l'oignon" dans la bibliothèque
    Quand je crée le plat " soupe a l’oignon "
    Alors le plat n'est pas enregistré
    Et un message m'indique qu'un plat porte déjà ce nom

  @P1 @P4
  Scénario: Les espaces en trop sont retirées des noms
    Quand je crée le plat "  Gratin   de  courgettes " avec les ingrédients :
      | ingrédient       | quantité | unité |
      | crème    fraîche | 20       | cl    |
    Alors le plat "Gratin de courgettes" figure dans la bibliothèque
    Et le référentiel contient un seul ingrédient "crème fraîche"

  @P1 @P4
  Plan du scénario: Un nom sans caractère visible ou avec un caractère de contrôle est refusé
    Quand je crée un plat dont le nom <de quoi> est <nom>
    Alors le plat n'est pas enregistré
    Et le nom <de quoi> est refusé comme <refus>

    Exemples:
      | de quoi         | nom                                           | refus    |
      | du plat         | l'espace sans chasse U+200B                   | vide     |
      | du plat         | "Soupe" suivi du caractère de contrôle U+0007 | invalide |
      | d'un ingrédient | l'espace sans chasse U+200B                   | vide     |
      | d'un ingrédient | "sel" suivi du caractère de contrôle U+0007   | invalide |

  @P1
  Scénario: Un ingrédient peut n'avoir aucune quantité
    Quand je crée le plat "Pâtes au beurre" avec les ingrédients :
      | ingrédient | quantité | unité |
      | pâtes      | 500      | g     |
      | beurre     | 30       | g     |
      | sel        |          |       |
    Alors le plat "Pâtes au beurre" figure dans la bibliothèque
    Et l'ingrédient "sel" du plat "Pâtes au beurre" n'a pas de quantité
    Et l'ingrédient "sel" n'a pas d'unité par défaut

  @P1 @P4
  Scénario: Un même ingrédient ne figure qu'une fois dans un plat
    Quand je crée le plat "Ratatouille" avec les ingrédients :
      | ingrédient | quantité | unité |
      | courgette  | 3        | pièce |
      | Courgette  | 200      | g     |
    Alors le plat n'est pas enregistré
    Et un message m'indique que l'ingrédient "courgette" figure déjà dans le plat

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

  @P2
  Scénario: Une modification faite entre-temps par un autre membre n'est pas écrasée
    Étant donné le plat "Chili con carne" défini pour 4 parts avec 500 g de "bœuf haché"
    Et j'ai ouvert le plat "Chili con carne"
    Et Bruno a modifié entre-temps la quantité de "bœuf haché" du plat "Chili con carne" à 600 g
    Quand j'ajoute 1 cuillère à café de "cumin" au plat "Chili con carne"
    Alors mon enregistrement est refusé
    Et un message m'indique que le plat a été modifié entre-temps
    Et le plat "Chili con carne" m'est présenté à jour, avec 600 g de "bœuf haché" et sans "cumin"

  @P3 @ui
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

  @P4 @ui
  Scénario: L'autocomplétion propose les ingrédients existants
    Étant donné l'ingrédient "tomate" existe dans le référentiel
    Quand je saisis "tom" comme ingrédient d'un plat
    Alors l'ingrédient "tomate" m'est proposé

  @P4 @ui
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

  @P4 @ui
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

  @P4 @ui
  Scénario: Un ingrédient créé pendant la saisie est proposé à la saisie suivante
    Étant donné l'ingrédient "gingembre" n'existe pas dans le référentiel
    Et j'ai ajouté l'ingrédient "gingembre" au plat "Curry de lentilles"
    Quand je saisis "ging" comme ingrédient d'un autre plat, sans avoir rechargé l'application
    Alors l'ingrédient "gingembre" m'est proposé
