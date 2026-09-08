package docker

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

type GameQueryResult struct {
	Online        bool     `json:"online"`
	MaxPlayers    int      `json:"max_players"`
	OnlinePlayers int      `json:"online_players"`
	CurrentMap    string   `json:"current_map"`
	PlayersOnline []Player `json:"players_online"`
	RuntimeStatus string   `json:"runtime_status"`
	PingMS        int      `json:"ping_ms"`
	TPS           float64  `json:"tps"`
}

type Player struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Ping  int     `json:"ping"`
}

func QueryGameServer(ctx context.Context, serverID, gameID string, limits map[string]any, port int) GameQueryResult {
	out := GameQueryResult{
		RuntimeStatus: DetailedStatus(ctx, serverID),
		MaxPlayers:    maxPlayersFromLimits(limits),
		PlayersOnline: []Player{},
	}
	if out.RuntimeStatus != "running" {
		return out
	}
	out.Online = true
	game := normalizeGame(gameID)
	basePort := PrimaryPortFromServer(port)
	host, queryPort := QueryTarget(ctx, serverID, basePort+gamecatalog.QueryPortOffset(game), "udp")

	// Отвечал ли сервер вообще. Без этого признака пинг ниже измерял бы
	// длительность собственного таймаута — и молчащий сервер попадал бы в базу
	// не как «нет ответа», а как «пинг 1200 мс».
	answered := false
	queryStart := time.Now()
	switch {
	case isSourceGame(game):
		if info, err := queryA2SInfo(host, queryPort); err == nil && info != nil {
			answered = true
			if info.MaxPlayers > 0 {
				out.MaxPlayers = info.MaxPlayers
			}
			out.OnlinePlayers = info.Players
			if info.Map != "" {
				out.CurrentMap = info.Map
			}
		}
		if players, err := queryA2SPlayers(host, queryPort); err == nil {
			out.PlayersOnline = players
			if len(players) > 0 {
				out.OnlinePlayers = len(players)
			}
		}
	case isMcJavaGame(game):
		players := queryMinecraftPlayers(ctx, serverID)
		out.PlayersOnline = players
		out.OnlinePlayers = len(players)
	case isMcBedrockGame(game):
		if info, err := queryBedrock(host, queryPort); err == nil && info != nil {
			answered = true
			out.OnlinePlayers = info.Players
			if info.MaxPlayers > 0 {
				out.MaxPlayers = info.MaxPlayers
			}
			out.CurrentMap = info.LevelName
		}
	case isSampGame(game):
		if info, err := querySampFull(host, queryPort); err == nil && info != nil {
			answered = true
			out.OnlinePlayers = info.Players
			if info.MaxPlayers > 0 {
				out.MaxPlayers = info.MaxPlayers
			}
			// У SA-MP карта одна на всю игру, поэтому в её строке показываем
			// режим — единственное, что там осмысленно различается.
			out.CurrentMap = info.GameMode
		}
	}

	if answered {
		out.PingMS = int(time.Since(queryStart).Milliseconds())
	}
	if isMcJavaGame(game) {
		out.TPS = queryMinecraftTPS(ctx, serverID)
	}

	if out.CurrentMap == "" {
		out.CurrentMap = readCurrentMap(ctx, serverID, gameID)
	}
	if out.OnlinePlayers == 0 && len(out.PlayersOnline) > 0 {
		out.OnlinePlayers = len(out.PlayersOnline)
	}
	return out
}

func queryMinecraftPlayers(ctx context.Context, serverID string) []Player {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
		`if command -v rcon-cli >/dev/null 2>&1; then rcon-cli list 2>/dev/null; elif command -v mcrcon >/dev/null 2>&1; then mcrcon list 2>/dev/null; fi`).CombinedOutput()
	if err != nil {
		return []Player{}
	}
	return parseMcListOutput(string(out))
}

func queryMinecraftTPS(ctx context.Context, serverID string) float64 {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
		`if command -v rcon-cli >/dev/null 2>&1; then rcon-cli tps 2>/dev/null; elif command -v mcrcon >/dev/null 2>&1; then mcrcon tps 2>/dev/null; fi`).CombinedOutput()
	if err != nil {
		return 0
	}
	m := tpsPattern.FindStringSubmatch(stripMcColors(string(out)))
	if len(m) < 2 {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil || v <= 0 || v > 100 {
		return 0
	}
	return v
}

var (
	tpsPattern   = regexp.MustCompile(`(?i)TPS[^:]*:\s*\*?([0-9]+(?:\.[0-9]+)?)`)
	mcColorCodes = regexp.MustCompile(`\x{00a7}[0-9a-fk-orA-FK-OR]|\x1b\[[0-9;]*m`)
)

func stripMcColors(s string) string {
	return mcColorCodes.ReplaceAllString(s, "")
}

func maxPlayersFromLimits(limits map[string]any) int {
	if limits == nil {
		return 0
	}
	if v, ok := limits["slots"]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

// readPropsValue читает значение ключа из файла вида «ключ=значение».
func readPropsValue(ctx context.Context, serverID, path, key string) string {
	out, err := exec.CommandContext(ctx, "docker", "exec", ContainerName(serverID),
		"sh", "-c", "grep -m1 '^"+key+"=' "+shellQuote(path)+" 2>/dev/null").Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	_, value, found := strings.Cut(line, "=")
	if !found {
		return ""
	}
	return strings.TrimSpace(value)
}

func readCurrentMap(ctx context.Context, serverID, gameID string) string {
	game := normalizeGame(gameID)
	// Minecraft ни в одну из веток сетевого опроса карту не отдаёт: у Java её
	// нет в протоколе вовсе, у Bedrock она приходит, но только когда сервер
	// ответил. Имя мира лежит в server.properties и читается одинаково для
	// обоих изданий.
	if isMcJavaGame(game) || isMcBedrockGame(game) {
		return readPropsValue(ctx, serverID, "/data/server.properties", "level-name")
	}
	var paths []string
	switch game {
	case "cs16", "cstrike":
		paths = []string{"/data/cstrike/server.cfg", "/data/server.cfg"}
	case "css":
		paths = []string{"/data/cstrike/cfg/server.cfg"}
	case "cs2":
		paths = []string{"/data/game/csgo/cfg/server.cfg"}
	default:
		return ""
	}
	cname := ContainerName(serverID)
	for _, p := range paths {
		out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", "grep -E '^\\s*map\\s+' "+shellQuote(p)+" 2>/dev/null | tail -1").Output()
		if err != nil || len(out) == 0 {
			continue
		}
		parts := strings.Fields(strings.TrimSpace(string(out)))
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func ExecConsoleCommand(ctx context.Context, serverID, gameID, command string) (string, error) {
	cname := ContainerName(serverID)
	game := normalizeGame(gameID)
	command = formatKickBanCommand(game, command)

	switch {
	case isMcJavaGame(game):
		return execMcRcon(ctx, cname, command)
	case isSourceGame(game):
		if out, err := execSourceRcon(ctx, cname, game, command); err == nil {
			return out, nil
		}
		fallthrough
	default:
		out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
			"echo "+shellQuote(command)+" >> /data/console_commands.log && echo ok").CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
}

func formatKickBanCommand(game, command string) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)
	if !strings.HasPrefix(lower, "kick ") && !strings.HasPrefix(lower, "ban ") {
		return cmd
	}
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return cmd
	}
	action := strings.ToLower(parts[0])
	name := parts[1]
	reason := strings.Join(parts[2:], " ")

	switch normalizeGame(game) {
	case "cs16", "cstrike":
		if action == "ban" {
			if reason != "" {
				return "banid 0 " + shellQuoteInner(name) + " " + shellQuoteInner(reason)
			}
			return "banid 0 " + shellQuoteInner(name)
		}
		if reason != "" {
			return "kick " + shellQuoteInner(name) + " " + shellQuoteInner(reason)
		}
		return "kick " + shellQuoteInner(name)
	case "css", "cs2", "tf2", "gmod", "garrysmod":
		if action == "ban" {
			if reason != "" {
				return "banid 0 " + shellQuoteInner(name) + " " + shellQuoteInner(reason)
			}
			return "banid 0 " + shellQuoteInner(name)
		}
		if reason != "" {
			return "kick " + shellQuoteInner(name) + " " + shellQuoteInner(reason)
		}
		return "kick " + shellQuoteInner(name)
	default:
		return cmd
	}
}

func shellQuoteInner(s string) string {
	if strings.ContainsAny(s, " \t\"") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

func execMcRcon(ctx context.Context, cname, command string) (string, error) {
	cmdQ := shellQuote(command)
	script := "if command -v rcon-cli >/dev/null 2>&1; then rcon-cli " + cmdQ + "; " +
		"elif command -v mcrcon >/dev/null 2>&1; then mcrcon " + cmdQ + "; else echo ok; fi"
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func execSourceRcon(ctx context.Context, cname, game, command string) (string, error) {
	pass := readRconPassword(ctx, cname, game)
	if pass == "" {
		return "", fmt.Errorf("rcon password not found")
	}
	port := readRconPort(ctx, cname, game)
	cmdQ := shellQuote(command)
	passQ := shellQuote(pass)
	script := fmt.Sprintf(
		`if command -v mcrcon >/dev/null 2>&1; then mcrcon -H 127.0.0.1 -P %d -p %s %s; `+
			`elif command -v rcon-cli >/dev/null 2>&1; then rcon-cli --host 127.0.0.1 --port %d --password %s %s; `+
			`else echo ok; fi`,
		port, passQ, cmdQ, port, passQ, cmdQ)
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func readRconPassword(ctx context.Context, cname, game string) string {
	paths := rconConfigPaths(game)
	for _, p := range paths {
		out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
			"grep -E '^\\s*rcon_password\\s+' "+shellQuote(p)+" 2>/dev/null | tail -1").Output()
		if err != nil || len(out) == 0 {
			continue
		}
		parts := strings.Fields(strings.TrimSpace(string(out)))
		if len(parts) >= 2 {
			return strings.Trim(parts[1], `"`)
		}
	}
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
		"grep -E '^rcon\\.password=' /data/server.properties 2>/dev/null | tail -1").Output()
	if err == nil && len(out) > 0 {
		if idx := strings.Index(string(out), "="); idx >= 0 {
			return strings.TrimSpace(string(out)[idx+1:])
		}
	}
	return ""
}

func readRconPort(ctx context.Context, cname, game string) int {
	if isMcJavaGame(game) {
		out, _ := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
			"grep -E '^rcon\\.port=' /data/server.properties 2>/dev/null | tail -1").Output()
		if len(out) > 0 {
			if idx := strings.Index(string(out), "="); idx >= 0 {
				if n, err := strconv.Atoi(strings.TrimSpace(string(out)[idx+1:])); err == nil && n > 0 {
					return n
				}
			}
		}
		return 25575
	}
	return 27015
}

func rconConfigPaths(game string) []string {
	switch normalizeGame(game) {
	case "cs16", "cstrike":
		return []string{"/data/server.cfg", "/data/cstrike/server.cfg"}
	case "css":
		return []string{"/data/cstrike/cfg/server.cfg"}
	case "cs2":
		return []string{"/data/game/csgo/cfg/server.cfg"}
	default:
		return []string{"/data/server.cfg"}
	}
}

var ansiEscape = regexp.MustCompile(`\x1B\[[0-?]*[ -/]*[@-~]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func ParseLimits(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	return raw
}

func ContainerRunning(ctx context.Context, serverID string) bool {
	ok, err := isRunning(ctx, ContainerName(serverID))
	return err == nil && ok
}

func PrimaryPortFromServer(port int) int {
	if port <= 0 {
		return 27015
	}
	return port
}

func SlotsFromString(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
