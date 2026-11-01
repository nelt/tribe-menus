-- Rollback trial (plan recette, step 32): a migration without effect on the schema, only
-- to raise the version of the tribe databases. Deployed in recette, never merged: the
-- rollback must then ask for the restoration of the data (D7).
CREATE TABLE rollback_trial (id INTEGER PRIMARY KEY) STRICT;
DROP TABLE rollback_trial;
