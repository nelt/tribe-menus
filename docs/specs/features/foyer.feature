# language: fr
Fonctionnalité: Foyer
  En tant que membre de la famille,
  je veux partager les plats, le planning et les listes de courses avec les autres membres,
  afin que chacun puisse participer à l'organisation des repas.

  Contexte:
    Étant donné Alice et Bruno sont membres du même foyer

  @F1
  Scénario: Les plats sont partagés entre membres
    Quand Alice crée le plat "Ratatouille"
    Alors Bruno voit le plat "Ratatouille" dans la bibliothèque, après rafraîchissement

  @F1
  Scénario: Le planning est partagé entre membres
    Quand Alice ajoute le plat "Ratatouille" au repas du mardi 6 octobre 2026 à midi
    Alors Bruno voit "Ratatouille" au repas du mardi 6 octobre 2026 à midi, après rafraîchissement

  @F2
  Scénario: Tous les membres ont les mêmes droits
    Étant donné qu'Alice a créé le plat "Ratatouille"
    Quand Bruno modifie le plat "Ratatouille"
    Alors la modification est enregistrée
