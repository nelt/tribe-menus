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
| **Repas** | Un créneau (date et moment) auquel on associe un plat et un **nombre de convives** (4 par défaut). |
| **Liste de courses** | Agrégation des ingrédients de tous les repas d'une période, quantités ajustées au nombre de convives. |

### Règle de calcul

Pour chaque ingrédient d'un plat servi à un repas :

```
quantité à acheter = quantité du plat × (convives du repas ÷ portions de référence du plat)
```

Exemple : plat « Chili » défini pour 4 avec 500 g de bœuf, repas pour 6 convives, soit 750 g de bœuf.

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

- **R1** : Je vois les repas des 7 prochains jours sous forme de grille.
- **R2** : Je choisis un plat pour un créneau. Le nombre de convives vaut 4 par défaut et reste modifiable repas par repas.
- **R3** : Je modifie ou retire le plat d'un repas.
- **R4** : Je navigue vers les semaines précédentes et suivantes.

### Liste de courses

- **C1** : Je demande la liste de courses en indiquant un jour de début et un jour de fin. Par défaut, la période couvre les 7 prochains jours.
- **C2** : La liste agrège les ingrédients de tous les repas de la période :
  - les quantités sont ajustées selon la règle de calcul ;
  - un même ingrédient avec la même unité donne une seule ligne dont les quantités sont additionnées ;
  - un même ingrédient avec des unités différentes donne des lignes distinctes (voir Q3).
- **C3** : Je coche les articles au fil des courses.

## Hors périmètre V1

- Recettes (étapes, temps de préparation, photos).
- Suggestions automatiques de menus.
- Gestion du stock ou du placard.
- Articles hors plats dans la liste de courses (lessive, pain…).
- Préférences et contraintes alimentaires par membre.
- Budget, historique, statistiques.

## Questions ouvertes

- **Q1. Moments du repas** : midi et soir uniquement, ou aussi petit-déjeuner et goûter ? Un repas contient-il un seul plat, ou plusieurs (entrée, plat, dessert) ?
- **Q2. Période par défaut** : « les 7 prochains jours » inclut-il aujourd'hui (J à J+6) ou commence-t-il demain (J+1 à J+7) ?
- **Q3. Unités** : faut-il une liste d'unités fermée (g, kg, ml, cl, l, pièce, c. à soupe…) et des conversions simples pour agréger (g/kg, ml/cl/l) ?
- **Q4. Arrondis** : les quantités « à la pièce » sont-elles arrondies au supérieur (1,5 oignon donne 2) ?
- **Q5. Suppression d'un plat planifié** : faut-il l'interdire, vider les repas concernés, ou archiver le plat ?
- **Q6. Foyer et comptes** : un foyer partagé où tous les membres ont les mêmes droits, ou des rôles différents ? La liste cochée est-elle partagée en temps réel entre membres ?
- **Q7. Hors-ligne** : la liste de courses doit-elle rester consultable et cochable sans réseau en magasin ?
