ALTER TABLE core.user_profiles
    ADD COLUMN IF NOT EXISTS middle_name  TEXT,
    ADD COLUMN IF NOT EXISTS country      TEXT,
    ADD COLUMN IF NOT EXISTS address_line TEXT,
    ADD COLUMN IF NOT EXISTS city         TEXT,
    ADD COLUMN IF NOT EXISTS region       TEXT,
    ADD COLUMN IF NOT EXISTS postal_code  TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'user_profiles_country_format'
    ) THEN
        ALTER TABLE core.user_profiles
            ADD CONSTRAINT user_profiles_country_format
            CHECK (country IS NULL OR country ~ '^[A-Z]{2}$');
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_user_profiles_country ON core.user_profiles (country)
    WHERE country IS NOT NULL;

ALTER TABLE core.pending_registrations
    ADD COLUMN IF NOT EXISTS middle_name  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS country      TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS address_line TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS city         TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS region       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS postal_code  TEXT NOT NULL DEFAULT '';

ALTER TABLE core.user_billing_profiles
    ADD COLUMN IF NOT EXISTS country            TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS tax_id             TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS tax_id_type        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS tax_id_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS tax_exempt         BOOLEAN NOT NULL DEFAULT false;
