-- The limit by tribe moves to the tribe databases (plan revue-securite, D2): the slug of
-- the requests is no longer needed here.
DROP INDEX code_requests_slug;
ALTER TABLE code_requests DROP COLUMN slug;
