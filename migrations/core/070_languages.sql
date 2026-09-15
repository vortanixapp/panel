CREATE TABLE IF NOT EXISTS core.languages (
    code        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    base        TEXT NOT NULL CHECK (base IN ('ru', 'en')),
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO core.languages (code, name, base, created_at) VALUES
    ('ru', 'Русский', 'ru', '2000-01-01 00:00:00+00'),
    ('en', 'English', 'en', '2000-01-01 00:00:01+00')
ON CONFLICT (code) DO NOTHING;

DELETE FROM core.translation_keys
WHERE btrim(value) = '' OR position('.' IN key) = 0;
