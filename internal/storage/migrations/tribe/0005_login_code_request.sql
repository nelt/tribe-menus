-- A login code is entered only from the browser that requested it (ENF-01; plan
-- revue-securite, D5): request_hash is the SHA-256 hash of the token of the cookie set by
-- the request. The pending codes, valid 10 minutes at most, are dropped with the table.
DROP TABLE login_codes;

CREATE TABLE login_codes (
    member_id INTEGER PRIMARY KEY REFERENCES members (id),
    code_hash BLOB NOT NULL,
    request_hash BLOB NOT NULL,
    expires_at TEXT NOT NULL,
    attempts_left INTEGER NOT NULL CHECK (attempts_left BETWEEN 0 AND 3)
) STRICT;
