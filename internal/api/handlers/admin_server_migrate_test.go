package handlers

import "testing"

func TestParseHumanSizeMB(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"", 0},
		{"7.6Gi", 7782.4},
		{"48G", 49152},
		{"512M", 512},
		{"512Mi", 512},
		{"2T", 2097152},
		{"1024K", 1},
		{"мусор", 0},
	}
	for _, c := range cases {
		got := parseHumanSizeMB(c.in)
		if diff := got - c.want; diff > 0.5 || diff < -0.5 {
			t.Errorf("parseHumanSizeMB(%q) = %v, ожидалось %v", c.in, got, c.want)
		}
	}
}

func TestIntFromLimits(t *testing.T) {
	if got := intFromLimits(map[string]any{"memory_mb": float64(2048)}, "memory_mb", "ram_mb"); got != 2048 {
		t.Errorf("memory_mb: получено %d", got)
	}
	if got := intFromLimits(map[string]any{"ram_mb": "4096"}, "memory_mb", "ram_mb"); got != 4096 {
		t.Errorf("строковый ram_mb: получено %d", got)
	}
	if got := intFromLimits(map[string]any{}, "memory_mb"); got != 0 {
		t.Errorf("пустые лимиты: получено %d", got)
	}
}
