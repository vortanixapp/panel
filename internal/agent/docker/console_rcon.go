package docker

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/rcon"
)

const (
	rconSource = ""
	rconWeb    = "web"
	rconTelnet = "telnet"
)

type rconTarget struct {
	kind     string
	port     int
	password string
}

func (t rconTarget) via() string {
	if t.kind == rconTelnet {
		return "telnet"
	}
	return "rcon"
}

type rconCacheEntry struct {
	key     string
	target  rconTarget
	found   bool
	expires time.Time
}

const (
	rconCacheTTL      = time.Minute
	rconFileLimit     = 1 << 20
	rustWebRconPort   = 28016
	minecraftRconPort = 25575
)

var (
	rconMu    sync.Mutex
	rconCache = map[string]rconCacheEntry{}
)

func findRcon(ctx context.Context, serverID, gameID, cname string, info *containerInfo) (rconTarget, bool) {
	key := info.State.StartedAt + "|" + gameID
	rconMu.Lock()
	entry, ok := rconCache[cname]
	rconMu.Unlock()
	if ok && entry.key == key && time.Now().Before(entry.expires) {
		return entry.target, entry.found
	}
	target, found := discoverRcon(ctx, serverID, gameID, cname, info)
	rconMu.Lock()
	rconCache[cname] = rconCacheEntry{key: key, target: target, found: found, expires: time.Now().Add(rconCacheTTL)}
	rconMu.Unlock()
	return target, found
}

func forgetRcon(cname string) {
	rconMu.Lock()
	delete(rconCache, cname)
	rconMu.Unlock()
}

func discoverRcon(ctx context.Context, serverID, gameID, cname string, info *containerInfo) (rconTarget, bool) {
	dir := serverDataDir(serverID)
	game := normalizeGame(gameID)
	env := containerEnv(info)
	fallback := catalogRconPort(game, env, info)

	if t, ok := rconFromProperties(dir); ok {
		return t, true
	}
	if t, ok := rconFromArgs(containerArgs(ctx, cname), fallback); ok {
		return t, true
	}
	if t, ok := rconFromGameIni(dir); ok {
		return t, true
	}
	if t, ok := telnetFromServerConfig(dir); ok {
		return t, true
	}
	if t, ok := rconFromSourceCfg(dir, fallback); ok {
		return t, true
	}
	if pass := env["RCON_PASSWORD"]; pass != "" {
		port := positiveInt(env["RCON_PORT"])
		if port == 0 {
			port = fallback
		}
		if port > 0 {
			return rconTarget{port: port, password: pass}, true
		}
	}
	return rconTarget{}, false
}

func (t rconTarget) run(ctx context.Context, info *containerInfo, command string) (string, error) {
	var lastErr error
	for _, addr := range rconAddrs(info, t.port) {
		var out string
		var err error
		switch t.kind {
		case rconWeb:
			out, err = rcon.WebRcon(ctx, addr, t.password, command, rcon.Options{})
		case rconTelnet:
			out, err = rcon.Telnet(ctx, addr, t.password, command, rcon.Options{})
		default:
			out, err = rcon.Source(ctx, addr, t.password, command, rcon.Options{})
		}
		if err == nil {
			return out, nil
		}
		lastErr = err
		var opErr *net.OpError
		if !errors.As(err, &opErr) || opErr.Op != "dial" {
			return "", err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("rcon: у контейнера нет доступного адреса")
	}
	return "", lastErr
}

func rconAddrs(info *containerInfo, port int) []string {
	var out []string
	seen := map[string]bool{}
	add := func(ip string, p int) {
		if ip == "" || p <= 0 {
			return
		}
		addr := net.JoinHostPort(ip, strconv.Itoa(p))
		if !seen[addr] {
			seen[addr] = true
			out = append(out, addr)
		}
	}
	for _, network := range info.NetworkSettings.Networks {
		add(network.IPAddress, port)
	}
	add(info.NetworkSettings.IPAddress, port)
	for _, binding := range info.NetworkSettings.Ports[strconv.Itoa(port)+"/tcp"] {
		ip := binding.HostIP
		if ip == "" || ip == "0.0.0.0" || ip == "::" {
			ip = defaultGateway()
		}
		add(ip, positiveInt(binding.HostPort))
	}
	return out
}

func defaultGateway() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "00000000" {
			continue
		}
		raw, err := strconv.ParseUint(fields[2], 16, 32)
		if err != nil || raw == 0 {
			continue
		}
		return net.IPv4(byte(raw), byte(raw>>8), byte(raw>>16), byte(raw>>24)).String()
	}
	return ""
}

func containerEnv(info *containerInfo) map[string]string {
	env := make(map[string]string, len(info.Config.Env))
	for _, item := range info.Config.Env {
		if k, v, ok := strings.Cut(item, "="); ok {
			env[k] = v
		}
	}
	return env
}

func catalogRconPort(game string, env map[string]string, info *containerInfo) int {
	offset := -1
	for _, spec := range gamecatalog.PortLayout(game) {
		if strings.EqualFold(spec.Protocol, "tcp") && strings.Contains(strings.ToUpper(spec.Purpose), "RCON") {
			offset = spec.Offset
			break
		}
	}
	if offset < 0 {
		return 0
	}
	primary := 0
	if name := gamecatalog.PortEnvName(game); name != "" {
		primary = positiveInt(env[name])
	}
	if primary == 0 {
		for key := range info.NetworkSettings.Ports {
			if p := positiveInt(strings.SplitN(key, "/", 2)[0]); p > 0 && (primary == 0 || p < primary) {
				primary = p
			}
		}
	}
	if primary == 0 {
		return 0
	}
	return primary + offset
}

func containerArgs(ctx context.Context, cname string) []string {
	script := `for f in /proc/[0-9]*/cmdline; do tr '\000' '\n' < "$f" 2>/dev/null; echo; done`
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).Output()
	if err != nil {
		return nil
	}
	var args []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			args = append(args, line)
		}
	}
	return args
}

func rconFromArgs(args []string, fallback int) (rconTarget, bool) {
	var t rconTarget
	rust := false
	webFlag := ""
	gamePort := 0
	var tokens []string
	for _, arg := range args {
		lower := strings.ToLower(arg)
		if strings.Contains(lower, "?rconenabled=true") {
			for _, part := range strings.Split(arg, "?") {
				k, v, ok := strings.Cut(part, "=")
				if !ok {
					continue
				}
				switch strings.ToLower(k) {
				case "rconport":
					t.port = positiveInt(v)
				case "serveradminpassword":
					t.password = v
				}
			}
		}
		tokens = append(tokens, strings.Fields(arg)...)
	}
	for i, token := range tokens {
		next := ""
		if i+1 < len(tokens) {
			next = strings.Trim(tokens[i+1], `"'`)
		}
		if strings.HasPrefix(next, "+") || strings.HasPrefix(next, "-") {
			next = ""
		}
		switch strings.ToLower(token) {
		case "+rcon.password":
			t.password, rust = next, true
		case "+rcon.port":
			t.port = positiveInt(next)
		case "+rcon.web":
			webFlag = next
		case "+rcon_password", "-rcon_password", "--rcon-password":
			t.password = next
		case "+rcon_port", "-rconport", "--rcon-port":
			t.port = positiveInt(next)
		case "-port", "+port", "+hostport", "-hostport":
			if gamePort == 0 {
				gamePort = positiveInt(next)
			}
		}
	}
	if t.password == "" {
		return rconTarget{}, false
	}
	if rust {
		if webFlag != "0" && !strings.EqualFold(webFlag, "false") {
			t.kind = rconWeb
		}
		if t.port == 0 {
			t.port = rustWebRconPort
		}
	}
	if t.port == 0 {
		t.port = gamePort
	}
	if t.port == 0 {
		t.port = fallback
	}
	return t, t.port > 0
}

func rconFromProperties(dir string) (rconTarget, bool) {
	content := readSmallFile(filepath.Join(dir, "server.properties"))
	if content == "" {
		return rconTarget{}, false
	}
	props := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			props[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if !strings.EqualFold(props["enable-rcon"], "true") || props["rcon.password"] == "" {
		return rconTarget{}, false
	}
	port := positiveInt(props["rcon.port"])
	if port == 0 {
		port = minecraftRconPort
	}
	return rconTarget{port: port, password: props["rcon.password"]}, true
}

var gameIniFiles = []string{
	"ShooterGame/Saved/Config/LinuxServer/GameUserSettings.ini",
	"ShooterGame/Saved/Config/WindowsServer/GameUserSettings.ini",
	"Pal/Saved/Config/LinuxServer/PalWorldSettings.ini",
	"Pal/Saved/Config/WindowsServer/PalWorldSettings.ini",
}

func rconFromGameIni(dir string) (rconTarget, bool) {
	for _, rel := range gameIniFiles {
		content := readSmallFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if content == "" || !strings.EqualFold(settingValue(content, "RCONEnabled"), "true") {
			continue
		}
		port := positiveInt(settingValue(content, "RCONPort"))
		pass := settingValue(content, "ServerAdminPassword")
		if pass == "" {
			pass = settingValue(content, "AdminPassword")
		}
		if port > 0 && pass != "" {
			return rconTarget{port: port, password: pass}, true
		}
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "Zomboid", "Server", "*.ini"))
	for _, path := range matches {
		content := readSmallFile(path)
		port := positiveInt(settingValue(content, "RCONPort"))
		pass := settingValue(content, "RCONPassword")
		if port > 0 && pass != "" {
			return rconTarget{port: port, password: pass}, true
		}
	}
	return rconTarget{}, false
}

func rconFromSourceCfg(dir string, port int) (rconTarget, bool) {
	if port <= 0 {
		return rconTarget{}, false
	}
	for _, pattern := range []string{"*/cfg/server.cfg", "game/*/cfg/server.cfg"} {
		matches, _ := filepath.Glob(filepath.Join(dir, filepath.FromSlash(pattern)))
		for _, path := range matches {
			if pass := cfgValue(readSmallFile(path), "rcon_password"); pass != "" {
				return rconTarget{port: port, password: pass}, true
			}
		}
	}
	return rconTarget{}, false
}

func telnetFromServerConfig(dir string) (rconTarget, bool) {
	content := readSmallFile(filepath.Join(dir, "serverconfig.xml"))
	if content == "" || strings.EqualFold(xmlProperty(content, "TelnetEnabled"), "false") {
		return rconTarget{}, false
	}
	pass := xmlProperty(content, "TelnetPassword")
	port := positiveInt(xmlProperty(content, "TelnetPort"))
	if pass == "" || port == 0 {
		return rconTarget{}, false
	}
	return rconTarget{kind: rconTelnet, port: port, password: pass}, true
}

func xmlProperty(content, name string) string {
	re := regexp.MustCompile(`(?i)<property\s+name\s*=\s*"` + regexp.QuoteMeta(name) + `"\s+value\s*=\s*"([^"]*)"`)
	if m := re.FindStringSubmatch(content); m != nil {
		return m[1]
	}
	return ""
}

func settingValue(content, key string) string {
	re := regexp.MustCompile(`(?im)(?:^|[\s,(?])` + regexp.QuoteMeta(key) + `[ \t]*=[ \t]*(?:"([^"\r\n]*)"|([^,)\r\n]*))`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	if m[1] != "" {
		return m[1]
	}
	return strings.TrimSpace(m[2])
}

func cfgValue(content, key string) string {
	value := ""
	for _, line := range strings.Split(content, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], key) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimSpace(line)[len(fields[0]):])
		value = strings.Trim(rest, `"`)
	}
	return value
}

func readSmallFile(path string) string {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() > rconFileLimit {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func positiveInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 || n > 65535 {
		return 0
	}
	return n
}
