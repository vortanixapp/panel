package handlers

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/vortanixapp/panel/pkg/relaytls"
)

func (h *Handler) agentRelayTarget(ctx context.Context, raw string) (string, string) {
	base := strings.TrimSuffix(strings.TrimSpace(raw), "/")
	pin := relaytls.Pin(ctx, h.db)
	plain := strings.HasPrefix(base, "ws://") || strings.HasPrefix(base, "http://")

	if pin != "" && plain {
		port := envOr("RELAY_TLS_PORT", "8443")
		if host := relayHostOnly(base); host != "" && port != "" && port != "0" {
			u := url.URL{Scheme: "wss", Host: net.JoinHostPort(host, port), Path: "/v1/agent/connect"}
			return u.String(), pin
		}
	}

	if !strings.HasPrefix(base, "ws") {
		base = "wss://" + strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	}
	return base + "/v1/agent/connect", ""
}

func relayHostOnly(raw string) string {
	trimmed := raw
	for _, scheme := range []string{"wss://", "ws://", "https://", "http://"} {
		trimmed = strings.TrimPrefix(trimmed, scheme)
	}
	if idx := strings.IndexByte(trimmed, '/'); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}
	return strings.Trim(strings.TrimSpace(trimmed), "[]")
}
