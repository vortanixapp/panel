CREATE TABLE IF NOT EXISTS core.tenant_databases (
    slug       TEXT PRIMARY KEY,
    db_name    TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO core.tenant_databases (slug, db_name)
SELECT slug, db_name FROM core.tenants WHERE db_name IS NOT NULL
ON CONFLICT (slug) DO NOTHING;
