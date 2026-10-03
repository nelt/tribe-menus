-- name: InsertCodeRequest :exec
INSERT INTO code_requests (at, slug, email_hash, ip_hash) VALUES (?, ?, ?, ?);

-- name: CountCodeRequestsByEmail :one
SELECT count(*) FROM code_requests WHERE email_hash = ? AND at > ?;

-- name: CountCodeRequestsByIP :one
SELECT count(*) FROM code_requests WHERE ip_hash = ? AND at > ?;

-- name: CountCodeRequestsByTribe :one
SELECT count(*) FROM code_requests WHERE slug = ? AND at > ?;

-- name: PurgeCodeRequests :execrows
DELETE FROM code_requests WHERE at <= ?;
