-- Login codes and sessions (ENF-01). Neither a code nor a session token is stored in
-- clear: only their SHA-256 hash.

-- At most one valid login code per member: a new request replaces the previous code.
CREATE TABLE login_codes (
    member_id INTEGER PRIMARY KEY REFERENCES members (id),
    code_hash BLOB NOT NULL,
    expires_at TEXT NOT NULL,
    attempts_left INTEGER NOT NULL CHECK (attempts_left BETWEEN 0 AND 3)
) STRICT;

-- Sessions of the members, one per device (EF-04). Deleted at sign-out or expiry.
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY,
    token_hash BLOB NOT NULL UNIQUE,
    member_id INTEGER NOT NULL REFERENCES members (id),
    opened_at TEXT NOT NULL,
    last_active_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    -- Detected device (EF-04): phone, tablet, computer or empty when unknown; operating
    -- system and browser as detected from the User-Agent header, empty when unknown.
    device_type TEXT NOT NULL,
    os TEXT NOT NULL,
    browser TEXT NOT NULL,
    -- 1 for the installed app, 0 for a browser tab, as told by the client.
    installed_app INTEGER NOT NULL CHECK (installed_app IN (0, 1)),
    -- Session label given by the member (EF-04).
    label TEXT
) STRICT;

CREATE INDEX sessions_member_id ON sessions (member_id);
CREATE INDEX sessions_expires_at ON sessions (expires_at);
