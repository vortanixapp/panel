CREATE TABLE IF NOT EXISTS core.node_images (
    node_id    UUID NOT NULL REFERENCES core.nodes(id) ON DELETE CASCADE,
    image_key  TEXT NOT NULL,
    image      TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'building', 'ready', 'failed')),
    error      TEXT NOT NULL DEFAULT '',
    recipe_ref TEXT NOT NULL DEFAULT '',
    queued_at  TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    built_at   TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (node_id, image_key)
);

INSERT INTO core.node_images (node_id, image_key, status, error, built_at, updated_at)
SELECT node_id, game_slug,
       CASE status WHEN 'pending' THEN 'queued' ELSE status END,
       COALESCE(error, ''), built_at, updated_at
FROM core.node_game_images
ON CONFLICT (node_id, image_key) DO NOTHING;

DROP TABLE IF EXISTS core.node_game_images;
