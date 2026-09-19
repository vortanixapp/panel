ALTER TABLE core.user_profiles ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT '';
ALTER TABLE core.user_profiles ADD COLUMN IF NOT EXISTS preferences JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE core.user_profiles ADD COLUMN IF NOT EXISTS avatar_version INTEGER NOT NULL DEFAULT 0;

ALTER TABLE core.users ADD COLUMN IF NOT EXISTS password_set BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE core.users ADD COLUMN IF NOT EXISTS pending_email TEXT;
ALTER TABLE core.users ADD COLUMN IF NOT EXISTS pending_email_hash TEXT;
ALTER TABLE core.users ADD COLUMN IF NOT EXISTS pending_email_expires TIMESTAMPTZ;
ALTER TABLE core.users ADD COLUMN IF NOT EXISTS referrer_id UUID REFERENCES core.users(id) ON DELETE SET NULL;
ALTER TABLE core.users ADD COLUMN IF NOT EXISTS referral_code TEXT;
ALTER TABLE core.users ADD COLUMN IF NOT EXISTS referred_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_referral_code ON core.users (referral_code) WHERE referral_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_referrer ON core.users (referrer_id) WHERE referrer_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_pending_email_hash ON core.users (pending_email_hash) WHERE pending_email_hash IS NOT NULL;

UPDATE core.users SET password_set = false
WHERE password_set AND email LIKE 'tg\_%@telegram.local' ESCAPE '\';

ALTER TABLE core.user_notification_channels ADD COLUMN IF NOT EXISTS quiet_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE core.user_notification_channels ADD COLUMN IF NOT EXISTS quiet_from SMALLINT NOT NULL DEFAULT 1380;
ALTER TABLE core.user_notification_channels ADD COLUMN IF NOT EXISTS quiet_to SMALLINT NOT NULL DEFAULT 480;
ALTER TABLE core.user_notification_channels ADD COLUMN IF NOT EXISTS quiet_critical BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE core.user_notification_channels ADD COLUMN IF NOT EXISTS quiet_tz TEXT NOT NULL DEFAULT '';
ALTER TABLE core.user_notification_channels ADD COLUMN IF NOT EXISTS balance_threshold NUMERIC(14, 2);

ALTER TABLE core.api_keys ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'admin';
ALTER TABLE core.api_keys ADD COLUMN IF NOT EXISTS last_used_ip TEXT;
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON core.api_keys (user_id, created_at DESC) WHERE user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS core.referral_rewards (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id  UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    referred_id  UUID REFERENCES core.users(id) ON DELETE SET NULL,
    payment_id   UUID NOT NULL UNIQUE REFERENCES core.payments(id) ON DELETE CASCADE,
    amount       NUMERIC(14, 2) NOT NULL,
    currency     TEXT NOT NULL,
    percent      NUMERIC(5, 2) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_referral_rewards_referrer ON core.referral_rewards (referrer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_attempts_user ON core.login_attempts (user_id, created_at DESC) WHERE user_id IS NOT NULL;
