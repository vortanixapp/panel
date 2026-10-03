package handlers

import (
	"fmt"
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

func expiringSoonSQL() string {
	return fmt.Sprintf("make_interval(days => %d)", settingsreg.BillingExpiringSoonDays.Int())
}
