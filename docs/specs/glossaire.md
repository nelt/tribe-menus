# Glossaire français → anglais

Les spécifications et les scénarios Gherkin sont en français ; le code est entièrement en anglais (ADR 0008). Ce glossaire fixe la traduction de chaque terme métier : le code utilise **exactement** ces mots, et tout nouveau terme y est ajouté avant d'apparaître dans le code.

## Tribu, membres et sessions

| Français | Anglais | Remarque |
| --- | --- | --- |
| tribu | tribe | |
| taille de la tribu | tribe size | 4 par défaut, T3 |
| identifiant d'URL (de la tribu) | slug | `/tribes/<slug>/` |
| registre (des tribus) | registry | base globale, ADR 0003 |
| membre | member | |
| nom d'affichage | display name | |
| actif / révoqué | active / revoked | statut d'un membre |
| révoquer / réactiver | revoke / reactivate | |
| quitter la tribu | leave the tribe | auto-révocation |
| code de connexion | login code | |
| session | session | |
| appareil | device | |
| déconnecter tous les autres appareils | sign out other devices | |
| journal d'audit | audit log | |
| script d'administration | admin command | sous-commande `admin` du binaire |
| supprimer une tribu | delete a tribe | EF-10 |
| anonymiser (un membre) | anonymize | EF-11 |

## Plats et ingrédients

| Français | Anglais | Remarque |
| --- | --- | --- |
| ingrédient | ingredient | |
| référentiel d'ingrédients | ingredient catalog | |
| bibliothèque de plats | dish library | |
| plat | dish | |
| parts de référence | reference servings | taille de la tribu par défaut |
| unité par défaut | default unit | |
| couleur (d'un plat) | color | teinte de la palette, Q19 |

## Planning

| Français | Anglais | Remarque |
| --- | --- | --- |
| planning | meal plan | |
| repas | meal | une date et un moment |
| moment : midi / soir | mealtime: lunch / dinner | |
| plat servi | served dish | |
| parts (d'un plat servi) | servings | |

## Listes de courses

| Français | Anglais | Remarque |
| --- | --- | --- |
| liste de courses | shopping list | |
| période (début, fin) | period (start, end) | bornes incluses |
| article | item | |
| article calculé | computed item | |
| article ajouté | manual item | saisi à la main, hors planning |
| cocher / coché | check / checked | |
| en cours / faite | open / completed | état d'une liste |
| déclarer les courses faites | complete | |
| abandonner | discard | supprime la liste |
| recalculer / recalcul | recompute / recomputation | |
| périmée | stale | |
| historique | history | |

## Quantités

| Français | Anglais | Remarque |
| --- | --- | --- |
| quantité | quantity | |
| unité | unit | |
| famille d'unités : masse, volume, pièce, cuillère à soupe, cuillère à café, pincée | unit family: mass, volume, piece, tablespoon, teaspoon, pinch | les quatre dernières ne comptent qu'une unité |
| pièce | piece | |
| cuillère à soupe / à café | tablespoon / teaspoon | |
| pincée | pinch | |
| unité de base | base unit | stockage en entiers, ADR 0003 |
