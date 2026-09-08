

ALTER TABLE core.users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE core.users ADD CONSTRAINT users_role_check
    CHECK (role IN ('owner', 'admin', 'support', 'user'));

ALTER TABLE core.users
    ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_login_at    TIMESTAMPTZ;

CREATE TABLE core.user_profiles (
    user_id       UUID PRIMARY KEY REFERENCES core.users(id) ON DELETE CASCADE,
    display_name  TEXT,
    first_name    TEXT,
    last_name     TEXT,
    phone         TEXT,
    avatar_url    TEXT,
    locale        TEXT NOT NULL DEFAULT 'ru',
    contacts      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE core.user_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    ip_address    TEXT,
    user_agent    TEXT,
    last_active   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_sessions_tenant ON core.user_sessions(tenant_id, user_id, last_active DESC);

CREATE TABLE core.two_factor_secrets (
    user_id         UUID PRIMARY KEY REFERENCES core.users(id) ON DELETE CASCADE,
    secret          TEXT NOT NULL,
    recovery_codes  JSONB NOT NULL DEFAULT '[]',
    enabled_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE core.tenant_settings (
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    key           TEXT NOT NULL,
    value         JSONB NOT NULL DEFAULT 'null',
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key)
);

CREATE TABLE core.password_reset_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    token_hash    TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_user ON core.password_reset_tokens(tenant_id, user_id);

ALTER TABLE core.user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.password_reset_tokens ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_user_sessions ON core.user_sessions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_tenant_settings ON core.tenant_settings
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_password_reset_tokens ON core.password_reset_tokens
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
