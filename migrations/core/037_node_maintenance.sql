

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS maintenance_mode BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS maintenance_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS maintenance_until TIMESTAMPTZ;

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS maintenance_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_nodes_maintenance
    ON core.nodes(tenant_id)
    WHERE maintenance_mode = true;
