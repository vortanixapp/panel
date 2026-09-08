package netaddr

import "testing"

func TestListenDefaultsToAllInterfaces(t *testing.T) {
	t.Setenv("BIND_ADDR", "")
	if got := Listen("8080"); got != ":8080" {
		t.Fatalf("Listen(8080) = %q, ожидалось \":8080\"", got)
	}
}

func TestListenHonorsBindAddr(t *testing.T) {
	t.Setenv("BIND_ADDR", "127.0.0.1")
	if got := Listen("8085"); got != "127.0.0.1:8085" {
		t.Fatalf("Listen(8085) = %q, ожидалось \"127.0.0.1:8085\"", got)
	}
}
