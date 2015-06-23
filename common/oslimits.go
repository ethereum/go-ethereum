// +build !windows

package common

import (
	"syscall"
)

func MaxOpenFileLimit() int {
	var nofile syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &nofile)
	if err != nil {
		return 1024
	}
	return int(nofile.Max)
}
