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
	root, err := serverRootFor(serverID)
	if err != nil {
		return err
	}
	defer root.Close()

	paramsPath := filepath.FromSlash(startupParamsRel)
	argvPath := filepath.FromSlash(startupArgvRel)

	line := firstLine(params)
	if line == "" {
		return removeAll(root, paramsPath, argvPath)
	}
	if err := root.MkdirAll(".vtx", 0o755); err != nil {
		return err
	}

	argv := gamesettings.SplitArgs(line)
	if len(argv) == 0 {
		return removeAll(root, paramsPath, argvPath)
	}
	var buf strings.Builder
	for _, tok := range argv {
		buf.WriteString(gamesettings.Unquote(tok))
		buf.WriteByte('\n')
	}
	if err := writeFileInRoot(root, argvPath, []byte(buf.String())); err != nil {
		return err
	}
	return writeFileInRoot(root, paramsPath, []byte(line+"\n"))
}

func writeFileInRoot(root *os.Root, name string, content []byte) error {
	_ = root.Remove(name)
	f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func removeAll(root *os.Root, paths ...string) error {
	for _, p := range paths {
		if err := root.Remove(p); err != nil && !os.IsNotExist(err) {
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
