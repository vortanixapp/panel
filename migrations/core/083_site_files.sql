CREATE TABLE IF NOT EXISTS core.site_files (
    name        TEXT PRIMARY KEY,
    content     BYTEA NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by  UUID
);
