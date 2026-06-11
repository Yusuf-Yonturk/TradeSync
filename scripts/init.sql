CREATE TABLE IF NOT EXISTS events(
    id  BIGSERIAL   PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    sequence BIGINT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_sequence ON events(sequence ASC);
CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_id, sequence ASC);

CREATE TABLE IF NOT EXISTS trades (
    id            BIGSERIAL   PRIMARY KEY,
    buy_order_id  VARCHAR(64) NOT NULL,
    sell_order_id VARCHAR(64) NOT NULL,
    buyer_id      VARCHAR(64) NOT NULL DEFAULT '',
    seller_id     VARCHAR(64) NOT NULL DEFAULT '',
    symbol        VARCHAR(32) NOT NULL,
    price         BIGINT NOT NULL,
    quantity      BIGINT NOT NULL,
    traded_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol, traded_at DESC);

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    display_name  VARCHAR(64),
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);