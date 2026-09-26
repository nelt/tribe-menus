# Spécifications fonctionnelles : menus de la semaine (V1)

- **Date** : 2026-09-26
- **Statut** : prêt

## Objectif

Permettre à une tribu (une famille, ou tout groupe qui partage ses repas) de planifier ses repas de la semaine en choisissant des plats, puis d'obtenir automatiquement la liste de courses correspondante pour une période donnée. Usage principal : mobile.

## Concepts

| Concept | Description |
| --- | --- |
| **Tribu** | Groupe de membres qui partagent la bibliothèque de plats, le référentiel d'ingrédients, le planning et les listes de courses. Rien n'est partagé entre tribus (voir `gestion-membres-et-sessions.md` et ENF-02). |
| **Ingrédient** | Élément du référentiel d'ingrédients de la tribu (ex. « tomate », « riz »), avec une unité par défaut. |
| **Bibliothèque de plats** | L'ensemble des plats de la tribu. Un plat y reste indépendamment de son utilisation dans les repas. |
| **Plat** | Un nom et une liste d'ingrédients avec quantités, exprimée pour un **nombre de parts de référence** (4 par défaut). Pas de recette (étapes) en V1. |
| **Repas** | Une case du planning : une date et un moment (**midi** ou **soir**). Elle contient zéro, un ou plusieurs plats servis. |
| **Plat servi** | Un plat de la bibliothèque placé dans un repas, avec son propre **nombre de parts** (toujours 4 par défaut). Exemple : chili 3 parts et chili végétarien 1 part. |
| **Liste de courses** | Agrégation des ingrédients de tous les plats servis d'une période, quantités ajustées au nombre de parts. |

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
  - autres : pièce, cuillère à soupe, cuillère à café, pincée…
- Pour agréger un même ingrédient saisi dans des unités différentes de la même famille, on convertit (ex. 500 g + 1 kg = 1,5 kg).
- Entre familles différentes (ex. « 2 pièces » et « 300 g » d'oignon), pas de conversion : lignes distinctes.
- Dans la liste de courses, les quantités à la pièce sont arrondies à l'entier supérieur, après agrégation (1,5 oignon donne 2).
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

- **C1** : Je demande la liste de courses pour une période ; par défaut, de demain à J+7 inclus.
- **C2** : La liste agrège les ingrédients des plats servis de la période, avec ajustement aux parts, conversions et arrondis.
- **C3** : Je coche les articles au fil des courses.
- **C4** : Une liste déjà affichée reste consultable et cochable sans réseau ; les coches se synchronisent au retour du réseau.

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
- Articles hors plats dans la liste de courses (lessive, pain…).
- Préférences et contraintes alimentaires par membre.
- Budget, historique, statistiques.

## Décisions

- **Q1** : deux moments par jour, midi et soir. Un repas peut contenir plusieurs plats, chacun avec son nombre de parts.
- **Q2** : « les 7 prochains jours » commencent demain (J+1 à J+7).
- **Q3** : conversion d'unités quand c'est nécessaire, au sein d'une même famille.
- **Q4** : les quantités à la pièce sont arrondies à l'entier supérieur.
- **Q5** : les plats vivent dans une bibliothèque ; retirer un plat d'un repas ne le supprime pas de la bibliothèque, mais perd sa configuration dans ce repas.
- **Q5b** : pas de suppression de plat de la bibliothèque en V1.
- **Q6** : pas de droits différenciés en V1 ; pas de temps réel nécessaire.
- **Q7** : la liste de courses doit pouvoir être consultée et cochée hors ligne.
- **Q8** : chaque plat servi part toujours de 4 parts.
- **Q9** : la bibliothèque de plats et le référentiel d'ingrédients sont propres à chaque tribu : partagés entre ses membres, jamais entre tribus.
