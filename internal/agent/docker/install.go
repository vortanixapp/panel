package docker

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const steamcmdImage = "steamcmd/steamcmd:latest"

const installTimeout = 2 * time.Hour

const maxArchiveBytes = 32 << 30

type InstallSpec struct {
	SourceType     string
	ArchiveURL     string
	SteamAppID     int64
	SteamBranch    string
	SteamModConfig string
}

func (s InstallSpec) HasSource() bool {
	return s.ArchiveURL != "" || s.SteamAppID > 0
}

func InstallSpecFromPayload(v any) InstallSpec {
	m, ok := v.(map[string]any)
	if !ok {
		return InstallSpec{}
	}
	spec := InstallSpec{}
	spec.SourceType, _ = m["source_type"].(string)
	spec.ArchiveURL, _ = m["archive_url"].(string)
	spec.SteamBranch, _ = m["steam_branch"].(string)
	spec.SteamModConfig, _ = m["steam_mod_config"].(string)
	spec.SteamAppID = int64(IntFromPayload(m["steam_app_id"]))
	spec.ArchiveURL = strings.TrimSpace(spec.ArchiveURL)
	return spec
}

var installLocks sync.Map

func lockServer(serverID string) *sync.Mutex {
	v, _ := installLocks.LoadOrStore(serverID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// installMarkerPath — метка незавершённой установки.
//
// Без неё оборванная установка выглядела как успешная: агент падал или его
// перезапускали посреди загрузки из SteamCMD, в томе оставались файлы, и
// NeedsInstall решал, что ставить больше нечего. Сервер стартовал на битой
// сборке, а вылечить это можно было только переустановкой с потерей данных.
func installMarkerPath(serverID string) string {
	return filepath.Join(serverDataDir(serverID), ".vtx", "install.inprogress")
}

func NeedsInstall(serverID string) bool {
	// Метка старше самой установки: если она на месте, прошлая попытка не
	// дошла до конца, и том надо ставить заново.
	if _, err := os.Stat(installMarkerPath(serverID)); err == nil {
		return true
	}
	entries, err := os.ReadDir(serverDataDir(serverID))
	if err != nil {
		return true
	}
	for _, e := range entries {
		if e.Name() != ".vtx" {
			return false
		}
	}
	return true
}

func Install(ctx context.Context, serverID string, spec InstallSpec, report ProgressFunc) error {
	if report == nil {
		report = noopProgress
	}
	if !spec.HasSource() {
		return fmt.Errorf("у версии игры нет ни архива, ни Steam App ID")
	}

	mu := lockServer(serverID)
	mu.Lock()
	defer mu.Unlock()

	if !NeedsInstall(serverID) {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	dataDir := serverDataDir(serverID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("создание тома: %w", err)
	}

	// Повторная установка начинается с чистого тома: остатки прошлой попытки
	// смешивались бы с новыми файлами, и получалась сборка, которой нет ни в
	// одной версии игры.
	marker := installMarkerPath(serverID)
	if _, err := os.Stat(marker); err == nil {
		report("prepare", 0, "предыдущая установка не завершилась, том очищается", "")
		if err := WipeData(serverID); err != nil {
			return fmt.Errorf("очистка после оборванной установки: %w", err)
		}
		if err := os.MkdirAll(dataDir, 0o755); err != nil {
			return fmt.Errorf("создание тома: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		return fmt.Errorf("служебный каталог: %w", err)
	}
	if err := os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644); err != nil {
		return fmt.Errorf("метка установки: %w", err)
	}

	var installErr error
	if spec.SteamAppID > 0 {
		installErr = installSteam(ctx, dataDir, spec, report)
	} else {
		installErr = installArchive(ctx, dataDir, spec.ArchiveURL, report)
	}
	if installErr != nil {
		// Метку оставляем: следующая попытка увидит её и начнёт с чистого тома.
		return installErr
	}
	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("снятие метки установки: %w", err)
	}
	return nil
}

// Update обновляет файлы игры поверх уже установленных.
//
// От Install отличается тем, что не требует пустого тома: обновление и есть
// повтор установки на существующие данные. SteamCMD в этом режиме докачивает
// изменившееся (app_update), архив — распаковывается поверх. Сохранения при
// этом не трогаются: удаления данных здесь нет, для чистой установки есть
// переустановка с wipe.
func Update(ctx context.Context, serverID string, spec InstallSpec, report ProgressFunc) error {
	if report == nil {
		report = noopProgress
	}
	if !spec.HasSource() {
		return fmt.Errorf("у версии игры нет ни архива, ни Steam App ID")
	}

	mu := lockServer(serverID)
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	dataDir := serverDataDir(serverID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("создание тома: %w", err)
	}

	if spec.SteamAppID > 0 {
		return installSteam(ctx, dataDir, spec, report)
	}
	return installArchive(ctx, dataDir, spec.ArchiveURL, report)
}

func installSteam(ctx context.Context, dataDir string, spec InstallSpec, report ProgressFunc) error {
	if report == nil {
		report = noopProgress
	}
	report(StageSteamCMD, 0, "Подготовка SteamCMD", "")
	if !imageExists(ctx, steamcmdImage) {
		if err := pullImage(ctx, steamcmdImage); err != nil {
			return fmt.Errorf("steamcmd образ: %w", explainPullError(err))
		}
	}

	report(StageSteamCMD, 5, "Загрузка файлов игры", "")
	sink := newLineScanner(func(line string) {
		if pct := steamPercent(line); pct >= 0 {
			report(StageSteamCMD, pct, "Загрузка файлов игры", line)
			return
		}
		if steamcmdNoise(line) {
			return
		}
		report(StageSteamCMD, -1, "", line)
	})
	err := runDockerStreaming(ctx, sink, steamcmdArgs(dataDir, spec)...)
	_ = sink.Close()
	if err != nil {
		// Вторая попытка не суеверие: первый запуск в свежем контейнере
		// обновляет сам steamcmd и нередко срывается на метаданных. К этому
		// моменту клиент уже обновлён и кэш прогрет, поэтому повтор проходит.
		report(StageSteamCMD, 5, "Повторная попытка загрузки", "")
		retrySink := newLineScanner(func(line string) {
			if pct := steamPercent(line); pct >= 0 {
				report(StageSteamCMD, pct, "Загрузка файлов игры", line)
				return
			}
			report(StageSteamCMD, -1, "", line)
		})
		retryErr := runDockerStreaming(ctx, retrySink, steamcmdArgs(dataDir, spec)...)
		_ = retrySink.Close()
		if retryErr != nil {
			return fmt.Errorf("steamcmd app_update %d: %w", spec.SteamAppID, retryErr)
		}
	}
	report(StageDone, 100, "Файлы сервера установлены", "")
	return nil
}

func steamcmdArgs(dataDir string, spec InstallSpec) []string {
	app := fmt.Sprint(spec.SteamAppID)
	args := []string{
		"run", "--rm",
		"-v", dataDir + ":/data",
		steamcmdImage,
		// Порядок как в документации Valve: сначала вход, потом каталог.
		"+login", "anonymous",
		"+force_install_dir", "/data",
		// «Missing configuration» — это не про игру, а про пустой кэш
		// метаданных: контейнер одноразовый, appinfo в нём каждый раз чистый.
		// app_info_update лишь помечает кэш устаревшим и не ждёт ответа, а
		// app_info_print блокируется, пока конфигурация приложения не придёт.
		// Без него app_update стартует раньше метаданных и падает сразу после
		// подключения — что и происходило на каждой установке из Steam.
		"+app_info_update", "1",
		"+app_info_print", app,
	}
	if spec.SteamModConfig != "" {
		args = append(args, "+app_set_config", app, "mod", spec.SteamModConfig)
	}
	if spec.SteamBranch != "" {
		args = append(args, "+app_update", app, "-beta", spec.SteamBranch, "validate")
	} else {
		args = append(args, "+app_update", app, "validate")
	}
	return append(args, "+quit")
}

// steamcmdNoise отсеивает выдачу app_info_print: это несколько тысяч строк
// VDF, и в журнале установки они прячут и прогресс, и ошибку.
func steamcmdNoise(line string) bool {
	s := strings.TrimSpace(line)
	if s == "{" || s == "}" {
		return true
	}
	return strings.HasPrefix(s, `"`)
}

func installArchive(ctx context.Context, dataDir, url string, report ProgressFunc) error {
	if report == nil {
		report = noopProgress
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("archive_url должен быть http(s) ссылкой")
	}

	tmpDir, err := os.MkdirTemp("", "vtx-archive-")
	if err != nil {
		return fmt.Errorf("временный каталог: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "archive")
	report(StageDownload, 0, "Скачивание файлов сервера", url)
	meta, err := download(ctx, url, archivePath, report)
	if err != nil {
		return err
	}

	kind := archiveKindFromURL(url)
	if kind == "" {
		kind = archiveKindFromResponse(meta)
	}

	switch kind {
	case archiveJar:
		report(StageExtract, 90, "Размещение server.jar", "")
		if err := moveFile(archivePath, filepath.Join(dataDir, "server.jar")); err != nil {
			return err
		}
		report(StageDone, 100, "Файлы сервера установлены", "")
		return nil
	case archiveZip, archiveTarGz:
	default:
		return fmt.Errorf("не удалось определить формат архива: нужен .zip, .tar.gz или .jar")
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return fmt.Errorf("каталог распаковки: %w", err)
	}
	report(StageExtract, 60, "Распаковка архива", "")
	if kind == archiveTarGz {
		err = extractTarGz(archivePath, extractDir)
	} else {
		err = extractZip(archivePath, extractDir)
	}
	if err != nil {
		return err
	}

	report(StageExtract, 90, "Размещение файлов сервера", "")
	if err := moveIntoData(extractDir, dataDir); err != nil {
		return err
	}
	report(StageDone, 100, "Файлы сервера установлены", "")
	return nil
}

const (
	archiveZip   = "zip"
	archiveTarGz = "tar.gz"
	archiveJar   = "jar"
)

func archiveKindFromURL(url string) string {
	lower := strings.ToLower(url)
	if i := strings.IndexAny(lower, "?#"); i >= 0 {
		lower = lower[:i]
	}
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return archiveZip
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return archiveTarGz
	case strings.HasSuffix(lower, ".jar"):
		return archiveJar
	default:
		return ""
	}
}

type downloadMeta struct {
	ContentType        string
	ContentDisposition string
}

func archiveKindFromResponse(meta downloadMeta) string {
	if name := filenameFromDisposition(meta.ContentDisposition); name != "" {
		if kind := archiveKindFromURL(name); kind != "" {
			return kind
		}
	}
	ct := strings.ToLower(strings.TrimSpace(meta.ContentType))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch ct {
	case "application/java-archive", "application/x-java-archive":
		return archiveJar
	case "application/zip", "application/x-zip-compressed":
		return archiveZip
	case "application/gzip", "application/x-gzip", "application/x-tgz":
		return archiveTarGz
	default:
		return ""
	}
}

func filenameFromDisposition(v string) string {
	for _, part := range strings.Split(v, ";") {
		part = strings.TrimSpace(part)
		if len(part) < len("filename=") || !strings.EqualFold(part[:len("filename=")], "filename=") {
			continue
		}
		return strings.Trim(strings.TrimSpace(part[len("filename="):]), `"`)
	}
	return ""
}

func moveFile(from, to string) error {
	_ = os.RemoveAll(to)
	if err := os.Rename(from, to); err == nil {
		return nil
	}
	if err := copyPath(from, to); err != nil {
		return fmt.Errorf("перенос %s: %w", filepath.Base(to), err)
	}
	return nil
}

func download(ctx context.Context, url, dest string, report ProgressFunc) (downloadMeta, error) {
	if report == nil {
		report = noopProgress
	}
	var meta downloadMeta
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return meta, fmt.Errorf("запрос архива: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return meta, fmt.Errorf("скачивание архива: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return meta, fmt.Errorf("скачивание архива: %s", resp.Status)
	}
	meta.ContentType = resp.Header.Get("Content-Type")
	meta.ContentDisposition = resp.Header.Get("Content-Disposition")

	f, err := os.Create(dest)
	if err != nil {
		return meta, fmt.Errorf("запись архива: %w", err)
	}
	defer f.Close()
	src := io.Reader(newProgressReader(resp.Body, resp.ContentLength, report))
	if _, err := io.Copy(f, io.LimitReader(src, maxArchiveBytes)); err != nil {
		return meta, fmt.Errorf("запись архива: %w", err)
	}
	return meta, nil
}

func extractZip(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("чтение zip: %w", err)
	}
	defer zr.Close()

	var written int64
	for _, f := range zr.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("чтение %s: %w", f.Name, err)
		}
		n, err := writeFile(target, rc, f.Mode())
		rc.Close()
		if err != nil {
			return err
		}
		written += n
		if written > maxArchiveBytes {
			return fmt.Errorf("архив распаковывается в слишком большой объём")
		}
	}
	return nil
}

func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("чтение архива: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("чтение gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var written int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("чтение tar: %w", err)
		}
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			n, err := writeFile(target, tr, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			written += n
			if written > maxArchiveBytes {
				return fmt.Errorf("архив распаковывается в слишком большой объём")
			}
		}
	}
}

func writeFile(path string, src io.Reader, mode os.FileMode) (int64, error) {
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return 0, fmt.Errorf("запись %s: %w", path, err)
	}
	defer out.Close()
	n, err := io.Copy(out, io.LimitReader(src, maxArchiveBytes))
	if err != nil {
		return n, fmt.Errorf("запись %s: %w", path, err)
	}
	return n, nil
}

func safeJoin(base, name string) (string, error) {
	clean := filepath.Clean(filepath.Join(base, filepath.FromSlash(name)))
	if clean != base && !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("архив содержит путь за пределами каталога: %s", name)
	}
	return clean, nil
}

func moveIntoData(extractDir, dataDir string) error {
	src := extractDir
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return fmt.Errorf("чтение распакованного: %w", err)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		src = filepath.Join(extractDir, entries[0].Name())
		entries, err = os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("чтение распакованного: %w", err)
		}
	}
	if len(entries) == 0 {
		return fmt.Errorf("архив пустой")
	}

	for _, e := range entries {
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dataDir, e.Name())
		_ = os.RemoveAll(to)
		if err := os.Rename(from, to); err != nil {
			if err := copyPath(from, to); err != nil {
				return fmt.Errorf("перенос %s: %w", e.Name(), err)
			}
		}
	}
	return nil
}

func copyPath(from, to string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(to, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(from)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(from, e.Name()), filepath.Join(to, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = writeFile(to, src, info.Mode())
	return err
}

func WipeData(serverID string) error {
	mu := lockServer(serverID)
	mu.Lock()
	defer mu.Unlock()

	dataDir := serverDataDir(serverID)
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("чтение тома: %w", err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dataDir, e.Name())); err != nil {
			return fmt.Errorf("удаление %s: %w", e.Name(), err)
		}
	}
	return nil
}

func explainPullError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "429"), strings.Contains(msg, "too many requests"):
		return fmt.Errorf("Docker Hub временно отказывает: с этого адреса исчерпан лимит "+
			"анонимных загрузок (100 за 6 часов). Подождите или выполните docker login "+
			"на ноде. Исходная ошибка: %w", err)
	case strings.Contains(msg, "pull access denied"), strings.Contains(msg, "repository does not exist"):
		return fmt.Errorf("образ не найден в реестре: соберите его на ноде "+
			"(sh scripts/build-game-images.sh --runtimes) или укажите реестр в "+
			"VORTANIX_REGISTRY. Исходная ошибка: %w", err)
	default:
		return err
	}
}
