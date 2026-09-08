-- Какой компонент владелец просит обновить.
--
-- Страница обновлений разложена на три вкладки — панель, агенты на нодах,
-- служба обновления, — и «Обновить сейчас» на каждой означает своё. Решение
-- сначала ложится сюда, а отсюда уезжает в сервис лицензий вместе с остальным
-- состоянием установки.
--
-- Пусто означает панель: так вели себя установки до появления вкладок, и
-- решение, записанное старой версией панели, должно сохранить прежний смысл.
ALTER TABLE core.installation
    ADD COLUMN IF NOT EXISTS update_component TEXT NOT NULL DEFAULT '';

ALTER TABLE core.installation
    DROP CONSTRAINT IF EXISTS installation_update_component_check;

ALTER TABLE core.installation
    ADD CONSTRAINT installation_update_component_check
    CHECK (update_component IN ('', 'panel-ui', 'agent', 'updater'));
