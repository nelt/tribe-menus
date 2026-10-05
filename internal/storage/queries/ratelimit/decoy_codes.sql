-- name: ReplaceDecoyCode :exec
INSERT INTO decoy_codes (email_hash, request_hash, expires_at, attempts_left)
VALUES (?, ?, ?, ?)
ON CONFLICT (email_hash) DO UPDATE SET
    request_hash = excluded.request_hash,
    expires_at = excluded.expires_at,
    attempts_left = excluded.attempts_left;

-- name: DecoyCode :one
SELECT * FROM decoy_codes WHERE email_hash = ?;

-- name: SetDecoyCodeAttempts :exec
UPDATE decoy_codes SET attempts_left = ? WHERE email_hash = ?;

-- name: DeleteDecoyCode :exec
DELETE FROM decoy_codes WHERE email_hash = ?;

-- name: PurgeDecoyCodes :execrows
DELETE FROM decoy_codes WHERE expires_at <= ? OR attempts_left = 0;
