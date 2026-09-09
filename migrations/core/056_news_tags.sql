ALTER TABLE core.news
    ADD COLUMN IF NOT EXISTS tag TEXT NOT NULL DEFAULT 'update',
    ADD COLUMN IF NOT EXISTS pinned BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_news_tag
    ON core.news (tenant_id, tag, published_at DESC);

CREATE INDEX IF NOT EXISTS idx_news_pinned
    ON core.news (tenant_id, published_at DESC) WHERE pinned;

ALTER TABLE core.users
    ADD COLUMN IF NOT EXISTS news_read_at TIMESTAMPTZ;
