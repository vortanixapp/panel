ALTER TABLE core.installation
    ADD COLUMN IF NOT EXISTS update_component TEXT NOT NULL DEFAULT '';

ALTER TABLE core.installation
    DROP CONSTRAINT IF EXISTS installation_update_component_check;

ALTER TABLE core.installation
    ADD CONSTRAINT installation_update_component_check
    CHECK (update_component IN ('', 'panel-ui', 'agent', 'updater'));
