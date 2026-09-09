CREATE TABLE IF NOT EXISTS core.server_backup_schedules (
    server_id    UUID PRIMARY KEY REFERENCES core.servers(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    enabled      BOOLEAN NOT NULL DEFAULT false,
    frequency    TEXT NOT NULL DEFAULT 'daily'
        CHECK (frequency IN ('daily', 'weekly')),

    hour_utc     SMALLINT NOT NULL DEFAULT 4
        CHECK (hour_utc BETWEEN 0 AND 23),

    day_of_week  SMALLINT NOT NULL DEFAULT 0
        CHECK (day_of_week BETWEEN 0 AND 6),

    keep_count   SMALLINT NOT NULL DEFAULT 7
        CHECK (keep_count BETWEEN 1 AND 30),
    last_run_at  TIMESTAMPTZ,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backup_schedules_due
    ON core.server_backup_schedules(last_run_at)
    WHERE enabled = true;
