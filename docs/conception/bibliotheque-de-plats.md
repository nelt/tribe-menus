# Conception : bibliothèque de plats

- **Mise à jour** : 2026-10-29
- **Statut** : brouillon, avec le plan [`bibliotheque`](../plans/2026-10-29-bibliotheque-de-plats.md) ; à confirmer à sa revue
- **Stories** : P1 à P4, T1 à T3 ; ENF-02 pour les plats et le référentiel

Ce document décrit la conception retenue pour la bibliothèque de plats, le référentiel d'ingrédients et la taille de la tribu : modèle, découpage du code, échanges. Il est tenu à jour par les plans qui touchent ce domaine (planning, listes de courses, hors-ligne) ; un plan, lui, se fige une fois terminé. Les règles fonctionnelles sont dans `docs/specs/fonctionnalites.md`, les décisions dans les ADR et dans les plans ; ce document les cite sans les répéter.

Noms anglais du code selon `docs/specs/glossaire.md` ; les termes marqués « nouveau » y sont ajoutés par le lot A du plan.

## Modèle du domaine

```mermaid
classDiagram
    direction LR

    class Tribe {
        name : string
        size : int
    }
    class Dish {
        id : DishID
        name : string
        color : Color
        referenceServings : int
        revision : int
    }
    class DishIngredient {
        position : int
        quantity : Quantity [0..1]
    }
    class Ingredient {
        name : string
        normalizedName : string
        defaultUnit : Unit [0..1]
    }
    class Quantity {
        amount : int
        unit : Unit
    }
    class Unit {
        <<enumeration>>
        g
        kg
        ml
        cl
        l
        piece
        tablespoon
        teaspoon
        pinch
    }
    class UnitFamily {
        <<enumeration>>
        mass
        volume
        piece
        tablespoon
        teaspoon
        pinch
    }
    class Color {
        <<enumeration>>
        9 teintes de la palette
    }

    Tribe "1" *-- "0..*" Dish : dish library
    Tribe "1" *-- "0..*" Ingredient : ingredient catalog
    Dish "1" *-- "0..40" DishIngredient : ingredients
    DishIngredient "0..*" --> "1" Ingredient
    DishIngredient ..> Quantity
    Quantity ..> Unit
    Unit ..> UnitFamily : belongs to
    Ingredient ..> Unit : default unit
    Dish ..> Color
```

- **Tribe** : la ligne unique de la base de la tribu (ADR 0003) gagne sa **taille** (T3), 4 par défaut, de 1 à 20 (plan, D12). Elle propose les parts d'un nouveau plat, puis, avec le planning, celles d'un plat servi.
- **Dish** : identifiant aléatoire (D10), nom unique dans la tribu au sens du nom normalisé (D7), couleur attribuée à la création et jamais changée (Q19), parts de référence, **révision** (nouveau terme : *revision*), augmentée à chaque modification, qui fait refuser un enregistrement fait sur une version dépassée (D5).
- **DishIngredient** : une ligne par ingrédient dans un plat (D6), dans l'ordre de saisie ; quantité **facultative** (D4) : sans quantité, pas d'unité.
- **Ingredient** : nom affiché de la première saisie, nom normalisé unique dans la tribu, unité par défaut = première unité saisie avec lui, absente tant qu'aucune ligne n'en a eu (Q20, D4). Créé seulement en saisissant un plat, ici, ou un article ajouté, au plan des listes.
- **Quantity** : un entier dans l'unité de base de la famille (milligramme, millilitre, millième de pièce, de cuillère ou de pincée ; ADR 0003, point 7) et l'unité choisie pour la ligne, qui sert à l'afficher comme elle a été saisie. Une saisie n'a que les décimales que l'unité de base représente : aucune en ml, une en cl, trois dans les autres unités ; au-delà, elle est refusée (plan, D20).
- **Pas de suppression** de plat (Q5b) ni d'ingrédient (pas d'écran du référentiel en V1).

### Nom normalisé

Une seule règle, pour les noms d'ingrédients (Q20), les noms de plats (D7), la recherche (P3) et l'autocomplétion (P4) :

1. minuscules, d'abord, pour que les étapes suivantes n'aient à connaître que des minuscules (« Œufs » devient « œufs ») ;
2. ligatures dépliées : œ → oe, æ → ae (D8, au-delà de Q20 : sans cela, « boeuf » ne trouve pas « bœuf ») ;
3. décomposition Unicode (NFD), puis retrait des seuls signes diacritiques combinants U+0300 à U+036F, comme la maquette `Doux-Bibliotheque` : le même ensemble en Go et en TypeScript, et non la catégorie Unicode `Mn` entière ;
4. apostrophes ramenées à `'` (U+0027) : ’ (U+2019), que la ponctuation intelligente de l'iPhone saisit par défaut, ‘ (U+2018) et ʼ (U+02BC) ;
5. espaces : l'espace insécable (U+00A0) et l'espace fine insécable (U+202F) comptent comme des espaces ; espaces en tête et en fin retirés, suites d'espaces réduites à une espace ordinaire.

La table de cas commune (D9) contient au moins « Œufs brouillés », « oeufs brouillés » et « œufs brouillés », qui donnent le même nom normalisé, des lettres accentuées en majuscule (« Éclair », « ÉPINARDS »), « piment d’Espelette » et « piment d'Espelette », et un nom avec une espace insécable.

Ces règles fixent l'identité d'un ingrédient et l'unicité d'un nom de plat, que les index uniques de la migration `0007` figent : les corriger après coup demanderait de fusionner des données.

Elle existe deux fois : en Go, qui décide de l'identité d'un ingrédient et de l'unicité d'un nom de plat, et en TypeScript, qui cherche et propose. La décomposition Unicode se fait par `golang.org/x/text/unicode/norm` en Go, seule dépendance d'exécution nouvelle, et par `String.prototype.normalize` dans le navigateur. Une table de cas commune, lue par `go test` et `node:test`, les garde d'accord (D9).

Pour la recherche et l'autocomplétion, le nom normalisé est découpé en **mots** sur tout caractère hors `a-z` et `0-9` ; un terme recherché trouve un mot qui **commence** par lui, et tous les termes doivent être trouvés (P3, comme la maquette `Doux-Bibliotheque`).

## Découpage du code

```mermaid
classDiagram
    direction TB

    namespace internal_dish {
        class Library {
            +List(ctx) []Dish
            +Create(ctx, DishInput) Dish
            +Update(ctx, DishID, revision, DishInput) Dish
            +Catalog(ctx) []Ingredient
        }
        class DishInput {
            name : string
            servings : int [0..1]
            lines : []LineInput
        }
        class LineInput {
            name : string
            quantity : Quantity [0..1]
        }
        class Store {
            <<interface>>
            +Dishes(ctx) []Dish
            +Ingredients(ctx) []Ingredient
            +InTx(ctx, func(TxStore) error) error
        }
        class rules {
            <<functions>>
            +NormalizeName(string) string
            +PickColor(counts) Color
            +ParseQuantity(string, Unit) Quantity
        }
    }
    namespace internal_storage {
        class DishStore {
            sqlc tribedb
        }
    }
    namespace internal_server {
        class dishAPI {
            GET dishes
            POST dishes
            PUT dishes/id
            GET ingredients
            GET PUT tribe
        }
    }
    namespace web_src {
        class dishes_ts {
            <<module pur>>
            +normalizeName(string) string
            +searchDishes(dishes, query) Dish[]
            +suggestIngredients(catalog, text, exclude) Ingredient[]
            +formatQuantity(Quantity) string
            +parseQuantity(string, Unit) Quantity
        }
        class DishLibraryScreen
        class DishScreen
        class TribeScreen
        class TabBar
    }

    dishAPI --> Library
    Library --> Store
    Library ..> rules
    Library ..> DishInput
    DishInput *-- LineInput
    DishStore ..|> Store
    DishLibraryScreen ..> dishes_ts
    DishScreen ..> dishes_ts
    DishLibraryScreen ..> dishAPI : HTTP JSON
    DishScreen ..> dishAPI : HTTP JSON
    TribeScreen ..> dishAPI : HTTP JSON
```

- **`internal/dish`** (nouveau paquet, ADR 0008 : un paquet par domaine) : la logique pure (règles, validation, choix de la couleur, résolution des ingrédients) derrière une interface `Store` ; aucune trace de HTTP ni de SQL (ADR 0002). Le paquet des listes de courses s'en servira pour le référentiel.
- **`internal/storage`** : migration `0007` de la base de la tribu, requêtes sqlc, adaptateur `DishStore`.
- **`internal/tribe`** : la taille de la tribu, avec le nom.
- **`internal/server`** : routes minces sous `/tribes/<slug>/api/`, derrière la session (ADR 0001, 0003).
- **`web/src`** : `dishes.ts`, module pur testé par `node:test` (ADR 0005, point 3) ; écrans en DOM classique, stepper de parts en shadow DOM avec `ElementInternals` (ADR 0004, point 3).

Les noms des méthodes sont indicatifs : le lot B les fixe.

## API

| Route | Corps | Réponses |
| --- | --- | --- |
| `GET dishes` | | 200 : tous les plats de la tribu, avec leurs lignes (D2) |
| `POST dishes` | nom, parts (facultatives : taille de la tribu), lignes `{ name, quantity?, unit? }` | 201 : le plat ; 400 : `name_required`, `name_taken`, `invalid_line`… |
| `PUT dishes/{id}` | révision lue, nom, parts, lignes | 200 : le plat ; 409 `dish_changed` avec le plat à jour (D5) ; 404 comme un identifiant inconnu (ENF-02) |
| `DELETE dishes/{id}` | | 405 : la route n'existe pas (P2, ADR 0005, point 4) |
| `GET ingredients` | | 200 : le référentiel, nom et unité par défaut |
| `GET tribe` | | 200 : nom, taille, nombre de membres actifs |
| `PUT tribe/size` | taille | 200 ; 400 hors de 1 à 20 |

Une ligne désigne son ingrédient **par son nom** : le serveur le rapproche d'un ingrédient existant de même nom normalisé, ou le crée (Q20). C'est déjà le mode prévu pour un article ajouté hors ligne (ADR 0004, point 7).

## Échanges

### Ouvrir l'onglet Plats et chercher

```mermaid
sequenceDiagram
    autonumber
    actor M as Membre
    participant S as Écran Plats (SPA)
    participant A as API
    participant L as dish.Library
    participant D as Base de la tribu

    M->>S: touche l'onglet Plats
    par
        S->>A: GET dishes
        A->>L: List
        L->>D: plats et lignes
        A-->>S: 200, tous les plats
    and
        S->>A: GET ingredients
        A-->>S: 200, le référentiel
    end
    S-->>M: « 8 plats », par ordre alphabétique
    M->>S: saisit « courg »
    Note over S: searchDishes, sans réseau :<br/>mots du nom et des ingrédients,<br/>préfixes, tous les termes
    S-->>M: « 2 résultats pour « courg » »
```

La recherche ne coûte aucune requête : elle fonctionnera hors ligne quand le plan hors-ligne mettra ces deux réponses en cache (ADR 0004, point 7 ; PT-04).

### Créer un plat

```mermaid
sequenceDiagram
    autonumber
    actor M as Membre
    participant S as Écran Nouveau plat (SPA)
    participant A as API
    participant L as dish.Library
    participant D as Base de la tribu

    M->>S: « Nouveau »
    S->>A: GET tribe
    A-->>S: 200, taille 5
    S-->>M: parts proposées = 5, la taille de la tribu (T3)
    M->>S: saisit « tom » comme ingrédient
    Note over S: suggestIngredients sur le référentiel chargé,<br/>sans ceux déjà dans le plat (D6)
    S-->>M: « tomate » (pièce), « tomate concassée » (g), Créer « tom »
    M->>S: choisit « tomate », 4 pièces, « Enregistrer le plat »
    S->>A: POST dishes { name, servings, lines }
    A->>L: Create
    L->>L: valide : nom, lignes, quantités, doublons
    L->>D: BEGIN IMMEDIATE
    L->>D: nom de plat déjà pris ?
    loop chaque ligne
        L->>D: ingrédient de même nom normalisé ?
        alt absent
            L->>D: crée l'ingrédient, unité par défaut = celle de la ligne
        else présent sans unité par défaut
            L->>D: unité par défaut = celle de la ligne, s'il y en a une
        end
    end
    L->>D: couleurs en usage, puis PickColor (Q19)
    L->>D: insère le plat et ses lignes, révision 1
    L->>D: COMMIT
    A-->>S: 201, le plat
    S-->>M: retour à la bibliothèque, le plat y figure
```

Tout se fait dans une transaction d'écriture (`BEGIN IMMEDIATE`, ADR 0003) : deux membres qui créent en même temps « tomate » ou un plat du même nom ne font ni doublon ni couleur choisie sur un état périmé ; le second reçoit `name_taken` pour le nom.

### Modifier un plat que quelqu'un vient de modifier

```mermaid
sequenceDiagram
    autonumber
    actor A1 as Alice
    actor B1 as Bruno
    participant A as API
    participant L as dish.Library
    participant D as Base de la tribu

    A1->>A: GET dishes (Chili, révision 3)
    B1->>A: GET dishes (Chili, révision 3)
    A1->>A: PUT dishes/chili { revision 3, 600 g de bœuf }
    A->>L: Update
    L->>D: révision actuelle = 3 : enregistre, révision 4
    A-->>A1: 200, révision 4
    B1->>A: PUT dishes/chili { revision 3, ajoute du cumin }
    A->>L: Update
    L->>D: révision actuelle = 4 ≠ 3
    A-->>B1: 409 dish_changed, le plat en révision 4
    Note over B1: « Ce plat a été modifié entre-temps. »<br/>Le formulaire montre le plat à jour,<br/>Bruno refait sa modification.
```

### Régler la taille de la tribu

```mermaid
sequenceDiagram
    autonumber
    actor M as Membre
    participant T as Écran Tribu (SPA)
    participant A as API
    participant D as Base de la tribu

    M->>T: pastille de la tribu, en tête du Planning
    T->>A: GET tribe
    A-->>T: nom, taille 4, membres actifs
    M->>T: « + » sur le stepper
    T->>A: PUT tribe/size { 5 }
    A->>D: taille = 5
    A-->>T: 200
    Note over T,D: Les plats existants gardent leurs parts,<br/>un nouveau plat est proposé pour 5 (T3).
```

## Ce que ce document ne couvre pas encore

- Le plat servi et le planning (R1 à R5), qui ajouteront au modèle le repas et le plat servi.
- Les listes de courses : article sans quantité issu d'une ligne sans quantité (plan, questions ouvertes).
- Le hors-ligne : cache des réponses `GET dishes` et `GET ingredients` par le service worker, effacement à la déconnexion.
