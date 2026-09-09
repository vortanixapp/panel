CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created
    ON core.audit_logs(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created
    ON core.audit_logs(user_id, created_at DESC)
    WHERE user_id IS NOT NULL;
