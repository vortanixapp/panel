package jobs

import "testing"

func TestRelayWebsocketURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			// Ровно этот случай ломал установку: в окружении стоит https, а
			// agent открывает вебсокет.
			name: "https переводится в wss",
			in:   "https://relay.vortanix.app",
			want: "wss://relay.vortanix.app/v1/agent/connect",
		},
		{
			name: "http переводится в ws",
			in:   "http://127.0.0.1:8082",
			want: "ws://127.0.0.1:8082/v1/agent/connect",
		},
		{
			name: "готовый адрес не трогаем",
			in:   "wss://relay.vortanix.app/v1/agent/connect",
			want: "wss://relay.vortanix.app/v1/agent/connect",
		},
		{
			name: "без схемы снаружи — wss",
			in:   "relay.vortanix.app",
			want: "wss://relay.vortanix.app/v1/agent/connect",
		},
		{
			name: "без схемы локально — ws",
			in:   "127.0.0.1:8082",
			want: "ws://127.0.0.1:8082/v1/agent/connect",
		},
		{
			name: "пусто — локальный relay",
			in:   "",
			want: "ws://127.0.0.1:8082/v1/agent/connect",
		},
		{
			name: "лишний слэш не удваивается",
			in:   "https://relay.vortanix.app/",
			want: "wss://relay.vortanix.app/v1/agent/connect",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := relayWebsocketURL(tc.in); got != tc.want {
				t.Errorf("relayWebsocketURL(%q) = %q, ожидалось %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRelayUnusableForNode(t *testing.T) {
	cases := []struct {
		name     string
		relay    string
		sshHost  string
		unusable bool
	}{
		{
			// Ровно эта настройка и оставила агента стучаться в себя.
			name:     "петлевой relay при удалённой ноде",
			relay:    "ws://127.0.0.1:8082/v1/agent/connect",
			sshHost:  "200.165.232.23",
			unusable: true,
		},
		{
			name:    "внешний relay при удалённой ноде",
			relay:   "wss://relay.vortanix.app/v1/agent/connect",
			sshHost: "200.165.232.23",
		},
		{
			name:    "всё на одной машине — петля законна",
			relay:   "ws://127.0.0.1:8082/v1/agent/connect",
			sshHost: "127.0.0.1",
		},
		{
			name:    "localhost как имя ноды",
			relay:   "ws://localhost:8082/v1/agent/connect",
			sshHost: "localhost",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := relayUnusableForNode(tc.relay, tc.sshHost); got != tc.unusable {
				t.Errorf("relayUnusableForNode(%q, %q) = %v, ожидалось %v",
					tc.relay, tc.sshHost, got, tc.unusable)
			}
		})
	}
}
