-- Образы игр собираются на ноде клиента, а не тянутся из реестра: держать 47
-- образов у себя негде, а собрать их на месте дёшево — контекст это Dockerfile
-- и entrypoint, всё остальное приезжает из apt.

-- Отдельный флаг, а не core.games.active: подключить игру к продаже и собирать
-- её образ — разные решения. Игру можно завести заранее, а собрать позже, и
-- наоборот — собрать про запас, не открывая продажу.
ALTER TABLE core.games
    ADD COLUMN IF NOT EXISTS build_image BOOLEAN NOT NULL DEFAULT false;

-- Состояние сборки по каждой ноде: один и тот же образ на разных нодах может
-- быть собран, собираться или упасть с ошибкой.
CREATE TABLE IF NOT EXISTS core.node_game_images (
    tenant_id  UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id    UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    game_slug  TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'building', 'ready', 'failed')),
    error      TEXT,
    built_at   TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (node_id, game_slug)
);

CREATE INDEX IF NOT EXISTS node_game_images_tenant_idx
    ON core.node_game_images (tenant_id, game_slug);
