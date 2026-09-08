

ALTER TABLE core.users
    ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true;

UPDATE core.nodes SET active = COALESCE(is_active, true);

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS docker_images JSONB NOT NULL DEFAULT '[]';

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_servers_idempotency
    ON core.servers(tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS steam_updatable BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS core.server_ports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    server_id   UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    port        INT NOT NULL,
    protocol    TEXT NOT NULL DEFAULT 'udp'
        CHECK (protocol IN ('udp', 'tcp', 'both')),
    purpose     TEXT NOT NULL DEFAULT 'game',
    meta        JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, port, protocol)
);

CREATE INDEX IF NOT EXISTS idx_server_ports_server ON core.server_ports(server_id, port);

ALTER TABLE core.jobs
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE core.jobs
    ADD COLUMN IF NOT EXISTS error_message TEXT;

CREATE INDEX IF NOT EXISTS idx_jobs_tenant_type_status
    ON core.jobs(tenant_id, type, status, created_at DESC);

ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS department_id TEXT;

ALTER TABLE core.support_messages
    ADD COLUMN IF NOT EXISTS body TEXT;

UPDATE core.support_messages SET body = message WHERE body IS NULL AND message IS NOT NULL;

CREATE TABLE IF NOT EXISTS core.support_message_attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    ticket_id       UUID NOT NULL REFERENCES core.support_tickets(id) ON DELETE CASCADE,
    message_id      UUID NOT NULL REFERENCES core.support_messages(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES core.users(id) ON DELETE SET NULL,
    is_staff        BOOLEAN NOT NULL DEFAULT false,
    disk            TEXT NOT NULL DEFAULT 'local',
    path            TEXT NOT NULL,
    original_name   TEXT NOT NULL,
    mime_type       TEXT,
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_support_attachments_ticket
    ON core.support_message_attachments(tenant_id, ticket_id, message_id);

ALTER TABLE core.mailings DROP CONSTRAINT IF EXISTS mailings_status_check;
ALTER TABLE core.mailings ADD CONSTRAINT mailings_status_check
    CHECK (status IN ('draft', 'scheduled', 'queued', 'sending', 'completed', 'failed', 'cancelled'));

ALTER TABLE core.mailings ALTER COLUMN title DROP NOT NULL;

ALTER TABLE core.hosting_emails ALTER COLUMN mailbox_name DROP NOT NULL;

CREATE TABLE IF NOT EXISTS core.translation_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    locale      TEXT NOT NULL DEFAULT 'ru',
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, locale, key)
);

CREATE INDEX IF NOT EXISTS idx_translation_keys_tenant_locale
    ON core.translation_keys(tenant_id, locale);

INSERT INTO core.translation_keys (tenant_id, locale, key, value)
SELECT t.id, v.locale, v.k, v.val
FROM core.tenants t
CROSS JOIN (VALUES
    ('ru', 'dashboard', 'Панель управления'),
    ('en', 'dashboard', 'Dashboard'),
    ('ru', 'billing', 'Биллинг'),
    ('en', 'billing', 'Billing'),
    ('ru', 'support', 'Тех. поддержка'),
    ('en', 'support', 'Support'),
    ('ru', 'my_servers', 'Мои серверы'),
    ('en', 'my_servers', 'My servers'),
    ('ru', 'rent_server', 'Аренда сервера'),
    ('en', 'rent_server', 'Rent a server')
) AS v(locale, k, val)
ON CONFLICT (tenant_id, locale, key) DO NOTHING;

ALTER TABLE core.daily_bonus_spins ALTER COLUMN reward_type SET DEFAULT 'balance';

INSERT INTO core.payment_providers (tenant_id, provider, enabled, config)
SELECT t.id, p.provider, false, '{}'::jsonb
FROM core.tenants t
CROSS JOIN (VALUES ('yookassa'), ('freekassa'), ('robokassa'), ('stripe')) AS p(provider)
ON CONFLICT (tenant_id, provider) DO NOTHING;

ALTER TABLE core.server_ports ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.support_message_attachments ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.translation_keys ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_server_ports ON core.server_ports
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_support_attachments ON core.support_message_attachments
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_translation_keys ON core.translation_keys
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
