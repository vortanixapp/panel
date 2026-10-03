CREATE OR REPLACE FUNCTION core.live_invalidate_jobs() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('vx_notifications', '*:inv:jobs');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION core.live_invalidate_server() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('vx_notifications', '*:inv:server:' || NEW.id::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS jobs_live_insert ON core.jobs;
CREATE TRIGGER jobs_live_insert
    AFTER INSERT ON core.jobs
    FOR EACH ROW EXECUTE FUNCTION core.live_invalidate_jobs();

DROP TRIGGER IF EXISTS jobs_live_status ON core.jobs;
CREATE TRIGGER jobs_live_status
    AFTER UPDATE ON core.jobs
    FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION core.live_invalidate_jobs();

DROP TRIGGER IF EXISTS node_tasks_live_insert ON core.node_tasks;
CREATE TRIGGER node_tasks_live_insert
    AFTER INSERT ON core.node_tasks
    FOR EACH ROW EXECUTE FUNCTION core.live_invalidate_jobs();

DROP TRIGGER IF EXISTS node_tasks_live_status ON core.node_tasks;
CREATE TRIGGER node_tasks_live_status
    AFTER UPDATE ON core.node_tasks
    FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION core.live_invalidate_jobs();

DROP TRIGGER IF EXISTS servers_live_status ON core.servers;
CREATE TRIGGER servers_live_status
    AFTER UPDATE ON core.servers
    FOR EACH ROW WHEN (
        OLD.status IS DISTINCT FROM NEW.status
        OR OLD.runtime_status IS DISTINCT FROM NEW.runtime_status
        OR OLD.provisioning_status IS DISTINCT FROM NEW.provisioning_status
    )
    EXECUTE FUNCTION core.live_invalidate_server();
