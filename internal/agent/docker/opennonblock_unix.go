//go:build unix

package docker

import "syscall"

const openNonBlock = syscall.O_NONBLOCK
