-- Alert counters (ADR 0023; plan production, D7): one row per ten-minute slot and per
-- event, incremented, without hash nor identifier: their size does not depend on the
-- number of requests. The events counted here do not depend on membership; those that
-- suppose a member are counted in the tribe database. Purged once out of the window of an
-- hour.
CREATE TABLE alert_counters (
    slot TEXT NOT NULL,
    event TEXT NOT NULL CHECK (event IN ('email_limit', 'ip_limit', 'decoy_code_exhausted')),
    n INTEGER NOT NULL CHECK (n > 0),
    PRIMARY KEY (slot, event)
) STRICT;

-- The last alert sent, by signal: an alert is not repeated within six hours. One row per
-- signal at most.
CREATE TABLE alerts_sent (
    signal TEXT PRIMARY KEY CHECK (signal IN ('refused_requests', 'exhausted_codes', 'repeated_requests')),
    sent_at TEXT NOT NULL
) STRICT;
