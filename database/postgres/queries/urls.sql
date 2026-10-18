-- name: GetURLByID :one
SELECT target_url
FROM urls
WHERE id = sqlc.arg(id);

-- name: CreateURL :one
INSERT INTO urls (target_url)
VALUES (sqlc.arg(target_url))
RETURNING id;
