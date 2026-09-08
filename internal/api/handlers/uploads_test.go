package handlers

import (
	"net/http/httptest"
	"testing"
)

// Адрес аватара складывался из API_PUBLIC_URL и уезжал в базу абсолютным.
// Переменную на проде не задали, а её значение по умолчанию —
// http://localhost:<порт> самого сервера, то есть адрес, по которому браузер
// клиента не достучится никогда.
func TestPublicBaseURLIgnoresLoopbackDefault(t *testing.T) {
	req := httptest.NewRequest("GET", "http://panel.example.ru/v1/account", nil)

	h := &Handler{apiPublicURL: "http://localhost:8080"}
	if got := h.publicBaseURL(req); got != "http://panel.example.ru" {
		t.Errorf("петлевой адрес должен уступать адресу запроса, получено %q", got)
	}

	h = &Handler{apiPublicURL: "https://api.vortanix.app"}
	if got := h.publicBaseURL(req); got != "https://api.vortanix.app" {
		t.Errorf("настроенный адрес обязан побеждать, получено %q", got)
	}
}

func TestPublicBaseURLHonoursProxyHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/v1/account", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")
	req.Header.Set("X-Forwarded-Host", "api.example.ru, internal")

	h := &Handler{apiPublicURL: ""}
	if got := h.publicBaseURL(req); got != "https://api.example.ru" {
		t.Errorf("адрес за прокси разобран неверно: %q", got)
	}
}

func TestUploadPublicURLRepairsStoredAddresses(t *testing.T) {
	req := httptest.NewRequest("GET", "http://panel.example.ru/v1/account", nil)
	h := &Handler{apiPublicURL: "http://localhost:8080"}

	cases := []struct {
		name  string
		in    string
		want  string
		about string
	}{
		{
			name:  "старая запись с localhost чинится",
			in:    "http://localhost:8080/v1/uploads/avatars/08b5ec57.jpg",
			want:  "http://panel.example.ru/v1/uploads/avatars/08b5ec57.jpg",
			about: "ровно то, что лежит в базе у уже загруженных аватаров",
		},
		{
			name: "относительный путь получает адрес",
			in:   "/v1/uploads/avatars/08b5ec57.jpg",
			want: "http://panel.example.ru/v1/uploads/avatars/08b5ec57.jpg",
		},
		{
			name:  "чужой адрес не трогаем",
			in:    "https://lh3.googleusercontent.com/a/photo.jpg",
			want:  "https://lh3.googleusercontent.com/a/photo.jpg",
			about: "аватар может приехать от Google или быть задан пользователем",
		},
		{
			name: "пусто остаётся пустым",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := h.uploadPublicURL(req, c.in); got != c.want {
				t.Errorf("uploadPublicURL(%q) = %q, ожидалось %q. %s", c.in, got, c.want, c.about)
			}
		})
	}

	if h.avatarPublicURL(req, nil) != nil {
		t.Error("отсутствующий аватар должен остаться отсутствующим")
	}
}
