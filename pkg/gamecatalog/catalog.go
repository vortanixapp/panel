package gamecatalog

import (
	"os"
	"sort"
	"strings"
)

type PortSpec struct {
	Offset   int
	Protocol string
	Purpose  string
}

type Install struct {
	SourceType     string
	SteamAppID     int64
	SteamBranch    string
	SteamModConfig string
	ArchiveURL     string
	Version        string
	Note           string
}

type Game struct {
	Key      string
	Aliases  []string
	ImageEnv string
	Image    string

	Name        string
	Description string
	Code        string
	MinPort     int
	MaxPort     int

	PortEnv    string
	Ports      []PortSpec
	QueryProto string
	MaxSlots   int

	MinRAMMB  int
	RecRAMMB  int
	MinDiskMB int
	MinCPU    float64

	DefaultStartup string
	Install        Install

	NoLinuxServer string
}

const (
	QueryA2S  = "a2s"
	QueryMC   = "mc"
	QuerySAMP = "samp"
	QueryNone = "none"
)

const (
	SourceSteam   = "steam"
	SourceArchive = "archive"
	SourceDocker  = "docker"
)

const (
	SmallNodeRAMMB  = 3328
	SmallNodeDiskMB = 40960
	SmallNodeCPU    = 2.0
)

const RegistryEnv = "VORTANIX_REGISTRY"

var byAlias = func() map[string]int {
	m := make(map[string]int, len(games)*3)
	for i, g := range games {
		m[g.Key] = i
		for _, a := range g.Aliases {
			m[strings.ToLower(strings.TrimSpace(a))] = i
		}
	}
	return m
}()

func All() []Game {
	out := make([]Game, len(games))
	copy(out, games)
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func Resolve(code string) (Game, bool) {
	i, ok := byAlias[strings.ToLower(strings.TrimSpace(code))]
	if !ok {
		return Game{}, false
	}
	return games[i], true
}

func Normalize(code string) string {
	if g, ok := Resolve(code); ok {
		return g.Key
	}
	return strings.ToLower(strings.TrimSpace(code))
}

func Repository(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	repo, _ := splitTag(g.Image)
	return withRegistry(repo)
}

func DefaultTag(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	_, tag := splitTag(g.Image)
	return tag
}

func Image(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	if g.ImageEnv != "" {
		if v := strings.TrimSpace(os.Getenv(g.ImageEnv)); v != "" {
			return v
		}
	}
	return withRegistry(g.Image)
}

func ImageWithTag(code, tag string) string {
	repo := Repository(code)
	if repo == "" {
		return ""
	}
	tag = strings.TrimSpace(tag)
	if !ValidTag(tag) {
		_, tag = splitTag(mustGame(code).Image)
	}
	return repo + ":" + tag
}

func ValidTag(tag string) bool {
	if tag == "" || len(tag) > 128 {
		return false
	}
	for i := 0; i < len(tag); i++ {
		c := tag[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case (c == '_' || c == '.' || c == '-') && i > 0:
		default:
			return false
		}
	}
	return true
}

func TagOf(image string) string {
	_, tag := splitTag(image)
	return tag
}

// BelongsTo сверяет образ с игрой по пути репозитория, не считая реестр
// частью имени: один и тот же образ у арендатора может быть записан и голым
// (vortanix/cs2), и с адресом нашего реестра.
func BelongsTo(code, image string) bool {
	repo := Repository(code)
	if repo == "" || image == "" {
		return false
	}
	got, _ := splitTag(strings.TrimSpace(image))
	return withoutRegistry(got) == withoutRegistry(repo)
}

// withoutRegistry убирает адрес реестра: это первый сегмент, если он похож на
// хост — с точкой, двоеточием или localhost.
func withoutRegistry(repo string) string {
	head, rest, ok := strings.Cut(repo, "/")
	if !ok {
		return repo
	}
	if strings.ContainsAny(head, ".:") || head == "localhost" {
		return rest
	}
	return repo
}

func mustGame(code string) Game {
	g, _ := Resolve(code)
	return g
}

func withRegistry(repo string) string {
	// Без переменной имя остаётся голым (vortanix/rust:latest) — это Docker
	// Hub, куда образы игр и публикуются: в своём реестре под них нет места.
	prefix := strings.Trim(strings.TrimSpace(os.Getenv(RegistryEnv)), "/")
	if prefix == "" || repo == "" {
		return repo
	}
	// Явно указанный образ уже несёт свой реестр — второй адрес поверх него
	// превратил бы ссылку в мусор.
	if withoutRegistry(repo) != repo {
		return repo
	}
	if strings.HasPrefix(repo, prefix+"/") {
		return repo
	}
	return prefix + "/" + repo
}

func splitTag(image string) (repo, tag string) {
	image = strings.TrimSpace(image)
	i := strings.LastIndex(image, ":")
	if i < 0 || strings.Contains(image[i+1:], "/") {
		return image, "latest"
	}
	return image[:i], image[i+1:]
}

func Slugs(code string) []string {
	g, ok := Resolve(code)
	if !ok {
		return []string{strings.ToLower(strings.TrimSpace(code))}
	}
	out := make([]string, 0, len(g.Aliases)+1)
	out = append(out, g.Key)
	out = append(out, g.Aliases...)
	return out
}

func PortLayout(code string) []PortSpec {
	g, ok := Resolve(code)
	if !ok || len(g.Ports) == 0 {
		return []PortSpec{{Offset: 0, Protocol: "udp", Purpose: "игровой сервер"}}
	}
	out := make([]PortSpec, len(g.Ports))
	copy(out, g.Ports)
	return out
}

func PortSpan(code string) int {
	span := 0
	for _, p := range PortLayout(code) {
		if p.Offset > span {
			span = p.Offset
		}
	}
	return span + 1
}

func PortEnvName(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	return g.PortEnv
}

func QueryProtocol(code string) string {
	g, ok := Resolve(code)
	if !ok || g.QueryProto == "" {
		return QueryNone
	}
	return g.QueryProto
}

func MaxSlots(code string) int {
	g, ok := Resolve(code)
	if !ok {
		return 0
	}
	return g.MaxSlots
}

func DefaultStartup(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	return g.DefaultStartup
}

func InstallOf(code string) (Install, bool) {
	g, ok := Resolve(code)
	if !ok {
		return Install{}, false
	}
	return g.Install, true
}

func FitsNode(code string, ramMB, diskMB int, cpu float64) bool {
	g, ok := Resolve(code)
	if !ok {
		return false
	}
	return g.MinRAMMB <= ramMB && g.MinDiskMB <= diskMB && g.MinCPU <= cpu
}

func FitsSmallNode(code string) bool {
	return FitsNode(code, SmallNodeRAMMB, SmallNodeDiskMB, SmallNodeCPU)
}

func InstallNote(g Game) string {
	if g.Install.Note != "" {
		return g.Install.Note
	}
	return g.NoLinuxServer
}

func LinuxSupported(code string) bool {
	g, ok := Resolve(code)
	if !ok {
		return true
	}
	return g.NoLinuxServer == ""
}

func UnsupportedReason(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	return g.NoLinuxServer
}

func Enabled(code string) bool {
	return LinuxSupported(code) && FitsSmallNode(code)
}

func DefaultLimits(code string) map[string]any {
	g, ok := Resolve(code)
	if !ok {
		return map[string]any{"memory_mb": 512, "cpu": 0.5}
	}
	mem := g.RecRAMMB
	if mem <= 0 {
		mem = g.MinRAMMB
	}
	if mem <= 0 {
		mem = 512
	}
	cpu := g.MinCPU
	if cpu <= 0 {
		cpu = 0.5
	}
	limits := map[string]any{"memory_mb": mem, "cpu": cpu}
	if g.MinDiskMB > 0 {
		limits["disk_mb"] = g.MinDiskMB
	}
	return limits
}

func QueryPortOffset(code string) int {
	for _, p := range PortLayout(code) {
		if p.Protocol != "udp" {
			continue
		}
		if strings.Contains(strings.ToLower(p.Purpose), "query") {
			return p.Offset
		}
	}
	return 0
}

func PosterURL(code string) string {
	g, ok := Resolve(code)
	if !ok {
		return ""
	}
	return "/games/" + g.Key + ".svg"
}
