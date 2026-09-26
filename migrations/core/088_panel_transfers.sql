CREATE TABLE IF NOT EXISTS core.panel_transfers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    mode          TEXT NOT NULL DEFAULT 'ssh'
        CHECK (mode IN ('ssh', 'export', 'import', 'agents')),
    status        TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    stage         TEXT NOT NULL DEFAULT '',

    target_host   TEXT NOT NULL DEFAULT '',
    target_port   INT NOT NULL DEFAULT 22,
    target_user   TEXT NOT NULL DEFAULT '',
    target_host_key TEXT NOT NULL DEFAULT '',
    target_secret_enc TEXT,
    target_secret_kind TEXT NOT NULL DEFAULT 'password'
        CHECK (target_secret_kind IN ('password', 'key')),

    new_address   TEXT NOT NULL DEFAULT '',
    same_address  BOOLEAN NOT NULL DEFAULT false,
    freeze_writes BOOLEAN NOT NULL DEFAULT true,

    bytes_total   BIGINT NOT NULL DEFAULT 0,
    bytes_done    BIGINT NOT NULL DEFAULT 0,

    agents_total  INT NOT NULL DEFAULT 0,
    agents_done   INT NOT NULL DEFAULT 0,
    agents_failed INT NOT NULL DEFAULT 0,

    log           TEXT NOT NULL DEFAULT '',
    error         TEXT,

    created_by    UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_panel_transfers_active
    ON core.panel_transfers((status IN ('pending', 'running')))
    WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS idx_panel_transfers_created
    ON core.panel_transfers(created_at DESC);
