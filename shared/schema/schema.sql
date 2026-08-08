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
-- Engine'in verdigi islem kimligi; ayni islem iki kez gelirse ikincisi yazilmaz.
ALTER TABLE trades ADD COLUMN IF NOT EXISTS trade_id VARCHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS idx_trades_trade_id ON trades(trade_id);
CREATE INDEX IF NOT EXISTS idx_trades_buyer ON trades(buyer_id, traded_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_seller ON trades(seller_id, traded_at DESC);

-- Deneme bakiyesinin verildigi an; bos ise henuz verilmemistir.
ALTER TABLE users ADD COLUMN IF NOT EXISTS funded_at TIMESTAMPTZ;

-- Butun tutarlar 10^8 olceklidir (fiyat 10^4 x miktar 10^4), boylece
-- fiyat*miktar bolme ve yuvarlama olmadan saklanir.
CREATE TABLE IF NOT EXISTS balances (
    user_id   VARCHAR(64) NOT NULL,
    asset     VARCHAR(16) NOT NULL,
    available NUMERIC(40, 0) NOT NULL DEFAULT 0 CHECK (available >= 0),
    locked    NUMERIC(40, 0) NOT NULL DEFAULT 0 CHECK (locked >= 0),
    PRIMARY KEY (user_id, asset)
);

-- Acik bir emir icin ayrilmis tutar. Islemler ve iptal bunu azaltir, sifira inince satir silinir.
CREATE TABLE IF NOT EXISTS order_locks (
    order_id  VARCHAR(64) PRIMARY KEY,
    user_id   VARCHAR(64) NOT NULL,
    asset     VARCHAR(16) NOT NULL,
    remaining NUMERIC(40, 0) NOT NULL CHECK (remaining >= 0),
    released  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
