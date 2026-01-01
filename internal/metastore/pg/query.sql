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

-- name: GetRef :one
SELECT * FROM refs WHERE repo_name = $1 AND ref_name = $2;

-- name: ListRefs :many
SELECT * FROM refs WHERE repo_name = $1;

-- name: PutRef :exec
INSERT INTO refs (repo_name, ref_name, type, hash, target)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (repo_name, ref_name)
DO UPDATE SET type = EXCLUDED.type, hash = EXCLUDED.hash, target = EXCLUDED.target;

-- name: DeleteRef :exec
DELETE FROM refs WHERE repo_name = $1 AND ref_name = $2;
