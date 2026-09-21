package gameconsole

import (
	"regexp"
	"sync"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

type Kind string

const (
	KindInfo  Kind = "info"
	KindWarn  Kind = "warn"
	KindError Kind = "error"
	KindChat  Kind = "chat"
	KindJoin  Kind = "join"
	KindLeave Kind = "leave"
	KindReady Kind = "ready"
	KindStop  Kind = "stop"
	KindMap   Kind = "map"
)

type Rule struct {
	Kind    Kind   `json:"kind"`
	Pattern string `json:"pattern"`
	Flags   string `json:"flags,omitempty"`
}

func (r Rule) goPattern() string {
	if r.Flags == "" {
		return r.Pattern
	}
	return "(?" + r.Flags + ")" + r.Pattern
}

type Arg struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
}

type Command struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Template string `json:"template"`
	Hint     string `json:"hint,omitempty"`
	Args     []Arg  `json:"args,omitempty"`
	Danger   bool   `json:"danger,omitempty"`
}

type Profile struct {
	Key      string   `json:"key"`
	SharedBy []string `json:"-"`

	Title     string    `json:"title"`
	Note      string    `json:"note,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
	Trim      []string  `json:"trim,omitempty"`
	Rules     []Rule    `json:"rules"`
	Commands  []Command `json:"commands,omitempty"`
}

func (p Profile) Games() []string {
	return append([]string{p.Key}, p.SharedBy...)
}

var (
	registryMu sync.RWMutex
	profiles   = map[string]*Profile{}
	registered []*Profile
)

func register(p Profile) {
	for i, rule := range p.Rules {
		p.Rules[i] = normalizeRule(rule)
	}
	if p.Timestamp != "" {
		regexp.MustCompile(p.Timestamp)
	}
	for _, pattern := range p.Trim {
		regexp.MustCompile(pattern)
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	stored := &p
	registered = append(registered, stored)
	for _, key := range p.Games() {
		norm := gamecatalog.Normalize(key)
		if existing, busy := profiles[norm]; busy {
			panic("gameconsole: игра " + norm + " уже обслуживается профилем " + existing.Key)
		}
		profiles[norm] = stored
	}
}

var inlineFlags = regexp.MustCompile(`^\(\?([a-zA-Z]+)\)`)

func normalizeRule(r Rule) Rule {
	if m := inlineFlags.FindStringSubmatch(r.Pattern); m != nil {
		r.Flags = m[1]
		r.Pattern = r.Pattern[len(m[0]):]
	}
	regexp.MustCompile(r.goPattern())
	return r
}

func For(gameKey string) *Profile {
	registryMu.RLock()
	p, ok := profiles[gamecatalog.Normalize(gameKey)]
	registryMu.RUnlock()
	if ok {
		return p
	}
	return Generic()
}

func Known(gameKey string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := profiles[gamecatalog.Normalize(gameKey)]
	return ok
}

func All() []*Profile {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Profile, len(registered))
	copy(out, registered)
	return out
}

var (
	genericOnce sync.Once
	genericBase *Profile
)

func Generic() *Profile {
	genericOnce.Do(func() {
		rules := genericRules()
		for i, rule := range rules {
			rules[i] = normalizeRule(rule)
		}
		genericBase = &Profile{
			Key:       "generic",
			Title:     "Общий разбор",
			Timestamp: genericTimestamp,
			Trim:      []string{genericTrim},
			Rules:     rules,
		}
	})
	return genericBase
}

const genericTimestamp = `^\[?(?:\d{4}[-./]\d{2}[-./]\d{2}[ T_])?(?<time>\d{1,2}:\d{2}:\d{2})(?:[.,:]\d+)?(?:\s+(?:INFO|WARN|WARNING|ERROR|SEVERE|FATAL|DEBUG|TRACE|NOTICE)\b)?\]?:?\s*`

const genericTrim = `^\[(?:[^\]]*/)?(?:INFO|WARN|WARNING|ERROR|SEVERE|FATAL|DEBUG|TRACE|NOTICE|CRITICAL)\]:?\s*`

func genericRules() []Rule {
	return []Rule{
		{Kind: KindError, Pattern: `\[(?:[^\]]*/)?(?:ERROR|SEVERE|FATAL)\]`},
		{Kind: KindWarn, Pattern: `\[(?:[^\]]*/)?WARN(?:ING)?\]`},
		{Kind: KindError, Pattern: `(?i)\b(error|fatal|severe|exception|traceback|segmentation fault|failed to)\b`},
		{Kind: KindWarn, Pattern: `(?i)\b(warn(?:ing)?|deprecated|can't keep up|cannot keep up)\b`},
	}
}
