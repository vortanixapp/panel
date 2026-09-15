CREATE TABLE IF NOT EXISTS core.staff_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    position    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_staff_groups_name ON core.staff_groups (lower(name));

ALTER TABLE core.users ADD COLUMN IF NOT EXISTS staff_group_id UUID REFERENCES core.staff_groups(id) ON DELETE RESTRICT;

ALTER TABLE core.users DROP CONSTRAINT IF EXISTS users_staff_group_role_check;
ALTER TABLE core.users ADD CONSTRAINT users_staff_group_role_check
    CHECK (staff_group_id IS NULL OR role = 'support');

CREATE INDEX IF NOT EXISTS idx_users_staff_group ON core.users (staff_group_id) WHERE staff_group_id IS NOT NULL;
