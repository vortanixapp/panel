

ALTER TABLE core.jobs
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE core.jobs
    ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE core.jobs
    ADD COLUMN IF NOT EXISTS last_actor_id UUID;

ALTER TABLE core.jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE core.jobs ADD CONSTRAINT jobs_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));

CREATE OR REPLACE FUNCTION core.jobs_touch_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS jobs_touch_updated_at ON core.jobs;
CREATE TRIGGER jobs_touch_updated_at
    BEFORE UPDATE ON core.jobs
    FOR EACH ROW EXECUTE FUNCTION core.jobs_touch_updated_at();

CREATE INDEX IF NOT EXISTS idx_jobs_tenant_status
    ON core.jobs(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_type_status
    ON core.jobs(type, status, created_at);

CREATE INDEX IF NOT EXISTS idx_jobs_running_updated
    ON core.jobs(updated_at)
    WHERE status = 'running';

CREATE OR REPLACE FUNCTION core.try_uuid(v TEXT) RETURNS UUID AS $$
BEGIN
    RETURN v::uuid;
EXCEPTION WHEN others THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
