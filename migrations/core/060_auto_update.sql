-- Состояние переключателя автообновления, полученное от сервиса лицензий.
--
-- Хранится у панели, чтобы вкладка обновлений рисовала переключатель сразу, не
-- дожидаясь очередной сверки с сервисом.
ALTER TABLE core.installation
    ADD COLUMN IF NOT EXISTS update_auto BOOLEAN NOT NULL DEFAULT false;
