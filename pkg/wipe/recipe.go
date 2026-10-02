package wipe

import (
	"errors"
	"regexp"
	"strings"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

type Target struct {
	Dir       string
	Patterns  []string
	Depth     int
	MatchDirs bool
}

type Recipe struct {
	Key           string
	Games         []string
	Kinds         []string
	Seed          bool
	AnnounceTool  string
	SaveTool      string
	targetsForKey func(kind string, env map[string]string) ([]Target, error)
}

var (
	ErrUnsupportedGame = errors.New("для этой игры вайп не поддерживается")
	ErrUnknownKind     = errors.New("неизвестный вид вайпа")
	ErrBadIdentity     = errors.New("идентификатор сохранения содержит недопустимые символы")
)

var identityPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

var recipes = []Recipe{
	{
		Key:          "rust",
		Games:        []string{"rust"},
		Kinds:        []string{"map", "map_bp", "full"},
		Seed:         true,
		AnnounceTool: "say",
		SaveTool:     "save",
		targetsForKey: func(kind string, env map[string]string) ([]Target, error) {
			identity := strings.TrimSpace(env["RUST_IDENTITY"])
			if identity == "" {
				identity = "server"
			}
			if !identityPattern.MatchString(identity) || identity == "." || identity == ".." {
				return nil, ErrBadIdentity
			}
			patterns := []string{
				"*.map", "*.sav", "*.sav.*",
				"sv.files.*.db", "player.deaths.*.db", "player.states.*.db",
			}
			switch kind {
			case "map":
			case "map_bp":
				patterns = append(patterns, "player.blueprints.*.db")
			case "full":
				patterns = append(patterns, "*.db")
			default:
				return nil, ErrUnknownKind
			}
			return []Target{{Dir: "/server/" + identity, Patterns: patterns}}, nil
		},
	},
	{
		Key:          "ark",
		Games:        []string{"arkse", "arksa"},
		Kinds:        []string{"world", "world_players"},
		AnnounceTool: "broadcast",
		SaveTool:     "save",
		targetsForKey: func(kind string, _ map[string]string) ([]Target, error) {
			patterns := []string{"*.ark"}
			switch kind {
			case "world":
			case "world_players":
				patterns = append(patterns, "*.arkprofile", "*.arktribe", "*.profilebak", "*.tribebak", "*.arktributetribe")
			default:
				return nil, ErrUnknownKind
			}
			return []Target{{Dir: "/ShooterGame/Saved/SavedArks", Patterns: patterns, Depth: 2}}, nil
		},
	},
	{
		Key:   "dayz",
		Games: []string{"dayz", "dayzdev"},
		Kinds: []string{"economy", "full"},
		targetsForKey: func(kind string, _ map[string]string) ([]Target, error) {
			switch kind {
			case "economy":
				return []Target{{Dir: "/mpmissions/*/storage_1", Patterns: []string{"data"}, MatchDirs: true}}, nil
			case "full":
				return []Target{{Dir: "/mpmissions/*", Patterns: []string{"storage_1"}, MatchDirs: true}}, nil
			}
			return nil, ErrUnknownKind
		},
	},
}

func RecipeFor(gameID string) (Recipe, bool) {
	game := gamecatalog.Normalize(gameID)
	for _, r := range recipes {
		for _, g := range r.Games {
			if g == game {
				return r, true
			}
		}
	}
	return Recipe{}, false
}

func Supported(gameID string) bool {
	_, ok := RecipeFor(gameID)
	return ok
}

func (r Recipe) HasKind(kind string) bool {
	for _, k := range r.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (r Recipe) Targets(kind string, env map[string]string) ([]Target, error) {
	if !r.HasKind(kind) {
		return nil, ErrUnknownKind
	}
	return r.targetsForKey(kind, env)
}

func (r Recipe) EnvFile() string {
	if r.Key == "rust" {
		return "/rust.env"
	}
	return ""
}

func (r Recipe) SeedKey() string {
	if r.Seed {
		return "RUST_SEED"
	}
	return ""
}
