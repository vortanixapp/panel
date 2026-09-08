package docker

import (
	"path/filepath"
	"strings"
	"testing"
)

// Путь в кэше приходит от панели, и по нему агент СОЗДАЁТ файл. Значит выход за
// пределы кэша означал бы запись куда угодно на ноде — от root, из контейнера
// с примонтированным docker.sock.
func TestArchiveCacheFileStaysInsideCacheDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VORTANIX_PLUGIN_CACHE_DIR", dir)

	good, err := archiveCacheFile("plugins/abc-100.archive")
	if err != nil {
		t.Fatalf("обычный путь отвергнут: %v", err)
	}
	if !strings.HasPrefix(good, dir) {
		t.Errorf("файл оказался вне кэша: %s", good)
	}
	if filepath.Base(good) != "abc-100.archive" {
		t.Errorf("имя файла потерялось: %s", good)
	}

	for _, bad := range []string{
		"../../etc/cron.d/evil",
		"plugins/../../../root/.ssh/authorized_keys",
		"",
		"   ",
	} {
		if _, err := archiveCacheFile(bad); err == nil {
			t.Errorf("выход за пределы кэша не пойман: %q", bad)
		}
	}
}
