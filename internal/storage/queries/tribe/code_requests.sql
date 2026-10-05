-- name: CountCodeRequests :one
SELECT count(*) FROM code_requests WHERE at > ?;

-- name: InsertCodeRequest :exec
INSERT INTO code_requests (at) VALUES (?);

-- name: PurgeCodeRequests :execrows
DELETE FROM code_requests WHERE at <= ?;
