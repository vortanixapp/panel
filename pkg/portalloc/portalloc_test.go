package portalloc

import "testing"

func TestPickBaseSteps(t *testing.T) {
	busy := map[int]bool{}

	first, ok := pickBase(8211, 8250, 11, busy)
	if !ok || first != 8211 {
		t.Fatalf("первый блок = %d, ok=%v; ожидалось 8211", first, ok)
	}
	for p := first; p < first+11; p++ {
		busy[p] = true
	}

	second, ok := pickBase(8211, 8250, 11, busy)
	if !ok || second != 8222 {
		t.Fatalf("второй блок = %d, ok=%v; ожидалось 8222", second, ok)
	}
}

func TestPickBaseSkipsBusy(t *testing.T) {
	busy := map[int]bool{8221: true}
	base, ok := pickBase(8211, 8250, 11, busy)
	if !ok || base != 8222 {
		t.Fatalf("base = %d, ok=%v; ожидалось 8222", base, ok)
	}
}

func TestPickBaseExhausted(t *testing.T) {
	if base, ok := pickBase(27015, 27020, 11, map[int]bool{}); ok {
		t.Fatalf("в диапазоне 27015-27020 блок из 11 портов не помещается, получено %d", base)
	}

	busy := map[int]bool{}
	for p := 27015; p <= 27030; p++ {
		busy[p] = true
	}
	if base, ok := pickBase(27015, 27030, 1, busy); ok {
		t.Fatalf("свободных портов нет, получено %d", base)
	}
}

func TestPortsForUsesCatalogLayout(t *testing.T) {
	ports := PortsFor("palworld", 8211)
	want := []Port{
		{Port: 8211, Protocol: "udp", Purpose: "игровой сервер"},
		{Port: 8212, Protocol: "udp", Purpose: "Query"},
		{Port: 8221, Protocol: "tcp", Purpose: "RCON"},
	}
	if len(ports) != len(want) {
		t.Fatalf("портов %d, ожидалось %d: %+v", len(ports), len(want), ports)
	}
	for i, w := range want {
		if ports[i] != w {
			t.Errorf("порт %d = %+v, ожидалось %+v", i, ports[i], w)
		}
	}

	cs := PortsFor("cs2", 27015)
	if len(cs) != 2 || cs[0].Port != 27015 || cs[1].Port != 27015 || cs[0].Protocol == cs[1].Protocol {
		t.Errorf("cs2 порты = %+v", cs)
	}

	unknown := PortsFor("unknown-game", 30000)
	if len(unknown) != 1 || unknown[0].Port != 30000 {
		t.Errorf("неизвестная игра = %+v", unknown)
	}
}

func TestPortsForClampsOutOfRange(t *testing.T) {
	for _, p := range PortsFor("mta", 65500) {
		if p.Port > 65535 {
			t.Errorf("порт %d вне допустимого диапазона", p.Port)
		}
	}
}
