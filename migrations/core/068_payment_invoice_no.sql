CREATE SEQUENCE IF NOT EXISTS core.payment_invoice_seq START WITH 100001;

ALTER TABLE core.payments ADD COLUMN IF NOT EXISTS invoice_no BIGINT;

UPDATE core.payments
SET invoice_no = nextval('core.payment_invoice_seq')
WHERE invoice_no IS NULL;

ALTER TABLE core.payments
    ALTER COLUMN invoice_no SET DEFAULT nextval('core.payment_invoice_seq'),
    ALTER COLUMN invoice_no SET NOT NULL;

ALTER SEQUENCE core.payment_invoice_seq OWNED BY core.payments.invoice_no;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_invoice_no ON core.payments (invoice_no);

CREATE INDEX IF NOT EXISTS idx_payments_provider_payment_id ON core.payments (provider, provider_payment_id);
