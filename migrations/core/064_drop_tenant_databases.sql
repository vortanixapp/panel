DROP TABLE IF EXISTS core.tenant_databases;

ALTER TABLE core.tenants
    DROP COLUMN IF EXISTS db_name;
