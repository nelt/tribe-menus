-- name: InsertTribe :exec
INSERT INTO tribe (id, name) VALUES (1, ?);

-- name: TribeName :one
SELECT name FROM tribe WHERE id = 1;

-- name: TribeSize :one
SELECT size FROM tribe WHERE id = 1;

-- name: SetTribeSize :exec
UPDATE tribe SET size = ? WHERE id = 1;
