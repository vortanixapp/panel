ALTER TABLE core.promotions
    ADD COLUMN IF NOT EXISTS max_uses_per_user INT CHECK (max_uses_per_user IS NULL OR max_uses_per_user > 0);

CREATE TABLE IF NOT EXISTS core.promotion_uses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID NOT NULL REFERENCES core.promotions(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_promotion_uses_user
    ON core.promotion_uses(promotion_id, user_id);

INSERT INTO core.promotion_uses (promotion_id, user_id, created_at)
SELECT promotion_id, user_id, COALESCE(credited_at, created_at)
FROM core.payments
WHERE promotion_id IS NOT NULL AND status = 'completed'
ON CONFLICT DO NOTHING;
