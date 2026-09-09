CREATE TABLE IF NOT EXISTS core.notification_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,

    notification_id UUID REFERENCES core.notifications(id) ON DELETE CASCADE,

    kind            TEXT NOT NULL,
    channel         TEXT NOT NULL CHECK (channel IN ('email', 'telegram', 'discord')),

    target          TEXT NOT NULL,

    subject         TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL DEFAULT '',
    action_label    TEXT NOT NULL DEFAULT '',
    action_href     TEXT NOT NULL DEFAULT '',

    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued', 'sent', 'failed')),
    attempts        INT NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',

    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_queue
    ON core.notification_deliveries (next_attempt_at, id)
    WHERE status = 'queued';

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user
    ON core.notification_deliveries (tenant_id, user_id, created_at DESC);

ALTER TABLE core.notifications
    ADD COLUMN IF NOT EXISTS dedupe_key TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_notifications_dedupe
    ON core.notifications (tenant_id, user_id, dedupe_key)
    WHERE dedupe_key <> '';

ALTER TABLE core.notifications
    ADD COLUMN IF NOT EXISTS action_label TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS action_href  TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_notifications_feed
    ON core.notifications (tenant_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread
    ON core.notifications (tenant_id, user_id)
    WHERE read_at IS NULL;
