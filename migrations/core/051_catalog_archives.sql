ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS uninstall_actions JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE core.plugins
SET uninstall_actions = meta -> 'uninstall_actions'
WHERE jsonb_typeof(meta -> 'uninstall_actions') = 'array'
  AND uninstall_actions = '[]'::jsonb;

ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS archive_location_id UUID REFERENCES core.nodes(id) ON DELETE SET NULL;
ALTER TABLE core.maps
    ADD COLUMN IF NOT EXISTS archive_location_id UUID REFERENCES core.nodes(id) ON DELETE SET NULL;

ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS archive_size BIGINT;
ALTER TABLE core.maps
    ADD COLUMN IF NOT EXISTS archive_size BIGINT;

ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS image_path TEXT;

ALTER TABLE core.plugins
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE core.maps
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_plugins_archive_location
    ON core.plugins (archive_location_id) WHERE archive_location_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_maps_archive_location
    ON core.maps (archive_location_id) WHERE archive_location_id IS NOT NULL;
