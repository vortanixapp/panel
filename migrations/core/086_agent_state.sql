ALTER TABLE core.node_daemons
    ADD COLUMN IF NOT EXISTS proto INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS caps TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS boot_id TEXT,
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS connected_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS disconnected_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS disconnect_reason TEXT,
    ADD COLUMN IF NOT EXISTS remote_addr TEXT,
    ADD COLUMN IF NOT EXISTS rtt_ms INT,
    ADD COLUMN IF NOT EXISTS host JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS stats_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS offline_notified_at TIMESTAMPTZ;

UPDATE core.node_daemons d
SET offline_notified_at = now()
FROM core.nodes n
WHERE d.node_id = n.id
  AND n.status <> 'online'
  AND d.offline_notified_at IS NULL;

UPDATE core.node_daemons
SET host = host || jsonb_build_object('agent',
        COALESCE(host->'agent', '{}'::jsonb) || jsonb_build_object('image', version)),
    version = CASE
        WHEN regexp_replace(version, '^.*:', '') ~ '^v?[0-9]+\.[0-9]+'
            THEN regexp_replace(regexp_replace(version, '^.*:', ''), '^v', '')
        ELSE NULL
    END
WHERE version ~ '[/:]';

CREATE TABLE IF NOT EXISTS core.node_events (
    id          BIGSERIAL PRIMARY KEY,
    node_id     UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    level       TEXT NOT NULL DEFAULT 'info'
        CHECK (level IN ('info', 'success', 'warn', 'error')),
    data        JSONB NOT NULL DEFAULT '{}'::jsonb,
    actor_id    UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_node_events_node ON core.node_events (node_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_node_events_created ON core.node_events (created_at);

CREATE TABLE IF NOT EXISTS core.node_tasks (
    id            UUID PRIMARY KEY,
    node_id       UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    action        TEXT NOT NULL,
    method        TEXT NOT NULL DEFAULT 'relay'
        CHECK (method IN ('relay', 'ssh')),
    status        TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'sent', 'running', 'done', 'failed', 'expired')),
    params        JSONB NOT NULL DEFAULT '{}'::jsonb,
    progress      JSONB,
    result        JSONB,
    error         TEXT,
    error_code    TEXT,
    boot_id       TEXT,
    job_id        UUID,
    requested_by  UUID REFERENCES core.users(id) ON DELETE SET NULL,
    deadline_at   TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_node_tasks_node ON core.node_tasks (node_id, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_node_tasks_open ON core.node_tasks (deadline_at)
    WHERE status IN ('queued', 'sent', 'running');
