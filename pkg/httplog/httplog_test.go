package httplog

import "testing"

func TestRedact(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "токен сессии в query",
			in:   "/v1/dashboard?token=eyJhbGciOiJIUzI1NiJ9.secret.value",
			want: "/v1/dashboard?token=redacted",
		},
		{
			name: "тикет консоли",
			in:   "/v1/console?ticket=1f0b6a1e-0000-4000-8000-000000000001",
			want: "/v1/console?ticket=redacted",
		},
		{
			name: "хеш подтверждения почты в пути",
			in:   "/v1/email/verify/8ac7b702-1437-4fdc-b6f8-a14d23449cf6/9f2b1c",
			want: "/v1/email/verify/8ac7b702-1437-4fdc-b6f8-a14d23449cf6/redacted",
		},
		{
			name: "обычный путь не трогаем",
			in:   "/v1/servers",
			want: "/v1/servers",
		},
		{
			name: "безобидные параметры остаются",
			in:   "/v1/admin/logs?limit=50&action=server.power",
			want: "/v1/admin/logs?action=server.power&limit=50",
		},
		{
			name: "секретный параметр вычищается, соседние — нет",
			in:   "/v1/admin/logs?stream=1&token=abc123",
			want: "/v1/admin/logs?stream=1&token=redacted",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Redact(c.in); got != c.want {
				t.Errorf("Redact(%q) = %q, ожидалось %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactUnparseable(t *testing.T) {
	got := Redact("://broken?token=secret")
	if got == "://broken?token=secret" {
		t.Fatalf("секрет остался в логе: %q", got)
	}
	if want := "://broken?redacted"; got != want {
		t.Errorf("Redact = %q, ожидалось %q", got, want)
	}
}
