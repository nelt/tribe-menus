-- Registry of the tribes (ADR 0003, point 2): only what must be known before
-- knowing which tribe a request is for. No personal data.
CREATE TABLE tribes (
    slug TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    -- Random file name of the tribe database, never derived from the slug.
    file TEXT NOT NULL UNIQUE
) STRICT;
