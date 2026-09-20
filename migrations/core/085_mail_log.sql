CREATE TABLE IF NOT EXISTS core.mail_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template    TEXT NOT NULL DEFAULT '',
    to_address  TEXT NOT NULL,
    subject     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    error       TEXT NOT NULL DEFAULT '',
    mailer      TEXT NOT NULL DEFAULT '',
    user_id     UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mail_log_created ON core.mail_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mail_log_status ON core.mail_log (status, created_at DESC);
