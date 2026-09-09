ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS service_kind TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS service_id   UUID;

ALTER TABLE core.support_tickets
    DROP CONSTRAINT IF EXISTS support_tickets_service_kind_check;

ALTER TABLE core.support_tickets
    ADD CONSTRAINT support_tickets_service_kind_check
    CHECK (service_kind IN ('', 'server', 'hosting'));

CREATE INDEX IF NOT EXISTS idx_support_tickets_service
    ON core.support_tickets (tenant_id, service_kind, service_id)
    WHERE service_kind <> '';

ALTER TABLE core.support_tickets
    ADD COLUMN IF NOT EXISTS notify BOOLEAN NOT NULL DEFAULT true;
