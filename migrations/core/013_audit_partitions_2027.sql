

DO $$
DECLARE
  rec RECORD;
BEGIN
  FOR rec IN
    SELECT * FROM (VALUES
      ('audit_logs_2027_01', '2027-01-01'::timestamptz, '2027-02-01'::timestamptz),
      ('audit_logs_2027_02', '2027-02-01'::timestamptz, '2027-03-01'::timestamptz),
      ('audit_logs_2027_03', '2027-03-01'::timestamptz, '2027-04-01'::timestamptz),
      ('audit_logs_2027_04', '2027-04-01'::timestamptz, '2027-05-01'::timestamptz),
      ('audit_logs_2027_05', '2027-05-01'::timestamptz, '2027-06-01'::timestamptz),
      ('audit_logs_2027_06', '2027-06-01'::timestamptz, '2027-07-01'::timestamptz)
    ) AS t(name, from_ts, to_ts)
  LOOP
    IF NOT EXISTS (
      SELECT 1 FROM pg_class c
      JOIN pg_namespace n ON n.oid = c.relnamespace
      WHERE n.nspname = 'core' AND c.relname = rec.name
    ) THEN
      EXECUTE format(
        'CREATE TABLE core.%I PARTITION OF core.audit_logs FOR VALUES FROM (%L) TO (%L)',
        rec.name, rec.from_ts, rec.to_ts
      );
    END IF;
  END LOOP;
END $$;
