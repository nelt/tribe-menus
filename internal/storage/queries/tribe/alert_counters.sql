-- name: IncrementAlertCounter :exec
INSERT INTO alert_counters (slot, event, n) VALUES (?, ?, 1)
ON CONFLICT (slot, event) DO UPDATE SET n = n + 1;

-- name: AlertCounts :many
SELECT event, CAST(sum(n) AS INTEGER) AS n FROM alert_counters WHERE slot > ? GROUP BY event;

-- name: PurgeAlertCounters :execrows
DELETE FROM alert_counters WHERE slot <= ?;
