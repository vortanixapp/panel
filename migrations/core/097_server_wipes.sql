CREATE TABLE IF NOT EXISTS core.server_wipe_plans (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id        UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    name             TEXT NOT NULL DEFAULT '',
    kind             TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    schedule_type    TEXT NOT NULL CHECK (schedule_type IN ('weekly', 'monthly', 'cron', 'once')),
    weekday          SMALLINT NOT NULL DEFAULT 4 CHECK (weekday BETWEEN 0 AND 6),
    nth              SMALLINT NOT NULL DEFAULT 1 CHECK (nth BETWEEN 1 AND 5),
    time_of_day      TEXT NOT NULL DEFAULT '19:00',
    cron_expr        TEXT NOT NULL DEFAULT '',
    run_at           TIMESTAMPTZ,
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    backup_before    BOOLEAN NOT NULL DEFAULT true,
    new_seed         BOOLEAN NOT NULL DEFAULT true,
    announce_minutes INT[] NOT NULL DEFAULT '{60,30,10,5,1}',
    announce_text    TEXT NOT NULL DEFAULT '',
    notify_owner     BOOLEAN NOT NULL DEFAULT true,
    skip_next        BOOLEAN NOT NULL DEFAULT false,
    next_run_at      TIMESTAMPTZ,
    last_run_at      TIMESTAMPTZ,
    last_status      TEXT NOT NULL DEFAULT '',
    created_by       UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_server_wipe_plans_server
    ON core.server_wipe_plans(server_id);

CREATE INDEX IF NOT EXISTS idx_server_wipe_plans_due
    ON core.server_wipe_plans(next_run_at)
    WHERE enabled AND next_run_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS core.server_wipe_runs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id          UUID REFERENCES core.server_wipe_plans(id) ON DELETE SET NULL,
    server_id        UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    kind             TEXT NOT NULL,
    source           TEXT NOT NULL CHECK (source IN ('schedule', 'manual', 'telegram')),
    status           TEXT NOT NULL DEFAULT 'announcing'
                     CHECK (status IN ('announcing', 'running', 'completed', 'failed', 'cancelled')),
    stage            TEXT NOT NULL DEFAULT '',
    starts_at        TIMESTAMPTZ NOT NULL,
    announced        INT[] NOT NULL DEFAULT '{}',
    backup_before    BOOLEAN NOT NULL DEFAULT true,
    new_seed         BOOLEAN NOT NULL DEFAULT true,
    announce_minutes INT[] NOT NULL DEFAULT '{}',
    announce_text    TEXT NOT NULL DEFAULT '',
    notify_owner     BOOLEAN NOT NULL DEFAULT true,
    details          JSONB NOT NULL DEFAULT '{}'::jsonb,
    error            TEXT NOT NULL DEFAULT '',
    created_by       UUID REFERENCES core.users(id) ON DELETE SET NULL,
    heartbeat_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_server_wipe_runs_active
    ON core.server_wipe_runs(server_id)
    WHERE status IN ('announcing', 'running');

CREATE INDEX IF NOT EXISTS idx_server_wipe_runs_server
    ON core.server_wipe_runs(server_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_server_wipe_runs_plan
    ON core.server_wipe_runs(plan_id, created_at DESC);
