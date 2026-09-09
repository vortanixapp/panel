ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'other';

CREATE INDEX IF NOT EXISTS idx_support_tickets_category
    ON core.support_tickets(tenant_id, category, status);

CREATE TABLE IF NOT EXISTS core.kb_articles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    slug         TEXT NOT NULL,
    title        TEXT NOT NULL,
    excerpt      TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT 'other',
    published    BOOLEAN NOT NULL DEFAULT true,
    views        INT NOT NULL DEFAULT 0,
    position     INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_articles_slug
    ON core.kb_articles(tenant_id, slug);

CREATE INDEX IF NOT EXISTS idx_kb_articles_category
    ON core.kb_articles(tenant_id, category, position);

CREATE INDEX IF NOT EXISTS idx_kb_articles_search
    ON core.kb_articles USING gin (
        to_tsvector('russian', title || ' ' || excerpt || ' ' || body)
    );

ALTER TABLE core.kb_articles ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_kb_articles ON core.kb_articles;

CREATE POLICY tenant_isolation_kb_articles ON core.kb_articles
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
