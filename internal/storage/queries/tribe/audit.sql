-- name: InsertAuditEntry :exec
INSERT INTO audit_log (at, operation, member_id, author_id, session_id, detected_device)
VALUES (?, ?, ?, ?, ?, ?);

-- name: AuditLog :many
SELECT audit_log.*, members.email AS member_email
FROM audit_log JOIN members ON members.id = audit_log.member_id
ORDER BY audit_log.id;
