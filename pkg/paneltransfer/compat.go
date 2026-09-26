package paneltransfer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vortanixapp/panel/pkg/buildinfo"
)

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = buildinfo.Normalize(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if v == "" || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func newer(a, b string) bool {
	left, okA := parseVersion(a)
	right, okB := parseVersion(b)
	if !okA || !okB {
		return false
	}
	for i := 0; i < 3; i++ {
		if left[i] != right[i] {
			return left[i] > right[i]
		}
	}
	return false
}

func CheckCompatible(m *Manifest, panelVersion string) error {
	if m == nil {
		return fmt.Errorf("в архиве нет описания")
	}
	if m.Schema > Schema {
		return fmt.Errorf("архив создан более новой панелью, обновите панель и повторите")
	}
	if newer(m.PanelVersion, panelVersion) {
		return fmt.Errorf("архив снят с панели %s, а здесь установлена %s: обновите панель до %s и повторите",
			m.PanelVersion, panelVersion, m.PanelVersion)
	}
	return nil
}
