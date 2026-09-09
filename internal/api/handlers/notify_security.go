package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) notifyNewLogin(ctx context.Context, r *http.Request, userID, email string) {
	if userID == "" {
		return
	}
	ip := clientIP(r)
	agent := r.Header.Get("User-Agent")
	if len(agent) > 300 {
		agent = agent[:300]
	}

	var seen bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM core.login_attempts
			WHERE user_id = $1::uuid AND success = true
			  AND ip = $2 AND user_agent = $3
			  AND created_at < now() - interval '5 seconds'
		)
	`, userID, ip, agent).Scan(&seen); err != nil || seen {
		return
	}

	where := ip
	if where == "" {
		where = "неизвестный адрес"
	}
	device := deviceFromUserAgent(agent)

	h.notifyUser(ctx, userID, notify.Event{
		Kind:  notify.KindNewLogin,
		Title: "Вход с нового устройства",
		Body: "В аккаунт " + email + " вошли с адреса " + where + " (" + device + "). " +
			"Если это были не вы — смените пароль и включите двухфакторную аутентификацию.",
		Action: h.panelAction("Проверить сеансы", "/settings?tab=sessions"),
		Meta:   map[string]any{"ip": ip, "user_agent": agent},
	})
}

func deviceFromUserAgent(agent string) string {
	low := strings.ToLower(agent)
	if low == "" {
		return "неизвестное устройство"
	}
	var os string
	switch {
	case strings.Contains(low, "android"):
		os = "Android"
	case strings.Contains(low, "iphone"), strings.Contains(low, "ipad"):
		os = "iOS"
	case strings.Contains(low, "windows"):
		os = "Windows"
	case strings.Contains(low, "mac os"), strings.Contains(low, "macintosh"):
		os = "macOS"
	case strings.Contains(low, "linux"):
		os = "Linux"
	}
	var browser string
	switch {
	case strings.Contains(low, "edg/"):
		browser = "Edge"
	case strings.Contains(low, "opr/"), strings.Contains(low, "opera"):
		browser = "Opera"
	case strings.Contains(low, "yabrowser"):
		browser = "Яндекс.Браузер"
	case strings.Contains(low, "firefox"):
		browser = "Firefox"
	case strings.Contains(low, "chrome"):
		browser = "Chrome"
	case strings.Contains(low, "safari"):
		browser = "Safari"
	}
	switch {
	case os != "" && browser != "":
		return browser + ", " + os
	case browser != "":
		return browser
	case os != "":
		return os
	default:
		return "неизвестное устройство"
	}
}
