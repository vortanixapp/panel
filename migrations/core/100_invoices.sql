CREATE TABLE IF NOT EXISTS core.invoices (
    transaction_id UUID PRIMARY KEY REFERENCES core.transactions(id) ON DELETE CASCADE,
    series         TEXT NOT NULL,
    number         BIGINT NOT NULL,
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (series, number)
);
