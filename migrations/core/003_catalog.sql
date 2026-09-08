

CREATE TABLE core.games (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT,
    image_url       TEXT,
    code            TEXT,
    query_protocol  TEXT,
    min_port        INT NOT NULL DEFAULT 1024,
    max_port        INT NOT NULL DEFAULT 65535,
    active          BOOLEAN NOT NULL DEFAULT true,
    settings_schema JSONB NOT NULL DEFAULT '{}',
    meta            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_games_tenant ON core.games(tenant_id, active);

CREATE TABLE core.game_versions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    game_id       UUID NOT NULL REFERENCES core.games(id) ON DELETE CASCADE,
    version       TEXT NOT NULL,
    source_type   TEXT NOT NULL DEFAULT 'archive'
        CHECK (source_type IN ('archive', 'steam', 'docker')),
    archive_url   TEXT,
    steam_app_id  BIGINT,
    steam_branch  TEXT,
    docker_image  TEXT,
    active        BOOLEAN NOT NULL DEFAULT true,
    sort_order    INT NOT NULL DEFAULT 0,
    meta          JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (game_id, version)
);

CREATE INDEX idx_game_versions_tenant ON core.game_versions(tenant_id, game_id, active);

CREATE TABLE core.tariffs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    node_id           UUID REFERENCES core.nodes(id) ON DELETE SET NULL,
    game_id           UUID REFERENCES core.games(id) ON DELETE SET NULL,
    name              TEXT NOT NULL,
    slug              TEXT NOT NULL,
    billing_type      TEXT NOT NULL DEFAULT 'fixed'
        CHECK (billing_type IN ('fixed', 'resources', 'slots')),
    price_monthly     NUMERIC(12, 2) NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT 'RUB',
    slots_min         INT,
    slots_max         INT,
    cpu_cores         NUMERIC(6, 2),
    cpu_shares        INT,
    ram_mb            INT,
    disk_mb           INT,
    rental_periods    JSONB NOT NULL DEFAULT '[]',
    renewal_periods   JSONB NOT NULL DEFAULT '[]',
    discounts         JSONB NOT NULL DEFAULT '{}',
    position          INT NOT NULL DEFAULT 0,
    active            BOOLEAN NOT NULL DEFAULT true,
    meta              JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_tariffs_tenant ON core.tariffs(tenant_id, active, position);

CREATE TABLE core.plugins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    category        TEXT,
    version         TEXT,
    description     TEXT,
    image_url       TEXT,
    archive_type    TEXT,
    archive_path    TEXT,
    install_path    TEXT NOT NULL DEFAULT '',
    supported_games TEXT[] NOT NULL DEFAULT '{}',
    file_actions    JSONB NOT NULL DEFAULT '[]',
    restart_required BOOLEAN NOT NULL DEFAULT false,
    active          BOOLEAN NOT NULL DEFAULT true,
    meta            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_plugins_tenant ON core.plugins(tenant_id, active);

CREATE TABLE core.maps (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    slug             TEXT NOT NULL,
    name             TEXT NOT NULL,
    category         TEXT,
    game_id          UUID REFERENCES core.games(id) ON DELETE SET NULL,
    version          TEXT,
    archive_path     TEXT,
    file_list        JSONB NOT NULL DEFAULT '[]',
    restart_required BOOLEAN NOT NULL DEFAULT false,
    active           BOOLEAN NOT NULL DEFAULT true,
    meta             JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_maps_tenant ON core.maps(tenant_id, active);

ALTER TABLE core.games ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.game_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.tariffs ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.plugins ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.maps ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_games ON core.games
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_game_versions ON core.game_versions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_tariffs ON core.tariffs
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_plugins ON core.plugins
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_maps ON core.maps
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
