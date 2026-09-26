# language: fr
Fonctionnalité: Liste de courses
  En tant que membre de la famille,
  je veux obtenir la liste de courses correspondant aux repas prévus,
  afin de faire les courses sans rien oublier ni acheter en trop.

  Contexte:
    Étant donné que nous sommes le lundi 5 octobre 2026
    Et la bibliothèque contient le plat "Chili con carne" défini pour 4 parts :
      | ingrédient | quantité | unité |
      | bœuf haché | 500      | g     |
      | oignon     | 1        | pièce |

  @C1
  Scénario: Période par défaut
    Quand je demande la liste de courses sans préciser de période
    Alors la période retenue va du mardi 6 au lundi 12 octobre 2026 inclus

  @C1
  Scénario: Seuls les repas de la période sont pris en compte
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre au soir
    Et le plat "Chili con carne" servi pour 4 parts le mardi 13 octobre au soir
    Quand je demande la liste de courses du 6 au 12 octobre 2026
    Alors la liste contient 500 g de "bœuf haché"

  @C2
  Scénario: Les quantités sont ajustées au nombre de parts
    Étant donné le plat "Chili con carne" servi pour 6 parts le mardi 6 octobre au soir
    Quand je demande la liste de courses du 6 au 12 octobre 2026
    Alors la liste contient 750 g de "bœuf haché"

  @C2
  Scénario: Les quantités d'un même ingrédient sont additionnées
    Étant donné le plat "Chili con carne" servi pour 3 parts le mardi 6 octobre au soir
    Et le plat "Chili con carne" servi pour 4 parts le jeudi 8 octobre à midi
    Quand je demande la liste de courses du 6 au 12 octobre 2026
    Alors la liste contient une seule ligne "bœuf haché" de 875 g

  @C2
  Scénario: Les unités d'une même famille sont converties
    Étant donné le plat "Crème de courgettes" défini pour 4 parts avec 50 cl de "crème"
    Et le plat "Gratin dauphinois" défini pour 4 parts avec 1 l de "crème"
    Et chacun de ces plats servi pour 4 parts dans la période
    Quand je demande la liste de courses de la période
    Alors la liste contient une seule ligne "crème" de 1,5 l

  @C2
  Scénario: Les unités de familles différentes restent sur des lignes distinctes
    Étant donné le plat "Soupe à l'oignon" défini pour 4 parts avec 300 g de "oignon"
    Et le plat "Chili con carne" servi pour 4 parts dans la période
    Et le plat "Soupe à l'oignon" servi pour 4 parts dans la période
    Quand je demande la liste de courses de la période
    Alors la liste contient 1 pièce de "oignon"
    Et la liste contient 300 g de "oignon"

  @C2
  Scénario: Les quantités à la pièce sont arrondies à l'entier supérieur
    Étant donné le plat "Chili con carne" servi pour 6 parts dans la période
    Quand je demande la liste de courses de la période
    Alors la liste contient 2 pièces de "oignon"

  @C2
  Scénario: L'arrondi s'applique après l'addition
    Étant donné le plat "Chili con carne" servi pour 2 parts le mardi 6 octobre au soir
    Et le plat "Chili con carne" servi pour 2 parts le jeudi 8 octobre à midi
    Quand je demande la liste de courses du 6 au 12 octobre 2026
    Alors la liste contient 1 pièce de "oignon"

  @C2
  Scénario: Un plat sans ingrédient n'ajoute rien à la liste
    Étant donné le plat "Restes" sans ingrédient servi pour 4 parts dans la période
    Quand je demande la liste de courses de la période
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
    Alors "bœuf haché" apparaît comme coché pour les autres membres du foyer

  @C4
  Scénario: Générer une liste nécessite le réseau
    Étant donné le réseau est coupé
    Quand je demande une nouvelle liste de courses
    Alors un message m'indique que le réseau est nécessaire
