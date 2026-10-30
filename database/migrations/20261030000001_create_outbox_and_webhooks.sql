-- migrate:up
CREATE TABLE event_outbox (
    sequence BIGSERIAL PRIMARY KEY,
    id TEXT NOT NULL UNIQUE,
    topic TEXT NOT NULL,
    payload BYTEA NOT NULL,
    correlation_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);
CREATE INDEX event_outbox_pending ON event_outbox(sequence) WHERE published_at IS NULL;

CREATE TABLE webhook_deliveries (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    destination TEXT NOT NULL,
    body BYTEA NOT NULL,
    correlation_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','succeeded','failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    last_status INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enqueue_after TIMESTAMPTZ NOT NULL DEFAULT '-infinity',
    locked_until TIMESTAMPTZ NOT NULL DEFAULT '-infinity',
    lease_token TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    UNIQUE(event_id, destination)
);
CREATE INDEX webhook_deliveries_pending ON webhook_deliveries(next_attempt_at) WHERE status='pending';

-- migrate:down
DROP TABLE webhook_deliveries;
DROP TABLE event_outbox;
