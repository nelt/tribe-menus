-- Shopping lists (plan courses, lot B).

INSERT INTO collection_versions (name, version) VALUES ('shopping_lists', 0);

-- Shopping lists (C1, C2, C12): at most 20 per tribe (D12), checked by the code. No author
-- nor date (D22).
CREATE TABLE shopping_lists (
    -- Random, 128 bits in hexadecimal (D10).
    id TEXT PRIMARY KEY,
    -- Name as entered, its spaces reduced; not unique (Q17).
    name TEXT NOT NULL,
    -- Rank of creation, increasing: orders the lists after the main one (C2).
    rank INTEGER NOT NULL CHECK (rank >= 1),
    -- 1 for the main list of the tribe, 0 otherwise: exactly one (Q23), at most one by the
    -- index below, at least one by the code, which never takes the role away without giving it.
    main INTEGER NOT NULL CHECK (main IN (0, 1))
) STRICT;

CREATE UNIQUE INDEX shopping_lists_main ON shopping_lists (main) WHERE main = 1;

-- Items of a list (C3 to C6): at most 300 per list (D12), checked by the code. A manual item
-- is entered by hand, without source; a computed item has at least one source (lot C).
CREATE TABLE shopping_items (
    -- Random, 128 bits in hexadecimal (D10).
    id TEXT PRIMARY KEY,
    list_id TEXT NOT NULL REFERENCES shopping_lists (id),
    ingredient_id INTEGER NOT NULL REFERENCES ingredients (id),
    -- Optional quantity, in the base unit of the family (ADR 0003, point 7), with the unit
    -- it is displayed in: both present or both absent (Q36).
    amount INTEGER CHECK (amount > 0),
    unit TEXT CHECK (unit IN ('g', 'kg', 'ml', 'cl', 'l', 'piece', 'tablespoon', 'teaspoon', 'pinch')),
    checked INTEGER NOT NULL CHECK (checked IN (0, 1)),
    manual INTEGER NOT NULL CHECK (manual IN (0, 1)),
    CHECK ((amount IS NULL) = (unit IS NULL))
) STRICT;

CREATE INDEX shopping_items_list ON shopping_items (list_id);

CREATE INDEX shopping_items_ingredient ON shopping_items (ingredient_id);

-- Sources of a computed item (C5): what a served dish brought to it, at the time of the
-- computation. No foreign key to served_dishes: a served dish removed from the meal plan
-- leaves its sources readable until the recomputation (D5).
CREATE TABLE shopping_item_sources (
    item_id TEXT NOT NULL REFERENCES shopping_items (id),
    served_dish_id TEXT NOT NULL,
    dish_id TEXT NOT NULL REFERENCES dishes (id),
    date TEXT NOT NULL CHECK (date BETWEEN '2000-01-01' AND '2099-12-31' AND date(date) IS date),
    mealtime TEXT NOT NULL CHECK (mealtime IN ('lunch', 'dinner')),
    servings INTEGER NOT NULL CHECK (servings BETWEEN 1 AND 50),
    -- Contributed quantity, absent for an ingredient without quantity (D3).
    amount INTEGER CHECK (amount > 0),
    unit TEXT CHECK (unit IN ('g', 'kg', 'ml', 'cl', 'l', 'piece', 'tablespoon', 'teaspoon', 'pinch')),
    CHECK ((amount IS NULL) = (unit IS NULL)),
    -- A served dish brings at most once to an item: a dish has each ingredient once (Q27).
    PRIMARY KEY (item_id, served_dish_id)
) STRICT;

-- Served dishes received by a list (Q22): never removed in V1, even when their items are
-- gone, nor when the served dish is removed from the meal plan (D5): no foreign key to
-- served_dishes. Its servings, the reference servings of the dish and its lines at the time
-- of the computation tell courses-suite whether the list is outdated (C8).
CREATE TABLE shopping_received (
    list_id TEXT NOT NULL REFERENCES shopping_lists (id),
    served_dish_id TEXT NOT NULL,
    dish_id TEXT NOT NULL REFERENCES dishes (id),
    servings INTEGER NOT NULL CHECK (servings BETWEEN 1 AND 50),
    dish_servings INTEGER NOT NULL CHECK (dish_servings BETWEEN 1 AND 50),
    PRIMARY KEY (list_id, served_dish_id)
) STRICT;

-- Lines of the dish of a received served dish, at the time of the computation (D5).
CREATE TABLE shopping_received_lines (
    list_id TEXT NOT NULL,
    served_dish_id TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    ingredient_id INTEGER NOT NULL REFERENCES ingredients (id),
    amount INTEGER CHECK (amount > 0),
    unit TEXT CHECK (unit IN ('g', 'kg', 'ml', 'cl', 'l', 'piece', 'tablespoon', 'teaspoon', 'pinch')),
    CHECK ((amount IS NULL) = (unit IS NULL)),
    PRIMARY KEY (list_id, served_dish_id, position),
    FOREIGN KEY (list_id, served_dish_id) REFERENCES shopping_received (list_id, served_dish_id)
) STRICT;

-- The main list "Courses" of every tribe, existing or created later (C1, D14): identifier in
-- the format of the others, as randomID writes it.
INSERT INTO shopping_lists (id, name, rank, main) VALUES (lower(hex(randomblob(16))), 'Courses', 1, 1);

-- Triggers of the collection shopping_lists, for its five tables (ADR 0029).
CREATE TRIGGER shopping_lists_insert AFTER INSERT ON shopping_lists
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_lists_update AFTER UPDATE ON shopping_lists
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_lists_delete AFTER DELETE ON shopping_lists
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;

CREATE TRIGGER shopping_items_insert AFTER INSERT ON shopping_items
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_items_update AFTER UPDATE ON shopping_items
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_items_delete AFTER DELETE ON shopping_items
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;

CREATE TRIGGER shopping_item_sources_insert AFTER INSERT ON shopping_item_sources
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_item_sources_update AFTER UPDATE ON shopping_item_sources
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_item_sources_delete AFTER DELETE ON shopping_item_sources
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;

CREATE TRIGGER shopping_received_insert AFTER INSERT ON shopping_received
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_received_update AFTER UPDATE ON shopping_received
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_received_delete AFTER DELETE ON shopping_received
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;

CREATE TRIGGER shopping_received_lines_insert AFTER INSERT ON shopping_received_lines
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_received_lines_update AFTER UPDATE ON shopping_received_lines
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
CREATE TRIGGER shopping_received_lines_delete AFTER DELETE ON shopping_received_lines
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'shopping_lists'; END;
