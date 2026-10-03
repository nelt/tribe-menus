-- Dates are UTC timestamps in RFC 3339 text, written by the application.

-- The tribe itself: a single row.
CREATE TABLE tribe (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    name TEXT NOT NULL
) STRICT;

-- Members of the tribe (EF-01 to EF-03, EF-06). A member is never deleted, only revoked.
CREATE TABLE members (
    id INTEGER PRIMARY KEY,
    -- Normalized address (no spaces, lower case). NULL once the member is anonymized (EF-11).
    email TEXT UNIQUE,
    display_name TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'revoked')),
    added_at TEXT NOT NULL,
    -- NULL when added by the admin command.
    added_by INTEGER REFERENCES members (id),
    revoked_at TEXT,
    -- NULL when revoked by the admin command, or not revoked.
    revoked_by INTEGER REFERENCES members (id),
    CHECK ((status = 'revoked') = (revoked_at IS NOT NULL))
) STRICT;

-- Audit log (EF-07): append only.
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY,
    at TEXT NOT NULL,
    -- One of the audit operations of the glossary, in snake case (tribe_initialized…).
    operation TEXT NOT NULL,
    member_id INTEGER NOT NULL REFERENCES members (id),
    -- NULL when the author is the admin command.
    author_id INTEGER REFERENCES members (id),
    -- For an operation on a session (lot B).
    session_id INTEGER,
    detected_device TEXT
) STRICT;

CREATE INDEX audit_log_at ON audit_log (at);
