package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func emitWebhook(ctx context.Context, db *pgxpool.Pool, tenantID, event string, payload map[string]any) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("relay: событие %s не сериализовано: %v", event, err)
		return
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO core.webhook_deliveries (tenant_id, webhook_id, event, payload, next_retry_at)
		SELECT $1, w.id, $2, $3::jsonb, now()
		FROM core.webhooks w
		WHERE w.tenant_id = $1 AND w.active = true
		  AND (jsonb_array_length(w.events) = 0 OR w.events ? $2)
	`, tenantID, event, body); err != nil {
		log.Printf("relay: событие %s не поставлено в очередь: %v", event, err)
	}
}
