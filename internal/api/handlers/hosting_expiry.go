package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/vortanixapp/panel/internal/api/hosting"
	"github.com/vortanixapp/panel/pkg/notify"
)

const hostingExpiryInterval = 5 * time.Minute

func (h *Handler) StartHostingExpirySweeper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hostingExpiryInterval)
		defer ticker.Stop()
		h.suspendExpiredHosting(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.suspendExpiredHosting(ctx)
			}
		}
	}()
}

func (h *Handler) suspendExpiredHosting(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, hostingExpiryInterval)
	defer cancel()

	rows, err := h.dbOf(ctx).Query(ctx, `
		UPDATE core.hosting_accounts ha
		SET suspended_at = now(), status = 'suspended', updated_at = now()
		WHERE ha.id IN (
			SELECT id FROM core.hosting_accounts
			WHERE expires_at IS NOT NULL
			  AND expires_at < now()
			  AND suspended_at IS NULL
			  AND status = 'active'
			ORDER BY expires_at ASC
			LIMIT 100
			FOR UPDATE SKIP LOCKED
		)
		RETURNING ha.id::text, COALESCE(ha.user_id::text, ''),
		          COALESCE(ha.panel_account_id, ha.username), ha.username,
		          ha.hosting_server_id::text
	`)
	if err != nil {
		log.Printf("hosting expiry: query: %v", err)
		return
	}
	type expiredAccount struct {
		id, userID, panelID, username, serverID string
	}
	var list []expiredAccount
	for rows.Next() {
		var a expiredAccount
		if rows.Scan(&a.id, &a.userID, &a.panelID, &a.username, &a.serverID) == nil {
			list = append(list, a)
		}
	}
	rows.Close()

	for _, a := range list {
		cfg, err := h.hostingServerConfig(ctx, a.serverID)
		if err == nil {
			if err := hosting.NewAdapter(cfg).Suspend(ctx, a.panelID, "expired"); err != nil {
				log.Printf("hosting expiry: account %s: panel suspend failed: %v", a.id, err)
			}
		}

		if a.userID != "" {
			h.notifyUser(ctx, a.userID, notify.Event{
				Kind:      notify.KindServerSuspended,
				Title:     "Хостинг приостановлен",
				Body:      "Оплаченный период аккаунта «" + a.username + "» закончился. Продлите его, чтобы сайт снова открывался.",
				Action:    h.panelAction("Продлить", "/hosting/"+a.id),
				Meta:      map[string]any{"hosting_id": a.id},
				DedupeKey: "hosting.suspended:" + a.id,
			})
		}

		meta, _ := json.Marshal(map[string]any{"reason": "expired"})
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.audit_logs ( user_id, action, resource, meta)
			VALUES ( NULL, 'hosting.suspend', $1, $2::jsonb)
		`, "hosting_account:"+a.id, meta)

		log.Printf("hosting expiry: account %s (%s) suspended", a.id, a.username)
	}
}

func (h *Handler) hostingServerConfig(ctx context.Context, hostingServerID string) (hosting.ServerConfig, error) {
	var cfg hosting.ServerConfig
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT panel_type, api_url, COALESCE(api_username, ''), COALESCE(api_token_enc, '')
		FROM core.hosting_servers WHERE id = $1
	`, hostingServerID).Scan(&cfg.PanelType, &cfg.APIURL, &cfg.APIUsername, &cfg.APIToken)
	if err != nil {
		return cfg, err
	}
	cfg.APIToken = h.secrets.MustDecrypt(cfg.APIToken)
	return cfg, nil
}
