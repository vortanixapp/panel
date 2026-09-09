INSERT INTO core.tenant_settings (key, value)
SELECT 'panel.name', to_jsonb(name)
FROM core.tenants
ORDER BY created_at
LIMIT 1
ON CONFLICT (key) DO NOTHING;

DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a
          ON a.attrelid = c.oid AND a.attname = 'tenant_id'
         AND a.attnum > 0 AND NOT a.attisdropped
        WHERE n.nspname = 'core'
          AND c.relkind IN ('r', 'p')
          AND NOT c.relispartition
    LOOP
        EXECUTE format('ALTER TABLE core.%I DROP COLUMN tenant_id', r.relname);
    END LOOP;
END $$;

DROP TABLE IF EXISTS core.tenants CASCADE;
