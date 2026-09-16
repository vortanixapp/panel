DELETE FROM core.notifications n
USING core.notifications d
WHERE n.dedupe_key <> ''
  AND n.user_id = d.user_id
  AND n.dedupe_key = d.dedupe_key
  AND (n.created_at, n.id) > (d.created_at, d.id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_notifications_user_dedupe
    ON core.notifications (user_id, dedupe_key)
    WHERE dedupe_key <> '';

CREATE INDEX IF NOT EXISTS idx_notifications_user_feed
    ON core.notifications (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON core.notifications (user_id)
    WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_created
    ON core.notifications (created_at);

CREATE INDEX IF NOT EXISTS idx_notifications_node
    ON core.notifications ((meta->>'node_id'), created_at DESC)
    WHERE type IN ('node.offline', 'staff.node_offline');

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user
    ON core.notification_deliveries (user_id, created_at DESC);

ALTER TABLE core.user_notification_channels
    ADD COLUMN IF NOT EXISTS routes JSONB NOT NULL DEFAULT '{}';
