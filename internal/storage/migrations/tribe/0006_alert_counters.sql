-- Alert counters of the events that suppose a member (ADR 0023; plan production, D7):
-- counted in the rate limit database, they would tell which hashes of addresses are
-- members'. One row per ten-minute slot and per event, incremented, without identifier.
-- Purged once out of the window of an hour.
CREATE TABLE alert_counters (
    slot TEXT NOT NULL,
    event TEXT NOT NULL CHECK (event IN ('tribe_limit', 'login_code_exhausted')),
    n INTEGER NOT NULL CHECK (n > 0),
    PRIMARY KEY (slot, event)
) STRICT;
