# language: fr
Fonctionnalité: Tribu
  En tant que membre de la tribu,
  je veux partager les plats, le planning et les listes de courses avec les autres membres,
  afin que chacun puisse participer à l'organisation des repas.

  Contexte:
    Étant donné Alice et Bruno sont membres de la même tribu

  @T1
  Scénario: Les plats sont partagés entre membres
    Quand Alice crée le plat "Ratatouille"
    Alors Bruno voit le plat "Ratatouille" dans la bibliothèque, après rafraîchissement

  @T1
  Scénario: Le planning est partagé entre membres
    Quand Alice ajoute le plat "Ratatouille" au repas du mardi 6 octobre 2026 à midi
    Alors Bruno voit "Ratatouille" au repas du mardi 6 octobre 2026 à midi, après rafraîchissement

  @T1
  Scénario: Le référentiel d'ingrédients est partagé entre membres
    Quand Alice ajoute le nouvel ingrédient "piment d'Espelette" au plat "Axoa"
    Et Bruno saisit "pim" comme ingrédient d'un plat
    Alors "piment d'Espelette" fait partie des propositions

  @T2
  Scénario: Tous les membres ont les mêmes droits
    Étant donné qu'Alice a créé le plat "Ratatouille"
    Quand Bruno modifie le plat "Ratatouille"
    Alors la modification est enregistrée

  @T3
  Scénario: La taille de la tribu vaut 4 par défaut
    Étant donné une tribu qui vient d'être initialisée
    Alors la taille de la tribu est 4

  @T3
  Scénario: La taille de la tribu fixe les parts des plats servis ajoutés ensuite
    Étant donné le plat "Ratatouille" servi pour 4 parts le mardi 6 octobre 2026 à midi
    Quand Alice règle la taille de la tribu à 5
    Et Bruno ajoute le plat "Ratatouille" au repas du mercredi 7 octobre 2026 à midi
    Alors le repas du mercredi 7 octobre 2026 à midi contient "Ratatouille" pour 5 parts
    Et le repas du mardi 6 octobre 2026 à midi contient toujours "Ratatouille" pour 4 parts

  @T3
  Scénario: La taille de la tribu est proposée pour un nouveau plat
    Étant donné la taille de la tribu est 5
    Et le plat "Chili con carne" défini pour 4 parts
    Quand je crée le plat "Curry de lentilles" sans modifier le nombre de parts proposé
    Alors le plat "Curry de lentilles" est défini pour 5 parts
    Et le plat "Chili con carne" est toujours défini pour 4 parts
