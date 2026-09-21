//go:build !windows

package docker

import "syscall"

func diskInfo(path string) (totalMB, usedMB int64, percent float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0
	}
	bs := int64(st.Bsize)
	total := int64(st.Blocks) * bs
	avail := int64(st.Bavail) * bs
	used := total - avail
	if total <= 0 {
		return 0, 0, 0
	}
	return total / (1024 * 1024), used / (1024 * 1024),
		clampPercent(float64(used) / float64(total) * 100)
}

func diskFreeInfo(path string) (freeMB int64, inodesPercent float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	freeMB = int64(st.Bavail) * int64(st.Bsize) / (1024 * 1024)
	if st.Files > 0 {
		usedInodes := float64(st.Files - st.Ffree)
		inodesPercent = clampPercent(usedInodes / float64(st.Files) * 100)
	}
	return freeMB, inodesPercent
}
