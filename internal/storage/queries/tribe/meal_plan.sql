-- name: ServedDishesAfter :many
SELECT id, date, mealtime, dish_id, servings, rank FROM served_dishes
WHERE id > ? ORDER BY id LIMIT ?;

-- name: MealServedDishes :many
SELECT id, date, mealtime, dish_id, servings, rank FROM served_dishes
WHERE date = ? AND mealtime = ? ORDER BY rank;

-- name: DishExists :one
SELECT EXISTS (SELECT 1 FROM dishes WHERE id = ?);

-- name: NextServedDishRank :one
SELECT CAST(coalesce(max(rank), 0) + 1 AS INTEGER) FROM served_dishes;

-- name: InsertServedDish :exec
INSERT INTO served_dishes (id, date, mealtime, dish_id, servings, rank)
VALUES (?, ?, ?, ?, ?, ?);

-- name: SetServedDishServings :one
UPDATE served_dishes SET servings = ? WHERE id = ?
RETURNING id, date, mealtime, dish_id, servings, rank;

-- name: DeleteServedDish :execrows
DELETE FROM served_dishes WHERE id = ?;
