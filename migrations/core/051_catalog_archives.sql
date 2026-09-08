-- Каталог плагинов и карт: архивы и их доставка до локаций.
--
-- В старой панели у плагина было поле uninstall_actions — что сделать с
-- конфигами при удалении плагина с сервера. В новой его не завели, а код
-- читает эти действия из meta->'uninstall_actions' (server_plugins_maps.go).
-- Писать туда было нечем, поэтому действия при удалении терялись целиком.
ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS uninstall_actions JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Переносим то, что могло попасть в meta вручную, чтобы колонка сразу стала
-- источником истины и meta больше не читалась.
UPDATE core.plugins
SET uninstall_actions = meta -> 'uninstall_actions'
WHERE jsonb_typeof(meta -> 'uninstall_actions') = 'array'
  AND uninstall_actions = '[]'::jsonb;

-- Архив плагина или карты хранится у панели, но перед установкой копируется в
-- кэш на самой локации: тянуть сотни мегабайт через панель на каждую установку
-- незачем. Здесь запоминается локация, в кэше которой архив уже лежит, — чтобы
-- при установке на сервер той же локации ничего не копировать заново.
ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS archive_location_id UUID REFERENCES core.nodes(id) ON DELETE SET NULL;
ALTER TABLE core.maps
    ADD COLUMN IF NOT EXISTS archive_location_id UUID REFERENCES core.nodes(id) ON DELETE SET NULL;

-- Размер архива показывается в админке и проверяется при загрузке.
ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS archive_size BIGINT;
ALTER TABLE core.maps
    ADD COLUMN IF NOT EXISTS archive_size BIGINT;

-- Превью плагина: в старой панели хранился путь к файлу, а URL собирался на
-- лету с хешем для сброса кэша. В новой было только image_url строкой, то есть
-- загрузить картинку было нечем.
ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS image_path TEXT;

-- Обеим таблицам не хватало отметки времени изменения: список в админке
-- сортируется по ней, и без неё не видно, что менялось последним.
ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE core.maps
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_plugins_archive_location
    ON core.plugins (archive_location_id) WHERE archive_location_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_maps_archive_location
    ON core.maps (archive_location_id) WHERE archive_location_id IS NOT NULL;
