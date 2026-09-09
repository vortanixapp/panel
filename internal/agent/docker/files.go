package docker

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"
)

const serverRoot = "/data"

type FileEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

func resolveServerPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	full := path.Clean(serverRoot + path.Clean(p))
	if full != serverRoot && !strings.HasPrefix(full, serverRoot+"/") {
		return "", fmt.Errorf("путь вне каталога сервера")
	}
	return full, nil
}

func isServerRoot(p string) bool {
	return p == serverRoot
}

func guardedScript(target, op string) string {
	return "P=" + shellQuote(target) + "; " +
		`R=$(realpath -m "$P" 2>/dev/null || echo "$P"); ` +
		`case "$R" in ` + serverRoot + `|` + serverRoot + `/*) ;; ` +
		`*) echo 'path outside server data' >&2; exit 2;; esac; ` + op
}

func execInServer(ctx context.Context, serverID, target, op string) *exec.Cmd {
	return exec.CommandContext(ctx, "docker", "exec", ContainerName(serverID),
		"sh", "-c", guardedScript(target, op))
}

func execInServerStdin(ctx context.Context, serverID, target, op string) *exec.Cmd {
	return exec.CommandContext(ctx, "docker", "exec", "-i", ContainerName(serverID),
		"sh", "-c", guardedScript(target, op))
}

func ListFiles(ctx context.Context, serverID, dir string) ([]FileEntry, error) {
	target, err := resolveServerPath(dir)
	if err != nil {
		return nil, err
	}
	return listFilesOnHost(serverID, target)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
