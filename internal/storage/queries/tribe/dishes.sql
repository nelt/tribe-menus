-- name: CollectionVersion :one
SELECT version FROM collection_versions WHERE name = ?;

-- name: SetCollectionVersion :exec
UPDATE collection_versions SET version = ? WHERE name = ?;

-- name: DishesAfter :many
SELECT id, name, color, servings, revision FROM dishes
WHERE id > ? ORDER BY id LIMIT ?;

-- name: DishByID :one
SELECT id, name, color, servings, revision FROM dishes WHERE id = ?;

-- name: DishIDByNormalizedName :one
SELECT id FROM dishes WHERE normalized_name = ?;

-- name: DishLines :many
SELECT l.dish_id, l.position, l.amount, l.unit, i.name, i.default_unit
FROM dish_ingredients l JOIN ingredients i ON i.id = l.ingredient_id
WHERE l.dish_id >= sqlc.arg(first) AND l.dish_id <= sqlc.arg(last)
ORDER BY l.dish_id, l.position;

-- name: ColorCounts :many
SELECT color, count(*) AS n FROM dishes GROUP BY color;

-- name: InsertDish :exec
INSERT INTO dishes (id, name, normalized_name, color, servings, revision)
VALUES (?, ?, ?, ?, ?, 1);

-- name: UpdateDish :exec
UPDATE dishes SET name = ?, normalized_name = ?, servings = ?, revision = revision + 1
WHERE id = ?;

-- name: DeleteDishLines :exec
DELETE FROM dish_ingredients WHERE dish_id = ?;

-- name: InsertDishLine :exec
INSERT INTO dish_ingredients (dish_id, position, ingredient_id, amount, unit)
VALUES (?, ?, ?, ?, ?);

-- name: IngredientsAfter :many
SELECT id, name, default_unit FROM ingredients
WHERE id > ? ORDER BY id LIMIT ?;

-- name: IngredientByNormalizedName :one
SELECT id, name, default_unit FROM ingredients WHERE normalized_name = ?;

-- name: InsertIngredient :one
INSERT INTO ingredients (name, normalized_name, default_unit) VALUES (?, ?, ?)
RETURNING id;

-- name: SetDefaultUnit :exec
UPDATE ingredients SET default_unit = ? WHERE id = ? AND default_unit IS NULL;
