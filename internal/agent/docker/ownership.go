package docker

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Владельца и права внутри каталога сервера правим только через os.Root: он не
// идёт за симлинками наружу. Раньше это делал шелл (stat и chown -h), а шелл
// раскрывает промежуточные звенья пути, и подложенный из контейнера симлинк
// выводил операцию за пределы тома.

func ownerOf(root *os.Root, name string) (uid, gid int, ok bool) {
	if name == "" {
		name = "."
	}
	info, err := root.Lstat(name)
	if err != nil {
		return 0, 0, false
	}
	return fileOwner(info)
}

// applyOwnership переносит владельца reference на target, не разыменовывая
// симлинки ни в одном звене пути.
func applyOwnership(root *os.Root, reference, target string) {
	uid, gid, ok := ownerOf(root, reference)
	if !ok {
		return
	}
	_ = root.Lchown(target, uid, gid)
}

// reapplyOwnershipTree заново назначает владельца всему дереву и открывает
// группе доступ к файлам: этим живёт доступ по SFTP. Симлинки получают только
// нового владельца — права у них не меняются, за ссылку не ходим.
func reapplyOwnershipTree(root *os.Root) error {
	uid, gid, ok := ownerOf(root, ".")
	if !ok {
		return nil
	}
	return fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := filepath.FromSlash(p)
		_ = root.Lchown(name, uid, gid)
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, statErr := root.Lstat(name)
		if statErr != nil {
			return nil
		}
		mode := info.Mode().Perm()
		next := mode | 0o060
		if d.IsDir() || mode&0o100 != 0 {
			next |= 0o010
		}
		if next != mode {
			_ = root.Chmod(name, next)
		}
		return nil
	})
}
