CREATE TABLE IF NOT EXISTS events(
    id  BIGSERIAL   PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    sequence BIGINT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_sequence ON events(sequence ASC);
CREATE INDEX idx_events_aggregate ON events(aggregate_id, sequence ASC);