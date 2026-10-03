-- name: InsertTribe :exec
INSERT INTO tribe (id, name) VALUES (1, ?);

-- name: TribeName :one
SELECT name FROM tribe WHERE id = 1;
