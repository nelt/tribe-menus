# Conception : listes de courses

- **Mise à jour** : 2026-11-14 (plan `courses-suite`, lot A)
- **Statut** : retenu avec le plan [`courses`](../plans/2026-11-06-listes-de-courses.md), accordé aux specs, au glossaire et au design par son lot A (2026-11-07) ; noms du serveur des listes et des articles fixés par le lot B, ceux du calcul par le lot C ; en production avec `v0.4.0` (2026-11-08) ; à tenir à jour par les plans qui touchent ce domaine ; recalcul, part prise et déplacement ajoutés par le plan [`courses-suite`](../plans/2026-11-14-recalcul-et-selection-des-courses.md), lot A (2026-11-14), noms du code à fixer par ses lots B et C ; hors-ligne à venir
- **Stories** : C1 à C6, C11, C12 (`courses`) ; C8 à C10 (`courses-suite`) ; P2 pour le calcul ; T1 et ENF-02 pour les listes. C7 attend le plan hors-ligne.

Ce document décrit la conception retenue pour les listes de courses : modèle, calcul, découpage du code, API, échanges. Il s'appuie sur la bibliothèque de plats ([`bibliotheque-de-plats.md`](bibliotheque-de-plats.md)), dont il réutilise les quantités, les unités, le référentiel et la copie des collections, et sur le planning ([`planning-des-repas.md`](planning-des-repas.md)), dont il lit les plats servis. Les règles fonctionnelles sont dans `docs/specs/fonctionnalites.md` (C1 à C12, « Ajout des courses d'une période », « Recalcul d'une liste », « Sélection d'articles », « Règle de calcul », « Unités, conversions et arrondis », Q10 à Q18, Q22 à Q24, Q36 à Q43), les décisions dans les ADR (0003, 0027, 0029, 0031) et dans les plans `courses` (D1 à D24) et `courses-suite` (D1 à D22).

Noms anglais du code selon `docs/specs/glossaire.md` : liste de courses (*shopping list*), liste principale (*main list*), article (*item*), article calculé (*computed item*), article ajouté (*manual item*), article sans quantité (*unquantified item*), part prise (*taken quantity*), provenance (*item source*), quantité apportée (*contributed quantity*), plats servis reçus (*received served dishes*), ajouter les courses des repas (*add meal shopping*), période (*period*), déclarer les courses faites (*check out*), choix des listes (*list picker*), destination des courses (*target list*), copie des listes (*cached shopping lists*), recalculer (*recompute*), liste périmée (*stale list*), lignes gardées (*received lines*), déplacer (*move*), liste de départ (*source list*), liste d'arrivée (*target list*), articles partis entre-temps (*missing items*).

## Modèle du domaine

```mermaid
classDiagram
    direction LR

    class Tribe {
        name : string
        size : int
    }
    class ShoppingList {
        id : ListID
        name : string
        main : bool
        rank : int
    }
    class Item {
        id : ItemID
        quantity : Quantity [0..1]
        taken : Quantity [0..1]
        checked : bool
        manual : bool
    }
    class ItemSource {
        servedDish : ServedDishID
        dish : DishID
        date : Date
        mealtime : Mealtime
        servings : int
        contributed : Quantity [0..1]
    }
    class ReceivedServedDish {
        servedDish : ServedDishID
        dish : DishID
        servings : int
        dishServings : int
    }
    class ReceivedLine {
        <<lignes du plat au calcul>>
        ingredient : Ingredient
        quantity : Quantity [0..1]
    }
    class Ingredient {
        name : string
        defaultUnit : Unit [0..1]
    }
    class Quantity {
        <<valeur, internal/dish>>
        amount : int, en unités de base
        unit : Unit
    }
    class ServedDish {
        <<planning>>
        id : ServedDishID
    }
    class Dish {
        <<bibliothèque>>
        id : DishID
    }

    Tribe "1" *-- "1..20" ShoppingList : exactement une principale (C1)
    ShoppingList "1" *-- "0..300" Item
    ShoppingList "1" *-- "0..*" ReceivedServedDish : Q22
    Item "0..*" --> "1" Ingredient
    Item ..> Quantity
    Item "1" *-- "0..*" ItemSource : article calculé seulement
    ItemSource ..> Quantity
    ItemSource ..> ServedDish : identifiant seul, sans clé étrangère (D5)
    ItemSource ..> Dish
    ReceivedServedDish ..> ServedDish : identifiant seul
    ReceivedServedDish "1" *-- "0..40" ReceivedLine : D5
```

- **ShoppingList** : identifiant aléatoire (D10) ; nom de 40 caractères au plus, non unique (Q17, D12) ; **rang** de création, qui ordonne les listes après la principale (C2) ; **principale** pour exactement une liste de la tribu (Q23), garantie par un index unique partiel. La liste « Courses », principale, est créée par la migration `0009` dans toute base de tribu (D14). 20 listes au plus (D12).
- **Item** : identifiant aléatoire ; un ingrédient du référentiel ; une quantité facultative, en unités de base avec son unité d'affichage (ADR 0003, point 7) ; coché ou non ; **ajouté** (saisi à la main, sans provenance) ou **calculé** (au moins une provenance). Une liste a au plus un article calculé par ingrédient et famille d'unités, plus au plus un article calculé sans quantité par ingrédient (Q24, D3) ; les articles ajoutés ne sont jamais fusionnés (Q13). 300 articles au plus par liste (D12). **Part prise** (`taken`, colonne `taken_amount` de la migration `0010`, `courses-suite`, D1 et D9) : la quantité qu'avait un article calculé coché quand sa quantité a augmenté, en unités de base ; présente seulement sur un article calculé avec quantité, non coché, et inférieure à sa quantité (Q41).
- **ItemSource** (provenance) : ce qu'un plat servi a apporté à un article calculé, **au moment du calcul** : le plat servi, le plat, la date, le moment, les parts et la quantité apportée (absente pour un ingrédient sans quantité). Pas de clé étrangère vers `served_dishes` : un plat servi retiré du planning laisse sa provenance lisible (C5) jusqu'au recalcul (D5).
- **ReceivedServedDish** : les plats servis dont la liste a reçu les courses (Q22), avec le plat, les parts servies, les **parts de référence du plat** (`dishServings`) et les **lignes du plat au moment du calcul** (`ReceivedLine`), y compris ceux sans ingrédient (« Restes ») et ceux dont les articles sont partis par les courses faites ; jamais retirés en V1, sans clé étrangère. C'est ce qui dit qu'une liste est périmée (C8) et ce que le recalcul refait (« Recalcul et péremption » ci-dessous) : les lignes gardées distinguent une ligne ajoutée au plat, que le recalcul crée, d'une ligne dont l'article est parti, qu'il ne recrée pas (`courses`, D5 ; `courses-suite`, D2 et D5). Le nom du plat n'y est pas : un renommage ne rend pas une liste périmée. Ces données restent dans le serveur ; l'API ne donne que les identifiants (`received`).
- **Ingredient**, **Quantity**, **Dish**, **ServedDish** : inchangés. Un article ajouté peut créer un ingrédient (Q13) et fixer son unité par défaut (Q20).

## Calcul des courses d'une période

Le calcul est **pur** (`internal/shopping`, D6) : il reçoit la liste, les plats servis de la période, les plats et leurs lignes, et rend la liste nouvelle et ses comptes. Toutes les quantités sont des entiers en unités de base : milligramme pour les masses, millilitre pour les volumes, millième de pièce, de cuillère à soupe, de cuillère à café, de pincée.

1. **Plats servis retenus** : ceux de la période (bornes incluses) qui ne sont pas déjà reçus par la liste ; les autres sont comptés comme ignorés (Q22). Tous les retenus deviennent des plats servis reçus, même sans ingrédient, avec leurs parts, les parts de référence du plat et ses lignes, lues dans la transaction (D5).
2. **Quantité apportée** par chaque ligne de chaque plat retenu : `quantité du plat × parts servies ÷ parts de référence`, arrondie à l'unité de base supérieure (D7). Une ligne sans quantité apporte une provenance sans quantité (D3).
3. **Regroupement** par ingrédient et famille d'unités (masse, volume, pièce, cuillère à soupe, cuillère à café, pincée), plus un groupe « sans quantité » par ingrédient.
4. **Versement** dans la liste : un groupe dont l'article calculé existe déjà (même ingrédient, même famille, ou tous deux sans quantité) s'y ajoute et lui donne de nouvelles provenances ; sinon un article calculé nouveau est créé. Les articles ajoutés ne sont pas touchés.
5. **Arrondi de l'article**, après l'addition des quantités apportées de toutes ses provenances, anciennes et nouvelles : pièces, cuillères et pincées à l'entier supérieur (Q4), masses au gramme supérieur (Q37, D7) ; les volumes sont déjà en millilitres entiers.
6. **Coche** : un article coché dont la quantité arrondie augmente est décoché, et garde sa quantité d'avant comme part prise (Q41, `courses-suite`, D9 ; `applyTaken`) ; un article sans quantité garde sa coche (Q36, D3).
7. **Limite** : si la liste dépasse 300 articles, rien n'est écrit (D12).

Exemple de la spec : « Chili con carne », défini pour 4 parts avec 500 g de bœuf haché et 1 oignon, servi pour 3 parts le mardi soir et pour 4 parts le jeudi midi. Bœuf : 375 000 mg et 500 000 mg apportés, 875 000 mg, affichés « 875 g ». Oignon : 750 et 1 000 millièmes apportés, 1 750, arrondis à 2 000 : « 2 pièces » ; le dépliage montre « 0,75 pièce » et « 1 pièce ».

L'**unité d'affichage** d'un article calculé est la plus grande unité de sa famille qui donne au moins 1 (spec) : 1 500 000 mg en kg (« 1,5 kg »), 750 ml en cl (« 75 cl »). Elle est choisie par le navigateur (`shopping.ts`) à partir de la quantité renvoyée par l'API dans la plus petite unité de la famille (ci-dessous), avec les règles d'affichage des quantités du design.

## Recalcul et péremption

Plan `courses-suite`, D2, D5, D6 et D9 ; Q41 et Q43. Une seule fonction pure, `recompute`, dans `internal/shopping` : elle reçoit la liste (articles, provenances, plats servis reçus et leurs lignes gardées) et l'état actuel des plats servis qu'elle a reçus (parts, ou retiré du planning) et de leurs plats (parts de référence, lignes), lus dans la même transaction ; elle rend la liste recalculée. Une liste est **périmée** quand `recompute` changerait ses articles (`stale`).

1. **Plats servis vivants** : les plats servis reçus qui ont encore au moins une provenance dans la liste. Les autres (tous leurs articles achetés ou déplacés, ou qui n'ont jamais rien apporté) ne sont ni lus ni recalculés : ils ne rendent pas la liste périmée (D2).
2. **Clé** d'une ligne : l'ingrédient et la famille d'unités de sa quantité, ou l'ingrédient seul pour une ligne sans quantité (Q36).
3. **Pour chaque plat servi vivant**, clé par clé, en comparant les lignes actuelles du plat aux lignes gardées :
   - clé gardée, et un article de la liste porte la provenance de ce plat servi pour cette clé : la provenance est recalculée sur l'état actuel (`Contribution` : quantité × parts servies ÷ parts de référence, arrondie à l'unité de base supérieure, D7) ;
   - clé gardée, sans article ni provenance pour elle : l'article a quitté la liste (courses faites, ou déplacé) ; rien n'est recréé ;
   - clé nouvelle (ligne ajoutée au plat, ou dont l'unité a changé de famille) : une provenance est créée et versée dans l'article calculé de cette clé, existant ou créé (D2) ;
   - clé absente du plat actuel, ou plat servi retiré du planning (R4) : la provenance disparaît.
4. **Articles calculés** : la quantité de chaque article est recalculée à partir de toutes ses provenances, puis arrondie (`RoundItem`, Q37) ; un article sans provenance disparaît, même coché.
5. **Coche et part prise** (D9, Q41) : un article coché dont la quantité augmente est décoché, et sa part prise devient sa quantité d'avant ; un article décoché avec une part prise la garde quand sa quantité augmente encore ; un article dont la quantité baisse jusqu'à sa part prise ou en dessous est coché, sans part prise. Le même pas sert à l'ajout des courses d'une période (« Calcul des courses d'une période », étape 6).
6. **Lignes gardées** : celles des plats servis vivants sont remplacées par l'état actuel (parts, parts de référence, lignes) ; celles d'un plat servi retiré du planning ou sans provenance restante ne changent pas.
7. **Limite** : si la liste dépasse 300 articles, rien n'est écrit (`409 too_many_items`, Q38).

**`stale(list, état actuel)`** vaut vrai quand l'étape 4 ou 5 change un article : existence, quantité, part prise, coche, provenances (plat servi, parts, quantité apportée), les quantités comparées en unités de base. L'étape 6 seule n'en est pas un changement : une quantité réécrite dans une autre unité de sa famille (500 g en 0,5 kg), les parts d'un plat servi dont aucune ligne n'a de quantité, un plat renommé (le nom n'est pas gardé) ne rendent pas la liste périmée (revue #92, point 6). Un plat servi ajouté ensuite au planning n'est pas reçu : il n'entre pas dans la péremption (spec).

Exemple : « Chili con carne », défini pour 4 parts avec 500 g de bœuf haché et 1 oignon, servi pour 3 parts le mardi soir et 4 le jeudi midi, ses courses dans « Courses », le bœuf coché (875 g). On passe le mardi à 6 parts et on ajoute 1 pièce d'ail au plat : la liste est périmée. Le recalcul donne bœuf 750 g + 500 g = 1,25 kg, décoché, part prise 875 g (« 1,25 kg · encore 375 g ») ; oignon 1,5 + 1 = 2,5 arrondi à 3 pièces ; ail, clé nouvelle, 1,5 + 1 = 3 pièces, créé. Si le bœuf avait été acheté (courses faites), il ne serait pas recréé, l'oignon et l'ail si.

## Déplacement

Plan `courses-suite`, D10 ; « Sélection d'articles » de la spec ; Q41, Q42. Fonction pure `move` : elle reçoit la liste de départ, la liste d'arrivée (ou le nom d'une liste nouvelle) et les identifiants sélectionnés ; elle rend les deux listes et les comptes.

1. **Articles déplacés** : les identifiants sélectionnés encore dans la liste de départ ; les autres sont comptés dans `missing` (Q42), sans erreur.
2. **Article calculé** : s'il existe dans la liste d'arrivée un article calculé de même clé, les deux sont **fusionnés** : provenances réunies par plat servi, une provenance déjà présente n'étant pas comptée deux fois ; quantité recalculée et arrondie ; coché si les deux l'étaient, sinon part prise = somme des parts prises (la quantité entière d'un article coché), s'il y en a une (Q41). Sinon l'article passe tel quel dans la liste d'arrivée, son identifiant gardé.
3. **Article ajouté** : passe tel quel, ligne distincte (Q13).
4. **Plats servis reçus** : ceux des provenances déplacées sont copiés, avec leurs lignes gardées, dans la liste d'arrivée qui ne les a pas encore reçus ; elle garde les siens sinon. La liste de départ les garde aussi (Q22) : ils n'y ont plus de provenance pour les articles partis, que le recalcul ne recrée pas (étape 3 ci-dessus).
5. **Nouvelle liste** : nom selon `ParseListName` (D12), rang suivant, non principale, créée dans la même transaction.
6. **Refus**, rien n'est écrit : aucun identifiant (`no_items`), liste d'arrivée égale à la liste de départ (`same_list`), liste d'arrivée inconnue ou supprimée entre-temps (`target_not_found`), plus de 300 articles dans la liste d'arrivée après fusion (`too_many_items`), 20 listes pour une liste nouvelle (`too_many_lists`).

## Découpage du code

```mermaid
classDiagram
    direction TB

    namespace internal_shopping {
        class Lists {
            +Page(ctx, after) version, Page~List~
            +Version(ctx) int
            +Create(ctx, name) List, Versions
            +Rename(ctx, ListID, name) List, Versions
            +SetMain(ctx, ListID) Versions
            +Delete(ctx, ListID) Versions
            +AddItem(ctx, ListID, dish.LineInput) List, Versions
            +RemoveItem(ctx, ListID, ItemID) List, Versions
            +SetChecked(ctx, ListID, ItemID, bool) List, Versions
            +CheckOut(ctx, ListID, seen []ItemID) List, removed, Versions
            +AddMealShopping(ctx, ListID, Period) List, added, ignored, Versions
            +Stale(ctx) []ListID
            +Recompute(ctx, ListID) List, Versions
            +Move(ctx, ListID, []ItemID, Target) List, List, moved, missing, Versions
        }
        class Store {
            <<interface>>
            +Version(ctx) int
            +Lists(ctx, after, limit) version, Page~List~
            +Write(ctx, func(Tx) error) Versions
        }
        class Tx {
            <<interface>>
            dish.Catalog : Ingredient, AddIngredient, SetDefaultUnit
            +List(ctx, ListID) List, ErrNotFound
            +ListCount(ctx) int
            +NextRank(ctx) int
            +InsertList(ctx, List)
            +RenameList(ctx, ListID, name)
            +SetMain(ctx, ListID)
            +DeleteList(ctx, ListID)
            +InsertItem(ctx, ListID, Item, ingredient)
            +SetChecked(ctx, ItemID, bool)
            +DeleteItem(ctx, ItemID)
            +ServedDishes(ctx, Period) []Served, avec parts de référence et lignes du plat
            +UpdateItem(ctx, Item)
            +InsertSource(ctx, ItemID, Source)
            +InsertReceived(ctx, ListID, Served) et lignes du plat
            +Received(ctx, ListID) plats servis reçus et lignes gardées
            +CurrentServed(ctx, []ServedDishID) parts ou retiré, plats et lignes
            +ReplaceReceived(ctx, ListID, Served) et lignes du plat
            +DeleteItem, DeleteSource, MoveItem(ctx, ItemID, ListID)
        }
        class compute {
            <<functions>>
            addMealShopping(List, []Served, newID) liste, changements, added, ignored
            +Contribution(Quantity, servings, reference) Quantity
            +RoundItem(Quantity) Quantity
            recompute(List, []Current) liste recalculée, changements
            stale(List, []Current) bool
            move(source, target, []ItemID) départ, arrivée, moved, missing
            applyTaken(before, after Item) Item : coche et part prise (Q41)
        }
        class rules {
            <<functions>>
            +ParseListName(string) string
            +ParsePeriod(from, to) Period
            +SortItems([]Item)
            MaxListNameLength, MaxLists, MaxItems
        }
        class errors {
            <<variables>>
            ErrListNameEmpty, ErrListNameInvalid, ErrListNameTooLong
            ErrTooManyLists, ErrTooManyItems
            ErrInvalidDate, ErrInvalidPeriod, ErrPeriodTooLong
            ErrNoItems, ErrSameList, ErrTargetNotFound
            NotEmptyError, ErrMainList, ErrComputedItem, ErrNotFound
            ItemError, code de dish.ErrorCode
        }
    }
    namespace internal_dish {
        class quantities {
            <<réutilisé>>
            Unit, Family, Quantity, ParseQuantity
            NormalizeName (Q29), CleanName (D30)
            ParseLine, ResolveIngredient (Q13, Q20)
        }
    }
    namespace internal_mealplan {
        class mealplan_rules {
            <<réutilisé>>
            ParseDate (Q35)
        }
    }
    namespace internal_storage {
        class ShoppingStore {
            sqlc tribedb
        }
        class tables {
            <<tables et déclencheurs>>
            shopping_lists, shopping_items
            shopping_item_sources, shopping_received
            shopping_received_lines
            collection shopping_lists
        }
    }
    namespace internal_server {
        class shoppingAPI {
            GET, POST shopping-lists
            PUT shopping-lists/id/name
            PUT shopping-lists/main
            DELETE shopping-lists/id
            POST, DELETE items
            PUT items/item/checked
            POST checkout
            POST meal-shopping
            GET shopping-lists/stale
            POST recompute
            POST move
        }
    }
    namespace web_src {
        class shopping_ts {
            <<module pur>>
            +sortLists(lists) List[]
            +sortItems(items) Item[]
            +itemLabel(item) string
            +defaultPeriod(today) Period
            +monthGrid(month, period, served) Day[]
            +alreadyReceived(list, served, period) int
            +sourceRows(item, dishes) Row[]
            +checkoutSummary(list) partent, restent
            +remainingLabel(item) string, part prise (Q41)
            +selection : toggle, all, actions, vanished
            +moveMessage(moved, missing, target) string
            +deletedMessage(list, write) string (D12)
        }
        class shopping_lists_ts {
            <<copie en mémoire>>
            lists : CachedCollection~List~
            +check() listes, avec planning et bibliothèque, puis stale
            stale : Set~ListID~
            +recompute(ListID)
            +move(ListID, ids, target)
            +write(…) applyWrite ou vérification
        }
        class CachedCollection~T~ {
            <<collections.ts, générique>>
        }
        class ShoppingListScreen
        class ListPickerScreen
        class MealShoppingScreen
    }

    shoppingAPI --> Lists
    Lists --> Store
    Store ..> Tx
    Lists ..> compute
    Lists ..> rules
    Lists ..> errors
    compute ..> quantities
    rules ..> quantities
    rules ..> mealplan_rules
    ShoppingStore ..|> Store
    ShoppingStore ..> tables
    ShoppingListScreen --> shopping_lists_ts
    ListPickerScreen --> shopping_lists_ts
    MealShoppingScreen --> shopping_lists_ts
    ShoppingListScreen ..> shopping_ts
    ListPickerScreen ..> shopping_ts
    MealShoppingScreen ..> shopping_ts
    shopping_lists_ts *-- CachedCollection~T~ : listes
    CachedCollection~T~ ..> shoppingAPI : HTTP JSON, pages, ETag
```

- **`internal/shopping`** (nouveau paquet, ADR 0008) : règles pures (nom, période, limites), calcul pur (`compute`) et `Lists`, qui écrit derrière une interface `Store`, sans HTTP ni SQL (ADR 0002). Il dépend de `internal/dish` pour les unités, les familles et leur facteur (`Unit.Factor`), les quantités, le nom normalisé, la règle des noms (`CleanName`), celle d'une ligne saisie (`ParseLine`) et la résolution d'un ingrédient dans le référentiel (`ResolveIngredient`, sur l'interface `dish.Catalog`), communes aux plats et aux articles ajoutés ; de `internal/mealplan` pour la seule règle des dates (`ParseDate`), le `Tx` lui rendant les plats servis de la période. Le `Tx` écrit par opérations ciblées (une liste, un article, une coche, une provenance, un plat servi reçu), lues puis vérifiées par `Lists` ; le calcul pur rend la liste après, les articles créés ou changés avec les provenances qu'ils gagnent, et les plats servis reçus, que `Lists` écrit. Table de cas des quantités apportées et des arrondis : `internal/shopping/testdata/meal-shopping.json`, lue par `go test` et par `shopping.ts` (lot E).
- **Plan `courses-suite`** : `Stale`, `Recompute` et `Move` dans `Lists` ; `recompute`, `stale`, `move` et `applyTaken` dans le calcul pur, avec leurs tables de cas (`internal/shopping/testdata/`, dont `taken.json`, lue aussi par `shopping.ts`) ; migration `0010` (`taken_amount`) ; noms du `Tx` à fixer par les lots B et C, comme l'a fait `courses`.
- **`internal/storage`** : migration `0009` (cinq tables, leurs déclencheurs, la liste « Courses », collection `shopping_lists` dans `storage.Collections`), requêtes sqlc, adaptateur `ShoppingStore`. Une écriture augmente la version d'une unité exactement (`writeCollections`, conception de la bibliothèque), qu'elle touche une ou cent lignes. Un article ajouté qui crée un ingrédient augmente aussi `ingredients`. Supprimer une liste supprime ses plats servis reçus et leurs lignes ; retirer un article, ses provenances.
- **`internal/collection`** : inchangé ; `GET shopping-lists` passe par `Serve`.
- **`internal/server`** : routes minces sous `/tribes/<slug>/api/`, derrière la session.
- **`web/src`** : `shopping.ts`, module pur testé par `node:test`, dont l'ordre des articles partage sa table de cas avec `SortItems` (`internal/shopping/testdata/sort-items.json`, revue de la PR #98, point 3) ; `shopping-lists.ts`, copie des listes sur `CachedCollection`, vérifiée avec `planning.ts` et `library.ts` sur l'écran des courses des repas et pour le dépliage (lot E) ; écrans `shopping-list-screen.ts`, `list-picker-screen.ts` (lot D), `meal-shopping-screen.ts` (lot E) ; la combobox d'ingrédient de l'écran d'un plat devient un contrôleur Lit, `ingredient-combobox.ts`, partagé par l'écran d'un plat et le formulaire d'un article ajouté ; `warningBanner` est repris.

## API

| Route | Corps | Réponses |
| --- | --- | --- |
| `GET shopping-lists?after=…` | | 200 : `{ items: [List], next? }`, 20 listes au plus par page, dans l'ordre des identifiants (D4) ; `ETag: "shopping_lists-<version>"`, `304`, `412` (ADR 0029) |
| `POST shopping-lists` | `{ name }` | 201 : `{ list, versions }` ; 400 : `list_name_empty`, `list_name_invalid`, `list_name_too_long` ; 409 `too_many_lists` |
| `PUT shopping-lists/{id}/name` | `{ name }` | 200 : `{ list, versions }` ; 400 comme ci-dessus ; 404 |
| `PUT shopping-lists/main` | `{ list }` | 200 : `{ versions }`, vide pour la liste déjà principale ; 404 |
| `DELETE shopping-lists/{id}` | | 200 : `{ versions }` ; 409 `{ error: "list_not_empty", list }`, la liste à jour ; 409 `main_list` ; 404 |
| `POST shopping-lists/{id}/items` | `{ ingredient, quantity?, unit? }` | 201 : `{ list, versions }` ; 400 : les codes d'une ligne de plat, `ingredient_name_required`, `ingredient_name_invalid`, `ingredient_name_too_long`, `unit_required`, `invalid_unit`, `invalid_quantity`, `quantity_too_precise` (même règle, `dish.ParseLine`, mêmes messages) ; 409 `too_many_items` ; 404 |
| `DELETE shopping-lists/{id}/items/{item}` | | 200 : `{ list, versions }` ; 409 `computed_item` ; 404, liste supprimée entre-temps ou article inconnu (l'écran le traite comme un retrait fait, D11) |
| `PUT shopping-lists/{id}/items/{item}/checked` | `{ checked }` | 200 : `{ list, versions }` ; 400 `bad_request` sans `checked` ; 404 |
| `POST shopping-lists/{id}/checkout` | `{ items: [ItemID] }`, ceux vus cochés | 200 : `{ list, removed, versions }` (D9) ; 404 |
| `POST shopping-lists/{id}/meal-shopping` | `{ from, to }` | 200 : `{ list, added, ignored, versions }`, `versions` vide sans plat servi ajouté ; 400 `invalid_date`, `invalid_period`, `period_too_long` ; 409 `too_many_items` ; 404 |
| `GET shopping-lists/stale` | | 200 : `{ lists: [ListID] }`, les listes périmées dans l'ordre des identifiants, calculées à chaque appel ; sans ETag ni `304` (ADR 0031) |
| `POST shopping-lists/{id}/recompute` | | 200 : `{ list, versions }`, `versions` vide si rien ne change ; 409 `too_many_items` ; 404 |
| `POST shopping-lists/{id}/move` | `{ items: [ItemID], to: ListID }` ou `{ items, name }` | 200 : `{ list, target, moved, missing, versions }`, la liste de départ et la liste d'arrivée à jour ; 400 `no_items`, `same_list`, les codes du nom d'une liste ; 404 (liste de départ), 404 `{ error: "target_not_found" }` ; 409 `too_many_items`, `too_many_lists` |

Une liste (`List`) : `{ id, name, main, rank, items: [Item], received: [ServedDishID] }`, ses articles dans l'ordre de Q40 (`SortItems` : non cochés puis cochés, nom normalisé, avec quantité avant sans quantité, calculé avant ajouté, puis identifiant pour un ordre stable) ; un article (`Item`) : `{ id, ingredient, quantity?, unit?, taken?, checked, manual, sources: [{ servedDish, dish, date, mealtime, servings, quantity?, unit? }] }`.

**Un seul format de quantité, en lecture comme en écriture**, celui des lignes des plats (`lineResponse`, `internal/server/api_dishes.go`) et de `POST …/items` : `quantity`, le texte de `Quantity.Text()` (point décimal, sans zéro inutile, relu par `ParseQuantity`), et `unit`, le code d'une unité de la liste fermée (revue #92, point 2). Un article ajouté garde l'unité saisie. Un article calculé et ses provenances sont exprimés dans la **plus petite unité de leur famille** : `g`, `ml`, `piece`, `tablespoon`, `teaspoon`, `pinch` (« 875 » `g`, « 0.75 » `piece`) ; `shopping.ts` en tire l'unité d'affichage. Aucune quantité en unités de base (milligrammes, millièmes) ne sort du serveur : elles restent dans la base et le calcul. Le nom de l'ingrédient est donné en clair, comme dans les lignes d'un plat (le référentiel n'a pas d'identifiant public, `bibliotheque`, D10).

Toutes ces routes répondent 401 `no_session` sans session et gardent `Cache-Control: no-store`. Corps limité à 4 Kio (`maxBodyBytes`), sauf ceux de `checkout` et de `move`, limités à 16 Kio (`maxCheckoutBodyBytes`, comme `maxDishBodyBytes` pour un plat) : il porte au plus 300 identifiants de 32 caractères hexadécimaux, soit environ 10,5 Kio en JSON (revue #92, point 1).

- **`404` identique** pour une liste ou un article inconnu, supprimé entre-temps ou d'une autre tribu (ENF-02) ; pour la liste d'arrivée d'un déplacement, `target_not_found`, que l'écran distingue d'une liste de départ disparue.
- **`taken`** : la part prise, au format des quantités d'un article calculé (plus petite unité de sa famille), absente sans part prise (Q41).
- **Écritures sur une liste** : la réponse porte la liste entière à jour, que la copie prend ; un `409 list_not_empty` aussi.
- **`versions`** : celles des seules collections que l'écriture a changées (`shopping_lists`, et `ingredients` quand un article ajouté crée son ingrédient) ; une écriture qui ne change rien les laisse vides (lot D ; revue de la PR #98, point 2).
- **`from`, `to`** : `AAAA-MM-JJ`, dates réelles de 2000-01-01 à 2099-12-31 (Q35, `invalid_date` sinon, comme le planning) ; `to` au plus tôt `from` (`invalid_period`), 31 jours au plus (`period_too_long`, D12). Le serveur ne connaît pas « aujourd'hui » : la période par défaut est calculée par l'appareil (D24).

## Données dans le navigateur

Les listes vivent dans une **copie en mémoire** (`web/src/shopping-lists.ts`), une `CachedCollection` de la collection `shopping_lists`, comme les plats et le planning (ADR 0029, point 4). Elle est vérifiée à chaque visite de l'onglet Courses et à chaque retour au premier plan ; avec la copie du planning et celle de la bibliothèque sur l'écran des courses des repas (jours ayant des repas, plats servis déjà reçus) et pour le dépliage (nom et couleur des plats). Après une écriture, la copie prend la liste renvoyée et la version, ou se vérifie si un autre membre a écrit entre-temps ; après un `404`, elle retire la liste ou se vérifie. `PUT shopping-lists/main` ne renvoie que les versions : la copie inverse elle-même les deux drapeaux `main` si la version suit la sienne, sinon elle se vérifie. Rien n'est écrit sur l'appareil (D2) : la copie disparaît au rechargement, à la déconnexion et sur un 401.

Réalisation (lot D) : une coche est montrée aussitôt dans la copie, triée de nouveau, puis remplacée par la liste renvoyée ; sans réseau, l'article revient à son état. Un refus `409` (`list_not_empty`, `main_list`, `too_many_lists`, `too_many_items`) dit que la copie est en retard : elle prend la liste renvoyée s'il y en a une, puis se vérifie. Un `404` fait vérifier la copie, qui dit ensuite si la liste a été supprimée entre-temps (l'écran ouvre la liste principale avec son bandeau, D11) ou seulement l'article (aucun message, D8). Une écriture dont les `versions` portent `ingredients` différente de celle de la copie du référentiel fait vérifier celle-ci : l'élément de liste ne porte pas l'unité par défaut d'un ingrédient créé (Q20). La copie des listes est vérifiée au retour au premier plan quand l'onglet Courses est affiché.

Réalisation (lot E) : la copie des listes se vérifie avec celle de la bibliothèque, pour le dépliage ; l'écran des courses des repas vérifie aussi la copie du planning, à son ouverture et au retour au premier plan. Après un refus ou un `404`, seule la copie des listes se vérifie. `POST …/meal-shopping` passe par la même lecture de réponse que les autres écritures, qui en garde `added` et `ignored`.

Plan `courses-suite` : après chaque vérification de la copie, `GET shopping-lists/stale` donne les listes périmées, gardées à part de la copie (`stale`), puisqu'elles dépendent du planning et des plats et non de la version des listes (ADR 0031). Une liste recalculée en sort aussitôt ; un ajout de courses n'en fait pas entrer. Un déplacement prend les deux listes de la réponse. Le message d'une liste supprimée entre-temps nomme la liste d'après la copie d'avant la vérification (D12). La sélection est un état de l'écran d'une liste, jamais de la copie : un article qui disparaît de la copie quitte la sélection.

Ce que l'écran montre de ces échanges (bandeaux, feuilles, messages, « Réseau nécessaire ») est décrit dans `docs/design/README.md`, « Listes de courses » et « Messages des listes de courses » (lot A). `shopping.ts` trie de nouveau les articles après une coche locale, avec la même règle que `SortItems`.

## Échanges

### Ajouter les courses des repas d'une période

```mermaid
sequenceDiagram
    autonumber
    actor M as Membre
    participant S as Écran des courses des repas (SPA)
    participant L as shopping-lists.ts
    participant A as API
    participant K as shopping.Lists
    participant D as Base de la tribu

    M->>S: « Ajouter les courses des repas » depuis « Courses »
    S->>L: check() avec planning et bibliothèque
    S->>S: defaultPeriod(lundi 5 octobre) : 6 au 12 octobre (D24)
    S-->>M: calendrier, points des repas, liste « Courses »,<br/>« 1 plat servi déjà reçu par cette liste sera ignoré »
    M->>S: « Ajouter à « Courses » »
    S->>L: addMealShopping(courses, 2026-10-06, 2026-10-12)
    L->>A: POST shopping-lists/courses/meal-shopping { from, to }
    A->>K: AddMealShopping
    K->>K: ParsePeriod : 7 jours, au plus 31 (D12)
    K->>D: BEGIN IMMEDIATE
    K->>D: la liste, ses articles, provenances, plats servis reçus
    K->>D: plats servis du 6 au 12 octobre, leurs plats et lignes (P2)
    K->>K: compute : retenus, ignorés (Q22), quantités apportées,<br/>regroupement, versement, arrondi, décoche (D7)
    alt plus de 300 articles
        A-->>L: 409 too_many_items, rien d'écrit
    else
        K->>D: articles créés ou changés, provenances,<br/>plats servis reçus et leurs lignes (D5)
        K->>D: COMMIT, shopping_lists passe de 31 à 32
        A-->>L: 200, la liste à jour, added 4, ignored 1, versions
        L->>L: applyWrite : 32 suit 31
        S-->>M: retour à « Courses » : « 4 plats servis ajoutés, 1 déjà reçu ignoré »
    end
```

La transaction immédiate (ADR 0003) sérialise deux ajouts simultanés : le second voit les plats servis reçus par le premier et les ignore (Q22, D6).

### Cocher, puis déclarer les courses faites à deux

```mermaid
sequenceDiagram
    autonumber
    actor A1 as Alice
    actor B1 as Bruno
    participant P as shopping-lists.ts (Bruno)
    participant A as API
    participant D as Base de la tribu

    Note over A1,B1: « Courses » ouverte sur les deux téléphones
    B1->>P: coche « pain »
    P->>A: PUT …/items/pain/checked { true }
    A->>D: coché, shopping_lists 32 → 33
    A-->>P: 200, la liste, versions
    B1->>P: « Courses faites » : la feuille annonce 1 article retiré, 5 restants
    A1->>A: PUT …/items/lait/checked { true }
    A->>D: coché, 33 → 34 (dernier enregistrement gagnant, D8)
    B1->>P: confirme
    P->>A: POST …/checkout { items: [pain] }
    A->>D: retire « pain », encore coché, « lait » n'était pas annoncé, il reste (D9)
    A-->>P: 200, la liste à jour, removed 1, versions { shopping_lists: 35 }
    P->>P: 35 ne suit pas 33 : la copie prend la liste et se vérifie
    P-->>B1: bandeau vert « 1 article retiré », « lait » coché, en bas
```

### Supprimer une liste devenue non vide entre-temps

```mermaid
sequenceDiagram
    autonumber
    actor A1 as Alice
    actor B1 as Bruno
    participant P as shopping-lists.ts (Bruno)
    participant A as API
    participant D as Base de la tribu

    Note over P: copie : « Marché » vide
    A1->>A: POST shopping-lists/marche/items { "pain" }
    A->>D: article ajouté, version augmentée
    B1->>P: choix des listes, « ⋮ » de « Marché », « Supprimer la liste »
    P->>A: DELETE shopping-lists/marche
    A->>D: la liste a un article
    A-->>P: 409 list_not_empty, la liste à jour
    P->>P: put(liste à jour), puis check()
    P-->>B1: « Cette liste n'est plus vide. », « Supprimer » inactif, avec sa raison
```

### Recalculer une liste périmée

```mermaid
sequenceDiagram
    autonumber
    actor B1 as Bruno
    actor A1 as Alice
    participant P as Écran d'une liste (Alice)
    participant L as shopping-lists.ts
    participant A as API
    participant K as shopping.Lists
    participant D as Base de la tribu

    B1->>A: PUT meal-plan/…/servings { 6 } (mardi soir)
    A->>D: meal_plan 40 → 41, shopping_lists inchangée
    A1->>P: onglet Courses
    P->>L: check()
    L->>A: GET shopping-lists, If-None-Match "shopping_lists-34"
    A-->>L: 304
    L->>A: GET shopping-lists/stale
    A->>K: Stale
    K->>D: listes, plats servis reçus vivants, état actuel des plats servis et des plats
    K->>K: stale(liste, état actuel) pour chaque liste
    A-->>L: 200 { lists: [courses] }
    P-->>A1: bandeau « Des repas ont changé… », « Recalculer »
    A1->>P: « Recalculer »
    P->>L: recompute(courses)
    L->>A: POST shopping-lists/courses/recompute
    A->>K: Recompute
    K->>D: BEGIN IMMEDIATE, la liste et l'état actuel
    K->>K: recompute : provenances recalculées, ail créé,<br/>bœuf décoché avec sa part prise (Q41)
    K->>D: articles, provenances, lignes gardées ; COMMIT, 34 → 35
    A-->>L: 200, la liste à jour, versions
    L->>L: applyWrite ; courses sort de stale
    P-->>A1: « Liste recalculée. », bœuf « 1,25 kg · encore 375 g »
```

La péremption n'est pas mise en cache : chaque vérification la redemande, la liste pouvant devenir périmée sans que sa version change (ADR 0031).

### Déplacer une sélection pendant qu'un autre membre retire un article

```mermaid
sequenceDiagram
    autonumber
    actor A1 as Alice
    actor B1 as Bruno
    participant P as Écran d'une liste (Alice)
    participant L as shopping-lists.ts
    participant A as API
    participant K as shopping.Lists
    participant D as Base de la tribu

    A1->>P: « Sélectionner », « pain » et « lessive »
    B1->>A: DELETE shopping-lists/courses/items/lessive
    A->>D: lessive retirée, 35 → 36
    A1->>P: « Déplacer vers… », « Marché »
    P->>L: move(courses, [pain, lessive], marche)
    L->>A: POST shopping-lists/courses/move { items, to: marche }
    A->>K: Move
    K->>D: BEGIN IMMEDIATE, les deux listes
    K->>K: move : pain déplacé, lessive comptée dans missing
    K->>D: pain dans « Marché » ; COMMIT, 36 → 37
    A-->>L: 200, les deux listes, moved 1, missing 1, versions
    L->>L: 37 ne suit pas 35 : check()
    P-->>A1: hors sélection, « 1 article déplacé vers « Marché ».<br/>1 article n'était plus dans la liste. »
```

## Preuves

- **Contre l'API** (godog) : listes (principale à la naissance, changement de principale, création, nom, renommage, suppression et ses refus), articles ajoutés (quantité facultative, référentiel enrichi, retrait, deux articles du même ingrédient), ordre des articles, coche, courses faites, toutes les règles de calcul de C4 (dont la période et ses bornes, les masses au gramme) et le décompte de C5, P2, ingrédient sans quantité, limites, écritures croisées, T1, ENF-02 (listes et articles) ; tableau « Scénarios couverts » du plan.
- **Dans le navigateur** (`@ui`, ADR 0028), horloge de Playwright au lundi 5 octobre 2026 : l'onglet Courses ouvre la liste principale (C1), ordre des listes (C2), période et liste par défaut, date de fin avant le début (C4), dépliage d'un article, article ajouté non dépliable (C5), sous Chromium et WebKit.
- **Plan `courses-suite`** : contre l'API, la péremption (par `GET shopping-lists/stale`) et ses changements sans effet, le recalcul (articles ajoutés, coches, part prise, articles achetés ou déplacés non recréés, ligne ajoutée au plat, 300 articles), la part prise à l'ajout des courses et aux courses faites, le déplacement (fusion, provenances, coche, plats servis reçus, articles partis entre-temps, liste d'arrivée supprimée, mêmes listes, limites), ENF-02 ; dans le navigateur, le bandeau et « Recalculer », la pastille « Repas modifiés », la sélection, le déplacement et la nouvelle liste depuis l'écran, sous les sept projets, et la sélection dans `formats-ecran.feature`.
- **Tests unitaires** : règles et calcul de `internal/shopping` (`go test`, table de cas `internal/shopping/testdata/` tirée des exemples de la spec) ; `shopping.ts` (`node:test`) : ordres, libellés et unités d'affichage, période par défaut, grille d'un mois, plats servis déjà reçus, lignes dépliées, résumé des courses faites.

## Ce que ce document ne couvre pas encore

- **Le hors-ligne** des listes (C7, Q16, Q18) : stockage sur l'appareil, file des coches et des ajouts, rejeu au retour du réseau ; plan hors-ligne, qui s'ajoutera à la copie des collections sans la remplacer (ADR 0029, conséquences). Ce plan reprendra aussi un défaut d'affichage noté à la revue de la PR #100 : pendant qu'une coche attend sa réponse, la réponse à une autre écriture remplace la liste entière, et l'article en attente s'affiche un instant dans son état d'avant ; la file des coches devra garder les coches en attente par-dessus les listes reçues.
- **Le delta d'une collection**, si les listes deviennent longues à recharger (plan, questions ouvertes).
