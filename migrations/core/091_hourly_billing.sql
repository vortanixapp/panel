ALTER TABLE core.tariffs
    ADD COLUMN IF NOT EXISTS payment_mode TEXT NOT NULL DEFAULT 'prepaid';

ALTER TABLE core.tariffs DROP CONSTRAINT IF EXISTS tariffs_payment_mode_check;
ALTER TABLE core.tariffs ADD CONSTRAINT tariffs_payment_mode_check
    CHECK (payment_mode IN ('prepaid', 'hourly'));

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS payment_mode TEXT NOT NULL DEFAULT 'prepaid';

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS hourly_rate NUMERIC(12, 4) NOT NULL DEFAULT 0;

ALTER TABLE core.servers
    ADD COLUMN IF NOT EXISTS billed_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_servers_hourly_due
    ON core.servers(billed_until)
    WHERE payment_mode = 'hourly';

CREATE TABLE IF NOT EXISTS core.server_hourly_charges (
    server_id    UUID NOT NULL REFERENCES core.servers(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    amount       NUMERIC(12, 4) NOT NULL,
    currency     TEXT NOT NULL DEFAULT 'RUB',
    tx_id        UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (server_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_server_hourly_charges_time
    ON core.server_hourly_charges(created_at DESC);
