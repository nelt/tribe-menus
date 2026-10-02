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
| nom (d'une session) | session label | nom donné par le membre, EF-04 |
| appareil détecté | detected device | EF-04 |
| essai (de code) | attempt | 3 par code, ENF-01 |
| limitation des demandes | rate limit | ENF-01 |
| anonymisé | anonymized | EF-11 |
| journal d'audit | audit log | |
| opération d'audit | audit operation | valeurs ci-dessous |
| initialisation de la tribu | tribe initialized | EF-08 |
| ajout / révocation / réactivation / anonymisation (d'un membre) | member added / member revoked / member reactivated / member anonymized | EF-07, EF-11 |
| ouverture de session | session opened | |
| révocation de session | session revoked | |
| déconnexion des autres appareils | other devices signed out | |
| déconnexion | signed out | |
| fermeture de session (suite à la révocation d'un membre) | session closed | |
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
| liste principale | main list | une seule par tribu, C1 |
| nom (d'une liste) | name | |
| ajouter les courses des repas | add meal shopping | C4 |
| période (début, fin) | period (start, end) | bornes incluses, celle de l'ajout des courses |
| plats servis reçus | received served dishes | plats servis dont une liste a reçu les courses, Q22 |
| article | item | |
| article calculé | computed item | |
| article ajouté | manual item | saisi à la main, hors planning |
| cocher / coché | check / checked | |
| déclarer les courses faites | check out | retire les articles cochés, C11 |
| supprimer (une liste) | delete | liste vide et non principale, C12 |
| recalculer / recalcul | recompute / recomputation | |
| périmée | stale | |
| sélection | selection | C9, C10 |
| déplacer | move | C9 |
| provenance (d'un article calculé) | item source | un plat servi, C5 |
| quantité apportée | contributed quantity | part d'une provenance, C5 |
| nombre de plats servis (d'un article) | source count | C5 |

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
