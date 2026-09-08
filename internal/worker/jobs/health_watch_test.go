package jobs

import "testing"

// Размеры дисков приходят строкой: сборщик пишет байты (df -B1), но в старых
// записях встречается «50G». Ошибка в единицах здесь означает письмо «мало
// места» там, где места полтерабайта.
func TestParseSizeBytes(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"53687091200", 53687091200, true},
		{"50G", 50 * 1 << 30, true},
		{"512M", 512 * 1 << 20, true},
		{"2T", 2 * 1 << 40, true},
		{"1.5G", 1.5 * (1 << 30), true},
		{"10GiB", 10 * 1 << 30, true},
		{" 100K ", 100 * 1 << 10, true},
		{"", 0, false},
		{"n/a", 0, false},
		{"-5", 0, false},
	}
	for _, c := range cases {
		got, ok := parseSizeBytes(c.in)
		if ok != c.ok {
			t.Errorf("%q: разобрано=%v, ожидалось %v", c.in, ok, c.ok)
			continue
		}
		if ok && got != c.want {
			t.Errorf("%q: %v байт, ожидалось %v", c.in, got, c.want)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[float64]string{
		512:             "512 Б",
		1536:            "1.5 КБ",
		5 * 1 << 20:     "5.0 МБ",
		50 * 1 << 30:    "50.0 ГБ",
		1.5 * (1 << 40): "1.5 ТБ",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("%v: %q, ожидалось %q", in, got, want)
		}
	}
}
