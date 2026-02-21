-- name: CreateWebhook :one
INSERT INTO webhook_events (event_id, type, payload) VALUES ($1, $2, $3)
RETURNING *;

-- name: ClaimNextWebhook :one
UPDATE webhook_events
SET status = 'processing', attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = (
  SELECT id FROM webhook_events
  WHERE status = 'received'
  ORDER BY received_at ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: MarkWebhookDone :one
UPDATE webhook_events
SET status = 'done', processed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: MarkWebhookFailed :one
UPDATE webhook_events
SET status = 'failed', last_error = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
