CREATE TABLE IF NOT EXISTS core.user_billing_profiles (
    user_id     UUID PRIMARY KEY REFERENCES core.users(id) ON DELETE CASCADE,
    payer_type  TEXT NOT NULL DEFAULT 'person' CHECK (payer_type IN ('person', 'ip', 'company')),
    legal_name  TEXT NOT NULL DEFAULT '',
    inn         TEXT NOT NULL DEFAULT '',
    kpp         TEXT NOT NULL DEFAULT '',
    ogrn        TEXT NOT NULL DEFAULT '',
    address     TEXT NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_billing_profiles_inn ON core.user_billing_profiles (inn) WHERE inn <> '';

CREATE SEQUENCE IF NOT EXISTS core.billing_act_seq START WITH 1;

CREATE TABLE IF NOT EXISTS core.billing_acts (
    user_id     UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    period      DATE NOT NULL,
    currency    TEXT NOT NULL,
    number      BIGINT NOT NULL DEFAULT nextval('core.billing_act_seq'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, period, currency)
);

ALTER SEQUENCE core.billing_act_seq OWNED BY core.billing_acts.number;

CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_acts_number ON core.billing_acts (number);
CREATE INDEX IF NOT EXISTS idx_transactions_type_created ON core.transactions (type, created_at);
CREATE INDEX IF NOT EXISTS idx_payments_credited_at ON core.payments (credited_at) WHERE credited_at IS NOT NULL;
