CREATE TABLE IF NOT EXISTS core.server_metric_rollups (
    server_id    UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    ts           TIMESTAMPTZ NOT NULL,
    cpu_avg      DOUBLE PRECISION NOT NULL,
    cpu_max      DOUBLE PRECISION NOT NULL,
    mem_used_avg INT NOT NULL,
    mem_used_max INT NOT NULL,
    mem_limit_mb INT NOT NULL,
    samples      INT NOT NULL,
    PRIMARY KEY (server_id, ts)
);

CREATE INDEX IF NOT EXISTS idx_server_metric_rollups_ts
    ON core.server_metric_rollups USING brin (ts);

DO $$
BEGIN
    CREATE INDEX IF NOT EXISTS idx_server_metric_points_ts
        ON core.server_metric_points USING brin (ts);
EXCEPTION WHEN others THEN
    RAISE NOTICE 'пропущено (%): индекс по времени для точек метрик', SQLERRM;
END $$;

CREATE INDEX IF NOT EXISTS idx_server_friends_user
    ON core.server_friends (user_id);

CREATE INDEX IF NOT EXISTS idx_jobs_open
    ON core.jobs (type, created_at)
    WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS idx_node_tasks_finished
    ON core.node_tasks ((COALESCE(finished_at, updated_at)))
    WHERE status IN ('done', 'failed', 'expired');

DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
EXCEPTION WHEN others THEN
    RAISE NOTICE 'pg_stat_statements недоступно: %', SQLERRM;
END $$;
