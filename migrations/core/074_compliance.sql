ALTER TABLE core.users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS retain_until DATE,
    ADD COLUMN IF NOT EXISTS identified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS identification_method TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS identification_note TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_users_deleted ON core.users (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS core.identification_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    action      TEXT NOT NULL CHECK (action IN ('identified', 'revoked')),
    method      TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    payment_id  UUID REFERENCES core.payments(id) ON DELETE SET NULL,
    actor_id    UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_identification_events_user ON core.identification_events (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS core.legal_documents (
    kind                 TEXT NOT NULL CHECK (kind IN ('offer', 'privacy', 'consent', 'cookies')),
    version              INT NOT NULL,
    title                TEXT NOT NULL,
    body                 TEXT NOT NULL,
    requires_acceptance  BOOLEAN NOT NULL DEFAULT true,
    published_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_by         UUID REFERENCES core.users(id) ON DELETE SET NULL,
    PRIMARY KEY (kind, version)
);

CREATE TABLE IF NOT EXISTS core.legal_consents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    version     INT NOT NULL,
    action      TEXT NOT NULL DEFAULT 'accepted' CHECK (action IN ('accepted', 'withdrawn')),
    source      TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_legal_consents_user ON core.legal_consents (user_id, kind, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_legal_consents_created ON core.legal_consents (created_at DESC);

CREATE SEQUENCE IF NOT EXISTS core.abuse_case_seq START WITH 1;

CREATE TABLE IF NOT EXISTS core.abuse_cases (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number       BIGINT NOT NULL DEFAULT nextval('core.abuse_case_seq'),
    source       TEXT NOT NULL CHECK (source IN ('rkn', 'court', 'police', 'copyright', 'abuse', 'other')),
    reference    TEXT NOT NULL DEFAULT '',
    subject      TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    target       TEXT NOT NULL DEFAULT '',
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadline_at  TIMESTAMPTZ,
    server_id    UUID REFERENCES core.servers(id) ON DELETE SET NULL,
    server_name  TEXT NOT NULL DEFAULT '',
    user_id      UUID REFERENCES core.users(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'notified', 'restricted', 'resolved', 'rejected')),
    resolution   TEXT NOT NULL DEFAULT '',
    created_by   UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at    TIMESTAMPTZ
);

ALTER SEQUENCE core.abuse_case_seq OWNED BY core.abuse_cases.number;

CREATE UNIQUE INDEX IF NOT EXISTS uq_abuse_cases_number ON core.abuse_cases (number);
CREATE INDEX IF NOT EXISTS idx_abuse_cases_status ON core.abuse_cases (status, deadline_at);

CREATE TABLE IF NOT EXISTS core.abuse_case_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id     UUID NOT NULL REFERENCES core.abuse_cases(id) ON DELETE CASCADE,
    action      TEXT NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    actor_id    UUID REFERENCES core.users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_abuse_case_events_case ON core.abuse_case_events (case_id, created_at);

CREATE SEQUENCE IF NOT EXISTS core.balance_refund_seq START WITH 1;

CREATE TABLE IF NOT EXISTS core.balance_refund_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number        BIGINT NOT NULL DEFAULT nextval('core.balance_refund_seq'),
    user_id       UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    currency      TEXT NOT NULL,
    amount        NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    refunded      NUMERIC(18, 2) NOT NULL DEFAULT 0,
    method        TEXT NOT NULL CHECK (method IN ('original', 'bank')),
    recipient     TEXT NOT NULL DEFAULT '',
    bank_account  TEXT NOT NULL DEFAULT '',
    bank_bik      TEXT NOT NULL DEFAULT '',
    bank_name     TEXT NOT NULL DEFAULT '',
    reason        TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'rejected', 'cancelled')),
    admin_note    TEXT NOT NULL DEFAULT '',
    reference     TEXT NOT NULL DEFAULT '',
    processed_by  UUID REFERENCES core.users(id) ON DELETE SET NULL,
    processed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER SEQUENCE core.balance_refund_seq OWNED BY core.balance_refund_requests.number;

CREATE UNIQUE INDEX IF NOT EXISTS uq_balance_refund_number ON core.balance_refund_requests (number);
CREATE UNIQUE INDEX IF NOT EXISTS uq_balance_refund_pending ON core.balance_refund_requests (user_id, currency) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS core.receipt_offsets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id  UUID NOT NULL,
    payment_id      UUID REFERENCES core.payments(id) ON DELETE CASCADE,
    amount          NUMERIC(18, 2) NOT NULL DEFAULT 0,
    provider        TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'manual', 'skipped')),
    attempts        INT NOT NULL DEFAULT 0,
    error           TEXT NOT NULL DEFAULT '',
    reference       TEXT NOT NULL DEFAULT '',
    locked_until    TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_receipt_offsets_charge ON core.receipt_offsets
    (transaction_id, COALESCE(payment_id, '00000000-0000-0000-0000-000000000000'::uuid));
CREATE INDEX IF NOT EXISTS idx_receipt_offsets_status ON core.receipt_offsets (status, created_at);
CREATE INDEX IF NOT EXISTS idx_receipt_offsets_payment ON core.receipt_offsets (payment_id) WHERE payment_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS core.bank_statement_lines (
    fingerprint  TEXT PRIMARY KEY,
    doc_number   TEXT NOT NULL DEFAULT '',
    doc_date     DATE,
    amount       NUMERIC(18, 2) NOT NULL,
    payer_name   TEXT NOT NULL DEFAULT '',
    payer_inn    TEXT NOT NULL DEFAULT '',
    purpose      TEXT NOT NULL DEFAULT '',
    payment_id   UUID REFERENCES core.payments(id) ON DELETE SET NULL,
    imported_by  UUID REFERENCES core.users(id) ON DELETE SET NULL,
    imported_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
