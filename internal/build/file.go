// Copyright 2024 The go-ethereum Authors
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

package build

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileExist checks if a file exists at path.
func FileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// HashFiles iterates the provided set of files, computing the hash of each.
func HashFiles(files []string) (map[string][32]byte, error) {
	res := make(map[string][32]byte)
	for _, filePath := range files {
		f, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
		if err != nil {
			return nil, err
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return nil, err
		}
		res[filePath] = [32]byte(hasher.Sum(nil))
	}
	return res, nil
}

// HashFolder iterates all files under the given directory, computing the hash
// of each.
func HashFolder(folder string, exlude []string) (map[string][32]byte, error) {
	res := make(map[string][32]byte)
	err := filepath.WalkDir(folder, func(path string, d os.DirEntry, _ error) error {
		// Skip anything that's exluded or not a regular file
		for _, skip := range exlude {
			if strings.HasPrefix(path, filepath.FromSlash(skip)) {
				return filepath.SkipDir
			}
		}
		if !d.Type().IsRegular() {
			return nil
		}
		// Regular file found, hash it
		f, err := os.OpenFile(path, os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}
		res[path] = [32]byte(hasher.Sum(nil))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// DiffHashes compares two maps of file hashes and returns the changed files.
func DiffHashes(a map[string][32]byte, b map[string][32]byte) []string {
	var updates []string

	for file := range a {
		if _, ok := b[file]; !ok || a[file] != b[file] {
			updates = append(updates, file)
		}
	}
	for file := range b {
		if _, ok := a[file]; !ok {
			updates = append(updates, file)
		}
	}
	sort.Strings(updates)
	return updates
}
