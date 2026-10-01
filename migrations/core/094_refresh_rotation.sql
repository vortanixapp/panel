ALTER TABLE core.user_sessions
    ADD COLUMN IF NOT EXISTS refresh_jti      TEXT,
    ADD COLUMN IF NOT EXISTS prev_refresh_jti TEXT,
    ADD COLUMN IF NOT EXISTS rotated_at       TIMESTAMPTZ;
