package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

type nodeCapacity struct {
	NodeID       string
	NodeName     string
	Servers      int
	MaxServers   *int
	AllocatedRAM int
	AllocatedCPU float64
	NodeRAMMB    float64
	NodeDiskMB   float64
	FreeDiskMB   float64
	Overcommit   float64
	ReservedRAM  int
	MetricsKnown bool
	DiskQuota    *bool
}

func (h *Handler) loadNodeCapacity(ctx context.Context, tenantID, nodeID string) (*nodeCapacity, error) {
	c := &nodeCapacity{NodeID: nodeID}
	var maxServers *int
	var overcommit float64
	var reserved int
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT name, max_servers, COALESCE(ram_overcommit, 1.0)::float8, COALESCE(reserved_ram_mb, 1024)
		FROM core.nodes WHERE id = $1 AND tenant_id = $2
	`, nodeID, tenantID).Scan(&c.NodeName, &maxServers, &overcommit, &reserved)
	if err != nil {
		return nil, err
	}
	c.MaxServers = maxServers
	c.Overcommit = overcommit
	c.ReservedRAM = reserved

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT limits FROM core.servers WHERE node_id = $1 AND tenant_id = $2
	`, nodeID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if rows.Scan(&raw) != nil {
			continue
		}
		var lim map[string]any
		_ = json.Unmarshal(raw, &lim)
		c.Servers++
		c.AllocatedRAM += intFromLimits(lim, "memory_mb", "ram_mb", "memory")
		if cpu, ok := lim["cpu"].(float64); ok {
			c.AllocatedCPU += cpu
		}
	}

	var ramTotal, diskTotal, diskUsed, diskAvail *string
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
			MAX(text_value) FILTER (WHERE metric_type = 'ram_total'),
			MAX(text_value) FILTER (WHERE metric_type = 'disk_total'),
			MAX(text_value) FILTER (WHERE metric_type = 'disk_used'),
			MAX(text_value) FILTER (WHERE metric_type = 'disk_available')
		FROM (
			SELECT DISTINCT ON (metric_type) metric_type, text_value, measured_at
			FROM core.node_metrics
			WHERE tenant_id = $1 AND node_id = $2
			  AND metric_type IN ('ram_total', 'disk_total', 'disk_used', 'disk_available')
			ORDER BY metric_type, measured_at DESC
		) latest
	`, tenantID, nodeID).Scan(&ramTotal, &diskTotal, &diskUsed, &diskAvail)

	var quotaVal *float64
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT value FROM core.node_metrics
		WHERE tenant_id = $1 AND node_id = $2 AND metric_type = 'disk_quota'
		ORDER BY measured_at DESC LIMIT 1
	`, tenantID, nodeID).Scan(&quotaVal)

	c.DiskQuota = boolPtrFromMetric(quotaVal)

	c.NodeRAMMB = parseHumanSizeMB(strPtr(ramTotal))
	c.NodeDiskMB = parseHumanSizeMB(strPtr(diskTotal))
	c.FreeDiskMB = parseHumanSizeMB(strPtr(diskAvail))
	if c.FreeDiskMB == 0 && c.NodeDiskMB > 0 {
		c.FreeDiskMB = c.NodeDiskMB - parseHumanSizeMB(strPtr(diskUsed))
	}
	c.MetricsKnown = c.NodeRAMMB > 0 || c.NodeDiskMB > 0
	return c, nil
}

func (c *nodeCapacity) freeRAMForNewServers() float64 {
	if c.NodeRAMMB <= 0 {
		return 0
	}
	usable := c.NodeRAMMB*c.Overcommit - float64(c.ReservedRAM)
	return usable - float64(c.AllocatedRAM)
}

func (c *nodeCapacity) slotsLeft() *int {
	if c.MaxServers == nil {
		return nil
	}
	left := *c.MaxServers - c.Servers
	if left < 0 {
		left = 0
	}
	return &left
}

func (h *Handler) AdminNodeCapacity(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")
	ctx := r.Context()

	c, err := h.loadNodeCapacity(ctx, claims.TenantID, nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	freeRAM := c.freeRAMForNewServers()
	games := []map[string]any{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT slug, name FROM core.games
		WHERE tenant_id = $1 AND active = true
		ORDER BY name
	`, claims.TenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slug, name string
			if rows.Scan(&slug, &name) != nil {
				continue
			}
			g, known := gamecatalog.Resolve(slug)
			if !known {
				continue
			}
			ramPer := g.RecRAMMB
			if ramPer <= 0 {
				ramPer = g.MinRAMMB
			}
			if ramPer <= 0 {
				continue
			}
			byRAM := int(math.Floor(freeRAM / float64(ramPer)))
			byDisk := -1
			if g.MinDiskMB > 0 && c.FreeDiskMB > 0 {
				byDisk = int(math.Floor(c.FreeDiskMB / float64(g.MinDiskMB)))
			}
			fits := byRAM
			limitedBy := "памятью"
			if byDisk >= 0 && byDisk < fits {
				fits, limitedBy = byDisk, "диском"
			}
			if slots := c.slotsLeft(); slots != nil && *slots < fits {
				fits, limitedBy = *slots, "лимитом серверов"
			}
			if fits < 0 {
				fits = 0
			}
			games = append(games, map[string]any{
				"slug": slug, "name": name,
				"ram_per_server_mb":  ramPer,
				"disk_per_server_mb": g.MinDiskMB,
				"fits":               fits,
				"limited_by":         limitedBy,
			})
		}
		sort.Slice(games, func(i, j int) bool {
			return games[i]["fits"].(int) > games[j]["fits"].(int)
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"node_id":          c.NodeID,
		"node_name":        c.NodeName,
		"servers":          c.Servers,
		"max_servers":      c.MaxServers,
		"slots_left":       c.slotsLeft(),
		"allocated_ram_mb": c.AllocatedRAM,
		"allocated_cpu":    c.AllocatedCPU,
		"node_ram_mb":      c.NodeRAMMB,
		"node_disk_mb":     c.NodeDiskMB,
		"free_disk_mb":     c.FreeDiskMB,
		"free_ram_mb":      freeRAM,
		"ram_overcommit":   c.Overcommit,
		"reserved_ram_mb":  c.ReservedRAM,
		"metrics_known":    c.MetricsKnown,
		"games":            games,
		"disk_quota":       c.DiskQuota,
	})
}

type nodeCapacityBody struct {
	MaxServers    *int     `json:"max_servers"`
	RAMOvercommit *float64 `json:"ram_overcommit"`
	ReservedRAMMB *int     `json:"reserved_ram_mb"`
}

func (h *Handler) AdminNodeCapacityUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")
	var body nodeCapacityBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.RAMOvercommit != nil && (*body.RAMOvercommit <= 0 || *body.RAMOvercommit > 4) {
		writeError(w, http.StatusBadRequest, "ram_overcommit must be between 0 and 4")
		return
	}
	if body.MaxServers != nil && *body.MaxServers < 0 {
		writeError(w, http.StatusBadRequest, "max_servers must be positive")
		return
	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.nodes SET
			max_servers     = CASE WHEN $3::bool THEN $4::int ELSE max_servers END,
			ram_overcommit  = COALESCE($5, ram_overcommit),
			reserved_ram_mb = COALESCE($6, reserved_ram_mb)
		WHERE id = $1 AND tenant_id = $2
	`, nodeID, claims.TenantID,
		body.MaxServers != nil, body.MaxServers,
		body.RAMOvercommit, body.ReservedRAMMB)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "node.capacity", "node:"+nodeID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) nodeCapacityReason(ctx context.Context, tenantID, nodeID, gameID string) string {
	c, err := h.loadNodeCapacity(ctx, tenantID, nodeID)
	if err != nil {
		return ""
	}
	if slots := c.slotsLeft(); slots != nil && *slots <= 0 {
		return "на локации достигнут лимит серверов"
	}
	if !c.MetricsKnown {
		return ""
	}
	need := 0
	if g, known := gamecatalog.Resolve(strings.TrimSpace(gameID)); known {
		need = g.RecRAMMB
		if need <= 0 {
			need = g.MinRAMMB
		}
	}
	if need <= 0 {
		return ""
	}
	if c.freeRAMForNewServers() < float64(need) {
		return "на локации не хватает памяти для этой игры"
	}
	return ""
}

func boolPtrFromMetric(v *float64) *bool {
	if v == nil {
		return nil
	}
	b := *v > 0
	return &b
}
