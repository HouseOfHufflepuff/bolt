-- +goose Up

CREATE TABLE IF NOT EXISTS assets (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol        TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL,
    decimals      INT         NOT NULL DEFAULT 8,
    asset_id      BYTEA,
    genesis_point TEXT,
    total_supply  BIGINT      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mint_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_symbol TEXT        NOT NULL,
    amount       BIGINT      NOT NULL,
    operator     TEXT        NOT NULL,
    batch_key    BYTEA,
    asset_proof  BYTEA,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS burn_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_symbol TEXT        NOT NULL,
    amount       BIGINT      NOT NULL,
    operator     TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS price_feed (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_symbol TEXT        NOT NULL,
    price_usd    FLOAT8      NOT NULL,
    source       TEXT        NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_price_feed_symbol_time ON price_feed (asset_symbol, recorded_at DESC);

CREATE TABLE IF NOT EXISTS reserve_snapshots (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_symbol   TEXT        NOT NULL,
    reserve_amount BIGINT      NOT NULL,
    supply_amount  BIGINT      NOT NULL,
    ratio          FLOAT8      NOT NULL,
    merkle_root    BYTEA,
    signature      BYTEA,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rebalance_log (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    triggered_by   TEXT        NOT NULL,
    weights_before JSONB       NOT NULL,
    weights_after  JSONB       NOT NULL,
    trade_queue    JSONB       NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'pending',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at   TIMESTAMPTZ
);

-- +goose Down

DROP TABLE IF EXISTS rebalance_log;
DROP TABLE IF EXISTS reserve_snapshots;
DROP TABLE IF EXISTS price_feed;
DROP TABLE IF EXISTS burn_events;
DROP TABLE IF EXISTS mint_events;
DROP TABLE IF EXISTS assets;
