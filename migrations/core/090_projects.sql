CREATE TABLE IF NOT EXISTS core.projects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    comment    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projects_owner
    ON core.projects(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS core.project_members (
    project_id  UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    tenant_id   UUID REFERENCES core.tenants(id) ON DELETE CASCADE,
    permissions JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_project_members_user
    ON core.project_members(user_id);

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES core.projects(id) ON DELETE SET NULL;

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS comment TEXT NOT NULL DEFAULT '';

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS delete_protection BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_servers_project
    ON core.servers(project_id)
    WHERE project_id IS NOT NULL;
