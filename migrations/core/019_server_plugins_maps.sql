CREATE TABLE IF NOT EXISTS core.server_plugins (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id     UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    plugin_id     UUID NOT NULL REFERENCES core.plugins(id) ON DELETE CASCADE,
    installed     BOOLEAN NOT NULL DEFAULT false,
    enabled       BOOLEAN NOT NULL DEFAULT true,
    installed_at  TIMESTAMPTZ,
    last_error    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, plugin_id)
);

CREATE INDEX IF NOT EXISTS idx_server_plugins_server ON core.server_plugins(server_id);

CREATE TABLE IF NOT EXISTS core.server_maps (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id     UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    map_id        UUID NOT NULL REFERENCES core.maps(id) ON DELETE CASCADE,
    installed     BOOLEAN NOT NULL DEFAULT false,
    is_active     BOOLEAN NOT NULL DEFAULT false,
    installed_at  TIMESTAMPTZ,
    last_error    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, map_id)
);

CREATE INDEX IF NOT EXISTS idx_server_maps_server ON core.server_maps(server_id);

ALTER TABLE core.server_plugins ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.server_maps ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    CREATE POLICY tenant_isolation_server_plugins ON core.server_plugins
        USING (EXISTS (
            SELECT 1 FROM core.servers s
            WHERE s.id = server_plugins.server_id
              AND s.tenant_id = current_setting('app.tenant_id', true)::uuid
        ));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE POLICY tenant_isolation_server_maps ON core.server_maps
        USING (EXISTS (
            SELECT 1 FROM core.servers s
            WHERE s.id = server_maps.server_id
              AND s.tenant_id = current_setting('app.tenant_id', true)::uuid
        ));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
