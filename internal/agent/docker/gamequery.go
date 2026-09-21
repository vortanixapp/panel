package docker

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
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
	command = formatKickBanCommand(normalizeGame(gameID), command)
	reply, err := SendConsoleCommand(ctx, serverID, gameID, command)
	if err != nil {
		return "", err
	}
	return reply.Output, nil
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
