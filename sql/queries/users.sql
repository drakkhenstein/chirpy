-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    now(),
    now(),
    $1,
    $2
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET
    updated_at = now(),
    email = $2,
    hashed_password = $3
WHERE id = $1
RETURNING *;

-- name: UpgradeToChirpyRed :one
UPDATE users
SET is_chirpy_red = true,
updated_at = now()
WHERE id = $1
RETURNING *;