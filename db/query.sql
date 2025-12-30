-- name: CreateRepository :one
INSERT INTO repositories (name) VALUES ($1) RETURNING *;

-- name: ListRepositories :many
SELECT * FROM repositories ORDER BY name;
