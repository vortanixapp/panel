DO $$
DECLARE
    stmt text;
BEGIN
    FOREACH stmt IN ARRAY ARRAY[
        'CREATE UNIQUE INDEX IF NOT EXISTS uq_servers_idempotency ON core.servers (idempotency_key) WHERE idempotency_key IS NOT NULL',
        'CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_receipt_number ON core.payments (receipt_number) WHERE receipt_number IS NOT NULL',
        'CREATE UNIQUE INDEX IF NOT EXISTS uq_kb_articles_slug ON core.kb_articles (slug)',
        'CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email ON core.users (lower(email))'
    ]
    LOOP
        BEGIN
            EXECUTE stmt;
        EXCEPTION WHEN others THEN
            RAISE NOTICE 'пропущено (%): %', SQLERRM, stmt;
        END;
    END LOOP;
END $$;

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user ON core.password_reset_tokens (user_id) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON core.user_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_created ON core.login_attempts (ip, created_at DESC) WHERE success = false;
