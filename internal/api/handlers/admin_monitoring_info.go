package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultGrafanaPort   = "3001"
	defaultGrafanaAddr   = "grafana:3000"
	defaultRetention     = "15d"
	defaultRetentionSize = "2GB"
	monitoringAuthDir    = "/monitoring"
	monitoringProbeLimit = 3 * time.Second
)

type monitoringTarget struct {
	Job   string `json:"job"`
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

type monitoringInfo struct {
	Prometheus struct {
		Ready         bool               `json:"ready"`
		InternalURL   string             `json:"internal_url"`
		Retention     string             `json:"retention"`
		RetentionSize string             `json:"retention_size"`
		Targets       []monitoringTarget `json:"targets"`
	} `json:"prometheus"`
	Grafana struct {
		Ready          bool   `json:"ready"`
		URL            string `json:"url"`
		Port           string `json:"port"`
		Login          string `json:"login"`
		Password       string `json:"password"`
		PasswordSource string `json:"password_source"`
	} `json:"grafana"`
}

func requestHostname(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	if i := strings.Index(host, ","); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return strings.Trim(host, "[]")
}

func formatHostForURL(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

func grafanaPassword() (string, string) {
	if raw, err := os.ReadFile(monitoringAuthDir + "/admin-password"); err == nil {
		if value := strings.TrimSpace(string(raw)); value != "" {
			return value, "generated"
		}
	}
	if value := strings.TrimSpace(os.Getenv("GRAFANA_ADMIN_PASSWORD")); value != "" {
		return value, "env"
	}
	return "", ""
}

func probeOK(ctx context.Context, client *http.Client, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func prometheusTargets(ctx context.Context, client *http.Client) []monitoringTarget {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, prometheusBase()+"/api/v1/targets?state=active", nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			ActiveTargets []struct {
				Labels    map[string]string `json:"labels"`
				Health    string            `json:"health"`
				LastError string            `json:"lastError"`
			} `json:"activeTargets"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return nil
	}
	targets := make([]monitoringTarget, 0, len(out.Data.ActiveTargets))
	for _, t := range out.Data.ActiveTargets {
		targets = append(targets, monitoringTarget{Job: t.Labels["job"], State: t.Health, Error: t.LastError})
	}
	return targets
}

func (h *Handler) AdminMonitoringInfo(w http.ResponseWriter, r *http.Request) {
	var info monitoringInfo
	info.Prometheus.InternalURL = prometheusBase()
	info.Prometheus.Retention = envOr("PROMETHEUS_RETENTION", defaultRetention)
	info.Prometheus.RetentionSize = envOr("PROMETHEUS_RETENTION_SIZE", defaultRetentionSize)
	info.Prometheus.Targets = []monitoringTarget{}

	port := envOr("GRAFANA_PORT", defaultGrafanaPort)
	if _, err := strconv.Atoi(port); err != nil {
		port = defaultGrafanaPort
	}
	info.Grafana.Port = port
	info.Grafana.Login = "admin"
	info.Grafana.URL = "https://" + formatHostForURL(requestHostname(r)) + ":" + port
	info.Grafana.Password, info.Grafana.PasswordSource = grafanaPassword()

	ctx, cancel := context.WithTimeout(r.Context(), monitoringProbeLimit)
	defer cancel()
	plain := &http.Client{Timeout: monitoringProbeLimit}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		info.Prometheus.Ready = probeOK(ctx, plain, prometheusBase()+"/-/ready")
	}()
	go func() {
		defer wg.Done()
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", envOr("GRAFANA_INTERNAL_ADDR", defaultGrafanaAddr))
		if err == nil {
			_ = conn.Close()
			info.Grafana.Ready = true
		}
	}()
	go func() {
		defer wg.Done()
		if targets := prometheusTargets(ctx, plain); targets != nil {
			info.Prometheus.Targets = targets
		}
	}()
	wg.Wait()

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, info)
}
