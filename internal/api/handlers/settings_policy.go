package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

func passwordMinLength() int {
	return int(settingsreg.AuthPasswordMinLength.Int())
}

func passwordTooShort(password string) bool {
	return len(password) < passwordMinLength()
}

func passwordTooShortMessage() string {
	return fmt.Sprintf("Пароль должен быть не короче %d символов", passwordMinLength())
}

func attemptsWindow() time.Duration {
	return settingsreg.AuthAttemptsWindow.Duration()
}

func intOf(s *settingsreg.Setting) int {
	return int(s.Int())
}

func (h *Handler) overServerLimit(ctx context.Context, w http.ResponseWriter, table, extra, serverID string, limit *settingsreg.Setting, message string) bool {
	max := limit.Int()
	if max <= 0 {
		return false
	}
	var n int64
	_ = h.dbOf(ctx).QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE server_id = $1"+extra, serverID).Scan(&n)
	if n < max {
		return false
	}
	writeCodedError(w, http.StatusConflict, "server_limit", message)
	return true
}

func (h *Handler) overUserServerLimit(ctx context.Context, w http.ResponseWriter, userID string, add int) bool {
	max := settingsreg.ServersMaxPerUser.Int()
	if max <= 0 {
		return false
	}
	var n int64
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE user_id = $1::uuid`, userID).Scan(&n)
	if n+int64(add) <= max {
		return false
	}
	writeCodedError(w, http.StatusConflict, "servers_limit",
		fmt.Sprintf("Достигнут предел числа серверов на аккаунт: %d", max))
	return true
}

func expiringSoonSQL() string {
	return fmt.Sprintf("make_interval(days => %d)", settingsreg.BillingExpiringSoonDays.Int())
}
