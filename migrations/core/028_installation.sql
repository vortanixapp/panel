

CREATE TABLE IF NOT EXISTS core.installation (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),


    singleton          BOOLEAN NOT NULL DEFAULT true UNIQUE CHECK (singleton),

    license_key        TEXT,
    license_token      TEXT,
    token_expires_at   TIMESTAMPTZ,
    license_expires_at TIMESTAMPTZ,
    license_status     TEXT NOT NULL DEFAULT 'unknown'
        CHECK (license_status IN
            ('unknown', 'legacy', 'active', 'past_due', 'suspended', 'revoked', 'expired')),
    plan               TEXT,
    revision           BIGINT NOT NULL DEFAULT 0,



    max_servers        INT,
    max_nodes          INT,
    max_admins         INT,
    api_rpm            INT,



    license_public_key TEXT,

    domain             TEXT,
    activated_at       TIMESTAMPTZ,
    last_verified_at   TIMESTAMPTZ,
    last_attempt_at    TIMESTAMPTZ,
    last_error         TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO core.installation
    (id, singleton, plan, license_status, max_servers, max_nodes,
     activated_at, last_verified_at)
SELECT (t.settings->>'installation_id')::uuid,
       true,
       t.settings->>'plan',
       'legacy',
       NULLIF((t.settings->'limits'->>'max_servers'), '')::int,
       NULLIF((t.settings->'limits'->>'max_nodes'), '')::int,
       now(),
       now()
FROM core.tenants t
WHERE t.settings ? 'installation_id'
  AND (t.settings->>'installation_id') ~
      '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
ORDER BY t.created_at
LIMIT 1
ON CONFLICT DO NOTHING;

INSERT INTO core.installation (singleton) VALUES (true) ON CONFLICT DO NOTHING;
