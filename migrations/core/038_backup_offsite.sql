ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES core.tenants(id) ON DELETE CASCADE;

UPDATE core.server_backups b
SET tenant_id = s.tenant_id
FROM core.servers s
WHERE b.server_id = s.id AND b.tenant_id IS NULL;

ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS remote_key TEXT;

ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS remote_status TEXT NOT NULL DEFAULT 'none'
        CHECK (remote_status IN ('none', 'pending', 'uploading', 'uploaded', 'failed', 'deleted'));

ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS remote_error TEXT;

ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS remote_size BIGINT;

ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMPTZ;

ALTER TABLE core.server_backups
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'manual';

CREATE INDEX IF NOT EXISTS idx_server_backups_remote
    ON core.server_backups(server_id, remote_status, created_at DESC);
