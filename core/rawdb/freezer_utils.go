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

package rawdb

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// openFileForAppend opens a file in append-mode and seeks to the end.
func openFileForAppend(filename string) (*os.File, error) {
	// Open the file without the O_APPEND flag
	// because it has differing behaviour during Truncate operations
	// on different OS's
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	// Seek to end for append
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}
	return file, nil
}

// openFileForReadOnly opens a file for read only access.
func openFileForReadOnly(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDONLY, 0644)
}

// openFileTruncated opens a file making sure it is truncated.
func openFileTruncated(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

// truncateFile resizes a file to the given size, and seeks to the end.
func truncateFile(file *os.File, size int64) error {
	if err := file.Truncate(size); err != nil {
		return err
	}
	// Seek to end for append
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return nil
}

// copyFrom copies data from 'srcPath' at offset 'offset' into 'destPath'.
// The 'destPath' is created if it doesn't exist, otherwise it is overwritten.
// It is perfectly valid to have destPath == srcPath.
func copyFrom(srcPath, destPath string, offset uint64) error {
	// Create a temp file in the same dir where we want it to wind up
	f, err := ioutil.TempFile(filepath.Dir(destPath), "copy-tmp-*")
	if err != nil {
		return err
	}
	tmpFilePath := f.Name()
	defer func() {
		if f != nil {
			f.Close()
		}
		os.Remove(tmpFilePath)
	}()
	// Open the source file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	if _, err = src.Seek(int64(offset), 0); err != nil {
		src.Close()
		return err
	}
	// io.Copy uses 32K buffer internally.
	_, err = io.Copy(f, src)
	// src may be same as dest, so needs to be closed before we do the
	// final move.
	src.Close()
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	f = nil
	// Now change tempfile into the actual destination file
	if err := os.Rename(tmpFilePath, destPath); err != nil {
		return err
	}
	return nil
}

func iterateIndexFile(from uint64, indexFile *os.File, callback func(entry *indexEntry) bool) error {
	// Apply the table-offset
	//from = from - t.itemOffset
	for {
		buffer := make([]byte, indexEntrySize)
		if _, err := indexFile.ReadAt(buffer, int64(from*indexEntrySize)); err != nil {
			return err
		}
		index := new(indexEntry)
		index.unmarshalBinary(buffer)
		if !callback(index) {
			break
		}
	}
	return nil
}
