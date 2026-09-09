DO $$
DECLARE
    r    record;
    cols text;
BEGIN
    FOR r IN
        SELECT c.conname, c.conrelid, c.conkey, c.conrelid::regclass AS tbl
        FROM pg_constraint c
        JOIN pg_namespace n ON n.oid = c.connamespace
        WHERE c.contype = 'u'
          AND n.nspname = 'core'
          AND EXISTS (
              SELECT 1 FROM pg_attribute a
              WHERE a.attrelid = c.conrelid
                AND a.attnum = ANY (c.conkey)
                AND a.attname = 'tenant_id'
          )
    LOOP
        SELECT string_agg(quote_ident(a.attname), ', ' ORDER BY x.ord)
          INTO cols
          FROM unnest(r.conkey) WITH ORDINALITY AS x(attnum, ord)
          JOIN pg_attribute a
            ON a.attrelid = r.conrelid AND a.attnum = x.attnum
         WHERE a.attname <> 'tenant_id';

        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', r.tbl, r.conname);

        IF cols IS NOT NULL THEN
            EXECUTE format('ALTER TABLE %s ADD CONSTRAINT %I UNIQUE (%s)',
                           r.tbl, left(r.conname, 55) || '_single', cols);
        END IF;
    END LOOP;
END $$;

DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT table_name
        FROM information_schema.columns
        WHERE table_schema = 'core'
          AND column_name = 'tenant_id'
          AND is_nullable = 'NO'
          AND NOT EXISTS (
              SELECT 1
              FROM pg_constraint c
              JOIN pg_attribute a
                ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
              WHERE c.contype = 'p'
                AND c.conrelid = ('core.' || quote_ident(table_name))::regclass
                AND a.attname = 'tenant_id'
          )
    LOOP
        EXECUTE format('ALTER TABLE core.%I ALTER COLUMN tenant_id DROP NOT NULL', r.table_name);
    END LOOP;
END $$;
