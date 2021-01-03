// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// +build !windows

package utils

import (
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sys/unix"
	"syscall"
)

var stat unix.Statfs_t

func getFreeDiskSpace(path string) uint64 {

	err := unix.Statfs(path, &stat)
	if err != nil {
		log.Warn("Failed to call Statfs", "path", path, "err", err)
		sigtermCh <- syscall.SIGTERM
		return 0
	}

	// Available blocks * size per block = available space in bytes
	return stat.Bavail * uint64(stat.Bsize)
}
