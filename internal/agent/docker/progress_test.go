package docker

import (
	"strings"
	"testing"
)

func TestSteamPercent(t *testing.T) {
	cases := map[string]int{
		"Update state (0x61) downloading, progress: 42.15 (1234 / 5678)": 42,
		"Update state (0x61) downloading, progress: 0.00 (0 / 5678)":     0,
		"Update state (0x81) verifying, progress: 100.00 (5678 / 5678)":  100,
		"Success! App '740' fully installed.":                            -1,
		"Logging in user ... OK":                                         -1,
		"":                                                               -1,
	}
	for line, want := range cases {
		if got := steamPercent(line); got != want {
			t.Errorf("steamPercent(%q) = %d, ожидалось %d", line, got, want)
		}
	}
}

func TestLineScannerSplitsOnCarriageReturn(t *testing.T) {
	var got []string
	l := newLineScanner(func(line string) { got = append(got, line) })
	_, _ = l.Write([]byte("первая\nвтора"))
	_, _ = l.Write([]byte("я\rтретья"))
	_ = l.Close()

	want := []string{"первая", "вторая", "третья"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("получено %v, ожидалось %v", got, want)
	}
}

func TestLineScannerSkipsEmpty(t *testing.T) {
	var got []string
	l := newLineScanner(func(line string) { got = append(got, line) })
	_, _ = l.Write([]byte("\n\n  \r\n"))
	_ = l.Close()
	if len(got) != 0 {
		t.Errorf("пустые строки не должны отдаваться: %v", got)
	}
}

func TestProgressReaderWithoutLengthStaysSilent(t *testing.T) {
	called := false
	r := newProgressReader(strings.NewReader("данные"), -1, func(string, int, string, string) {
		called = true
	})
	buf := make([]byte, 64)
	for {
		if _, err := r.Read(buf); err != nil {
			break
		}
	}
	if called {
		t.Error("без известного размера прогресс сообщать нечего")
	}
}
