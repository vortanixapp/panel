

ALTER TABLE core.payments
    ADD COLUMN IF NOT EXISTS receipt_number BIGINT;

ALTER TABLE core.payments
    ADD COLUMN IF NOT EXISTS receipt_issued_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_receipt_number
    ON core.payments(tenant_id, receipt_number)
    WHERE receipt_number IS NOT NULL;

CREATE TABLE IF NOT EXISTS core.document_counters (
    tenant_id UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    kind      TEXT NOT NULL,
    value     BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, kind)
);
