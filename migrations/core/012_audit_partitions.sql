

DO $$
DECLARE
  part RECORD;
BEGIN
  DROP TABLE IF EXISTS _audit_part_move;
  CREATE TEMP TABLE _audit_part_move ON COMMIT DROP AS
    SELECT * FROM core.audit_logs LIMIT 0;

  FOR part IN
    SELECT * FROM (VALUES
      ('audit_logs_2026_07', '2026-07-01'::timestamptz, '2026-08-01'::timestamptz),
      ('audit_logs_2026_08', '2026-08-01'::timestamptz, '2026-09-01'::timestamptz),
      ('audit_logs_2026_09', '2026-09-01'::timestamptz, '2026-10-01'::timestamptz),
      ('audit_logs_2026_10', '2026-10-01'::timestamptz, '2026-11-01'::timestamptz),
      ('audit_logs_2026_11', '2026-11-01'::timestamptz, '2026-12-01'::timestamptz),
      ('audit_logs_2026_12', '2026-12-01'::timestamptz, '2027-01-01'::timestamptz)
    ) AS t(name, from_ts, to_ts)
  LOOP
    IF to_regclass('core.' || part.name) IS NOT NULL THEN
      CONTINUE;
    END IF;

    TRUNCATE _audit_part_move;
    INSERT INTO _audit_part_move
      SELECT * FROM core.audit_logs
      WHERE created_at >= part.from_ts AND created_at < part.to_ts;

    DELETE FROM core.audit_logs
      WHERE created_at >= part.from_ts AND created_at < part.to_ts;

    EXECUTE format(
      'CREATE TABLE core.%I PARTITION OF core.audit_logs FOR VALUES FROM (%L) TO (%L)',
      part.name, part.from_ts, part.to_ts
    );

    EXECUTE format(
      'INSERT INTO core.%I SELECT * FROM _audit_part_move',
      part.name
    );
  END LOOP;
END $$;
