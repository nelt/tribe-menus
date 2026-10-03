-- name: InsertMember :one
INSERT INTO members (email, display_name, status, added_at, added_by)
VALUES (?, ?, 'active', ?, ?)
RETURNING id;

-- name: MemberByEmail :one
SELECT * FROM members WHERE email = ?;
