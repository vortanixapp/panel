

CREATE TABLE IF NOT EXISTS core.server_ftp_accounts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    server_id   uuid NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,

    username    text NOT NULL,
    password    text NOT NULL,
    status      text NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'active', 'failed', 'deleting')),
    error_message text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS server_ftp_accounts_username_key
    ON core.server_ftp_accounts (username);

CREATE INDEX IF NOT EXISTS server_ftp_accounts_server_idx
    ON core.server_ftp_accounts (server_id);
