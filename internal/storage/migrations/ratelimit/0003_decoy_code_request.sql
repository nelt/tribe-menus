-- A decoy code keeps the hash of the token of its request, as a real code does (ENF-01;
-- plan revue-securite, D5): an attempt from another browser gets the same answer for any
-- address. The pending decoy codes, valid 10 minutes at most, are dropped with the table.
DROP TABLE decoy_codes;

CREATE TABLE decoy_codes (
    email_hash BLOB PRIMARY KEY,
    request_hash BLOB NOT NULL,
    expires_at TEXT NOT NULL,
    attempts_left INTEGER NOT NULL CHECK (attempts_left BETWEEN 0 AND 3)
) STRICT;
