package docker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vortanixapp/panel/pkg/gamesettings"
)

const (
	startupParamsRel = ".vtx/startup_params"
	startupArgvRel   = ".vtx/startup_argv"
)

func WriteStartupParams(serverID, params string) error {
	dir := serverDataDir(serverID)
	paramsPath := filepath.Join(dir, startupParamsRel)
	argvPath := filepath.Join(dir, startupArgvRel)

	line := firstLine(params)
	if line == "" {
		return removeAll(paramsPath, argvPath)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".vtx"), 0o755); err != nil {
		return err
	}

	argv := gamesettings.SplitArgs(line)
	if len(argv) == 0 {
		return removeAll(paramsPath, argvPath)
	}
	var buf strings.Builder
	for _, tok := range argv {
		buf.WriteString(gamesettings.Unquote(tok))
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(argvPath, []byte(buf.String()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(paramsPath, []byte(line+"\n"), 0o644)
}

func removeAll(paths ...string) error {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
