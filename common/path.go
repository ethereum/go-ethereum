// Copyright 2014 The go-ethereum Authors
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

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// MakeName creates a node name that follows the ethereum convention
// for such names. It adds the operation system name and Go runtime version
// the name.
func MakeName(name, version string) string {
	return fmt.Sprintf("%s/v%s/%s/%s", name, version, runtime.GOOS, runtime.Version())
}

// FileExist checks if a file exists at filePath.
func FileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

// AbsolutePath returns datadir + filename, or filename if it is absolute.
func AbsolutePath(datadir string, filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(datadir, filename)
}

// VerifyPath sanitizes the path to avoid Path Traversal vulnerability
func VerifyPath(path string) (string, error) {
	c := filepath.Clean(path)

	r, err := filepath.EvalSymlinks(c)
	if err != nil {
		return c, fmt.Errorf("unsafe or invalid path specified: %s", path)
	} else {
		return r, nil
	}
}

// VerifyCrasher sanitizes the path to avoid Path Traversal vulnerability and reads the file from that path, returning its content
func VerifyCrasher(crasher string) []byte {
	canonicalPath, err := VerifyPath(crasher)
	if err != nil {
		fmt.Println("path not verified: " + err.Error())
		return nil
	}

	data, err := os.ReadFile(canonicalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading crasher %v: %v", canonicalPath, err)
		os.Exit(1)
	}

	return data
}
