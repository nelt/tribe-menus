-- Code requests addressed to the active members of the tribe, for the limit by tribe
-- (ENF-01; plan revue-securite, D2). Only their time: the requests for other addresses
-- are counted by the rate limit database alone, which never learns who is a member.
-- Purged once out of the window of the limit.
CREATE TABLE code_requests (
    id INTEGER PRIMARY KEY,
    at TEXT NOT NULL
) STRICT;

CREATE INDEX code_requests_at ON code_requests (at);
