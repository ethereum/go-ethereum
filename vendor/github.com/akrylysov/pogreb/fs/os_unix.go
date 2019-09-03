// +build !windows

package fs

import (
	"os"
	"syscall"
	"unsafe"
)

func mmap(f *os.File, fileSize int64, mmapSize int64) ([]byte, int64, error) {
	p, err := syscall.Mmap(int(f.Fd()), 0, int(mmapSize), syscall.PROT_READ, syscall.MAP_SHARED)
	return p, mmapSize, err
}

func munmap(data []byte) error {
	return syscall.Munmap(data)
}

func madviceRandom(data []byte) error {
	_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(syscall.MADV_RANDOM))
	if errno != 0 {
		return errno
	}
	return nil
}

func createLockFile(name string, perm os.FileMode) (LockFile, bool, error) {
	acquiredExisting := false
	if _, err := os.Stat(name); err == nil {
		acquiredExisting = true
	}
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			err = os.ErrExist
		}
		return nil, false, err
	}
	return &oslockfile{f, name}, acquiredExisting, nil
}
