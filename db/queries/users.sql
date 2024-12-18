-- name: CreateUser :one
INSERT INTO users (id, username, fullname, email, password_hash)
VALUES (uuid_generate_v4(), $1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: UpdateUser :one
UPDATE users
SET username = $1, fullname = $2, email = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $4
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
