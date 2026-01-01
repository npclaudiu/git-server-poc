-- name: CreateRepository :one
INSERT INTO repositories (name) VALUES ($1) RETURNING *;

-- name: ListRepositories :many
SELECT * FROM repositories ORDER BY name;

-- name: GetRepository :one
SELECT * FROM repositories WHERE name = $1;

-- name: UpdateRepository :one
UPDATE repositories SET name = sqlc.arg(new_name) WHERE name = sqlc.arg(old_name) RETURNING *;

-- name: DeleteRepository :exec
DELETE FROM repositories WHERE name = $1;
