-- name: InsertMember :one
INSERT INTO members (email, display_name, status, added_at, added_by)
VALUES (?, ?, 'active', ?, ?)
RETURNING id;

-- name: MemberByEmail :one
SELECT * FROM members WHERE email = ?;

-- name: MemberByID :one
SELECT * FROM members WHERE id = ?;

-- name: RevokeMember :execrows
UPDATE members SET status = 'revoked', revoked_at = ?, revoked_by = ?
WHERE id = ? AND status = 'active';
