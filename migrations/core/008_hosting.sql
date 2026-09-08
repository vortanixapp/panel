

CREATE TABLE core.hosting_servers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    hostname          TEXT NOT NULL,
    ip_address        TEXT NOT NULL,
    port              INT NOT NULL DEFAULT 443,
    panel_type        TEXT NOT NULL
        CHECK (panel_type IN ('fastpanel', 'ispmanager', 'cpanel', 'plesk')),
    api_url           TEXT NOT NULL,
    api_key_enc       TEXT,
    api_username      TEXT,
    api_token_enc     TEXT,
    use_ssl           BOOLEAN NOT NULL DEFAULT true,
    nameservers       JSONB NOT NULL DEFAULT '[]',
    max_accounts      INT NOT NULL DEFAULT 0,
    current_accounts  INT NOT NULL DEFAULT 0,
    active            BOOLEAN NOT NULL DEFAULT true,
    description       TEXT,
    meta              JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hosting_servers_tenant ON core.hosting_servers(tenant_id, active);

CREATE TABLE core.hosting_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    hosting_server_id   UUID NOT NULL REFERENCES core.hosting_servers(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    panel_package_name  TEXT NOT NULL,
    disk_mb             INT NOT NULL DEFAULT 1024,
    bandwidth_mb        INT NOT NULL DEFAULT 0,
    max_domains         INT NOT NULL DEFAULT 1,
    max_subdomains      INT NOT NULL DEFAULT 5,
    max_databases       INT NOT NULL DEFAULT 1,
    max_email_accounts  INT NOT NULL DEFAULT 5,
    max_ftp_accounts    INT NOT NULL DEFAULT 1,
    has_ssl             BOOLEAN NOT NULL DEFAULT true,
    has_ssh             BOOLEAN NOT NULL DEFAULT false,
    has_cron            BOOLEAN NOT NULL DEFAULT true,
    has_backup          BOOLEAN NOT NULL DEFAULT true,
    php_version         TEXT,
    price_monthly       NUMERIC(12, 2) NOT NULL DEFAULT 0,
    rental_periods      JSONB NOT NULL DEFAULT '[]',
    renewal_periods     JSONB NOT NULL DEFAULT '[]',
    discounts           JSONB NOT NULL DEFAULT '{}',
    position            INT NOT NULL DEFAULT 0,
    active              BOOLEAN NOT NULL DEFAULT true,
    description         TEXT,
    meta                JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hosting_plans_tenant ON core.hosting_plans(tenant_id, hosting_server_id, active, position);

CREATE TABLE core.hosting_accounts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    hosting_server_id   UUID NOT NULL REFERENCES core.hosting_servers(id) ON DELETE CASCADE,
    hosting_plan_id     UUID NOT NULL REFERENCES core.hosting_plans(id) ON DELETE RESTRICT,
    wallet_id           UUID REFERENCES core.wallets(id) ON DELETE SET NULL,
    username            TEXT NOT NULL,
    primary_domain      TEXT,
    panel_account_id    TEXT,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'active', 'suspended', 'terminated', 'error')),
    ip_address          TEXT,
    panel_password_enc  TEXT,
    panel_login_url     TEXT,
    disk_used_mb        BIGINT NOT NULL DEFAULT 0,
    bandwidth_used_mb   BIGINT NOT NULL DEFAULT 0,
    expires_at          TIMESTAMPTZ,
    suspended_at        TIMESTAMPTZ,
    suspension_reason   TEXT,
    extra_data          JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hosting_server_id, username)
);

CREATE INDEX idx_hosting_accounts_tenant ON core.hosting_accounts(tenant_id, user_id, status);
CREATE INDEX idx_hosting_accounts_expires ON core.hosting_accounts(tenant_id, expires_at)
    WHERE expires_at IS NOT NULL;

CREATE TABLE core.hosting_domains (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    hosting_account_id  UUID NOT NULL REFERENCES core.hosting_accounts(id) ON DELETE CASCADE,
    domain              TEXT NOT NULL,
    is_primary          BOOLEAN NOT NULL DEFAULT false,
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('pending', 'active', 'suspended', 'removed')),
    panel_domain_id     TEXT,
    ssl_enabled         BOOLEAN NOT NULL DEFAULT false,
    meta                JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hosting_account_id, domain)
);

CREATE INDEX idx_hosting_domains_tenant ON core.hosting_domains(tenant_id, hosting_account_id, status);

CREATE TABLE core.hosting_databases (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    hosting_account_id  UUID NOT NULL REFERENCES core.hosting_accounts(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    db_user             TEXT,
    creds_ref           TEXT,
    panel_database_id   TEXT,
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('pending', 'active', 'removed')),
    meta                JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hosting_account_id, name)
);

CREATE INDEX idx_hosting_databases_tenant ON core.hosting_databases(tenant_id, hosting_account_id);

CREATE TABLE core.hosting_emails (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    hosting_account_id  UUID NOT NULL REFERENCES core.hosting_accounts(id) ON DELETE CASCADE,
    address             TEXT NOT NULL,
    mailbox_name        TEXT NOT NULL,
    creds_ref           TEXT,
    panel_mailbox_id    TEXT,
    quota_mb            INT,
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('pending', 'active', 'suspended', 'removed')),
    meta                JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hosting_account_id, address)
);

CREATE INDEX idx_hosting_emails_tenant ON core.hosting_emails(tenant_id, hosting_account_id, status);

ALTER TABLE core.hosting_servers ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.hosting_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.hosting_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.hosting_domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.hosting_databases ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.hosting_emails ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_hosting_servers ON core.hosting_servers
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_hosting_plans ON core.hosting_plans
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_hosting_accounts ON core.hosting_accounts
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_hosting_domains ON core.hosting_domains
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_hosting_databases ON core.hosting_databases
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_hosting_emails ON core.hosting_emails
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
