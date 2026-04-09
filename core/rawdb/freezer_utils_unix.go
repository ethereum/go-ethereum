// Copyright 2022 The go-ethereum Authors
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

package rawdb

import (
	"errors"
	"os"
	"syscall"
)

// syncDir ensures that the directory metadata (e.g. newly renamed files)
// is flushed to durable storage.
func syncDir(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	// Some file systems do not support fsyncing directories (e.g. some FUSE
	// mounts). Ignore EINVAL in those cases.
	if err := f.Sync(); err != nil {
		if errors.Is(err, os.ErrInvalid) {
			return nil
		}
		if patherr, ok := err.(*os.PathError); ok && patherr.Err == syscall.EINVAL {
			return nil
		}
		return err
	}
	return nil
}
