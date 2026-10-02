ALTER TABLE core.user_notification_channels
    ADD COLUMN IF NOT EXISTS telegram_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS telegram_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS telegram_control BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS uq_notification_channels_telegram_user
    ON core.user_notification_channels (telegram_user_id)
    WHERE telegram_user_id IS NOT NULL;

ALTER TABLE core.notification_deliveries
    ADD COLUMN IF NOT EXISTS buttons JSONB NOT NULL DEFAULT '[]'::jsonb;
