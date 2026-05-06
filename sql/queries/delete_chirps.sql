-- name: DeleteChirps :exec
DELETE FROM chirps
WHERE id = $1;