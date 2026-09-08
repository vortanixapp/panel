package handlers

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Путь к архиву лежит в базе и приходит из формы администратора. Разворачивать
// его в абсолютный можно только с проверкой: иначе '..' в значении выводит
// чтение и удаление файлов за пределы хранилища.
func TestResolveCatalogPathStaysInsideStorage(t *testing.T) {
	h := &Handler{uploadDir: t.TempDir()}

	good, err := h.resolveCatalogPath("plugins/amxmodx/plugin_ab12.zip")
	if err != nil {
		t.Fatalf("обычный путь отвергнут: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(good), "/catalog/plugins/amxmodx/") {
		t.Errorf("путь развернулся не туда: %s", good)
	}

	for _, bad := range []string{
		"../../etc/passwd",
		"plugins/../../../etc/shadow",
		"plugins/x/../../../../root/.ssh/id_rsa",
		"",
		"   ",
	} {
		if _, err := h.resolveCatalogPath(bad); err == nil {
			t.Errorf("выход за пределы хранилища не пойман: %q", bad)
		}
	}
}

func TestCatalogSafeSlugStripsPathCharacters(t *testing.T) {
	cases := map[string]string{
		"amxmodx":      "amxmodx",
		"AmxModX":      "amxmodx",
		"../../etc":    "etc",
		"a/b/c":        "abc",
		"":             "item",
		"...":          "item",
		"plug in_v1-2": "plugin_v1-2",
		"кириллица":    "item",
	}
	for in, want := range cases {
		if got := catalogSafeSlug(in); got != want {
			t.Errorf("catalogSafeSlug(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

func TestCatalogArchiveExtDetection(t *testing.T) {
	cases := []struct{ typ, name, wantType, wantExt string }{
		{"zip", "x.zip", "zip", ".zip"},
		{"targz", "x.tar.gz", "targz", ".tar.gz"},
		{"", "map.zip", "zip", ".zip"},
		{"", "bundle.tar.gz", "targz", ".tar.gz"},
		{"", "bundle.tgz", "targz", ".tar.gz"},
		{"", "plain.tar", "tar", ".tar"},
	}
	for _, c := range cases {
		gotType, gotExt, err := catalogArchiveExtFor(c.typ, c.name)
		if err != nil || gotType != c.wantType || gotExt != c.wantExt {
			t.Errorf("catalogArchiveExtFor(%q,%q) = (%q,%q,%v)", c.typ, c.name, gotType, gotExt, err)
		}
	}
	if _, _, err := catalogArchiveExtFor("", "readme.txt"); err == nil {
		t.Error("посторонний файл должен отвергаться")
	}
}

// Список файлов карты строится из zip и потом служит единственным основанием
// для удаления карты с сервера. Каталоги, выходы через '..' и дубликаты в него
// попадать не должны.
func TestZipFileListMatchesOldPanelRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "map.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{
		"maps/de_dust2.bsp",
		"maps/",
		"/maps/de_dust2.txt",
		"maps\\overviews\\de_dust2.bmp",
		"../../etc/passwd",
		"maps/de_dust2.bsp",
		"MAPS/aaa.res",
	} {
		if _, err := zw.Create(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	list, err := zipFileList(path)
	if err != nil {
		t.Fatalf("zipFileList: %v", err)
	}
	want := []string{
		"MAPS/aaa.res",
		"maps/de_dust2.bsp",
		"maps/de_dust2.txt",
		"maps/overviews/de_dust2.bmp",
	}
	if len(list) != len(want) {
		t.Fatalf("получено %v, ожидалось %v", list, want)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Errorf("позиция %d: %q, ожидалось %q (весь список: %v)", i, list[i], want[i], list)
		}
	}
}
