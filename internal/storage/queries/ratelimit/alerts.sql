-- name: IncrementAlertCounter :exec
INSERT INTO alert_counters (slot, event, n) VALUES (?, ?, 1)
ON CONFLICT (slot, event) DO UPDATE SET n = n + 1;

-- name: AlertCounts :many
SELECT event, CAST(sum(n) AS INTEGER) AS n FROM alert_counters WHERE slot > ? GROUP BY event;

-- name: PurgeAlertCounters :execrows
DELETE FROM alert_counters WHERE slot <= ?;

-- name: CountRepeatedRequests :one
-- The hashes of addresses with at least the given number of requests in the window.
SELECT count(*) FROM (
    SELECT email_hash FROM code_requests WHERE at > sqlc.arg(since) GROUP BY email_hash HAVING count(*) >= CAST(sqlc.arg(min_requests) AS INTEGER)
);

-- name: AlertsSent :many
SELECT signal, sent_at FROM alerts_sent;

-- name: SetAlertSent :exec
INSERT INTO alerts_sent (signal, sent_at) VALUES (?, ?)
ON CONFLICT (signal) DO UPDATE SET sent_at = excluded.sent_at;
