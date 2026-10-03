-- name: InsertSession :one
INSERT INTO sessions (token_hash, member_id, opened_at, last_active_at, expires_at, device_type, os, browser, installed_app)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: SessionByTokenHash :one
SELECT * FROM sessions WHERE token_hash = ? AND expires_at > ?;

-- name: SessionByID :one
SELECT * FROM sessions WHERE id = ?;

-- name: TouchSession :exec
UPDATE sessions SET last_active_at = ?, expires_at = ? WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: MemberSessions :many
SELECT * FROM sessions WHERE member_id = ? ORDER BY id;

-- name: PurgeSessions :execrows
DELETE FROM sessions WHERE expires_at <= ?;
