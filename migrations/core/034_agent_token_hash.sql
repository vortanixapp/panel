ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS agent_token_hash TEXT;

UPDATE core.nodes
SET agent_token_hash = encode(sha256(agent_token::bytea), 'hex')
WHERE agent_token IS NOT NULL
  AND agent_token <> ''
  AND agent_token NOT LIKE 'enc:v1:%'
  AND agent_token_hash IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_agent_token_hash
    ON core.nodes(agent_token_hash)
    WHERE agent_token_hash IS NOT NULL;
