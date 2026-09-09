package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func hostServerPath(serverID, containerTarget string) (string, error) {
	base := serverDataDir(serverID)
	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(containerTarget, serverRoot), "/")
	full := filepath.Join(realBase, filepath.FromSlash(rel))

	probe := full
	for {
		resolved, evalErr := filepath.EvalSymlinks(probe)
		if evalErr == nil {
			if !underDir(realBase, resolved) {
				return "", fmt.Errorf("путь вне каталога сервера")
			}
			return full, nil
		}
		if !os.IsNotExist(evalErr) {
			return "", evalErr
		}
		parent := filepath.Dir(probe)
		if parent == probe || len(parent) < len(realBase) {
			return "", fmt.Errorf("путь вне каталога сервера")
		}
		probe = parent
	}
}

func underDir(base, p string) bool {
	if p == base {
		return true
	}
	return strings.HasPrefix(p, base+string(os.PathSeparator))
}

func readFileFromHost(serverID, containerTarget string) ([]byte, error) {
	path, err := hostServerPath(serverID, containerTarget)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func writeFileToHost(ctx context.Context, serverID, containerTarget string, content []byte) error {
	path, err := hostServerPath(serverID, containerTarget)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	mode := os.FileMode(0o644)
	reference := dir
	if st, statErr := os.Stat(path); statErr == nil {
		mode = st.Mode().Perm()
		reference = path
	}

	tmp, err := os.CreateTemp(dir, ".vtx-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	copyOwnership(ctx, reference, tmpName)
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	tmpName = ""
	return nil
}

func copyOwnership(ctx context.Context, reference, target string) {
	const script = `o=$(stat -c '%u:%g' "$1") || exit 0
chown -h "$o" "$2" || true`
	_ = exec.CommandContext(ctx, "sh", "-c", script, "sh", reference, target).Run()
}

func listFilesOnHost(serverID, containerTarget string) ([]FileEntry, error) {
	path, err := hostServerPath(serverID, containerTarget)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]FileEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		entry := FileEntry{Name: e.Name(), IsDir: e.IsDir()}
		if info, infoErr := e.Info(); infoErr == nil {
			entry.Size = info.Size()
			if info.Mode()&os.ModeSymlink != 0 {
				if st, statErr := os.Stat(filepath.Join(path, e.Name())); statErr == nil {
					entry.IsDir = st.IsDir()
					entry.Size = st.Size()
				}
			}
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
