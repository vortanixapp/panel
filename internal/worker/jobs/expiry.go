package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/worker/relay"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (r *Runner) ExpiryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.heartbeat.Beat(LoopExpiry)
			r.suspendExpiredServers(ctx)
			r.cleanupTrialServers(ctx)
		}
	}
}

type expiredServer struct {
	id        string
	nodeID    string
	userID    string
	name      string
	gameID    string
	limits    []byte
	status    string
	expiresAt time.Time
}

func (r *Runner) suspendExpiredServers(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, node_id::text, COALESCE(user_id::text, ''),
		       name, game_id, limits, status, expires_at
		FROM core.servers
		WHERE expires_at IS NOT NULL
		  AND expires_at < now()
		  AND suspended_at IS NULL
		ORDER BY expires_at ASC
		LIMIT 200
	`)
	if err != nil {
		return
	}
	var list []expiredServer
	for rows.Next() {
		var s expiredServer
		if rows.Scan(&s.id, &s.nodeID, &s.userID, &s.name, &s.gameID,
			&s.limits, &s.status, &s.expiresAt) == nil {
			list = append(list, s)
		}
	}
	rows.Close()

	for _, s := range list {
		r.suspendServer(ctx, s)
	}
}

func (r *Runner) suspendServer(ctx context.Context, s expiredServer) {
	if _, err := r.db.Exec(ctx, `
		UPDATE core.servers SET suspended_at = now() WHERE id = $1 AND suspended_at IS NULL
	`, s.id); err != nil {
		return
	}

	if s.status != "stopped" && s.status != "error" {
		var lim map[string]any
		_ = json.Unmarshal(s.limits, &lim)
		err := r.relay.SendCommand(ctx, s.nodeID, relay.CommandRequest{
			CommandID: uuid.NewString(),
			Action:    "power",
			ServerID:  s.id,
			Payload: map[string]any{
				"power_action": "stop",
				"name":         s.name,
				"game_id":      s.gameID,
				"limits":       lim,
			},
		})
		if err != nil {
			log.Printf("expiry: server %s: relay stop failed: %v", s.id, err)
		} else {
			_, _ = r.db.Exec(ctx, `UPDATE core.servers SET status = 'stopping' WHERE id = $1`, s.id)
		}
	}

	if s.userID != "" {
		r.notifyUser(ctx, s.userID, notify.Event{
			Kind:      notify.KindServerSuspended,
			Title:     "Сервер приостановлен",
			Body:      "Оплаченный период сервера «" + s.name + "» закончился, сервер остановлен. Продлите аренду, чтобы запустить его снова.",
			Action:    r.serverAction("Продлить", s.id, "/tariff"),
			Meta:      map[string]any{"server_id": s.id},
			DedupeKey: "server.suspended:" + s.id + ":" + s.expiresAt.Format(time.RFC3339),
		})
	}

	meta, _ := json.Marshal(map[string]any{"reason": "expired"})
	_, _ = r.db.Exec(ctx, `
		INSERT INTO core.audit_logs ( user_id, action, resource, meta)
		VALUES ( NULL, 'server.suspend', $1, $2::jsonb)
	`, "server:"+s.id, meta)

	log.Printf("expiry: server %s (%s) suspended", s.id, s.name)
}

const trialKeepAfterSuspend = 72 * time.Hour

func (r *Runner) cleanupTrialServers(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, node_id::text, COALESCE(user_id::text, ''), name
		FROM core.servers
		WHERE is_trial = true
		  AND suspended_at IS NOT NULL
		  AND suspended_at < now() - $1::interval
		LIMIT 50
	`, trialKeepAfterSuspend.String())
	if err != nil {
		return
	}
	type doomed struct{ id, nodeID, userID, name string }
	list := []doomed{}
	for rows.Next() {
		var d doomed
		if rows.Scan(&d.id, &d.nodeID, &d.userID, &d.name) == nil {
			list = append(list, d)
		}
	}
	rows.Close()

	for _, d := range list {
		if err := r.relay.SendCommand(ctx, d.nodeID, relay.CommandRequest{
			CommandID: uuid.NewString(),
			Action:    "destroy",
			ServerID:  d.id,
			Payload:   map[string]any{"wipe": true},
		}); err != nil {
			log.Printf("trial cleanup: сервер %s: агент недоступен: %v", d.id, err)
			continue
		}
		r.enqueueFtpCleanup(ctx, d.id, d.nodeID)
		if _, err := r.db.Exec(ctx, `
			UPDATE core.ip_pools SET status = 'free', server_id = NULL WHERE server_id = $1
		`, d.id); err != nil {
			log.Printf("trial cleanup: адрес сервера %s не возвращён в пул: %v", d.id, err)
		}
		if _, err := r.db.Exec(ctx, `DELETE FROM core.servers WHERE id = $1`, d.id); err != nil {
			continue
		}
		if d.userID != "" {
			r.notifyUser(ctx, d.userID, notify.Event{
				Kind:  notify.KindServerDeleted,
				Title: "Пробный сервер удалён",
				Body:  "Пробный сервер «" + d.name + "» удалён вместе с файлами: аренда не продлевалась. Новый сервер можно арендовать в любой момент.",
				Meta:  map[string]any{"server_name": d.name},
			})
		}
		log.Printf("trial cleanup: удалён пробный сервер %s", d.id)
	}
}
