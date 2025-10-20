// Copyright 2025 The go-ethereum Authors
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

//go:build !windows
// +build !windows

package pathdb

import (
	"errors"
	"os"
	"syscall"
)

func isErrInvalid(err error) bool {
	if errors.Is(err, os.ErrInvalid) {
		return true
	}
	// Go >= 1.8 returns *os.PathError instead
	if patherr, ok := err.(*os.PathError); ok && patherr.Err == syscall.EINVAL {
		return true
	}
	return false
}

func syncDir(name string) error {
	// As per fsync manpage, Linux seems to expect fsync on directory, however
	// some system don't support this, so we will ignore syscall.EINVAL.
	//
	// From fsync(2):
	//   Calling fsync() does not necessarily ensure that the entry in the
	//   directory containing the file has also reached disk. For that an
	//   explicit fsync() on a file descriptor for the directory is also needed.
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := f.Sync(); err != nil && !isErrInvalid(err) {
		return err
	}
	return nil
}
