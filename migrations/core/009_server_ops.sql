CREATE TABLE IF NOT EXISTS core.server_backups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    filename    TEXT,
    size_bytes  BIGINT,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_server_backups_server ON core.server_backups(server_id, created_at DESC);

CREATE TABLE IF NOT EXISTS core.server_cron_jobs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id  UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    schedule   TEXT NOT NULL,
    command    TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_server_cron_server ON core.server_cron_jobs(server_id);

CREATE TABLE IF NOT EXISTS core.server_firewall_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id  UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    protocol   TEXT NOT NULL DEFAULT 'tcp',
    port_from  INT NOT NULL,
    port_to    INT,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_server_firewall_server ON core.server_firewall_rules(server_id);

CREATE TABLE IF NOT EXISTS core.server_friends (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    permissions JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_server_friends_server ON core.server_friends(server_id);
