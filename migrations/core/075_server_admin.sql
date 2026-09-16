CREATE TABLE IF NOT EXISTS core.server_notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id  UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    author_id  UUID REFERENCES core.users(id) ON DELETE SET NULL,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_server_notes_server
    ON core.server_notes(server_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_created
    ON core.audit_logs(resource, created_at DESC);
