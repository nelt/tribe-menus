-- Rate limit database (ADR 0021): code requests and decoy codes, for every tribe of the
-- instance, existing or not. No address nor IP in clear: an address is hashed with the
-- slug of the tribe, an IP alone (SHA-256). Every row is purged once out of its window,
-- an hour at most.
-- Dates: same format as the tribe databases (2006-01-02T15:04:05.000Z).

-- Accepted code requests (ENF-01).
CREATE TABLE code_requests (
    id INTEGER PRIMARY KEY,
    at TEXT NOT NULL,
    slug TEXT NOT NULL,
    email_hash BLOB NOT NULL,
    ip_hash BLOB NOT NULL
) STRICT;

CREATE INDEX code_requests_at ON code_requests (at);
CREATE INDEX code_requests_slug ON code_requests (slug, at);
CREATE INDEX code_requests_email_hash ON code_requests (email_hash, at);
CREATE INDEX code_requests_ip_hash ON code_requests (ip_hash, at);

-- Decoy codes: for an unknown tribe or an address that is not an active member, a code
-- never sent and that cannot succeed, with the deadline and attempts of a real one.
CREATE TABLE decoy_codes (
    email_hash BLOB PRIMARY KEY,
    expires_at TEXT NOT NULL,
    attempts_left INTEGER NOT NULL CHECK (attempts_left BETWEEN 0 AND 3)
) STRICT;
