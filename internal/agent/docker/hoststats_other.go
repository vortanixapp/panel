//go:build windows

package docker

func diskInfo(string) (totalMB, usedMB int64, percent float64) {
	return 0, 0, 0
}

func diskFreeInfo(string) (freeMB int64, inodesPercent float64) {
	return 0, 0
}
