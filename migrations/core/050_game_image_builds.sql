ALTER TABLE core.games
    ADD COLUMN IF NOT EXISTS build_image BOOLEAN NOT NULL DEFAULT false;

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
