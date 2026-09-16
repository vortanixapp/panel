package jobs

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/vortanixapp/panel/pkg/relaytls"
)

func relayTLSPort() string {
	return envOr("RELAY_TLS_PORT", "8443")
}

func secureRelayURL(raw, pin string) (string, string) {
	base := strings.TrimSpace(raw)
	if pin == "" || base == "" {
		return relayWebsocketURL(base), ""
	}
	if strings.HasPrefix(base, "wss://") || strings.HasPrefix(base, "https://") {
		return relayWebsocketURL(base), ""
	}
	port := relayTLSPort()
	if port == "" || port == "0" {
		return relayWebsocketURL(base), ""
	}
	trimmed := base
	for _, scheme := range []string{"wss://", "ws://", "https://", "http://"} {
		trimmed = strings.TrimPrefix(trimmed, scheme)
	}
	if idx := strings.IndexAny(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}
	trimmed = strings.Trim(trimmed, "[]")
	if trimmed == "" {
		return relayWebsocketURL(base), ""
	}
	u := url.URL{Scheme: "wss", Host: net.JoinHostPort(trimmed, port), Path: "/v1/agent/connect"}
	return u.String(), pin
}

func (r *Runner) relayTarget(ctx context.Context) (string, string) {
	raw := envOr("RELAY_PUBLIC_URL", envOr("RELAY_URL", ""))
	return secureRelayURL(raw, relaytls.Pin(ctx, r.db))
}
