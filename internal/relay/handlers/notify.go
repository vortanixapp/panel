package handlers

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) notifyServerOwner(ctx context.Context, db *pgxpool.Pool, serverID string, e notify.Event) {
	r, err := notify.LoadServerOwner(ctx, db, serverID)
	if err != nil {
		return
	}
	if _, err := notify.Dispatch(ctx, db, r, e); err != nil {
		log.Printf("relay: оповещение %s по серверу %s: %v", e.Kind, serverID, err)
	}
}

func (h *Handler) serverAction(label, serverID, suffix string) *notify.Action {
	if h.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: h.panelURL + "/servers/" + serverID + suffix}
}

func (h *Handler) notifyNodeOwners(ctx context.Context, db *pgxpool.Pool, nodeID, nodeName string) {
	rows, err := db.Query(ctx, `
		SELECT DISTINCT user_id::text
		FROM core.servers
		WHERE node_id = $1 AND user_id IS NOT NULL
		  AND COALESCE(runtime_status, status) = 'running'
	`, nodeID)
	if err != nil {
		return
	}
	defer rows.Close()

	owners := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			owners = append(owners, id)
		}
	}
	if len(owners) == 0 {
		return
	}

	name := nodeName
	if name == "" {
		name = "локация"
	}
	for _, userID := range owners {
		r, err := notify.LoadRecipient(ctx, db, userID)
		if err != nil {
			continue
		}
		_, _ = notify.Dispatch(ctx, db, r, notify.Event{
			Kind:  notify.KindNodeOffline,
			Title: "Локация недоступна",
			Body: "Связь с локацией «" + name + "» потеряна. Серверы на ней могут быть недоступны. " +
				"Мы уже разбираемся — отдельных действий от вас не требуется.",
			Meta:      map[string]any{"node_id": nodeID, "node_name": name},
			DedupeKey: "node.offline:" + nodeID,
		})
	}
}

func pendingServerOperation(ctx context.Context, rdb *redis.Client, serverID string) bool {
	if rdb == nil {
		return false
	}
	v, err := rdb.Get(ctx, "srv:"+serverID+":status").Result()
	if err != nil {
		return false
	}
	switch v {
	case "stopping", "starting", "reinstalling", "installing":
		return true
	}
	return false
}
