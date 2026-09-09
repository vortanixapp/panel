CREATE EXTENSION IF NOT EXISTS timescaledb;

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

ALTER TABLE core.server_metric_points DROP CONSTRAINT IF EXISTS server_metric_points_pkey;

SELECT create_hypertable('core.server_metric_points', 'ts', if_not_exists => TRUE);

SELECT add_retention_policy('core.server_metric_points', INTERVAL '90 days', if_not_exists => TRUE);

ALTER TABLE core.server_metric_points SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server_id'
);

SELECT add_compression_policy('core.server_metric_points', INTERVAL '7 days', if_not_exists => TRUE);
