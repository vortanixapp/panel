ALTER TABLE core.nodes
    ADD COLUMN country          TEXT,
    ADD COLUMN city             TEXT,
    ADD COLUMN region           TEXT,
    ADD COLUMN ip_address       TEXT,
    ADD COLUMN ssh_host         TEXT,
    ADD COLUMN ssh_user         TEXT,
    ADD COLUMN ssh_port         INT NOT NULL DEFAULT 22,
    ADD COLUMN ssh_password_enc TEXT,
    ADD COLUMN description      TEXT,
    ADD COLUMN sort_order       INT NOT NULL DEFAULT 0,
    ADD COLUMN is_active        BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE core.node_setups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id         UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    component       TEXT NOT NULL
        CHECK (component IN ('packages', 'docker', 'mysql', 'ftp', 'daemon', 'agent')),
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'installing', 'installed', 'failed')),
    installed_at    TIMESTAMPTZ,
    error_message   TEXT,
    meta            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (node_id, component)
);

CREATE INDEX idx_node_setups_tenant ON core.node_setups(tenant_id, node_id, status);

CREATE TABLE core.node_daemons (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id       UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'unknown'
        CHECK (status IN ('online', 'offline', 'unknown')),
    version       TEXT,
    pid           INT,
    uptime_sec    DOUBLE PRECISION,
    platform      TEXT,
    last_seen_at  TIMESTAMPTZ,
    meta          JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (node_id)
);

CREATE INDEX idx_node_daemons_tenant ON core.node_daemons(tenant_id, status);

CREATE TABLE core.ip_pools (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id         UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    address         INET NOT NULL,
    status          TEXT NOT NULL DEFAULT 'free'
        CHECK (status IN ('free', 'reserved', 'assigned')),
    server_id       UUID REFERENCES core.servers(id) ON DELETE SET NULL,
    reserved_until  TIMESTAMPTZ,
    meta            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (node_id, address)
);

CREATE INDEX idx_ip_pools_tenant ON core.ip_pools(tenant_id, node_id, status);

ALTER TABLE core.node_setups ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.node_daemons ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.ip_pools ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_node_setups ON core.node_setups
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_node_daemons ON core.node_daemons
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_ip_pools ON core.ip_pools
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
