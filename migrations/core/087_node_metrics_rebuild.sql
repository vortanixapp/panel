ALTER TABLE core.node_metrics RENAME TO node_metrics_old;
ALTER INDEX IF EXISTS core.node_metrics_pkey RENAME TO node_metrics_old_pkey;
ALTER SEQUENCE IF EXISTS core.node_metrics_id_seq RENAME TO node_metrics_old_id_seq;

CREATE TABLE core.node_metrics (
    id           BIGSERIAL PRIMARY KEY,
    node_id      UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    metric_type  TEXT NOT NULL,
    value        DOUBLE PRECISION NOT NULL DEFAULT 0,
    text_value   TEXT,
    measured_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO core.node_metrics (node_id, metric_type, value, text_value, measured_at, created_at)
SELECT DISTINCT ON (node_id, metric_type)
    node_id, metric_type, value, text_value, measured_at, created_at
FROM core.node_metrics_old
WHERE text_value IS NOT NULL
  AND measured_at < now() - interval '8 days'
ORDER BY node_id, metric_type, measured_at DESC;

INSERT INTO core.node_metrics (node_id, metric_type, value, text_value, measured_at, created_at)
SELECT node_id, metric_type, value, text_value, measured_at, created_at
FROM core.node_metrics_old
WHERE measured_at >= now() - interval '8 days'
ORDER BY measured_at, id;

CREATE INDEX idx_node_metrics_series ON core.node_metrics (node_id, metric_type, measured_at DESC);
CREATE INDEX idx_node_metrics_measured ON core.node_metrics (measured_at);

DROP TABLE core.node_metrics_old;
