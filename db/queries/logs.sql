-- name: CreateLog :one
INSERT INTO llm_logs (prompt, response, latency_ms)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRecentLogs :many
SELECT * FROM llm_logs 
ORDER BY created_at DESC 
LIMIT $1;