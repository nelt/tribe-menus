-- name: TribeFile :one
SELECT file FROM tribes WHERE slug = ?;

-- name: CountTribesWithSlug :one
SELECT count(*) FROM tribes WHERE slug = ?;

-- name: InsertTribe :exec
INSERT INTO tribes (slug, name, file) VALUES (?, ?, ?);

-- name: TribeFiles :many
SELECT file FROM tribes ORDER BY slug;

-- name: TribeSlugs :many
SELECT slug FROM tribes ORDER BY slug;
