

ALTER TABLE core.servers DROP CONSTRAINT IF EXISTS servers_status_check;

ALTER TABLE core.servers ADD CONSTRAINT servers_status_check
    CHECK (status IN ('running', 'stopped', 'starting', 'stopping', 'error', 'installing'));
