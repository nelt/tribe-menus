-- name: ShoppingListsAfter :many
SELECT id, name, rank, main FROM shopping_lists
WHERE id > ? ORDER BY id LIMIT ?;

-- name: ShoppingListByID :one
SELECT id, name, rank, main FROM shopping_lists WHERE id = ?;

-- name: ShoppingItemsOfLists :many
SELECT i.id, i.list_id, i.amount, i.unit, i.checked, i.manual, g.name
FROM shopping_items i JOIN ingredients g ON g.id = i.ingredient_id
WHERE i.list_id >= sqlc.arg(first) AND i.list_id <= sqlc.arg(last)
ORDER BY i.list_id, i.id;

-- name: ShoppingItemSourcesOfLists :many
SELECT s.item_id, s.served_dish_id, s.dish_id, s.date, s.mealtime, s.servings, s.amount, s.unit
FROM shopping_item_sources s JOIN shopping_items i ON i.id = s.item_id
WHERE i.list_id >= sqlc.arg(first) AND i.list_id <= sqlc.arg(last)
ORDER BY s.item_id, s.date, CASE s.mealtime WHEN 'lunch' THEN 0 ELSE 1 END, s.served_dish_id;

-- name: ShoppingReceivedOfLists :many
SELECT list_id, served_dish_id FROM shopping_received
WHERE list_id >= sqlc.arg(first) AND list_id <= sqlc.arg(last)
ORDER BY list_id, served_dish_id;

-- name: ShoppingListCount :one
SELECT count(*) FROM shopping_lists;

-- name: NextShoppingListRank :one
SELECT CAST(coalesce(max(rank), 0) + 1 AS INTEGER) FROM shopping_lists;

-- name: InsertShoppingList :exec
INSERT INTO shopping_lists (id, name, rank, main) VALUES (?, ?, ?, 0);

-- name: RenameShoppingList :exec
UPDATE shopping_lists SET name = ? WHERE id = ?;

-- name: ClearMainShoppingList :exec
UPDATE shopping_lists SET main = 0 WHERE main = 1;

-- name: SetMainShoppingList :exec
UPDATE shopping_lists SET main = 1 WHERE id = ?;

-- name: DeleteShoppingReceivedLinesOfList :exec
DELETE FROM shopping_received_lines WHERE list_id = ?;

-- name: DeleteShoppingReceivedOfList :exec
DELETE FROM shopping_received WHERE list_id = ?;

-- name: DeleteShoppingList :exec
DELETE FROM shopping_lists WHERE id = ?;

-- name: InsertShoppingItem :exec
INSERT INTO shopping_items (id, list_id, ingredient_id, amount, unit, checked, manual)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: SetShoppingItemChecked :exec
UPDATE shopping_items SET checked = ? WHERE id = ?;

-- name: DeleteShoppingItemSources :exec
DELETE FROM shopping_item_sources WHERE item_id = ?;

-- name: DeleteShoppingItem :exec
DELETE FROM shopping_items WHERE id = ?;

-- name: ServedDishesBetween :many
SELECT s.id, s.date, s.mealtime, s.dish_id, s.servings, d.servings AS dish_servings
FROM served_dishes s JOIN dishes d ON d.id = s.dish_id
WHERE s.date >= sqlc.arg(first) AND s.date <= sqlc.arg(last)
ORDER BY s.date, CASE s.mealtime WHEN 'lunch' THEN 0 ELSE 1 END, s.rank;

-- name: DishLinesServedBetween :many
SELECT l.dish_id, l.position, l.ingredient_id, l.amount, l.unit, i.name
FROM dish_ingredients l JOIN ingredients i ON i.id = l.ingredient_id
WHERE l.dish_id IN (
    SELECT dish_id FROM served_dishes WHERE date >= sqlc.arg(first) AND date <= sqlc.arg(last)
)
ORDER BY l.dish_id, l.position;

-- name: UpdateShoppingItem :exec
UPDATE shopping_items SET amount = ?, unit = ?, checked = ? WHERE id = ?;

-- name: InsertShoppingItemSource :exec
INSERT INTO shopping_item_sources (item_id, served_dish_id, dish_id, date, mealtime, servings, amount, unit)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertShoppingReceived :exec
INSERT INTO shopping_received (list_id, served_dish_id, dish_id, servings, dish_servings)
VALUES (?, ?, ?, ?, ?);

-- name: InsertShoppingReceivedLine :exec
INSERT INTO shopping_received_lines (list_id, served_dish_id, position, ingredient_id, amount, unit)
VALUES (?, ?, ?, ?, ?, ?);
