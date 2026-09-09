ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS auto_renew BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS dunning_stage SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS dunning_for TIMESTAMPTZ;

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS rental_period_days SMALLINT NOT NULL DEFAULT 30
        CHECK (rental_period_days BETWEEN 1 AND 365);

CREATE INDEX IF NOT EXISTS idx_servers_dunning
    ON core.servers(tenant_id, expires_at)
    WHERE expires_at IS NOT NULL;
