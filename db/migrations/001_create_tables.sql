CREATE EXTENSION IF NOT EXISTS timescaledb;
-- Registry of all known bots

CREATE TABLE IF NOT EXISTS bots (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    languages   TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- Orders-full order lifecycle

CREATE TABLE IF NOT EXISTS orders(
    id          UUID PRIMARY KEY,
    bot_id      TEXT NOT NULL,
    symbol      TEXT NOT NULL,
    side        TEXT NOT NULL,
    type        TEXT NOT NULL,
    price       NUMERIC(20,8) NOT NULL,
    quantity    NUMERIC (20,8) NOT NULL,
    remaining   NUMERIC (20,8) NOT NULL,
    status      TEXT  NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);


CREATE INDEX IF NOT EXISTS idx_orders_bot_id ON orders(bot_id);
CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);


-- trades -- every executed match ( TimescaleDB hypertable)
CREATE TABLE IF NOT EXISTS trades(
    id UUID NOT NULL,
    symbol TEXT NOT NULL,
    buy_order_id UUID NOT NULL,
    sell_order_id UUID NOT NULL,
    buy_bot_id TEXT NOT NULL,
    sell_bot_id TEXT NOT NULL,
    price NUMERIC(20,8) NOT NULL,
    quantity NUMERIC(20,8) NOT NULL,
    notional NUMERIC(20,8) NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id,executed_at)
);

-- Convert to TimescaleDB hypertable partitioned by time
SELECT create_hypertable('trades', 'executed_at', if_not_exists=>TRUE);
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol,executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_buy_bot ON trades(buy_bot_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_sell_bot ON trades(sell_bot_id, executed_at DESC);


-- Price candles- OHCLV aggregated from trades (TsDB hypertable)
-- One row per symbol per minute

CREATE TABLE IF NOT EXISTS price_candles(
    time TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    open NUMERIC(20,8) NOT NULL,
    high NUMERIC(20,8) NOT NULL,
    low  NUMERIC(20,8) NOT NULL,
    close NUMERIC(20,8) NOT NULL,
    volume NUMERIC(20,8) NOT NULL,
    trade_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (time, symbol)
);

SELECT create_hypertable('price_candles', 'time', if_not_exists=>TRUE);

CREATE INDEX IF NOT EXISTS idx_candles_symbol ON price_candles(symbol, time DESC);

-- ml snapshots

CREATE TABLE IF NOT EXISTS ml_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    n_samples INT NOT NULL,
    accuracy FLOAT NOT NULL,
    weights JSONB,
    signal_dist JSONB,
    feature_importance JSONB,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ml_snapshots_symbol ON ml_snapshots(symbol, recorded_at DESC);
-- Optional: Automatically drop trade data older than 30 days
SELECT add_retention_policy('trades', INTERVAL '30 days');