// +build !windows,!appengine

package maxminddb

import (
	"golang.org/x/sys/unix"
)

func mmap(fd int, length int) (data []byte, err error) {
	return unix.Mmap(fd, 0, length, unix.PROT_READ, unix.MAP_SHARED)
}

func munmap(b []byte) (err error) {
	return unix.Munmap(b)
}
