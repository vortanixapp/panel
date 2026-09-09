CREATE TABLE IF NOT EXISTS core.login_attempts (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  UUID REFERENCES core.tenants(id) ON DELETE CASCADE,
    email      TEXT NOT NULL DEFAULT '',
    user_id    UUID REFERENCES core.users(id) ON DELETE SET NULL,
    ip         TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    success    BOOLEAN NOT NULL DEFAULT false,

    reason     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_login_attempts_ip
    ON core.login_attempts(ip, created_at DESC)
    WHERE success = false;

CREATE INDEX IF NOT EXISTS idx_login_attempts_recent
    ON core.login_attempts(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS core.ip_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    ip         TEXT NOT NULL,
    reason     TEXT NOT NULL DEFAULT '',

    auto       BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,
    created_by UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, ip)
);

CREATE INDEX IF NOT EXISTS idx_ip_blocks_lookup
    ON core.ip_blocks(ip, expires_at);
