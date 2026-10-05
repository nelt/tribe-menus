-- name: ReplaceLoginCode :exec
INSERT INTO login_codes (member_id, code_hash, request_hash, expires_at, attempts_left)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (member_id) DO UPDATE SET
    code_hash = excluded.code_hash,
    request_hash = excluded.request_hash,
    expires_at = excluded.expires_at,
    attempts_left = excluded.attempts_left;

-- name: LoginCode :one
SELECT * FROM login_codes WHERE member_id = ?;

-- name: SetLoginCodeAttempts :exec
UPDATE login_codes SET attempts_left = ? WHERE member_id = ?;

-- name: DeleteLoginCode :exec
DELETE FROM login_codes WHERE member_id = ?;

-- name: PurgeLoginCodes :execrows
DELETE FROM login_codes WHERE expires_at <= ? OR attempts_left = 0;
