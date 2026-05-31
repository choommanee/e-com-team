CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subscriptions (
    user_id           TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plan              TEXT NOT NULL DEFAULT 'free',
    status            TEXT NOT NULL DEFAULT 'active',
    ls_subscription   TEXT NOT NULL DEFAULT '',
    period_start      TIMESTAMPTZ NOT NULL DEFAULT now(),
    period_end        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS jobs (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_name TEXT NOT NULL,
    lang         TEXT NOT NULL DEFAULT 'th',
    status       TEXT NOT NULL DEFAULT 'pending',
    result_json  JSONB,
    image_path   TEXT NOT NULL DEFAULT '',
    error        TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS jobs_user_created_idx ON jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS jobs_status_idx ON jobs (status);

CREATE TABLE IF NOT EXISTS usage (
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_start TEXT NOT NULL,
    count        INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, period_start)
);

CREATE TABLE IF NOT EXISTS affiliates (
    user_id      TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code         TEXT NOT NULL UNIQUE,
    status       TEXT NOT NULL DEFAULT 'approved',
    bio          TEXT NOT NULL DEFAULT '',
    niche        TEXT NOT NULL DEFAULT '',
    pitch        TEXT NOT NULL DEFAULT '',
    clicks       INT NOT NULL DEFAULT 0,
    signups      INT NOT NULL DEFAULT 0,
    earnings_thb INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS referrals (
    referred_user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    affiliate_code   TEXT NOT NULL,
    converted        BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS referrals_code_idx ON referrals (affiliate_code);
