

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_servers_expired_active
    ON core.servers(expires_at)
    WHERE expires_at IS NOT NULL AND suspended_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_hosting_accounts_expired_active
    ON core.hosting_accounts(expires_at)
    WHERE expires_at IS NOT NULL AND suspended_at IS NULL;
