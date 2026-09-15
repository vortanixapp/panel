ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS billing_source TEXT NOT NULL DEFAULT 'panel'
        CHECK (billing_source IN ('panel', 'whmcs'));

CREATE TABLE IF NOT EXISTS core.whmcs_clients (
    client_id   BIGINT PRIMARY KEY,
    user_id     UUID NOT NULL UNIQUE REFERENCES core.users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS core.whmcs_services (
    service_id        BIGINT PRIMARY KEY,
    client_id         BIGINT NOT NULL,
    user_id           UUID REFERENCES core.users(id) ON DELETE SET NULL,
    server_id         UUID UNIQUE REFERENCES core.servers(id) ON DELETE SET NULL,
    status            TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'active', 'suspended', 'terminated')),
    blocked_by_whmcs  BOOLEAN NOT NULL DEFAULT false,
    suspend_reason    TEXT,
    product           TEXT NOT NULL DEFAULT '',
    billing_cycle     TEXT NOT NULL DEFAULT '',
    next_due_date     DATE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_whmcs_services_client ON core.whmcs_services(client_id);
CREATE INDEX IF NOT EXISTS idx_whmcs_services_user ON core.whmcs_services(user_id);

CREATE TABLE IF NOT EXISTS core.whmcs_sso_tokens (
    token_hash  TEXT PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    redirect    TEXT NOT NULL DEFAULT '/servers',
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_whmcs_sso_tokens_expires ON core.whmcs_sso_tokens(expires_at);
