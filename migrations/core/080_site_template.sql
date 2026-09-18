CREATE TABLE IF NOT EXISTS core.site_template (
    id                SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    draft             JSONB NOT NULL DEFAULT '{}'::jsonb,
    published         JSONB NOT NULL DEFAULT '{}'::jsonb,
    revision          INTEGER NOT NULL DEFAULT 1,
    draft_updated_at  TIMESTAMPTZ,
    draft_updated_by  UUID,
    published_at      TIMESTAMPTZ,
    published_by      UUID
);

CREATE TABLE IF NOT EXISTS core.site_template_versions (
    id          BIGSERIAL PRIMARY KEY,
    document    JSONB NOT NULL,
    texts       JSONB NOT NULL DEFAULT '{}'::jsonb,
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  UUID
);

DO $$
DECLARE
    raw_blocks  jsonb;
    raw_hero    jsonb;
    blocks      jsonb := '{}'::jsonb;
    hero        jsonb := '{}'::jsonb;
    doc         jsonb := '{}'::jsonb;
    landing     jsonb := '[]'::jsonb;
    lang        text;
    section     text;
    hero_field  text;
    phrase      text;
BEGIN
    SELECT s.value INTO raw_blocks FROM core.tenant_settings s WHERE s.key = 'app.site.template.blocks';
    SELECT s.value INTO raw_hero FROM core.tenant_settings s WHERE s.key = 'appearance.landing.hero';

    BEGIN
        IF jsonb_typeof(raw_blocks) = 'string' THEN
            blocks := (raw_blocks #>> '{}')::jsonb;
        ELSIF jsonb_typeof(raw_blocks) = 'object' THEN
            blocks := raw_blocks;
        END IF;
    EXCEPTION WHEN others THEN
        blocks := '{}'::jsonb;
    END;
    IF jsonb_typeof(blocks) IS DISTINCT FROM 'object' THEN
        blocks := '{}'::jsonb;
    END IF;

    IF EXISTS (SELECT 1 FROM jsonb_each(blocks) AS e WHERE e.value = 'false'::jsonb) THEN
        FOREACH section IN ARRAY ARRAY['hero', 'games', 'steps', 'hardware', 'panel', 'locations', 'pricing', 'faq', 'cta'] LOOP
            landing := landing || jsonb_build_array(
                jsonb_build_object('id', section, 'type', 'landing.' || section)
                || CASE WHEN blocks -> section = 'false'::jsonb THEN jsonb_build_object('hidden', true) ELSE '{}'::jsonb END
            );
        END LOOP;
        doc := jsonb_build_object('pages', jsonb_build_object('/', jsonb_build_object('blocks', landing)));
    END IF;

    INSERT INTO core.site_template (id, draft, published, published_at)
    VALUES (1, doc, doc, now())
    ON CONFLICT (id) DO NOTHING;

    BEGIN
        IF jsonb_typeof(raw_hero) = 'string' THEN
            hero := (raw_hero #>> '{}')::jsonb;
        ELSIF jsonb_typeof(raw_hero) = 'object' THEN
            hero := raw_hero;
        END IF;
    EXCEPTION WHEN others THEN
        hero := '{}'::jsonb;
    END;
    IF jsonb_typeof(hero) IS DISTINCT FROM 'object' THEN
        hero := '{}'::jsonb;
    END IF;

    SELECT l.code INTO lang
    FROM core.languages l
    WHERE l.code = COALESCE(
        NULLIF(btrim((SELECT s.value #>> '{}' FROM core.tenant_settings s WHERE s.key = 'i18n.default_locale')), ''),
        'ru'
    );
    IF lang IS NULL THEN
        lang := 'ru';
    END IF;

    FOREACH hero_field IN ARRAY ARRAY['badge', 'title', 'subtitle', 'cta_primary', 'cta_secondary'] LOOP
        phrase := btrim(COALESCE(hero ->> hero_field, ''));
        IF phrase <> '' THEN
            INSERT INTO core.translation_keys (locale, key, value)
            VALUES (lang, 'landing.hero.' || hero_field, phrase)
            ON CONFLICT (locale, key) DO NOTHING;
        END IF;
    END LOOP;

    DELETE FROM core.tenant_settings WHERE key IN ('app.site.template.blocks', 'appearance.landing.hero');
END $$;
