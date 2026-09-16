package handlers

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const nodeOfflineGrace = 2 * time.Minute

func (h *Handler) notifyServerOwner(ctx context.Context, db *pgxpool.Pool, serverID string, e notify.Event) {
	r, err := notify.LoadServerOwner(ctx, db, serverID)
	if err != nil {
		return
	}
	if _, err := notify.Dispatch(ctx, db, r, e); err != nil {
		log.Printf("relay: оповещение %s по серверу %s: %v", e.Kind, serverID, err)
	}
}

func (h *Handler) serverAction(labelKey, serverID, suffix string) *notify.Action {
	if h.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: i18n.Key(labelKey), Href: h.panelURL + "/servers/" + serverID + suffix}
}

func (h *Handler) panelAction(labelKey, path string) *notify.Action {
	if h.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: i18n.Key(labelKey), Href: h.panelURL + path}
}

func nodeLabel(nodeName string) i18n.Msg {
	if nodeName == "" {
		return i18n.Key("notify.node_offline.unnamed")
	}
	return i18n.Raw(nodeName)
}

func (h *Handler) scheduleNodeOffline(db *pgxpool.Pool, nodeID, nodeName string, lastSeen *time.Time) {
	time.AfterFunc(nodeOfflineGrace, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		var still bool
		if err := db.QueryRow(ctx, `
			SELECT status = 'offline' AND last_seen_at IS NOT DISTINCT FROM $2
			FROM core.nodes WHERE id = $1
		`, nodeID, lastSeen).Scan(&still); err != nil || !still {
			return
		}
		h.notifyNodeOffline(ctx, db, nodeID, nodeName, lastSeen)
	})
}

func (h *Handler) notifyNodeOffline(ctx context.Context, db *pgxpool.Pool, nodeID, nodeName string, lastSeen *time.Time) {
	episode := "0"
	if lastSeen != nil {
		episode = strconv.FormatInt(lastSeen.Unix(), 10)
	}
	name := nodeLabel(nodeName)
	meta := map[string]any{"node_id": nodeID, "node_name": nodeName}

	rows, err := db.Query(ctx, `
		SELECT DISTINCT user_id::text
		FROM core.servers
		WHERE node_id = $1 AND user_id IS NOT NULL
		  AND COALESCE(runtime_status, status) = 'running'
	`, nodeID)
	if err == nil {
		var owners []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				owners = append(owners, id)
			}
		}
		rows.Close()
		for _, userID := range owners {
			r, err := notify.LoadRecipient(ctx, db, userID)
			if err != nil {
				continue
			}
			_, _ = notify.Dispatch(ctx, db, r, notify.Event{
				Kind:      notify.KindNodeOffline,
				Title:     i18n.Key("notify.node_offline.title"),
				Body:      i18n.Key("notify.node_offline.body", i18n.Params{"node": name}),
				Meta:      meta,
				DedupeKey: "node.offline:" + nodeID + ":" + episode,
			})
		}
	}

	if _, err := notify.DispatchStaff(ctx, db, notify.StaffAdmins, "", notify.Event{
		Kind:      notify.KindStaffNodeOffline,
		Title:     i18n.Key("notify.staff_node_offline.title", i18n.Params{"node": name}),
		Body:      i18n.Key("notify.staff_node_offline.body", i18n.Params{"node": name}),
		Action:    h.panelAction("notify.action.location", "/admin/locations/"+nodeID),
		Meta:      meta,
		DedupeKey: "staff.node_offline:" + nodeID + ":" + episode,
	}); err != nil {
		log.Printf("relay: оповещение персоналу о недоступности узла %s: %v", nodeID, err)
	}
}

func (h *Handler) notifyNodeRecovered(ctx context.Context, db *pgxpool.Pool, nodeID, nodeName string) {
	rows, err := db.Query(ctx, `
		SELECT n.id::text, n.user_id::text, n.type
		FROM core.notifications n
		WHERE n.type IN ('node.offline', 'staff.node_offline')
		  AND n.meta->>'node_id' = $1
		  AND n.created_at > now() - interval '24 hours'
		  AND NOT EXISTS (
		      SELECT 1 FROM core.notifications o
		      WHERE o.user_id = n.user_id
		        AND o.dedupe_key = 'node.online:' || n.id::text
		  )
	`, nodeID)
	if err != nil {
		log.Printf("relay: поиск оповещений о недоступности узла %s: %v", nodeID, err)
		return
	}
	type offlineNotice struct{ id, userID, kind string }
	var notices []offlineNotice
	for rows.Next() {
		var n offlineNotice
		if rows.Scan(&n.id, &n.userID, &n.kind) == nil {
			notices = append(notices, n)
		}
	}
	rows.Close()

	name := nodeLabel(nodeName)
	meta := map[string]any{"node_id": nodeID, "node_name": nodeName}
	for _, n := range notices {
		r, err := notify.LoadRecipient(ctx, db, n.userID)
		if err != nil {
			continue
		}
		e := notify.Event{
			Kind:      notify.KindNodeOnline,
			Title:     i18n.Key("notify.node_online.title"),
			Body:      i18n.Key("notify.node_online.body", i18n.Params{"node": name}),
			Meta:      meta,
			DedupeKey: "node.online:" + n.id,
		}
		if n.kind == string(notify.KindStaffNodeOffline) {
			e.Kind = notify.KindStaffNodeOnline
			e.Title = i18n.Key("notify.staff_node_online.title", i18n.Params{"node": name})
			e.Body = i18n.Key("notify.staff_node_online.body", i18n.Params{"node": name})
			e.Action = h.panelAction("notify.action.location", "/admin/locations/"+nodeID)
		}
		if _, err := notify.Dispatch(ctx, db, r, e); err != nil {
			log.Printf("relay: оповещение о восстановлении узла %s: %v", nodeID, err)
		}
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
