package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func randomSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func serverRootFor(serverID string) (*os.Root, error) {
	base := serverDataDir(serverID)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return os.OpenRoot(base)
}

func relativeInServer(containerTarget string) string {
	rel := strings.TrimPrefix(strings.TrimPrefix(containerTarget, serverRoot), "/")
	if rel == "" {
		return "."
	}
	return filepath.FromSlash(rel)
}

func openServerPath(serverID, containerTarget string) (*os.Root, string, error) {
	root, err := serverRootFor(serverID)
	if err != nil {
		return nil, "", err
	}
	rel := relativeInServer(containerTarget)
	if rel == "." {
		return root, rel, nil
	}
	if !filepath.IsLocal(rel) {
		root.Close()
		return nil, "", fmt.Errorf("путь вне каталога сервера")
	}
	return root, rel, nil
}

func readFileFromHost(serverID, containerTarget string) ([]byte, error) {
	root, rel, err := openServerPath(serverID, containerTarget)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(rel)
}

func writeFileToHost(ctx context.Context, serverID, containerTarget string, content []byte) error {
	root, rel, err := openServerPath(serverID, containerTarget)
	if err != nil {
		return err
	}
	defer root.Close()
	if rel == "." {
		return fmt.Errorf("нельзя записать в корень данных сервера")
	}

	dir := filepath.Dir(rel)
	if dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	mode := os.FileMode(0o644)
	reference := dir
	if st, statErr := root.Stat(rel); statErr == nil {
		mode = st.Mode().Perm()
		reference = rel
	}

	name := ".vtx-write-" + randomSuffix()
	tmp := name
	if dir != "." {
		tmp = filepath.Join(dir, name)
	}
	f, err := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	written := false
	defer func() {
		if !written {
			_ = root.Remove(tmp)
		}
	}()
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := root.Chmod(tmp, mode); err != nil {
		return err
	}
	copyOwnership(ctx, root, reference, tmp)
	if err := root.Rename(tmp, rel); err != nil {
		return err
	}
	written = true
	return nil
}

func mkdirOnHost(ctx context.Context, serverID, containerTarget string) error {
	root, rel, err := openServerPath(serverID, containerTarget)
	if err != nil {
		return err
	}
	defer root.Close()
	if rel == "." {
		return nil
	}
	if err := root.MkdirAll(rel, 0o755); err != nil {
		return err
	}
	copyOwnership(ctx, root, filepath.Dir(rel), rel)
	return nil
}

func deleteOnHost(serverID, containerTarget string) error {
	root, rel, err := openServerPath(serverID, containerTarget)
	if err != nil {
		return err
	}
	defer root.Close()
	if rel == "." {
		return fmt.Errorf("нельзя удалить корень данных сервера")
	}
	if _, err := root.Lstat(rel); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return root.RemoveAll(rel)
}

func copyOwnership(ctx context.Context, root *os.Root, reference, target string) {
	base := root.Name()
	if reference == "." {
		reference = ""
	}
	const script = `o=$(stat -c '%u:%g' "$1") || exit 0
chown -h "$o" "$2" || true`
	_ = exec.CommandContext(ctx, "sh", "-c", script, "sh",
		filepath.Join(base, reference), filepath.Join(base, target)).Run()
}

func listFilesOnHost(serverID, containerTarget string) ([]FileEntry, error) {
	root, rel, err := openServerPath(serverID, containerTarget)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dir, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	dirEntries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	out := make([]FileEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		entry := FileEntry{Name: e.Name(), IsDir: e.IsDir()}
		if info, infoErr := e.Info(); infoErr == nil {
			entry.Size = info.Size()
			if info.Mode()&os.ModeSymlink != 0 {
				if st, statErr := root.Stat(filepath.Join(rel, e.Name())); statErr == nil {
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
