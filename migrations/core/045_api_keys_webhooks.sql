

CREATE TABLE IF NOT EXISTS core.api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES core.users(id) ON DELETE SET NULL,
    name         TEXT NOT NULL,


    prefix       TEXT NOT NULL,
    key_hash     TEXT NOT NULL,


    scopes       JSONB NOT NULL DEFAULT '[]',
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_hash ON core.api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON core.api_keys(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS core.webhooks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,

    events      JSONB NOT NULL DEFAULT '[]',


    secret      TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT true,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_tenant ON core.webhooks(tenant_id, active);

CREATE TABLE IF NOT EXISTS core.webhook_deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    webhook_id   UUID NOT NULL REFERENCES core.webhooks(id) ON DELETE CASCADE,
    event        TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'delivered', 'failed')),
    attempts     SMALLINT NOT NULL DEFAULT 0,
    response_code INTEGER,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_pending
    ON core.webhook_deliveries(status, next_retry_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_hook
    ON core.webhook_deliveries(webhook_id, created_at DESC);
