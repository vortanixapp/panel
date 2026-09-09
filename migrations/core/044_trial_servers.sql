ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS is_trial BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS core.trial_grants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    server_id  UUID REFERENCES core.servers(id) ON DELETE SET NULL,
    game_id    TEXT NOT NULL DEFAULT '',
    hours      SMALLINT NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trial_grants_user
    ON core.trial_grants(tenant_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_servers_trial
    ON core.servers(tenant_id, is_trial)
    WHERE is_trial = true;
