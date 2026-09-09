CREATE TABLE IF NOT EXISTS core.server_metric_points (
    id           BIGSERIAL PRIMARY KEY,
    server_id    UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    ts           TIMESTAMPTZ NOT NULL DEFAULT now(),
    cpu_pct      DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_used_mb  INT NOT NULL DEFAULT 0,
    mem_limit_mb INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_server_metric_points_server_ts
    ON core.server_metric_points(server_id, ts DESC);
