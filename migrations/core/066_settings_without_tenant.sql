DELETE FROM core.tenant_settings a
 USING core.tenant_settings b
 WHERE a.key = b.key AND a.ctid < b.ctid;

ALTER TABLE core.tenant_settings DROP CONSTRAINT IF EXISTS tenant_settings_pkey;
ALTER TABLE core.tenant_settings DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE core.tenant_settings ADD PRIMARY KEY (key);

DELETE FROM core.document_counters a
 USING core.document_counters b
 WHERE a.kind = b.kind AND a.ctid < b.ctid;

ALTER TABLE core.document_counters DROP CONSTRAINT IF EXISTS document_counters_pkey;
ALTER TABLE core.document_counters DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE core.document_counters ADD PRIMARY KEY (kind);
