ALTER TABLE core.installation
    ADD COLUMN IF NOT EXISTS update_auto BOOLEAN NOT NULL DEFAULT false;
