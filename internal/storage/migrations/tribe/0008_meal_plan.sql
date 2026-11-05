-- Meal plan (plan planning, lot B).

INSERT INTO collection_versions (name, version) VALUES ('meal_plan', 0);

-- Served dishes (R2 to R5): a dish of the library in a meal, a date and a mealtime. A meal
-- is not stored: it exists as soon as it is named, empty until a dish is served in it. No
-- author nor date of change (D18), no revision (D3).
CREATE TABLE served_dishes (
    -- Random, 128 bits in hexadecimal (D10). A dish removed then served again is a new
    -- served dish (D2).
    id TEXT PRIMARY KEY,
    -- A real date from 2000-01-01 to 2099-12-31, without time zone (D4, Q35).
    date TEXT NOT NULL CHECK (date BETWEEN '2000-01-01' AND '2099-12-31' AND date(date) IS date),
    mealtime TEXT NOT NULL CHECK (mealtime IN ('lunch', 'dinner')),
    dish_id TEXT NOT NULL REFERENCES dishes (id),
    servings INTEGER NOT NULL CHECK (servings BETWEEN 1 AND 50),
    -- Rank of addition in the tribe, increasing: orders the dishes of a meal (D11).
    rank INTEGER NOT NULL CHECK (rank >= 1)
) STRICT;

-- A dish once in a meal (Q21); also the index of the dishes of a meal.
CREATE UNIQUE INDEX served_dishes_meal ON served_dishes (date, mealtime, dish_id);

CREATE INDEX served_dishes_dish ON served_dishes (dish_id);

-- Triggers of the collection meal_plan (ADR 0029).
CREATE TRIGGER served_dishes_insert AFTER INSERT ON served_dishes
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'meal_plan'; END;
CREATE TRIGGER served_dishes_update AFTER UPDATE ON served_dishes
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'meal_plan'; END;
CREATE TRIGGER served_dishes_delete AFTER DELETE ON served_dishes
BEGIN UPDATE collection_versions SET version = version + 1 WHERE name = 'meal_plan'; END;
