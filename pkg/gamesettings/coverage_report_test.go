package gamesettings

import (
	"testing"

	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

// Не проверка, а сводка: сколько игр каталога уже описано. Удобно смотреть в
// выводе `go test -v`, когда нужно понять, сколько работы осталось.
func TestCoverageReport(t *testing.T) {
	var linux, typed, small, smallTyped, fields int
	for _, g := range gamecatalog.All() {
		if !gamecatalog.LinuxSupported(g.Key) {
			continue
		}
		linux++
		if Typed(g.Key) {
			typed++
		}
		if gamecatalog.FitsSmallNode(g.Key) {
			small++
			if Typed(g.Key) {
				smallTyped++
			}
		}
	}
	for _, p := range All() {
		fields += len(p.Fields)
	}
	t.Logf("игр с Linux-сервером: %d, из них с полями: %d", linux, typed)
	t.Logf("игр малой ноды: %d, из них с полями: %d", small, smallTyped)
	t.Logf("профилей: %d, полей всего: %d", len(All()), fields)
}
