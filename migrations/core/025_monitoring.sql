

CREATE TABLE IF NOT EXISTS core.server_monitoring (
    server_id      UUID PRIMARY KEY REFERENCES core.servers(id) ON DELETE CASCADE,
    tenant_id      UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    public_enabled BOOLEAN NOT NULL DEFAULT false,
    title          TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    tags           TEXT[] NOT NULL DEFAULT '{}',
    discord_url    TEXT NOT NULL DEFAULT '',
    website_url    TEXT NOT NULL DEFAULT '',
    show_players   BOOLEAN NOT NULL DEFAULT true,
    show_chart     BOOLEAN NOT NULL DEFAULT true,
    show_incidents BOOLEAN NOT NULL DEFAULT true,
    show_address   BOOLEAN NOT NULL DEFAULT true,
    show_version   BOOLEAN NOT NULL DEFAULT false,
    votes          INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_server_monitoring_public
    ON core.server_monitoring(tenant_id, public_enabled);

CREATE TABLE IF NOT EXISTS core.server_online_points (
    id          BIGSERIAL PRIMARY KEY,
    server_id   UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    ts          TIMESTAMPTZ NOT NULL DEFAULT now(),
    online      INT NOT NULL DEFAULT 0,
    max_players INT NOT NULL DEFAULT 0,
    ping_ms     INT NOT NULL DEFAULT 0,
    tps         DOUBLE PRECISION NOT NULL DEFAULT 0,
    up          BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_server_online_points_server_ts
    ON core.server_online_points(server_id, ts DESC);

CREATE TABLE IF NOT EXISTS core.server_incidents (
    id           BIGSERIAL PRIMARY KEY,
    server_id    UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    level        TEXT NOT NULL DEFAULT 'info'
        CHECK (level IN ('info', 'warn', 'bad')),
    body         TEXT NOT NULL DEFAULT '',
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at  TIMESTAMPTZ,
    auto         BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_server_incidents_server_started
    ON core.server_incidents(server_id, started_at DESC);

CREATE TABLE IF NOT EXISTS core.server_monitoring_visits (
    server_id UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    day       DATE NOT NULL,
    visits    INT NOT NULL DEFAULT 0,
    PRIMARY KEY (server_id, day)
);
