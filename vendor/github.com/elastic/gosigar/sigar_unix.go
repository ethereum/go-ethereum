// Copyright (c) 2012 VMware, Inc.

// +build darwin freebsd linux

package gosigar

import (
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func (self *FileSystemUsage) Get(path string) error {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return err
	}

	self.Total = uint64(stat.Blocks) * uint64(stat.Bsize)
	self.Free = uint64(stat.Bfree) * uint64(stat.Bsize)
	self.Avail = uint64(stat.Bavail) * uint64(stat.Bsize)
	self.Used = self.Total - self.Free
	self.Files = stat.Files
	self.FreeFiles = uint64(stat.Ffree)

	return nil
}

func (r *Rusage) Get(who int) error {
	ru, err := getResourceUsage(who)
	if err != nil {
		return err
	}

	uTime := convertRtimeToDur(ru.Utime)
	sTime := convertRtimeToDur(ru.Stime)

	r.Utime = uTime
	r.Stime = sTime
	r.Maxrss = int64(ru.Maxrss)
	r.Ixrss = int64(ru.Ixrss)
	r.Idrss = int64(ru.Idrss)
	r.Isrss = int64(ru.Isrss)
	r.Minflt = int64(ru.Minflt)
	r.Majflt = int64(ru.Majflt)
	r.Nswap = int64(ru.Nswap)
	r.Inblock = int64(ru.Inblock)
	r.Oublock = int64(ru.Oublock)
	r.Msgsnd = int64(ru.Msgsnd)
	r.Msgrcv = int64(ru.Msgrcv)
	r.Nsignals = int64(ru.Nsignals)
	r.Nvcsw = int64(ru.Nvcsw)
	r.Nivcsw = int64(ru.Nivcsw)

	return nil
}

func getResourceUsage(who int) (unix.Rusage, error) {
	r := unix.Rusage{}
	err := unix.Getrusage(who, &r)

	return r, err
}

func convertRtimeToDur(t unix.Timeval) time.Duration {
	return time.Duration(t.Nano())
}
