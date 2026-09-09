CREATE TABLE IF NOT EXISTS core.news_images (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    news_id    UUID NOT NULL REFERENCES core.news(id) ON DELETE CASCADE,
    path       TEXT NOT NULL,
    sort       INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_news_images_news ON core.news_images (news_id, sort);

ALTER TABLE core.news_images ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_news_images ON core.news_images;
CREATE POLICY tenant_isolation_news_images ON core.news_images
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
