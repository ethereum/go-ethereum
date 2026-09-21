// Copyright 2026 The go-ethereum Authors
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

// Package cp provides file and directory copying helpers.
package cp

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile copies the file at src to dst. The new file is created with the
// same mode as src. It fails if dst already exists or if src is a directory.
func CopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("cp: cannot copy directory %s", src)
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// CopyAll recursively copies the file or directory tree rooted at src to dst.
// Directories are created with the same mode as their counterparts in src.
// It fails if dst (or any file within it) already exists.
func CopyAll(dst, src string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, strings.TrimPrefix(path, src))
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.Mkdir(dstPath, info.Mode())
		}
		return CopyFile(dstPath, path)
	})
}
