package gamesettings

import (
	"sort"
	"sync"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

var (
	registryMu sync.RWMutex
	profiles   = map[string]*Profile{}
	registered []*Profile
)

func register(p Profile) {
	registryMu.Lock()
	defer registryMu.Unlock()

	stored := &p
	registered = append(registered, stored)
	for _, key := range p.Games() {
		norm := gamecatalog.Normalize(key)
		if existing, busy := profiles[norm]; busy {
			panic("gamesettings: игра " + norm + " уже обслуживается профилем " + existing.Key)
		}
		profiles[norm] = stored
	}
}

func For(gameKey string) (*Profile, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := profiles[gamecatalog.Normalize(gameKey)]
	return p, ok
}

func Supported(gameKey string) bool {
	_, ok := For(gameKey)
	return ok
}

func Typed(gameKey string) bool {
	p, ok := For(gameKey)
	return ok && len(p.Fields) > 0
}

func All() []*Profile {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Profile, len(registered))
	copy(out, registered)
	return out
}

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
