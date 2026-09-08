package config

import (
	"os"
	"testing"
)

func TestMailDevExposeURLDefaultsOff(t *testing.T) {
	os.Unsetenv("MAIL_DEV_EXPOSE_URL")
	if Load().MailDevExposeURL {
		t.Fatal("MAIL_DEV_EXPOSE_URL по умолчанию должен быть выключен: " +
			"иначе при ненастроенном SMTP ссылка подтверждения почты уходит " +
			"в JSON и позволяет подтвердить чужой адрес")
	}

	t.Setenv("MAIL_DEV_EXPOSE_URL", "true")
	if !Load().MailDevExposeURL {
		t.Fatal("явное включение для локальной разработки не работает")
	}
}

func TestCORSOpenByDefault(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")

	got := Load().CORSOrigins

	if len(got) != 1 || got[0] != "*" {
		t.Fatalf("по умолчанию источники должны быть открыты, получено %v", got)
	}
}

func TestCORSExplicitListStillWorks(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "https://panel.example.ru, https://panel.example.com")

	got := Load().CORSOrigins

	if len(got) != 2 || got[0] != "https://panel.example.ru" || got[1] != "https://panel.example.com" {
		t.Fatalf("явный список должен разбираться как прежде, получено %v", got)
	}
}
