-- +goose Up

-- ---------- users & personal data ----------
CREATE TABLE users (
    id          BIGSERIAL PRIMARY KEY,
    email       TEXT UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watchlist (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code        VARCHAR(10) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, code)
);

CREATE TABLE holdings (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code        VARCHAR(10) NOT NULL,
    shares      BIGINT NOT NULL CHECK (shares >= 0),
    avg_cost    DOUBLE PRECISION NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, code)
);

-- ---------- market data ----------
-- Delisted stocks stay in this table (delisted_on set) to avoid survivorship bias.
CREATE TABLE stocks (
    code         VARCHAR(10) PRIMARY KEY,
    name         TEXT NOT NULL,
    exchange     VARCHAR(8) NOT NULL,          -- TWSE | TPEx
    industry     TEXT,
    capital      BIGINT,                       -- paid-in capital, NT$
    listed_on    DATE,
    delisted_on  DATE,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE daily_prices (
    code        VARCHAR(10) NOT NULL,
    trade_date  DATE NOT NULL,
    open        DOUBLE PRECISION NOT NULL,
    high        DOUBLE PRECISION NOT NULL,
    low         DOUBLE PRECISION NOT NULL,
    close       DOUBLE PRECISION NOT NULL,
    volume      BIGINT NOT NULL,               -- shares
    value       BIGINT NOT NULL,               -- NT$
    PRIMARY KEY (code, trade_date)
);
CREATE INDEX daily_prices_date_idx ON daily_prices (trade_date);

-- announced_on is when the figure became public; backtests must key on it.
CREATE TABLE monthly_revenue (
    code          VARCHAR(10) NOT NULL,
    year_month    DATE NOT NULL,               -- first day of the reported month
    announced_on  DATE NOT NULL,
    revenue       BIGINT NOT NULL,             -- NT$ thousand
    yoy           DOUBLE PRECISION NOT NULL DEFAULT 0,
    mom           DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (code, year_month)
);
CREATE INDEX monthly_revenue_announced_idx ON monthly_revenue (announced_on);

CREATE TABLE institutional (
    code         VARCHAR(10) NOT NULL,
    trade_date   DATE NOT NULL,
    foreign_net  BIGINT NOT NULL DEFAULT 0,    -- shares
    trust_net    BIGINT NOT NULL DEFAULT 0,
    dealer_net   BIGINT NOT NULL DEFAULT 0,
    total_net    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (code, trade_date)
);

CREATE TABLE valuations (
    code            VARCHAR(10) NOT NULL,
    trade_date      DATE NOT NULL,
    pe              DOUBLE PRECISION NOT NULL DEFAULT 0,
    dividend_yield  DOUBLE PRECISION NOT NULL DEFAULT 0,
    pb              DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (code, trade_date)
);

-- Used to compute back-adjusted prices.
CREATE TABLE corporate_actions (
    code           VARCHAR(10) NOT NULL,
    ex_date        DATE NOT NULL,
    cash_dividend  DOUBLE PRECISION NOT NULL DEFAULT 0,
    stock_ratio    DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (code, ex_date)
);

-- Tracks which days the historical backfill has finished, so it can resume.
CREATE TABLE backfill_progress (
    source      VARCHAR(16) NOT NULL,          -- twse | tpex
    trade_date  DATE NOT NULL,
    rows        INT NOT NULL,
    done_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source, trade_date)
);

-- ---------- screener & backtests ----------
CREATE TABLE strategies (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    definition  JSONB NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE screen_results (
    screen_date  DATE NOT NULL,
    strategy_id  BIGINT NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    code         VARCHAR(10) NOT NULL,
    name         TEXT NOT NULL,
    close        DOUBLE PRECISION NOT NULL,
    matched      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (screen_date, strategy_id, code)
);

CREATE TABLE backtests (
    id           BIGSERIAL PRIMARY KEY,
    strategy_id  BIGINT REFERENCES strategies(id) ON DELETE SET NULL,
    config       JSONB NOT NULL,
    status       VARCHAR(16) NOT NULL,         -- running | done | failed
    result       JSONB,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ
);

-- Use TimescaleDB for daily_prices when the extension is available
-- (plain PostgreSQL still works without it).
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'timescaledb') THEN
        CREATE EXTENSION IF NOT EXISTS timescaledb;
        PERFORM create_hypertable('daily_prices', 'trade_date',
            chunk_time_interval => INTERVAL '1 year', if_not_exists => TRUE, migrate_data => TRUE);
    END IF;
END
$$;
-- +goose StatementEnd

-- ---------- seed data ----------
-- Development account (id = 1); remove once real login exists.
INSERT INTO users (id, email) VALUES (1, 'dev@local');
SELECT setval('users_id_seq', 1);

-- Placeholder strategy until the real conditions are decided.
INSERT INTO strategies (name, definition) VALUES (
    '放量突破（範例）',
    '{
      "name": "放量突破（範例）",
      "universe": {"common_stock_only": true, "min_price": 10, "min_avg_volume_20": 200000},
      "rule": {"op": "and", "rules": [
        {"label": "創 60 日新高", "left": {"indicator": "close"}, "cmp": ">", "right": {"indicator": "highest_close", "period": 60}},
        {"label": "量增 2 倍", "left": {"indicator": "volume_ratio", "period": 20}, "cmp": ">=", "right": {"value": 2}},
        {"label": "站上月線", "left": {"indicator": "close"}, "cmp": ">", "right": {"indicator": "sma", "period": 20}}
      ]}
    }'
);

-- +goose Down
DROP TABLE IF EXISTS backtests, screen_results, strategies, backfill_progress,
    corporate_actions, valuations, institutional, monthly_revenue, daily_prices,
    stocks, holdings, watchlist, users;
