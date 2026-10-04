ALTER TABLE core.server_firewall_rules
    ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT 'deny'
        CHECK (action IN ('allow', 'deny')),
    ADD COLUMN IF NOT EXISTS source TEXT;
