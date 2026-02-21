CREATE TABLE webhook_events (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "event_id" TEXT NOT NULL UNIQUE,
    "type" TEXT,
    "payload" JSONB NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'received',
    "attempts" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "received_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "processed_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT webhook_events_status_valid
    CHECK (status IN ('received', 'processing', 'done', 'failed')),

  CONSTRAINT webhook_events_attempts_non_negative
    CHECK (attempts >= 0)
);

CREATE INDEX webhook_events_status_idx
  ON webhook_events (status, received_at);
