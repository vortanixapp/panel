package docker

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// serverFiles читает файлы внутри каталога сервера через os.Root: симлинк,
// подложенный из контейнера, не выведет чтение за пределы тома. Раньше конфиги
// открывались обычным os.ReadFile по склеенному пути, и ссылка вида
// server.properties -> /любой/файл заставляла агента прочитать чужой файл.
type serverFiles struct {
	root *os.Root
}

func openServerFiles(serverID string) (*serverFiles, bool) {
	root, err := serverRootFor(serverID)
	if err != nil {
		return nil, false
	}
	return &serverFiles{root: root}, true
}

func (s *serverFiles) Close() {
	if s != nil && s.root != nil {
		_ = s.root.Close()
	}
}

// readSmall возвращает содержимое небольшого файла или пустую строку.
func (s *serverFiles) readSmall(rel string) string {
	if s == nil || s.root == nil {
		return ""
	}
	name := filepath.FromSlash(rel)
	if !filepath.IsLocal(name) {
		return ""
	}
	info, err := s.root.Lstat(name)
	if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Size() > rconFileLimit {
		return ""
	}
	data, err := s.root.ReadFile(name)
	if err != nil {
		return ""
	}
	return string(data)
}

// glob подбирает файлы по шаблону, не выходя из каталога сервера.
func (s *serverFiles) glob(pattern string) []string {
	if s == nil || s.root == nil {
		return nil
	}
	matches, err := fs.Glob(s.root.FS(), path.Clean(pattern))
	if err != nil {
		return nil
	}
	return matches
}
