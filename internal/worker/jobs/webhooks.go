package jobs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	webhookTimeout     = 10 * time.Second
	webhookMaxAttempts = 6
	webhookInterval    = 30 * time.Second
)

func (r *Runner) WebhookLoop(ctx context.Context) {
	ticker := time.NewTicker(webhookInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.heartbeat.Beat(LoopWebhook)
			r.deliverWebhooks(ctx)
		}
	}
}

type webhookDelivery struct {
	id       string
	url      string
	secret   string
	event    string
	payload  []byte
	attempts int
}

func (r *Runner) deliverWebhooks(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		UPDATE core.webhook_deliveries d
		SET next_retry_at = now() + interval '2 minutes'
		FROM core.webhooks w
		WHERE w.id = d.webhook_id
		  AND d.id IN (
			SELECT dd.id FROM core.webhook_deliveries dd
			JOIN core.webhooks ww ON ww.id = dd.webhook_id
			WHERE dd.status = 'pending'
			  AND (dd.next_retry_at IS NULL OR dd.next_retry_at <= now())
			  AND ww.active = true
			ORDER BY dd.created_at
			LIMIT 50
			FOR UPDATE OF dd SKIP LOCKED
		)
		RETURNING d.id::text, w.url, w.secret, d.event, d.payload, d.attempts
	`)
	if err != nil {
		return
	}
	list := []webhookDelivery{}
	for rows.Next() {
		var d webhookDelivery
		if rows.Scan(&d.id, &d.url, &d.secret, &d.event, &d.payload, &d.attempts) == nil {
			list = append(list, d)
		}
	}
	rows.Close()

	for _, d := range list {
		r.deliverOne(ctx, d)
	}
}

func (r *Runner) deliverOne(ctx context.Context, d webhookDelivery) {
	code, err := postWebhook(ctx, d)
	attempts := d.attempts + 1

	if err == nil {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.webhook_deliveries
			SET status = 'delivered', attempts = $2, response_code = $3,
			    delivered_at = now(), error = NULL
			WHERE id = $1
		`, d.id, attempts, code)
		return
	}

	if attempts >= webhookMaxAttempts {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.webhook_deliveries
			SET status = 'failed', attempts = $2, response_code = $3, error = $4
			WHERE id = $1
		`, d.id, attempts, code, err.Error())
		log.Printf("webhook: доставка %s окончательно не удалась: %v", d.id, err)
		return
	}

	delay := time.Duration(1<<uint(attempts-1)) * time.Minute
	_, _ = r.db.Exec(ctx, `
		UPDATE core.webhook_deliveries
		SET attempts = $2, response_code = $3, error = $4, next_retry_at = now() + $5::interval
		WHERE id = $1
	`, d.id, attempts, code, err.Error(), delay.String())
}

func postWebhook(ctx context.Context, d webhookDelivery) (*int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, webhookTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, d.url, bytes.NewReader(d.payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Vortanix-Webhook/1")
	req.Header.Set("X-Vortanix-Event", d.event)
	req.Header.Set("X-Vortanix-Delivery", d.id)
	mac := hmac.New(sha256.New, []byte(d.secret))
	mac.Write(d.payload)
	req.Header.Set("X-Vortanix-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	code := resp.StatusCode
	if code >= 200 && code < 300 {
		return &code, nil
	}
	return &code, fmt.Errorf("получатель ответил %d", code)
}

func (r *Runner) EmitWebhook(ctx context.Context, tenantID, event string, payload map[string]any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = r.db.Exec(ctx, `
		INSERT INTO core.webhook_deliveries (tenant_id, webhook_id, event, payload, next_retry_at)
		SELECT $1, w.id, $2, $3::jsonb, now()
		FROM core.webhooks w
		WHERE w.tenant_id = $1 AND w.active = true
		  AND (jsonb_array_length(w.events) = 0 OR w.events ? $2)
	`, tenantID, event, body)
}
