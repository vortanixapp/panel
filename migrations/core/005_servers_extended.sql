ALTER TABLE core.servers
    ADD COLUMN user_id              UUID REFERENCES core.users(id) ON DELETE SET NULL,
    ADD COLUMN tariff_id            UUID REFERENCES core.tariffs(id) ON DELETE SET NULL,
    ADD COLUMN game_version_id      UUID REFERENCES core.game_versions(id) ON DELETE SET NULL,
    ADD COLUMN runtime_status       TEXT,
    ADD COLUMN provisioning_status  TEXT NOT NULL DEFAULT 'pending'
        CHECK (provisioning_status IN ('pending', 'provisioning', 'ready', 'failed', 'deprovisioning')),
    ADD COLUMN provisioning_error   TEXT,
    ADD COLUMN expires_at           TIMESTAMPTZ,
    ADD COLUMN ports                JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN ftp_creds_ref      TEXT,
    ADD COLUMN mysql_creds_ref    TEXT,
    ADD COLUMN container_id         TEXT,
    ADD COLUMN container_name       TEXT,
    ADD COLUMN ip_address           TEXT,
    ADD COLUMN primary_port         INT;

CREATE INDEX idx_servers_user ON core.servers(tenant_id, user_id, created_at DESC);
CREATE INDEX idx_servers_tariff ON core.servers(tenant_id, tariff_id);
CREATE INDEX idx_servers_expires ON core.servers(tenant_id, expires_at)
    WHERE expires_at IS NOT NULL;
CREATE INDEX idx_servers_provisioning ON core.servers(tenant_id, provisioning_status);
