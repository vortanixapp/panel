CREATE TABLE IF NOT EXISTS core.user_notification_channels (
    user_id          UUID PRIMARY KEY REFERENCES core.users(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    email_enabled    BOOLEAN NOT NULL DEFAULT true,
    telegram_enabled BOOLEAN NOT NULL DEFAULT false,
    discord_enabled  BOOLEAN NOT NULL DEFAULT false,
    telegram_chat_id TEXT NOT NULL DEFAULT '',
    discord_webhook  TEXT NOT NULL DEFAULT '',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_notification_channels_tenant
    ON core.user_notification_channels(tenant_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_action_created
    ON core.audit_logs(tenant_id, action, created_at DESC);

INSERT INTO core.daily_bonus_prizes
    (tenant_id, label, type, value, discount_type, weight, color, icon, prize_duration_hours, sort_order)
SELECT t.id, p.label, p.type, p.value, p.discount_type, p.weight, p.color, p.icon, p.hours, p.sort_order
FROM core.tenants t
CROSS JOIN (VALUES
    ('50 ₽ на баланс',       'balance',       50.00,  NULL,      34, '#3a3b3d', 'ri-coin-line',     NULL, 1),
    ('150 ₽ на баланс',      'balance',       150.00, NULL,      26, '#555658', 'ri-coin-line',     NULL, 2),
    ('3 дня хостинга',       'promo_hosting', 0.00,   NULL,      18, '#757678', 'ri-global-line',   72,   3),
    ('500 ₽ на баланс',      'balance',       500.00, NULL,      14, '#9a9b9d', 'ri-coins-line',    NULL, 4),
    ('Скидка 20% на аренду', 'promo_rent',    20.00,  'percent', 6,  '#c9cacc', 'ri-price-tag-line', NULL, 5),
    ('Месяц веб-хостинга',   'promo_hosting', 0.00,   NULL,      2,  '#e8a03c', 'ri-vip-crown-line', 720,  6)
) AS p(label, type, value, discount_type, weight, color, icon, hours, sort_order)
WHERE NOT EXISTS (
    SELECT 1 FROM core.daily_bonus_prizes d WHERE d.tenant_id = t.id
);
