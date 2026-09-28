# Spécifications fonctionnelles : menus de la semaine (V1)

- **Date** : 2026-09-26 (listes de courses révisées le 2026-09-27)
- **Statut** : prêt

## Objectif

Permettre à une tribu (une famille, ou tout groupe qui partage ses repas) de planifier ses repas de la semaine en choisissant des plats, puis d'obtenir automatiquement la liste de courses correspondante pour une période donnée. Usage principal : mobile.

## Concepts

| Concept | Description |
| --- | --- |
| **Tribu** | Groupe de membres qui partagent la bibliothèque de plats, le référentiel d'ingrédients, le planning et les listes de courses. Rien n'est partagé entre tribus (voir `gestion-membres-et-sessions.md` et ENF-02). |
| **Ingrédient** | Élément du référentiel d'ingrédients de la tribu (ex. « tomate », « riz »), avec une unité par défaut. |
| **Bibliothèque de plats** | L'ensemble des plats de la tribu. Un plat y reste indépendamment de son utilisation dans les repas. |
| **Plat** | Un nom et une liste d'ingrédients avec quantités, exprimée pour un **nombre de parts de référence** (4 par défaut). Il porte une **couleur**, attribuée automatiquement à sa création (Q19). Pas de recette (étapes) en V1. |
| **Repas** | Une case du planning : une date et un moment (**midi** ou **soir**). Elle contient zéro, un ou plusieurs plats servis. |
| **Plat servi** | Un plat de la bibliothèque placé dans un repas, avec son propre **nombre de parts** (toujours 4 par défaut). Exemple : chili 3 parts et chili végétarien 1 part. |
| **Liste de courses** | Liste créée pour une **période** (date de début et date de fin incluses). Elle contient les ingrédients agrégés des plats servis de la période, quantités ajustées au nombre de parts, plus d'éventuels articles ajoutés à la main. C'est un **instantané** : calculée à sa création, elle ne change ensuite que par un recalcul explicite. Plusieurs listes peuvent exister en même temps. |
| **Article** | Une ligne d'une liste de courses : un ingrédient, une quantité et une unité. Un article **calculé** provient d'un ou plusieurs repas de la période ; un article **ajouté** a été saisi à la main, hors planning. |
| **État d'une liste** | **En cours** (visible dans l'onglet Courses, modifiable) ou **faite** (figée, consultable dans l'historique). Une liste abandonnée est supprimée. |

### Recalcul d'une liste

Un recalcul (C5 ou C6) reconstruit les articles calculés à partir des repas de la période, puis :

- les **articles ajoutés** à la main sont conservés tels quels ;
- un article calculé encore présent **reste coché** s'il l'était et que sa quantité n'a pas augmenté ; si sa quantité augmente, il est décoché (il faut en racheter). « Encore présent » signifie même ingrédient et même famille d'unités ;
- les articles calculés qui ne sont plus nécessaires disparaissent, même cochés.

Une liste est **périmée** (C6) quand le calcul pour sa période, fait maintenant, ne donnerait pas les mêmes articles calculés (ingrédients ou quantités) : plat ajouté ou retiré d'un repas, parts modifiées, ingrédients d'un plat servi modifiés (P2). Une liste faite n'est jamais signalée périmée.

### Règle de calcul

Pour chaque ingrédient d'un plat servi :

```
quantité à acheter = quantité du plat × (parts du plat servi ÷ parts de référence du plat)
```

Exemple : plat « Chili » défini pour 4 parts avec 500 g de bœuf, servi pour 6 parts, soit 750 g de bœuf.

### Unités, conversions et arrondis

- Les unités forment une liste fermée, regroupée par famille :
  - masse : g, kg ;
  - volume : ml, cl, l ;
  - pièce ;
  - cuillère à soupe ;
  - cuillère à café ;
  - pincée.

  Pièce, cuillère à soupe, cuillère à café et pincée forment chacune une famille à elles seules : aucune conversion entre elles (1 cuillère à soupe et 2 cuillères à café d'huile donnent deux lignes).
- Pour agréger un même ingrédient saisi dans des unités différentes de la même famille, on convertit (ex. 500 g + 1 kg = 1,5 kg).
- Entre familles différentes (ex. « 2 pièces » et « 300 g » d'oignon), pas de conversion : lignes distinctes.
- Dans la liste de courses, les quantités en pièces, cuillères et pincées sont arrondies à l'entier supérieur, après agrégation (1,5 oignon donne 2 ; 1,5 cuillère à soupe donne 2). Les masses et les volumes ne sont pas arrondis à l'entier.
- Une quantité agrégée est affichée dans la plus grande unité de sa famille qui donne une valeur d'au moins 1 (1 500 g s'affiche 1,5 kg ; 750 g reste 750 g).

## User stories

Les critères d'acceptation de chaque story sont écrits en Gherkin dans `features/`. Chaque scénario porte l'identifiant de sa story en tag (ex. `@C2`).

### Bibliothèque de plats (`features/bibliotheque-plats.feature`)

- **P1** : Je crée un plat avec un nom et une liste d'ingrédients (ingrédient, quantité, unité), pour 4 parts par défaut.
- **P2** : Je modifie un plat ; la modification s'applique à tous les repas où il est servi. Pas de suppression en V1.
- **P3** : Je retrouve un plat en cherchant sur les mots de son nom et sur ses ingrédients.
- **P4** : La saisie d'un ingrédient propose ceux qui existent déjà (autocomplétion) ; je peux en créer un nouveau.

### Planning des repas (`features/planning-repas.feature`)

- **R1** : Je vois les repas des 7 prochains jours (de demain à J+7), avec deux cases par jour : midi et soir.
- **R2** : J'ajoute un ou plusieurs plats de la bibliothèque à un repas ; chaque plat servi part de 4 parts.
- **R3** : Je modifie le nombre de parts d'un plat servi.
- **R4** : Je retire un plat d'un repas ; il reste dans la bibliothèque, mais sa configuration dans ce repas est perdue.
- **R5** : Je navigue vers les semaines précédentes et suivantes.

### Liste de courses (`features/liste-courses.feature`)

- **C1** : Je crée une liste de courses en choisissant la date de début et la date de fin des repas pris en compte ; par défaut, de demain à J+7 inclus.
- **C2** : La liste agrège les ingrédients des plats servis de la période, avec ajustement aux parts, conversions et arrondis.
- **C3** : Je coche les articles au fil des courses.
- **C4** : Une liste déjà affichée reste consultable et cochable sans réseau ; les coches et les articles ajoutés se synchronisent au retour du réseau.
- **C5** : Je modifie les dates d'une liste en cours ; la liste est aussitôt recalculée.
- **C6** : Si les repas de la période ont changé depuis le dernier calcul, un bandeau en haut de la liste le signale, avec une action « Recalculer ».
- **C7** : Plusieurs listes peuvent être en cours en même temps, y compris sur des périodes qui se chevauchent.
- **C8** : J'ajoute à une liste des articles hors planning (ex. pain, lessive) ; je peux les retirer.
- **C9** : Un article calculé indique de combien de repas il provient ; en touchant la ligne, elle se déplie et montre les repas et plats concernés, avec la quantité apportée par chacun.
- **C10** : Je déclare les courses faites : la liste quitte l'onglet Courses et n'est plus consultable que dans l'historique, en lecture seule.
- **C11** : J'abandonne une liste en cours : après confirmation, elle est supprimée.

### Tribu (`features/tribu.feature`)

- **T1** : Les membres de la tribu partagent la bibliothèque de plats, le référentiel d'ingrédients, le planning et les listes de courses ; les autres tribus n'y ont pas accès.
- **T2** : Tous les membres ont les mêmes droits.

### Membres, sessions et administration

Stories EF-01 à EF-09, détaillées dans `gestion-membres-et-sessions.md` ; critères dans `features/membres-et-sessions.feature` et `features/administration.feature`. Les exigences ENF-01 (connexion et session) et ENF-02 (compartimentage entre tribus) ont leurs critères dans `features/authentification.feature` et `features/compartimentage-tribus.feature`.

## Hors périmètre V1

- Recettes (étapes, temps de préparation, photos).
- Suppression d'un plat de la bibliothèque.
- Rôles et droits différenciés entre membres.
- Synchronisation en temps réel entre membres (un rafraîchissement suffit).
- Suggestions automatiques de menus.
- Gestion du stock ou du placard.
- Préférences et contraintes alimentaires par membre.
- Budget, statistiques, historique des repas (seul existe l'historique des listes de courses faites, C10).
- Réouverture d'une liste faite ; suppression d'une liste de l'historique.
- Retrait d'un article calculé d'une liste (on peut le laisser non coché).

## Décisions

- **Q1** : deux moments par jour, midi et soir. Un repas peut contenir plusieurs plats, chacun avec son nombre de parts.
- **Q2** : « les 7 prochains jours » commencent demain (J+1 à J+7).
- **Q3** : conversion d'unités quand c'est nécessaire, au sein d'une même famille. Seules la masse et le volume comptent plusieurs unités ; pièce, cuillère à soupe, cuillère à café et pincée sont chacune leur propre famille (PT-01, 2026-09-28).
- **Q4** : les quantités en pièces, cuillères et pincées sont arrondies à l'entier supérieur (PT-01, 2026-09-28).
- **Q5** : les plats vivent dans une bibliothèque ; retirer un plat d'un repas ne le supprime pas de la bibliothèque, mais perd sa configuration dans ce repas.
- **Q5b** : pas de suppression de plat de la bibliothèque en V1.
- **Q6** : pas de droits différenciés en V1 ; pas de temps réel nécessaire.
- **Q7** : la liste de courses doit pouvoir être consultée et cochée hors ligne.
- **Q8** : chaque plat servi part toujours de 4 parts.
- **Q9** : la bibliothèque de plats et le référentiel d'ingrédients sont propres à chaque tribu : partagés entre ses membres, jamais entre tribus.
- **Q10** : une liste de courses est un instantané, recalculé seulement sur action explicite (changement de dates ou « Recalculer »), jamais automatiquement : la liste ne bouge pas sous les yeux de celui qui fait les courses.
- **Q11** : au recalcul, les articles ajoutés à la main sont conservés, et les coches aussi, sauf quand la quantité d'un article augmente.
- **Q12** : plusieurs listes en cours sont permises, sans contrôle de chevauchement : un même repas peut compter dans deux listes.
- **Q13** : un article ajouté à la main est un ingrédient du référentiel (autocomplétion et création comme en P4), avec quantité et unité facultatives. Il reste une ligne distincte, jamais fusionnée avec un article calculé du même ingrédient.
- **Q14** : « courses faites » est possible même s'il reste des articles non cochés. La liste faite est figée : ni coche, ni ajout, ni recalcul.
- **Q15** : abandonner une liste la supprime définitivement, après confirmation ; seule une liste en cours peut être abandonnée.
- **Q16** : sans réseau, on peut cocher et ajouter des articles (synchronisés au retour du réseau) ; créer une liste, changer ses dates, recalculer, la déclarer faite ou l'abandonner nécessitent le réseau.
- **Q18** : au retour du réseau, les coches et les articles ajoutés hors ligne sont rejoués (PT-03, 2026-09-28) :
  - si la liste a été entre-temps déclarée faite ou abandonnée, ils sont ignorés et un message le signale ;
  - si un recalcul a entre-temps supprimé un article coché hors ligne, la coche est ignorée et un message le signale ;
  - un ingrédient créé hors ligne par un article ajouté est rapproché d'un ingrédient existant de même nom normalisé plutôt que dupliqué.
- **Q19** : à sa création, un plat reçoit la teinte de la palette (`design/README.md`) la moins utilisée dans la bibliothèque, la première dans l'ordre de la palette en cas d'égalité ; elle est enregistrée avec le plat et ne change plus. Pas de choix de couleur par le membre en V1 (PT-05, 2026-09-28).
- **Q17** : une liste est désignée par sa période (ex. « 6 → 12 oct. ») ; les listes en cours sont triées par date de début, l'historique par date de courses faites, la plus récente en premier.
