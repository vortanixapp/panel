ALTER TABLE core.servers DROP CONSTRAINT IF EXISTS servers_provisioning_status_check;
ALTER TABLE core.servers ADD CONSTRAINT servers_provisioning_status_check
    CHECK (provisioning_status IN (
        'pending', 'provisioning', 'ready', 'failed', 'deprovisioning', 'migrating'
    ));

CREATE TABLE IF NOT EXISTS core.server_migrations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    server_id    UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,

    from_node_id UUID REFERENCES core.nodes(id) ON DELETE SET NULL,
    to_node_id   UUID REFERENCES core.nodes(id) ON DELETE SET NULL,

    from_node_name TEXT NOT NULL DEFAULT '',
    to_node_name   TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed')),

    stage        TEXT NOT NULL DEFAULT '',
    error        TEXT,
    bytes        BIGINT NOT NULL DEFAULT 0,

    remove_source BOOLEAN NOT NULL DEFAULT true,
    created_by   UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_server_migrations_server
    ON core.server_migrations(tenant_id, server_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_server_migrations_active
    ON core.server_migrations(server_id)
    WHERE status IN ('pending', 'running');
