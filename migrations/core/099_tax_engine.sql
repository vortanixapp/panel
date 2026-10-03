CREATE TABLE IF NOT EXISTS core.tax_rates (
    id          BIGSERIAL PRIMARY KEY,
    country     TEXT NOT NULL CHECK (country ~ '^[A-Z]{2}$'),
    subdivision TEXT NOT NULL DEFAULT '',
    label       TEXT NOT NULL DEFAULT '',
    rate        NUMERIC(6, 3) NOT NULL CHECK (rate >= 0 AND rate <= 100),
    enabled     BOOLEAN NOT NULL DEFAULT false,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (country, subdivision)
);

INSERT INTO core.tax_rates (country, subdivision, label, rate) VALUES
    ('AT', '', 'VAT', 20), ('BE', '', 'VAT', 21), ('BG', '', 'VAT', 20), ('HR', '', 'VAT', 25),
    ('CY', '', 'VAT', 19), ('CZ', '', 'VAT', 21), ('DK', '', 'VAT', 25), ('EE', '', 'VAT', 24),
    ('FI', '', 'VAT', 25.5), ('FR', '', 'VAT', 20), ('DE', '', 'VAT', 19), ('GR', '', 'VAT', 24),
    ('HU', '', 'VAT', 27), ('IE', '', 'VAT', 23), ('IT', '', 'VAT', 22), ('LV', '', 'VAT', 21),
    ('LT', '', 'VAT', 21), ('LU', '', 'VAT', 17), ('MT', '', 'VAT', 18), ('NL', '', 'VAT', 21),
    ('PL', '', 'VAT', 23), ('PT', '', 'VAT', 23), ('RO', '', 'VAT', 21), ('SK', '', 'VAT', 23),
    ('SI', '', 'VAT', 22), ('ES', '', 'VAT', 21), ('SE', '', 'VAT', 25),
    ('GB', '', 'VAT', 20), ('CH', '', 'VAT', 8.1), ('NO', '', 'VAT', 25), ('IS', '', 'VAT', 24),
    ('LI', '', 'VAT', 8.1), ('AL', '', 'VAT', 20), ('AD', '', 'IGI', 4.5), ('BA', '', 'VAT', 17),
    ('ME', '', 'VAT', 21), ('MK', '', 'VAT', 18), ('RS', '', 'VAT', 20),
    ('BY', '', 'VAT', 20), ('KZ', '', 'VAT', 16), ('UA', '', 'VAT', 20), ('AM', '', 'VAT', 20),
    ('AZ', '', 'VAT', 18), ('GE', '', 'VAT', 18), ('KG', '', 'VAT', 12), ('MD', '', 'VAT', 20),
    ('TJ', '', 'VAT', 15), ('TM', '', 'VAT', 15), ('UZ', '', 'VAT', 12)
ON CONFLICT (country, subdivision) DO NOTHING;

INSERT INTO core.tax_rates (country, subdivision, label, rate) VALUES
    ('US', 'AL', 'Sales tax', 4), ('US', 'AK', 'Sales tax', 0), ('US', 'AZ', 'Sales tax', 5.6),
    ('US', 'AR', 'Sales tax', 6.5), ('US', 'CA', 'Sales tax', 7.25), ('US', 'CO', 'Sales tax', 2.9),
    ('US', 'CT', 'Sales tax', 6.35), ('US', 'DE', 'Sales tax', 0), ('US', 'DC', 'Sales tax', 6),
    ('US', 'FL', 'Sales tax', 6), ('US', 'GA', 'Sales tax', 4), ('US', 'HI', 'Sales tax', 4),
    ('US', 'ID', 'Sales tax', 6), ('US', 'IL', 'Sales tax', 6.25), ('US', 'IN', 'Sales tax', 7),
    ('US', 'IA', 'Sales tax', 6), ('US', 'KS', 'Sales tax', 6.5), ('US', 'KY', 'Sales tax', 6),
    ('US', 'LA', 'Sales tax', 4.45), ('US', 'ME', 'Sales tax', 5.5), ('US', 'MD', 'Sales tax', 6),
    ('US', 'MA', 'Sales tax', 6.25), ('US', 'MI', 'Sales tax', 6), ('US', 'MN', 'Sales tax', 6.875),
    ('US', 'MS', 'Sales tax', 7), ('US', 'MO', 'Sales tax', 4.225), ('US', 'MT', 'Sales tax', 0),
    ('US', 'NE', 'Sales tax', 5.5), ('US', 'NV', 'Sales tax', 6.85), ('US', 'NH', 'Sales tax', 0),
    ('US', 'NJ', 'Sales tax', 6.625), ('US', 'NM', 'Sales tax', 4.875), ('US', 'NY', 'Sales tax', 4),
    ('US', 'NC', 'Sales tax', 4.75), ('US', 'ND', 'Sales tax', 5), ('US', 'OH', 'Sales tax', 5.75),
    ('US', 'OK', 'Sales tax', 4.5), ('US', 'OR', 'Sales tax', 0), ('US', 'PA', 'Sales tax', 6),
    ('US', 'RI', 'Sales tax', 7), ('US', 'SC', 'Sales tax', 6), ('US', 'SD', 'Sales tax', 4.2),
    ('US', 'TN', 'Sales tax', 7), ('US', 'TX', 'Sales tax', 6.25), ('US', 'UT', 'Sales tax', 4.85),
    ('US', 'VT', 'Sales tax', 6), ('US', 'VA', 'Sales tax', 4.3), ('US', 'WA', 'Sales tax', 6.5),
    ('US', 'WV', 'Sales tax', 6), ('US', 'WI', 'Sales tax', 5), ('US', 'WY', 'Sales tax', 4)
ON CONFLICT (country, subdivision) DO NOTHING;

CREATE TABLE IF NOT EXISTS core.transaction_taxes (
    transaction_id UUID PRIMARY KEY REFERENCES core.transactions(id) ON DELETE CASCADE,
    country        TEXT NOT NULL DEFAULT '',
    subdivision    TEXT NOT NULL DEFAULT '',
    regime         TEXT NOT NULL,
    rate           NUMERIC(6, 3) NOT NULL DEFAULT 0,
    net            NUMERIC(18, 2) NOT NULL,
    tax            NUMERIC(18, 2) NOT NULL,
    gross          NUMERIC(18, 2) NOT NULL,
    reverse_charge BOOLEAN NOT NULL DEFAULT false,
    buyer_tax_id   TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transaction_taxes_country ON core.transaction_taxes (country, regime);
