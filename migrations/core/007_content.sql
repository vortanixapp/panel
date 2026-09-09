CREATE TABLE core.support_tickets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    assigned_admin_id   UUID REFERENCES core.users(id) ON DELETE SET NULL,
    subject             TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'pending', 'answered', 'closed')),
    priority            TEXT NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    last_message_at     TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,
    meta                JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_support_tickets_tenant ON core.support_tickets(tenant_id, status, priority);
CREATE INDEX idx_support_tickets_user ON core.support_tickets(tenant_id, user_id, status);

CREATE TABLE core.support_messages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    ticket_id     UUID NOT NULL REFERENCES core.support_tickets(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES core.users(id) ON DELETE SET NULL,
    is_staff      BOOLEAN NOT NULL DEFAULT false,
    message       TEXT NOT NULL,
    meta          JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_support_messages_ticket ON core.support_messages(tenant_id, ticket_id, created_at);

CREATE TABLE core.news (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    slug          TEXT NOT NULL,
    excerpt       TEXT,
    body          TEXT,
    published_at  TIMESTAMPTZ,
    active        BOOLEAN NOT NULL DEFAULT true,
    meta          JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_news_tenant ON core.news(tenant_id, active, published_at DESC);

CREATE TABLE core.notifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    type          TEXT NOT NULL DEFAULT 'info',
    title         TEXT NOT NULL,
    body          TEXT,
    read_at       TIMESTAMPTZ,
    meta          JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON core.notifications(tenant_id, user_id, read_at, created_at DESC);

CREATE TABLE core.mailings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    title             TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'scheduled', 'sending', 'completed', 'failed', 'cancelled')),
    channels          JSONB NOT NULL DEFAULT '[]',
    audience          JSONB NOT NULL DEFAULT '{}',
    subject           TEXT,
    body              TEXT,
    is_html           BOOLEAN NOT NULL DEFAULT true,
    scheduled_at      TIMESTAMPTZ,
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ,
    total_recipients  INT NOT NULL DEFAULT 0,
    sent_count        INT NOT NULL DEFAULT 0,
    failed_count      INT NOT NULL DEFAULT 0,
    skipped_count     INT NOT NULL DEFAULT 0,
    last_error        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mailings_tenant ON core.mailings(tenant_id, status, scheduled_at);

CREATE TABLE core.mailing_deliveries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    mailing_id    UUID NOT NULL REFERENCES core.mailings(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES core.users(id) ON DELETE SET NULL,
    channel       TEXT NOT NULL,
    address       TEXT,
    status        TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'sent', 'failed', 'skipped')),
    attempts      INT NOT NULL DEFAULT 0,
    sent_at       TIMESTAMPTZ,
    error         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mailing_deliveries_mailing ON core.mailing_deliveries(tenant_id, mailing_id, status);

CREATE TABLE core.daily_bonus_prizes (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    label                 TEXT NOT NULL,
    type                  TEXT NOT NULL
        CHECK (type IN ('balance', 'promo_rent', 'promo_renew', 'promo_hosting', 'promo_game')),
    value                 NUMERIC(12, 2) NOT NULL DEFAULT 0,
    discount_type         TEXT
        CHECK (discount_type IS NULL OR discount_type IN ('percent', 'fixed')),
    weight                INT NOT NULL DEFAULT 1,
    color                 TEXT,
    icon                  TEXT,
    prize_duration_hours  INT,
    active                BOOLEAN NOT NULL DEFAULT true,
    sort_order            INT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_daily_bonus_prizes_tenant ON core.daily_bonus_prizes(tenant_id, active, sort_order);

CREATE TABLE core.daily_bonus_spins (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    prize_id          UUID REFERENCES core.daily_bonus_prizes(id) ON DELETE SET NULL,
    prize_snapshot    JSONB NOT NULL DEFAULT '{}',
    reward_type       TEXT NOT NULL,
    reward_value      NUMERIC(12, 2) NOT NULL DEFAULT 0,
    promotion_id      UUID REFERENCES core.promotions(id) ON DELETE SET NULL,
    promotion_code    TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_daily_bonus_spins_user ON core.daily_bonus_spins(tenant_id, user_id, created_at DESC);

ALTER TABLE core.support_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.support_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.news ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.mailings ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.mailing_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.daily_bonus_prizes ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.daily_bonus_spins ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_support_tickets ON core.support_tickets
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_support_messages ON core.support_messages
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_news ON core.news
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_notifications ON core.notifications
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_mailings ON core.mailings
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_mailing_deliveries ON core.mailing_deliveries
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_daily_bonus_prizes ON core.daily_bonus_prizes
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_daily_bonus_spins ON core.daily_bonus_spins
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
