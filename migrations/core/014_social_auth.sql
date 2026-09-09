CREATE TABLE IF NOT EXISTS core.user_social_accounts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email            TEXT,
    name             TEXT,
    avatar_url       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id),
    UNIQUE (tenant_id, user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_user_social_accounts_user
    ON core.user_social_accounts(tenant_id, user_id);

ALTER TABLE core.user_social_accounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_user_social_accounts ON core.user_social_accounts
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
