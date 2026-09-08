

ALTER TABLE core.payments
    ADD COLUMN IF NOT EXISTS refunded_amount NUMERIC(18, 2) NOT NULL DEFAULT 0
        CHECK (refunded_amount >= 0);

ALTER TABLE core.payments
    ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ;

ALTER TABLE core.payments
    ADD COLUMN IF NOT EXISTS refund_reference TEXT;

ALTER TABLE core.payments
    ADD COLUMN IF NOT EXISTS refund_reason TEXT;
