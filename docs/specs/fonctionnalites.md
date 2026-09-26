# Spécifications fonctionnelles : menus de la semaine (V1)

- **Date** : 2026-09-26
- **Statut** : prêt

## Objectif

Permettre à une famille de planifier ses repas de la semaine en choisissant des plats, puis d'obtenir automatiquement la liste de courses correspondante pour une période donnée. Usage principal : mobile.

## Concepts

| Concept | Description |
| --- | --- |
| **Ingrédient** | Élément du référentiel partagé (ex. « tomate », « riz »), avec une unité par défaut. |
| **Bibliothèque de plats** | L'ensemble des plats de la famille. Un plat y reste indépendamment de son utilisation dans les repas. |
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

## User stories

### Bibliothèque de plats

- **P1** : En tant que membre, je crée un plat avec un nom et une liste d'ingrédients (ingrédient, quantité, unité).
  - Les parts de référence valent 4 par défaut et sont modifiables.
  - Un plat doit avoir un nom ; la liste d'ingrédients peut être vide (ex. « restes »).
- **P2** : Je modifie un plat de la bibliothèque. Les modifications s'appliquent à tous les repas où il est servi. Un plat ne peut pas être supprimé de la bibliothèque en V1.
- **P3** : Je retrouve un plat par recherche sur son nom.
- **P4** : En saisissant un ingrédient, l'application me propose ceux qui existent déjà (autocomplétion), pour éviter les doublons (« tomate » / « tomates ») qui fausseraient l'agrégation. Je peux créer un nouvel ingrédient s'il n'existe pas.

### Planning des repas

- **R1** : Je vois les repas des 7 prochains jours (de demain à J+7) sous forme de grille, avec deux cases par jour : midi et soir.
- **R2** : J'ajoute un ou plusieurs plats de la bibliothèque à un repas. Chaque plat servi part de 4 parts, modifiable.
- **R3** : Je modifie le nombre de parts d'un plat servi.
- **R4** : Je retire un plat d'un repas. Le plat reste dans la bibliothèque ; seule sa configuration dans ce repas (nombre de parts) est perdue.
- **R5** : Je navigue vers les semaines précédentes et suivantes.

### Liste de courses

- **C1** : Je demande la liste de courses en indiquant un jour de début et un jour de fin. Par défaut, la période va de demain à J+7 inclus.
- **C2** : La liste agrège les ingrédients de tous les plats servis de la période :
  - les quantités sont ajustées selon la règle de calcul ;
  - un même ingrédient donne une seule ligne dès que ses unités sont convertibles entre elles ;
  - sinon, une ligne par famille d'unités ;
  - les quantités à la pièce sont arrondies à l'entier supérieur.
- **C3** : Je coche les articles au fil des courses.
- **C4** : Une liste de courses déjà affichée reste consultable et cochable sans réseau. Les coches faites hors ligne sont synchronisées au retour du réseau. Générer une nouvelle liste nécessite le réseau.

### Foyer

- **F1** : Les membres de la famille partagent la même bibliothèque, le même planning et les mêmes listes de courses.
- **F2** : Tous les membres ont les mêmes droits.

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
