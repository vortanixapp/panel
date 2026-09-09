ALTER TABLE core.tenants
    ADD COLUMN IF NOT EXISTS db_name TEXT UNIQUE;

UPDATE core.tenants
   SET db_name = current_database()
 WHERE db_name IS NULL;
