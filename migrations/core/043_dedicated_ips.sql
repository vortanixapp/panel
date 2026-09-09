ALTER TABLE core.ip_pools
    ADD COLUMN IF NOT EXISTS label TEXT NOT NULL DEFAULT '';

ALTER TABLE core.ip_pools
    ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS uq_ip_pools_server
    ON core.ip_pools(server_id)
    WHERE server_id IS NOT NULL;
