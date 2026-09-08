-- Решения об автоматической установке обновлений.
--
-- Ограничение на update_action знало только 'defer' и 'apply_now', поэтому
-- переключатель «Обновлять автоматически» падал на записи решения: панель
-- показывала «не удалось сохранить решение», а причина была в проверке
-- значения, а не в самой ручке.
ALTER TABLE core.installation
    DROP CONSTRAINT IF EXISTS installation_update_action_check;

ALTER TABLE core.installation
    ADD CONSTRAINT installation_update_action_check
    CHECK (update_action IN ('defer', 'apply_now', 'auto_on', 'auto_off'));
