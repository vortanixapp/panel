package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/vortanixapp/panel/pkg/notify"
)

// notifyNewLogin сообщает владельцу аккаунта о входе с незнакомого устройства.
//
// Событие security.new_login было объявлено в справочнике, но не отправлялось
// ниоткуда: вход с чужого адреса выглядел для владельца ровно так же, как его
// собственный. Письмо и есть тот сигнал, по которому успевают сменить пароль.
//
// «Незнакомый» считается по паре адрес плюс браузер: тот же адрес с другого
// устройства и тот же браузер с другого адреса одинаково стоят внимания.
func (h *Handler) notifyNewLogin(ctx context.Context, r *http.Request, tenantID, userID, email string) {
	if tenantID == "" || userID == "" {
		return
	}
	ip := clientIP(r)
	agent := r.Header.Get("User-Agent")
	if len(agent) > 300 {
		agent = agent[:300]
	}

	// Текущая попытка уже записана recordLoginAttempt, поэтому ищем совпадение
	// среди прежних: без исключения по времени сравнение шло бы с самим собой.
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

	h.notifyUser(ctx, tenantID, userID, notify.Event{
		Kind:  notify.KindNewLogin,
		Title: "Вход с нового устройства",
		Body: "В аккаунт " + email + " вошли с адреса " + where + " (" + device + "). " +
			"Если это были не вы — смените пароль и включите двухфакторную аутентификацию.",
		Action: h.panelAction("Проверить сеансы", "/settings?tab=sessions"),
		Meta:   map[string]any{"ip": ip, "user_agent": agent},
	})
}

// deviceFromUserAgent — короткое человеческое описание вместо всей строки
// браузера: в письме от неё пользы нет, а место она занимает всё.
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
	// Порядок важен: Edge и Opera представляются ещё и как Chrome, а Chrome —
	// как Safari. При обратном порядке всё превратилось бы в Safari.
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
