

CREATE TABLE core.wallets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    currency      TEXT NOT NULL,
    balance       NUMERIC(18, 2) NOT NULL DEFAULT 0,
    is_default    BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, currency)
);

CREATE INDEX idx_wallets_tenant ON core.wallets(tenant_id, user_id, is_default);

CREATE TABLE core.transactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    wallet_id     UUID NOT NULL REFERENCES core.wallets(id) ON DELETE CASCADE,
    type          TEXT NOT NULL
        CHECK (type IN ('debit', 'credit')),
    amount        NUMERIC(18, 2) NOT NULL,
    description   TEXT,
    meta          JSONB NOT NULL DEFAULT '{}',
    source_type   TEXT,
    source_id     UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_wallet ON core.transactions(tenant_id, wallet_id, created_at DESC);
CREATE INDEX idx_transactions_source ON core.transactions(tenant_id, source_type, source_id);

CREATE TABLE core.payment_providers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT false,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, provider)
);

CREATE INDEX idx_payment_providers_tenant ON core.payment_providers(tenant_id, enabled);

CREATE TABLE core.payments (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id               UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    wallet_id             UUID REFERENCES core.wallets(id) ON DELETE SET NULL,
    provider              TEXT NOT NULL,
    currency              TEXT NOT NULL DEFAULT 'RUB',
    amount                NUMERIC(18, 2) NOT NULL,
    status                TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled', 'refunded')),
    provider_payment_id   TEXT,
    provider_order_id     TEXT,
    promotion_id          UUID,
    credited_at           TIMESTAMPTZ,
    meta                  JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_tenant ON core.payments(tenant_id, user_id, status, created_at DESC);
CREATE INDEX idx_payments_provider ON core.payments(tenant_id, provider, provider_payment_id);

CREATE TABLE core.promotions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    code            TEXT,
    active          BOOLEAN NOT NULL DEFAULT true,
    starts_at       TIMESTAMPTZ,
    ends_at         TIMESTAMPTZ,
    applies_to      JSONB NOT NULL DEFAULT '[]',
    discount_type   TEXT
        CHECK (discount_type IS NULL OR discount_type IN ('percent', 'fixed')),
    discount_value  NUMERIC(10, 2) NOT NULL DEFAULT 0,
    bonus_percent   NUMERIC(5, 2) NOT NULL DEFAULT 0,
    bonus_fixed     NUMERIC(10, 2) NOT NULL DEFAULT 0,
    max_uses        INT,
    used_count      INT NOT NULL DEFAULT 0,
    min_amount      NUMERIC(10, 2),
    only_new_users  BOOLEAN NOT NULL DEFAULT false,
    filters         JSONB NOT NULL DEFAULT '{}',
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

CREATE INDEX idx_promotions_tenant ON core.promotions(tenant_id, active, starts_at, ends_at);

ALTER TABLE core.payments
    ADD CONSTRAINT payments_promotion_id_fkey
    FOREIGN KEY (promotion_id) REFERENCES core.promotions(id) ON DELETE SET NULL;

ALTER TABLE core.wallets ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.payment_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.promotions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_wallets ON core.wallets
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_transactions ON core.transactions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_payment_providers ON core.payment_providers
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_payments ON core.payments
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_promotions ON core.promotions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
