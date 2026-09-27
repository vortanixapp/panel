//go:build unix

package docker

import (
	"io/fs"
	"syscall"
)

func fileOwner(info fs.FileInfo) (uid, gid int, ok bool) {
	st, cast := info.Sys().(*syscall.Stat_t)
	if !cast {
		return 0, 0, false
	}
	return int(st.Uid), int(st.Gid), true
}
