package handlers

import (
	"io"
	"os"
	"strings"
	"testing"
)

func readSource(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("не открыть %s: %v", path, err)
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("не прочитать %s: %v", path, err)
	}
	return string(raw)
}

func TestStripeWebhookRefusesWithoutSecret(t *testing.T) {
	src := readSource(t, "webhooks_stripe.go")

	if !strings.Contains(src, `if secret == ""`) {
		t.Error("нет явного отказа при незаданном секрете подписи")
	}
	if !strings.Contains(src, "StatusServiceUnavailable") {
		t.Error("ненастроенный шлюз должен отвечать отказом, а не принимать платёж")
	}
	if strings.Contains(src, "accepting unsigned Stripe webhooks") {
		t.Error("вернулась ветка приёма неподписанных вебхуков")
	}
}

func TestForgotPasswordDoesNotReturnToken(t *testing.T) {
	src := readSource(t, "auth_ext.go")

	start := strings.Index(src, "func (h *Handler) ForgotPassword")
	if start < 0 {
		t.Fatal("не найден обработчик ForgotPassword")
	}
	end := strings.Index(src[start:], "\nfunc ")
	if end < 0 {
		end = len(src) - start
	}
	body := src[start : start+end]

	if strings.Contains(body, `"token":`) {
		t.Error("токен сброса снова уходит в ответ — это захват чужого аккаунта")
	}
	if !strings.Contains(body, "mail.PasswordResetBody") {
		t.Error("ссылка восстановления должна уходить письмом")
	}
	if strings.Count(body, `"status": "sent"`) < 2 {
		t.Error("ответ должен быть одинаков и при найденном, и при ненайденном адресе")
	}
}

func TestRefreshIsNotBehindAccessTokenAuth(t *testing.T) {
	src := readSource(t, "routes.go")

	protectedStart := strings.Index(src, "func (h *Handler) mountProtected")
	if protectedStart < 0 {
		t.Fatal("не найден mountProtected")
	}
	protectedEnd := strings.Index(src[protectedStart:], "\nfunc ")
	if protectedEnd < 0 {
		protectedEnd = len(src) - protectedStart
	}
	protected := src[protectedStart : protectedStart+protectedEnd]

	if strings.Contains(protected, `"/v1/auth/refresh"`) {
		t.Error("обновление сессии снова стоит за authWithAPIKey: истёкший access-токен " +
			"нечем будет обменять, и пользователя выкинет на вход при живом refresh-токене")
	}

	publicStart := strings.Index(src, "func (h *Handler) mountPublicAuth")
	if publicStart < 0 {
		t.Fatal("не найден mountPublicAuth")
	}
	if !strings.Contains(src[publicStart:], `r.Post("/v1/auth/refresh", h.Refresh)`) {
		t.Error("обновление сессии должно быть смонтировано рядом с остальными публичными входами")
	}
}

func TestRefreshAuthenticatesByRefreshTokenOnly(t *testing.T) {
	src := readSource(t, "auth_ext.go")

	start := strings.Index(src, "func (h *Handler) Refresh")
	if start < 0 {
		t.Fatal("не найден обработчик Refresh")
	}
	end := strings.Index(src[start:], "\nfunc ")
	if end < 0 {
		end = len(src) - start
	}
	body := src[start : start+end]

	// Раз маршрут публичный, хендлер обязан проверять всё сам.
	if !strings.Contains(body, "ParseRefreshDetails") {
		t.Error("подпись и срок refresh-токена должны проверяться в самом хендлере")
	}
	if !strings.Contains(body, "core.user_sessions") {
		t.Error("закрытая сессия не должна воскресать обновлением токена")
	}
	if !strings.Contains(body, "status = 'active'") {
		t.Error("токен не должен выдаваться заблокированному пользователю")
	}
	if strings.Contains(body, "tenantClaims(") {
		t.Error("публичный хендлер не может опираться на claims access-токена")
	}
}
