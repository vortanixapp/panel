package gamesettings

import (
	"sort"
	"sync"

	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

// Реестр профилей.
//
// Профили регистрируются файлами profiles_*.go через register в init. Так
// добавление игры — это один новый файл, а не правка общей карты, в которой
// легко потерять запятую и вместе с ней игру.
var (
	registryMu sync.RWMutex
	profiles   = map[string]*Profile{} // ключ игры каталога → профиль
	registered []*Profile              // порядок объявления, для обхода в тестах
)

func register(p Profile) {
	registryMu.Lock()
	defer registryMu.Unlock()

	stored := &p
	registered = append(registered, stored)
	for _, key := range p.Games() {
		norm := gamecatalog.Normalize(key)
		if existing, busy := profiles[norm]; busy {
			// Тихо перетереть чужой профиль значило бы получить игру, которая
			// настраивается не своими полями, и заметить это только по жалобе.
			panic("gamesettings: игра " + norm + " уже обслуживается профилем " + existing.Key)
		}
		profiles[norm] = stored
	}
}

// For возвращает профиль игры. Ключ приводится к каноническому виду каталогом,
// поэтому mcpaper, mcspigot и mcjava найдут один и тот же профиль, а отдельного
// словаря синонимов, как раньше на фронте, держать не нужно.
func For(gameKey string) (*Profile, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := profiles[gamecatalog.Normalize(gameKey)]
	return p, ok
}

// Supported сообщает, описаны ли для игры настройки.
func Supported(gameKey string) bool {
	_, ok := For(gameKey)
	return ok
}

// Typed сообщает, есть ли у игры типизированные поля. Профиль может состоять из
// одних файлов: тогда вкладка показывает сырой редактор и параметры запуска, но
// форм не будет.
func Typed(gameKey string) bool {
	p, ok := For(gameKey)
	return ok && len(p.Fields) > 0
}

// All возвращает профили в порядке объявления.
func All() []*Profile {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Profile, len(registered))
	copy(out, registered)
	return out
}

// GroupBySection раскладывает поля профиля по разделам в порядке словаря.
//
// Порядок вкладок задаёт словарь, а не профиль: игре достаточно перечислить поля
// как удобно автору, а последовательность разделов у всех игр одинаковая.
func (p Profile) GroupBySection() []struct {
	Section Section
	Fields  []Field
} {
	byID := map[SectionID][]Field{}
	for _, f := range p.Fields {
		byID[f.Section] = append(byID[f.Section], f)
	}

	ids := make([]SectionID, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return SectionRank(ids[i]) < SectionRank(ids[j])
	})

	out := make([]struct {
		Section Section
		Fields  []Field
	}, 0, len(ids))
	for _, id := range ids {
		out = append(out, struct {
			Section Section
			Fields  []Field
		}{Section: Section{ID: id, Title: SectionTitle(id)}, Fields: byID[id]})
	}
	return out
}
