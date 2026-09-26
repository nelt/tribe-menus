# Spécifications fonctionnelles : menus de la semaine (V1)

- **Date** : 2026-09-26
- **Statut** : brouillon, à valider

## Objectif

Permettre à une famille de planifier ses repas de la semaine en choisissant des plats, puis d'obtenir automatiquement la liste de courses correspondante pour une période donnée. Usage principal : mobile.

## Concepts

| Concept | Description |
| --- | --- |
| **Ingrédient** | Élément du référentiel partagé (ex. « tomate », « riz »), avec une unité par défaut. |
| **Plat** | Un nom et une liste d'ingrédients avec quantités, exprimée pour un **nombre de portions de référence** (4 par défaut). Pas de recette (étapes) en V1. |
| **Repas** | Un créneau : une date et un moment (**midi** ou **soir**). Il contient un ou plusieurs plats servis. |
| **Plat servi** | Un plat au sein d'un repas, avec son propre **nombre de convives** (4 par défaut). Exemple : chili pour 3 et chili végétarien pour 1. |
| **Liste de courses** | Agrégation des ingrédients de tous les plats servis d'une période, quantités ajustées au nombre de convives. |

### Règle de calcul

Pour chaque ingrédient d'un plat servi :

```
quantité à acheter = quantité du plat × (convives du plat servi ÷ portions de référence du plat)
```

Exemple : plat « Chili » défini pour 4 avec 500 g de bœuf, servi à 6 convives, soit 750 g de bœuf.

### Unités et conversions

- Les unités forment une liste fermée, regroupée par famille :
  - masse : g, kg ;
  - volume : ml, cl, l ;
  - autres : pièce, cuillère à soupe, cuillère à café, pincée…
- Pour agréger un même ingrédient saisi dans des unités différentes de la même famille, on convertit (ex. 500 g + 1 kg = 1,5 kg).
- Entre familles différentes (ex. « 2 pièces » et « 300 g » d'oignon), pas de conversion : lignes distinctes.

## User stories

### Plats

- **P1** : En tant que membre, je crée un plat avec un nom et une liste d'ingrédients (ingrédient, quantité, unité).
  - Les portions de référence valent 4 par défaut et sont modifiables.
  - Un plat doit avoir un nom ; la liste d'ingrédients peut être vide (ex. « restes »).
- **P2** : Je modifie ou supprime un plat.
  - Supprimer un plat déjà planifié demande une confirmation. Le comportement sur les repas concernés reste à définir (voir Q5).
- **P3** : Je retrouve un plat par recherche sur son nom.
- **P4** : En saisissant un ingrédient, l'application me propose ceux qui existent déjà (autocomplétion), pour éviter les doublons (« tomate » / « tomates ») qui fausseraient l'agrégation. Je peux créer un nouvel ingrédient s'il n'existe pas.

### Planning des repas

- **R1** : Je vois les repas des 7 prochains jours (de demain à J+7) sous forme de grille, avec deux créneaux par jour : midi et soir.
- **R2** : J'ajoute un ou plusieurs plats à un repas. Chaque plat servi a son nombre de convives : 4 par défaut, modifiable.
- **R3** : Je modifie le nombre de convives d'un plat servi, ou je le retire du repas.
- **R4** : Je navigue vers les semaines précédentes et suivantes.

### Liste de courses

- **C1** : Je demande la liste de courses en indiquant un jour de début et un jour de fin. Par défaut, la période va de demain à J+7 inclus.
- **C2** : La liste agrège les ingrédients de tous les plats servis de la période :
  - les quantités sont ajustées selon la règle de calcul ;
  - un même ingrédient donne une seule ligne dès que ses unités sont convertibles entre elles ;
  - sinon, une ligne par famille d'unités.
- **C3** : Je coche les articles au fil des courses.

## Hors périmètre V1

- Recettes (étapes, temps de préparation, photos).
- Suggestions automatiques de menus.
- Gestion du stock ou du placard.
- Articles hors plats dans la liste de courses (lessive, pain…).
- Préférences et contraintes alimentaires par membre.
- Budget, historique, statistiques.

## Questions ouvertes

- **Q4. Arrondis** : les quantités « à la pièce » sont-elles arrondies au supérieur (1,5 oignon donne 2) ?
- **Q5. Suppression d'un plat planifié** : faut-il l'interdire, vider les repas concernés, ou archiver le plat ?
- **Q6. Foyer et comptes** : un foyer partagé où tous les membres ont les mêmes droits, ou des rôles différents ? La liste cochée est-elle partagée en temps réel entre membres ?
- **Q7. Hors-ligne** : la liste de courses doit-elle rester consultable et cochable sans réseau en magasin ?

## Décisions

- **Q1** : deux moments par jour, midi et soir. Un repas peut contenir plusieurs plats, chacun avec son nombre de convives.
- **Q2** : « les 7 prochains jours » commencent demain (J+1 à J+7).
- **Q3** : conversion d'unités quand c'est nécessaire, au sein d'une même famille.
