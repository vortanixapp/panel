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

// Доступ к файлам сервера в обход контейнера.
//
// Раньше чтение и запись шли только через `docker exec`, а он на остановленном
// контейнере просто падает. Из-за этого самый естественный порядок работы —
// выключить сервер, поправить настройки, запустить — не работал вовсе: панель
// показывала пустой конфиг и молча не сохраняла правки.
//
// Ходить напрямую можно всегда: каталог данных монтируется в контейнер как
// `-v serverDataDir(serverID):/data` (docker.go:157), то есть путь внутри
// контейнера и путь на хосте — одно и то же место. Самому агенту этот каталог
// тоже виден: его контейнер запускается с
// `-v /var/lib/vortanix/servers:/var/lib/vortanix/servers`.
//
// Побочно это убирает зависимость от наличия `sh` и `cat` в образе игры и
// снимает накладные расходы на docker exec при каждой операции.

// hostServerPath переводит путь внутри контейнера в путь на хосте и проверяет,
// что он не выводит за каталог сервера.
//
// Проверка здесь своя и обязательная. В контейнере от выхода наружу защищал
// `realpath -m` внутри guardedScript, но на хосте его нет, а содержимое каталога
// принадлежит клиенту: по SFTP он может положить туда ссылку на /etc и без этой
// проверки прочитать её через настройки.
func hostServerPath(serverID, containerTarget string) (string, error) {
	base := serverDataDir(serverID)
	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(containerTarget, serverRoot), "/")
	full := filepath.Join(realBase, filepath.FromSlash(rel))

	// Разрешаем ближайшего существующего предка: самого файла может ещё не быть,
	// а вот каталог на пути к нему уже может оказаться подменённой ссылкой.
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

// writeFileToHost пишет через временный файл рядом и переименование.
//
// Прежняя запись — `cat > "$R"` — обрезала файл сразу, и обрыв связи посреди
// передачи оставлял клиента с усечённым конфигом, из-за которого сервер уже не
// поднимался. Переименование в пределах одного каталога атомарно: наблюдатель
// видит либо старое содержимое целиком, либо новое.
func writeFileToHost(ctx context.Context, serverID, containerTarget string, content []byte) error {
	path, err := hostServerPath(serverID, containerTarget)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Права и владельца берём с прежнего файла, а если его нет — с каталога.
	// Иначе сохранение настроек отбирало бы у клиента доступ к файлу по SFTP:
	// новый файл создался бы от root, тогда как остальные данные принадлежат
	// пользователю доступа (см. ReapplyOwnership).
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

// copyOwnership переносит владельца с образца на файл.
//
// Через os.Chown это потребовало бы syscall.Stat_t, который есть не на всех
// платформах сборки. Тот же приём уже используется в ReapplyOwnership, поэтому
// повторяем его: busybox-совместимый stat, ошибки не считаем фатальными —
// владелец мог просто не отличаться от текущего.
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
			// Ссылку на каталог показываем каталогом, как это делал `ls -la`
			// внутри контейнера: клиент видит структуру, а не тип inode.
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
