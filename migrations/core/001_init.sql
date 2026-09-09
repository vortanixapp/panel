CREATE SCHEMA IF NOT EXISTS core;

CREATE TABLE core.tenants (
    id          UUID PRIMARY KEY,
    slug        TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'revoked')),
    settings    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE core.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'owner'
        CHECK (role IN ('owner', 'admin', 'user')),
    status        TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON core.users(tenant_id);

CREATE TABLE core.nodes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    fqdn         TEXT NOT NULL,
    agent_token  TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'offline'
        CHECK (status IN ('online', 'offline', 'maintenance')),
    meta         JSONB NOT NULL DEFAULT '{}',
    last_seen_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_nodes_tenant ON core.nodes(tenant_id, status);

CREATE TABLE core.servers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id    UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    game_id    TEXT NOT NULL DEFAULT 'minecraft',
    name       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'stopped'
        CHECK (status IN ('running', 'stopped', 'starting', 'stopping', 'error')),
    config     JSONB NOT NULL DEFAULT '{}',
    limits     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_servers_tenant_status ON core.servers(tenant_id, status, created_at DESC);
CREATE INDEX idx_servers_node ON core.servers(node_id);

CREATE TABLE core.jobs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    payload    JSONB NOT NULL DEFAULT '{}',
    result     JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_tenant ON core.jobs(tenant_id, created_at DESC);

CREATE TABLE core.audit_logs (
    id         BIGSERIAL,
    tenant_id  UUID NOT NULL,
    user_id    UUID,
    action     TEXT NOT NULL,
    resource   TEXT NOT NULL,
    meta       JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE core.audit_logs_default PARTITION OF core.audit_logs DEFAULT;

CREATE INDEX idx_audit_tenant_time ON core.audit_logs(tenant_id, created_at DESC);

ALTER TABLE core.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.servers ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_users ON core.users
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_nodes ON core.nodes
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_servers ON core.servers
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
