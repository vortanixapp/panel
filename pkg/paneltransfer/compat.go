package paneltransfer

import (
	"fmt"

	"github.com/vortanixapp/panel/pkg/updates"
)

func CheckCompatible(m *Manifest, panelVersion string) error {
	if m == nil {
		return fmt.Errorf("в архиве нет описания")
	}
	if m.Schema > Schema {
		return fmt.Errorf("архив создан более новой панелью, обновите панель и повторите")
	}
	if updates.IsSemver(m.PanelVersion) && updates.IsSemver(panelVersion) && updates.IsNewer(m.PanelVersion, panelVersion) {
		return fmt.Errorf("архив снят с панели %s, а здесь установлена %s: обновите панель до %s и повторите",
			m.PanelVersion, panelVersion, m.PanelVersion)
	}
	return nil
}
