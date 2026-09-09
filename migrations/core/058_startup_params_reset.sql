UPDATE core.servers
SET config = config - 'startup_params'
WHERE config ? 'startup_params';
