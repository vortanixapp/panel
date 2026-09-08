-- Категория обращения и база знаний.
--
-- Форма создания тикета спрашивает категорию и приоритет. Приоритет в таблице
-- есть с самого начала, а категории не было: обращения про оплату и про
-- упавший сервер попадали в одну кучу, и разбирать очередь приходилось глазами
-- по теме письма.
--
-- Значение по умолчанию 'other', а не NULL: колонка участвует в фильтре
-- очереди, и строки без категории пришлось бы всюду вылавливать отдельно.
--
-- Без CHECK намеренно. Набор отделов настраивается отдельно для каждой панели
-- (core.tenant_settings, ключ support.departments), и список значений в схеме
-- воевал бы с этой настройкой: у клиента, добавившего свой отдел, вставка
-- падала бы на ограничении.
ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'other';

CREATE INDEX IF NOT EXISTS idx_support_tickets_category
    ON core.support_tickets(tenant_id, category, status);

-- База знаний.
--
-- Половина обращений повторяется дословно, а отвечать на них было нечем, кроме
-- как заново руками: статей в продукте не существовало. Держим их рядом с
-- тикетами и в той же схеме арендатора — у каждой панели свой набор, и статьи
-- одного клиента не должны быть видны другому.
CREATE TABLE IF NOT EXISTS core.kb_articles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    slug         TEXT NOT NULL,
    title        TEXT NOT NULL,
    excerpt      TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    -- Категория из того же набора, что и у тикетов: статья должна находиться
    -- по тому же отделу, в который клиент собирался писать.
    category     TEXT NOT NULL DEFAULT 'other',
    published    BOOLEAN NOT NULL DEFAULT true,
    -- Сколько раз статью открывали. По этому счётчику страница показывает
    -- «частые вопросы»: список, отсортированный руками, устаревает молча.
    views        INT NOT NULL DEFAULT 0,
    position     INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ссылка на статью строится по slug, и он обязан быть уникальным внутри
-- арендатора, но не между ними: у разных панелей может быть своя «оплата».
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_articles_slug
    ON core.kb_articles(tenant_id, slug);

CREATE INDEX IF NOT EXISTS idx_kb_articles_category
    ON core.kb_articles(tenant_id, category, position);

-- Поиск идёт по заголовку и тексту сразу: искать только по заголовку значит не
-- находить статью, названную иначе, чем спрашивает клиент.
CREATE INDEX IF NOT EXISTS idx_kb_articles_search
    ON core.kb_articles USING gin (
        to_tsvector('russian', title || ' ' || excerpt || ' ' || body)
    );

ALTER TABLE core.kb_articles ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_kb_articles ON core.kb_articles;

CREATE POLICY tenant_isolation_kb_articles ON core.kb_articles
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
