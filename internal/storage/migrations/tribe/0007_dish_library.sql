-- Dish library, ingredient catalog and tribe size (plan bibliotheque, lot B).

-- Tribe size (T3): proposes the servings of a new dish, 4 by default, from 1 to 50 (D12).
ALTER TABLE tribe ADD COLUMN size INTEGER NOT NULL DEFAULT 4 CHECK (size BETWEEN 1 AND 50);

-- Version of each collection of the API (ADR 0029): increased by the triggers of the tables
-- of the collection on every write, whatever the code that writes.
CREATE TABLE collection_versions (
    name TEXT PRIMARY KEY,
    version INTEGER NOT NULL CHECK (version >= 0)
) STRICT;

INSERT INTO collection_versions (name, version) VALUES ('dishes', 0), ('ingredients', 0);

-- Units of the closed list, by code (glossary): g, kg, ml, cl, l, piece, tablespoon,
-- teaspoon, pinch. Quantities are integers in the base unit of the family (ADR 0003,
-- point 7): milligrams, millilitres, thousandths of a piece, a spoon or a pinch.

-- Ingredient catalog (Q20): created only by entering a dish or an added item, never deleted.
-- Never designated by its identifier in the API, but by its name (D10).
CREATE TABLE ingredients (
    id INTEGER PRIMARY KEY,
    -- Name of the first entry, as displayed.
    name TEXT NOT NULL,
    -- Normalized name (Q29): the identity of the ingredient.
    normalized_name TEXT NOT NULL UNIQUE,
    -- First unit entered with the ingredient; NULL until a line has one (D4).
    default_unit TEXT CHECK (default_unit IN ('g', 'kg', 'ml', 'cl', 'l', 'piece', 'tablespoon', 'teaspoon', 'pinch'))
) STRICT;

-- Dishes (P1, P2): never deleted (Q5b); no author nor history (D23).
CREATE TABLE dishes (
    -- Random, 128 bits in hexadecimal (D10).
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    -- Normalized name: unique in the tribe (Q28).
    normalized_name TEXT NOT NULL UNIQUE,
    -- Hue of the palette, given at creation and never changed (Q19).
    color TEXT NOT NULL,
    -- Reference servings.
    servings INTEGER NOT NULL CHECK (servings BETWEEN 1 AND 50),
    -- Increased on every change; a save made on an outdated revision is refused (Q25).
    revision INTEGER NOT NULL CHECK (revision >= 1)
) STRICT;

-- Lines of a dish, in the order of entry: one per ingredient (Q27), the quantity optional
-- (Q26): amount and unit both present or both absent.
CREATE TABLE dish_ingredients (
    dish_id TEXT NOT NULL REFERENCES dishes (id),
    position INTEGER NOT NULL CHECK (position >= 0),
    ingredient_id INTEGER NOT NULL REFERENCES ingredients (id),
    amount INTEGER CHECK (amount > 0),
    unit TEXT CHECK (unit IN ('g', 'kg', 'ml', 'cl', 'l', 'piece', 'tablespoon', 'teaspoon', 'pinch')),
    CHECK ((amount IS NULL) = (unit IS NULL)),
    PRIMARY KEY (dish_id, position),
    UNIQUE (dish_id, ingredient_id)
) STRICT;

CREATE INDEX dish_ingredients_ingredient ON dish_ingredients (ingredient_id);

-- Triggers of the collections (ADR 0029): dishes for the tables dishes and dish_ingredients,
-- ingredients for the table ingredients. A test checks that each table of a collection has
-- its three triggers.
CREATE TRIGGER dishes_insert AFTER INSERT ON dishes
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'dishes'; END;
CREATE TRIGGER dishes_update AFTER UPDATE ON dishes
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'dishes'; END;
CREATE TRIGGER dishes_delete AFTER DELETE ON dishes
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'dishes'; END;

CREATE TRIGGER dish_ingredients_insert AFTER INSERT ON dish_ingredients
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'dishes'; END;
CREATE TRIGGER dish_ingredients_update AFTER UPDATE ON dish_ingredients
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'dishes'; END;
CREATE TRIGGER dish_ingredients_delete AFTER DELETE ON dish_ingredients
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'dishes'; END;

CREATE TRIGGER ingredients_insert AFTER INSERT ON ingredients
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'ingredients'; END;
CREATE TRIGGER ingredients_update AFTER UPDATE ON ingredients
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'ingredients'; END;
CREATE TRIGGER ingredients_delete AFTER DELETE ON ingredients
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'ingredients'; END;
