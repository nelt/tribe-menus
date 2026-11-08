# language: fr
Fonctionnalité: Formats d'écran
  En tant que membre de la tribu,
  je veux utiliser l'application sur un téléphone, une tablette ou un ordinateur,
  afin de préparer les menus et faire les courses sur l'écran que j'ai sous la main.

  Un scénario sans tag de format vaut pour tous les formats ; un tag de format
  le limite aux projets Playwright de ce format (ENF-03, conventions-gherkin.md).

  Contexte:
    Étant donné que nous sommes le lundi 5 octobre 2026
    Et la bibliothèque contient les plats "Chili con carne" et "Chili végétarien"
    Et la liste principale de la tribu est la liste "Courses"

  @ENF-03 @ui @compact
  Scénario: Sur téléphone, les onglets sont en bas de l'écran
    Quand j'ouvre le planning
    Alors les onglets sont en bas de l'écran

  @ENF-03 @ui @moyen @etendu
  Scénario: Sur tablette et ordinateur, les onglets sont sur le côté
    Quand j'ouvre le planning
    Alors les onglets sont sur le côté gauche de l'écran
    Et la pastille de la tribu, à côté des onglets, ouvre la tribu

  @ENF-03 @ui @etendu
  Scénario: Une fenêtre large mais basse garde la mise en page du téléphone
    Étant donné une fenêtre de 1366 px de large sur 520 px de haut
    Quand j'ouvre le planning
    Alors les onglets sont en bas de l'écran

  @ENF-03 @ui @etendu
  Scénario: Le planning montre la semaine en semainier
    Quand j'ouvre le planning
    Alors les jours du lundi 5 au dimanche 11 octobre 2026 sont côte à côte
    Et chaque jour montre le repas du midi au-dessus de celui du soir

  @ENF-03 @ui @etendu
  Scénario: Le semainier défile vers les jours qui ne tiennent pas
    Étant donné une fenêtre de 1024 px de large sur 768 px de haut
    Quand j'ouvre le planning
    Alors le dimanche 11 octobre 2026 n'est pas entièrement visible
    Quand je fais défiler le semainier d'un jour vers les jours suivants
    Alors le mardi 6 octobre 2026 est le premier jour visible, entièrement
    Quand je fais défiler le semainier jusqu'au dernier jour
    Alors le dimanche 11 octobre 2026 est entièrement visible

  @ENF-03 @ui @etendu
  Scénario: Sur un grand écran, les 7 jours tiennent sans défilement
    Étant donné une fenêtre de 1440 px de large sur 900 px de haut
    Quand j'ouvre le planning
    Alors les 7 jours sont visibles sans défilement

  @ENF-03 @ui @etendu
  Scénario: Le détail d'un repas s'ouvre en panneau par-dessus le semainier
    Étant donné le plat "Chili con carne" servi pour 4 parts le mardi 6 octobre 2026 au soir
    Quand j'ouvre le planning
    Et j'ouvre le repas du mardi 6 octobre 2026 au soir
    Alors le repas du mardi 6 octobre 2026 au soir s'affiche en panneau par-dessus le semainier
    Quand je ferme le panneau du repas
    Alors je vois les jours du lundi 5 au dimanche 11 octobre 2026

  @ENF-03 @ui @etendu
  Scénario: Échap ferme le panneau d'un repas
    Quand j'ouvre le planning
    Et j'ouvre le repas du mardi 6 octobre 2026 au soir
    Et j'appuie sur Échap
    Alors je vois les jours du lundi 5 au dimanche 11 octobre 2026
    Et aucun repas n'est ouvert

  @ENF-03 @ui @etendu
  Scénario: La bibliothèque reste affichée à côté du plat ouvert
    Quand j'ouvre la bibliothèque de plats
    Alors l'invitation à choisir un plat s'affiche à côté de la bibliothèque
    Quand j'ouvre le plat "Chili con carne"
    Alors le plat "Chili con carne" s'affiche à côté de la bibliothèque
    Et le plat "Chili con carne" est marqué dans la bibliothèque

  @ENF-03 @ui @etendu
  Scénario: La liste ouverte s'affiche à côté du choix des listes
    Étant donné les listes "Courses" et "Marché"
    Quand j'ouvre l'onglet Courses
    Alors la liste "Courses" s'affiche à côté du choix des listes
    Quand j'ouvre la liste "Marché" depuis le choix des listes
    Alors la liste "Marché" s'affiche à côté du choix des listes

  @ENF-03 @ui @compact
  Scénario: Sur téléphone, une confirmation s'affiche en bas de l'écran
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Et "pain" est coché dans la liste "Courses"
    Quand j'ouvre l'onglet Courses
    Et je demande à terminer les courses
    Alors la confirmation s'affiche en bas de l'écran

  @ENF-03 @ui @moyen @etendu
  Scénario: Sur tablette et ordinateur, une confirmation s'affiche au milieu de l'écran
    Étant donné la liste "Courses" contenant l'article ajouté "pain"
    Et "pain" est coché dans la liste "Courses"
    Quand j'ouvre l'onglet Courses
    Et je demande à terminer les courses
    Alors la confirmation s'affiche au milieu de l'écran

  @ENF-03 @ui @moyen @etendu
  Scénario: Quitter une modification non enregistrée demande confirmation
    Quand j'ouvre le plat "Chili con carne"
    Et je renomme le plat en "Chili du dimanche" sans l'enregistrer
    Et j'ouvre l'onglet Planning
    Alors on me demande si j'abandonne la modification
    Quand je continue la modification
    Alors le plat "Chili con carne" est ouvert avec le nom "Chili du dimanche"

  @ENF-03 @ui @moyen @etendu
  Scénario: Abandonner une modification la perd
    Quand j'ouvre le plat "Chili con carne"
    Et je renomme le plat en "Chili du dimanche" sans l'enregistrer
    Et j'ouvre l'onglet Planning
    Et j'abandonne la modification
    Alors je vois les jours du lundi 5 au dimanche 11 octobre 2026
    Et la bibliothèque contient toujours le plat "Chili con carne"

  @ENF-03 @ui @etendu
  Scénario: Ouvrir un autre plat pendant une modification demande confirmation
    Quand j'ouvre le plat "Chili con carne"
    Et je renomme le plat en "Chili du dimanche" sans l'enregistrer
    Et j'ouvre le plat "Chili végétarien" dans la bibliothèque
    Alors on me demande si j'abandonne la modification

  @ENF-03 @ui @moyen @etendu
  Scénario: Une saisie revenue à ce qui est enregistré ne demande pas de confirmation
    Quand j'ouvre le plat "Chili con carne"
    Et je renomme le plat en "Chili du dimanche" sans l'enregistrer
    Et je renomme le plat en "Chili con carne" sans l'enregistrer
    Et j'ouvre l'onglet Planning
    Alors je vois les jours du lundi 5 au dimanche 11 octobre 2026

  @ENF-03 @ui @compact
  Scénario: Sur téléphone, quitter une modification ne demande pas de confirmation
    Quand j'ouvre le plat "Chili con carne"
    Et je renomme le plat en "Chili du dimanche" sans l'enregistrer
    Et je reviens à la bibliothèque de plats
    Alors la bibliothèque contient toujours le plat "Chili con carne"

  @ENF-03 @ui @moyen
  Scénario: Tourner la tablette ne perd pas la saisie d'un plat
    Quand j'ouvre le plat "Chili con carne"
    Et je renomme le plat en "Chili du dimanche" sans l'enregistrer
    Et je tourne la tablette
    Alors le plat "Chili con carne" est ouvert avec le nom "Chili du dimanche"
    Et le plat "Chili con carne" s'affiche à côté de la bibliothèque

  @ENF-03 @ui @moyen
  Scénario: Tourner la tablette ne perd pas la saisie d'un article
    Quand j'ouvre l'onglet Courses
    Et je saisis l'article "pain" à la main sans l'ajouter
    Et je tourne la tablette
    Alors la saisie de l'article "pain" est toujours en cours
    Et la liste "Courses" s'affiche à côté du choix des listes

  @ENF-03 @manuel
  Scénario: Le clavier virtuel ne change pas la mise en page d'une tablette
    Étant donné une tablette tenue en paysage
    Quand j'ouvre le plat "Chili con carne"
    Et je touche le champ du nom du plat
    Alors le clavier virtuel s'ouvre
    Et la bibliothèque reste affichée à côté du plat
    Et le bouton « Enregistrer le plat » reste visible au-dessus du clavier
