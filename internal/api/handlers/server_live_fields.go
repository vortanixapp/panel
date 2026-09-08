package handlers

import (
	"context"
	"fmt"
	"time"
)

func (h *Handler) deriveProvisioningProgress(ctx context.Context, serverID, provStatus, serverStatus string) map[string]any {
	var cached map[string]any
	if ok, _ := h.cache.GetJSON(ctx, "srv:"+serverID+":provisioning_progress", &cached); ok && len(cached) > 0 {
		return cached
	}

	switch provStatus {
	case "pending":
		return map[string]any{
			"percent": 0, "stage": "pending", "derived": true,
			"message": "Ожидание в очереди...",
		}
	case "provisioning":
		msg := "Установка..."
		if serverStatus == "reinstalling" {
			msg = "Переустановка..."
		} else if serverStatus == "updating" {
			msg = "Обновление..."
		}
		return map[string]any{
			"percent": 0, "stage": "provisioning", "derived": true, "message": msg,
		}
	case "failed":
		return map[string]any{
			"percent": 0, "stage": "failed", "derived": true,
			"message": "Ошибка установки",
		}
	default:
		return nil
	}
}

func (h *Handler) latestServerMetricExtras(ctx context.Context, serverID string) (diskUsedMB, diskTotalMB int, uptime string) {
	points, err := h.cache.GetMetrics(ctx, serverID, 1)
	if err != nil || len(points) == 0 {
		return 0, 0, ""
	}
	latest := points[len(points)-1]
	if v, ok := latest["disk_used_mb"]; ok {
		diskUsedMB = intFromAny(v)
	}
	if v, ok := latest["disk_total_mb"]; ok {
		diskTotalMB = intFromAny(v)
	}
	if v, ok := latest["uptime"]; ok {
		uptime = fmt.Sprint(v)
	}
	if uptime == "" {
		if started, ok := latest["started_at"].(string); ok && started != "" {
			if t, err := time.Parse(time.RFC3339, started); err == nil {
				uptime = formatUptime(time.Since(t))
			}
		}
	}
	return diskUsedMB, diskTotalMB, uptime
}

func (h *Handler) enrichServerLiveFields(ctx context.Context, tenantID, serverID string, item map[string]any) {
	prov, _ := item["provisioning_status"].(string)
	status, _ := item["status"].(string)
	if progress := h.deriveProvisioningProgress(ctx, serverID, prov, status); progress != nil {
		item["provisioning_progress"] = progress
	}

	diskUsed, diskTotal, uptime := h.latestServerMetricExtras(ctx, serverID)
	if uptime != "" {
		item["uptime"] = uptime
	}
	if diskUsed > 0 {
		item["disk_used_mb"] = diskUsed
	}
	if diskTotal > 0 {
		item["disk_total_mb"] = diskTotal
	} else if limits, ok := item["limits"].(map[string]any); ok {
		if mb := intFromAny(limits["disk_mb"]); mb > 0 {
			item["disk_total_mb"] = mb
		}
	}

	if uptime != "" && diskUsed > 0 {
		return
	}
	running := false
	if runtime, _ := item["runtime_status"].(string); runtime == "running" || status == "running" {
		running = true
	}
	if !running && diskUsed > 0 {
		return
	}
	nodeID, err := h.serverNodeID(ctx, tenantID, serverID)
	if err != nil || nodeID == "" {
		return
	}
	limits, _ := item["limits"].(map[string]any)
	result, agentErr := h.agentCommand(ctx, nodeID, serverID, "stats", map[string]any{"limits": limits})
	if agentErr != nil || result == nil {
		return
	}
	if uptime == "" {
		if v, ok := result["uptime"].(string); ok && v != "" {
			item["uptime"] = v
		}
	}
	if diskUsed <= 0 {
		if v, ok := result["disk_used_mb"]; ok {
			item["disk_used_mb"] = intFromAny(v)
		}
	}
	if diskTotal <= 0 {
		if v, ok := result["disk_total_mb"]; ok {
			item["disk_total_mb"] = intFromAny(v)
		}
	}
}

func formatUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Seconds())
	days := sec / 86400
	sec %= 86400
	hours := sec / 3600
	sec %= 3600
	mins := sec / 60
	if days > 0 {
		return fmt.Sprintf("%dd %02dh %02dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
