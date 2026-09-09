ALTER TABLE core.installation
    ADD COLUMN IF NOT EXISTS update_target_version  TEXT,
    ADD COLUMN IF NOT EXISTS update_mandatory_after TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS update_notes           TEXT,

    ADD COLUMN IF NOT EXISTS update_defer_until     TIMESTAMPTZ,

    ADD COLUMN IF NOT EXISTS update_action          TEXT
        CHECK (update_action IN ('defer', 'apply_now'));
