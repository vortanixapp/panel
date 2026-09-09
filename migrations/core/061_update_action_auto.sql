ALTER TABLE core.installation
    DROP CONSTRAINT IF EXISTS installation_update_action_check;

ALTER TABLE core.installation
    ADD CONSTRAINT installation_update_action_check
    CHECK (update_action IN ('defer', 'apply_now', 'auto_on', 'auto_off'));
