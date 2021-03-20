// Copyright 2021 The go-ethereum Authors
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

// +build openbsd

package utils

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func getFreeDiskSpace(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to call Statfs: %v", err)
	}

	// Available blocks * size per block = available space in bytes
	var bavail = stat.F_bavail
	// Not sure if the following check is necessary for OpenBSD
	if stat.F_bavail < 0 {
		// FreeBSD can have a negative number of blocks available
		// because of the grace limit.
		bavail = 0
	}
	//nolint:unconvert
	return uint64(bavail) * uint64(stat.F_bsize), nil
}
