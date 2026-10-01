CREATE TABLE IF NOT EXISTS core.pending_registrations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    first_name    TEXT NOT NULL DEFAULT '',
    last_name     TEXT NOT NULL DEFAULT '',
    referral_code TEXT NOT NULL DEFAULT '',
    consents      TEXT[] NOT NULL DEFAULT '{}',
    ip            TEXT NOT NULL DEFAULT '',
    user_agent    TEXT NOT NULL DEFAULT '',
    token_hash    TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_registrations_email
    ON core.pending_registrations (email);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_registrations_token
    ON core.pending_registrations (token_hash);
