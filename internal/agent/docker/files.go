package docker

import (
	"bytes"
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

func ListFiles(ctx context.Context, serverID, dir string) ([]FileEntry, error) {
	target, err := resolveServerPath(dir)
	if err != nil {
		return nil, err
	}
	return listFilesOnHost(serverID, target)
}

func runCommand(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		return err
	}
	if len(msg) > 400 {
		msg = msg[:400]
	}
	return fmt.Errorf("%s", msg)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
