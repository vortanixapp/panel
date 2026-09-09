DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT schemaname, tablename, policyname
          FROM pg_policies
         WHERE schemaname = 'core'
           AND policyname LIKE 'tenant_isolation_%'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I',
                       r.policyname, r.schemaname, r.tablename);
    END LOOP;

    FOR r IN
        SELECT n.nspname AS schemaname, c.relname AS tablename
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'core'
           AND c.relkind = 'r'
           AND c.relrowsecurity
    LOOP
        EXECUTE format('ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY',
                       r.schemaname, r.tablename);
    END LOOP;
END $$;
