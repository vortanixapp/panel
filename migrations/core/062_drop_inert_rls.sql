-- Снятие RLS, которая никогда не работала.
--
-- Политики tenant_isolation_* писались под старую схему «одна база на всех
-- арендаторов». Сейчас у каждого арендатора своя база vx_<slug> (см.
-- apps/core-api/internal/tenantdb/registry.go), и этот же каталог миграций
-- накатывается в каждую из них. Внутри такой базы фильтр по tenant_id
-- бессмысленен: чужих строк там просто нет.
--
-- Вдобавок политики и не действовали ни дня: таблицы принадлежат роли
-- vortanix, приложение подключается той же ролью, а владелец таблицы обходит
-- RLS, пока не задан FORCE ROW LEVEL SECURITY — его нигде нет. Проверено:
--   select relrowsecurity, relforcerowsecurity, pg_get_userbyid(relowner)
--     from pg_class where relname = 'servers';
--   -> t | f | vortanix
--
-- Оставлять их — держать ложный сигнал защиты: при чтении миграций кажется,
-- что изоляция арендаторов обеспечена базой, тогда как на деле её держит
-- только руками написанное tenant_id = $N в запросах и разделение по базам.
--
-- Снимаем циклом, а не списком: набор таблиц в базах разных арендаторов
-- может отличаться, если какая-то отстала по миграциям.

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
