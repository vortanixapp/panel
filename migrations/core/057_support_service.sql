-- Услуга, к которой относится обращение, и уведомления по нему.
--
-- Половина обращений начинается со слов «не работает сервер», и первым ответом
-- оператор всегда спрашивает какой. Поле в форме убирает этот круг переписки,
-- а в очереди показывает, о чём речь, ещё до открытия тикета.
--
-- Ссылка полиморфная: услугой может быть и игровой сервер, и веб-аккаунт, а они
-- лежат в разных таблицах. Поэтому внешнего ключа здесь нет — вместо него вид
-- услуги и её идентификатор. Название разрешается при чтении: хранить его
-- копией значило бы получить расхождение при первом же переименовании сервера.
ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS service_kind TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS service_id   UUID;

ALTER TABLE core.support_tickets
    DROP CONSTRAINT IF EXISTS support_tickets_service_kind_check;

ALTER TABLE core.support_tickets
    ADD CONSTRAINT support_tickets_service_kind_check
    CHECK (service_kind IN ('', 'server', 'hosting'));

CREATE INDEX IF NOT EXISTS idx_support_tickets_service
    ON core.support_tickets (tenant_id, service_kind, service_id)
    WHERE service_kind <> '';

-- Уведомления по конкретному обращению.
--
-- Длинная переписка по мелкому вопросу заваливает почту, и отписаться было
-- нечем — только не читать. Признак на тикете, а не на пользователе: отключать
-- хотят один разговор, а не поддержку целиком.
ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS notify BOOLEAN NOT NULL DEFAULT true;
