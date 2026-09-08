package handlers

import (
	"strings"
	"testing"
)

// Соцвход выдавал токены сразу: аккаунт с паролем и включённой 2FA открывался
// соцаккаунтом с тем же адресом, минуя второй фактор.
func TestSocialLoginHonoursTwoFactor(t *testing.T) {
	body := funcBody(t, readSource(t, "auth_social.go"), "loginOrRegisterSocial")

	if !strings.Contains(body, "two_factor_enabled") {
		t.Fatal("соцвход не смотрит на второй фактор")
	}
	if !strings.Contains(body, "requires_2fa") {
		t.Error("соцвход не отправляет на подтверждение кодом")
	}
	if !strings.Contains(body, "recordLoginAttempt") {
		t.Error("соцвход не попадает в журнал попыток — захват останется незаметным")
	}
}

// Смена пароля не рвала чужие сеансы: refresh жил до тридцати дней и продлевался
// сам, поэтому угнанный аккаунт не возвращался сменой пароля.
func TestPasswordChangeClosesOtherSessions(t *testing.T) {
	body := funcBody(t, readSource(t, "me.go"), "ChangePassword")

	if !strings.Contains(body, "DELETE FROM core.user_sessions") {
		t.Fatal("чужие сеансы переживают смену пароля")
	}
	if !strings.Contains(body, "id != $3") {
		t.Error("свой сеанс тоже закрывается — человека выкинет из панели после смены пароля")
	}
}

// Сброс пароля — способ вернуть угнанный аккаунт, поэтому закрываются все
// сеансы без исключения: своего у человека сейчас нет.
func TestPasswordResetClosesAllSessions(t *testing.T) {
	body := funcBody(t, readSource(t, "auth_ext.go"), "ResetPassword")

	if !strings.Contains(body, "DELETE FROM core.user_sessions") {
		t.Error("после сброса пароля старые сеансы остаются у того, кто увёл доступ")
	}
}

// Второй фактор снимался одной кнопкой: тому, кто увёл живой сеанс, этого
// хватало, чтобы убрать защиту.
func TestTwoFactorDisableRequiresPassword(t *testing.T) {
	body := funcBody(t, readSource(t, "account.go"), "Disable2FA")

	if !strings.Contains(body, "CompareHashAndPassword") {
		t.Fatal("отключение второго фактора не спрашивает пароль")
	}
	if !strings.Contains(body, "recordLoginAttempt") {
		t.Error("неудачная попытка отключения не попадает в журнал")
	}
	if !strings.Contains(body, "audit(") {
		t.Error("снятие второго фактора не попадает в аудит")
	}
}
