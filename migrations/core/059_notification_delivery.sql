-- Доставка оповещений во внешние каналы.
--
-- Уведомление внутри панели и его доставка наружу — разные вещи, и раньше это
-- различие никак не выражалось: h.notify делала один INSERT в core.notifications
-- и на этом заканчивалась. Почта уходила отдельными вызовами из тех мест, где
-- об этом кто-то вспомнил, а Telegram и Discord не уходили никуда, хотя каналы
-- были заведены в базе и предлагались в интерфейсе.
--
-- Очередь нужна именно как таблица, а не как горутина. Отправка наружу может не
-- удаться по причинам, которые пройдут сами: сеть, недоступный SMTP, лимит
-- Telegram. Без записи о попытке такое письмо теряется молча — ровно это сейчас
-- и происходит в биллинге, где при пустом SMTP_HOST письмо помечается
-- отправленным и исчезает.
CREATE TABLE IF NOT EXISTS core.notification_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,

    -- Ссылка на запись в панели. Может быть пустой: бывают оповещения, которые
    -- идут только наружу (например письмо о смене пароля администратором).
    notification_id UUID REFERENCES core.notifications(id) ON DELETE CASCADE,

    kind            TEXT NOT NULL,
    channel         TEXT NOT NULL CHECK (channel IN ('email', 'telegram', 'discord')),

    -- Адрес доставки на момент постановки в очередь. Храним копией намеренно:
    -- если клиент сменит адрес, пока письмо ждёт отправки, оно должно уйти туда,
    -- куда предназначалось, а не туда, где оказался новый адрес.
    target          TEXT NOT NULL,

    subject         TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL DEFAULT '',
    action_label    TEXT NOT NULL DEFAULT '',
    action_href     TEXT NOT NULL DEFAULT '',

    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued', 'sent', 'failed')),
    attempts        INT NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',

    -- Когда можно пробовать снова. Именно этого поля не хватает очереди писем в
    -- биллинге: там упавшая строка остаётся queued и её берут в ближайшем же
    -- проходе, так что пять попыток сгорают за минуты и упираются в тот же сбой.
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индекс под единственный горячий запрос — выборку готовых к отправке.
-- Частичный: отправленное и провалившееся в него не попадает, а их со временем
-- станет подавляющее большинство.
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_queue
    ON core.notification_deliveries (next_attempt_at, id)
    WHERE status = 'queued';

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user
    ON core.notification_deliveries (tenant_id, user_id, created_at DESC);

-- Защита от повторов.
--
-- Остановку сервера за неоплату сейчас сообщают двое: воркер и подметатель в
-- core-api, — и клиент получает два разных уведомления об одном событии. Ключ
-- задаёт источник, и повторная постановка того же ключа отбрасывается.
ALTER TABLE core.notifications
    ADD COLUMN IF NOT EXISTS dedupe_key TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_notifications_dedupe
    ON core.notifications (tenant_id, user_id, dedupe_key)
    WHERE dedupe_key <> '';

-- Действие уведомления отдельными колонками, а не внутри meta.
--
-- Раньше кнопку клали в meta объектом, а читали строкой через fmt.Sprint: в
-- панель приезжало `map[label:Продлить url:…]`, ссылка оставалась пустой, и
-- кнопка не появлялась вовсе. Разнести на две колонки дешевле, чем каждый раз
-- договариваться о форме значения внутри JSONB.
ALTER TABLE core.notifications
    ADD COLUMN IF NOT EXISTS action_label TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS action_href  TEXT NOT NULL DEFAULT '';

-- Индекс под список и под счётчик непрочитанного.
--
-- Прежний idx_notifications_user(tenant_id, user_id, read_at, created_at DESC)
-- не обслуживал ни один запрос: read_at стоит перед created_at, поэтому
-- сортировку по дате индекс не давал, а счётчик непрочитанного вообще ходит без
-- tenant_id и читал таблицу целиком — при опросе колокольчика раз в минуту с
-- каждой вкладки.
CREATE INDEX IF NOT EXISTS idx_notifications_feed
    ON core.notifications (tenant_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread
    ON core.notifications (tenant_id, user_id)
    WHERE read_at IS NULL;
