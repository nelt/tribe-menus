-- The tribe itself: a single row.
CREATE TABLE tribe (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    name TEXT NOT NULL
) STRICT;
