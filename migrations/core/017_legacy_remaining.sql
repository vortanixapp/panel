ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS auto_start BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS blocked_reason TEXT,
    ADD COLUMN IF NOT EXISTS blocked_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_servers_blocked ON core.servers(tenant_id, is_blocked)
    WHERE is_blocked = true;

CREATE TABLE IF NOT EXISTS core.node_metrics (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id      UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    metric_type  TEXT NOT NULL,
    value        DOUBLE PRECISION NOT NULL DEFAULT 0,
    text_value   TEXT,
    measured_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_node_metrics_node
    ON core.node_metrics(tenant_id, node_id, metric_type, measured_at DESC);

ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS user_last_read_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS staff_last_read_at TIMESTAMPTZ;

ALTER TABLE core.support_messages
    ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS core.http_activity_logs (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES core.users(id) ON DELETE SET NULL,
    scope         TEXT NOT NULL DEFAULT 'api',
    event         TEXT NOT NULL,
    method        TEXT NOT NULL DEFAULT 'GET',
    status_code   INT NOT NULL DEFAULT 200,
    is_success    BOOLEAN NOT NULL DEFAULT true,
    request_path  TEXT NOT NULL,
    route_name    TEXT,
    ip_address    TEXT,
    user_agent    TEXT,
    duration_ms   INT NOT NULL DEFAULT 0,
    meta          JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_http_activity_tenant
    ON core.http_activity_logs(tenant_id, created_at DESC);

ALTER TABLE core.node_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.http_activity_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_node_metrics ON core.node_metrics
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_http_activity ON core.http_activity_logs
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

UPDATE core.tenants
SET settings = COALESCE(settings, '{}'::jsonb) || '{
  "migrated_modules": {
    "auth": true,
    "catalog": true,
    "servers": true,
    "billing": true,
    "support": true,
    "hosting": true,
    "admin": true
  }
}'::jsonb
WHERE NOT (COALESCE(settings, '{}'::jsonb) ? 'migrated_modules');
