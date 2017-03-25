// Package cp offers simple file and directory copying for Go.
package cp

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var errCopyFileWithDir = errors.New("dir argument to CopyFile")

// CopyFile copies the file with path src to dst. The new file must not exist.
// It is created with the same permissions as src.
func CopyFile(dst, src string) error {
	rf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer rf.Close()
	rstat, err := rf.Stat()
	if err != nil {
		return err
	}
	if rstat.IsDir() {
		return errCopyFileWithDir
	}

	wf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, rstat.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(wf, rf); err != nil {
		wf.Close()
		return err
	}
	return wf.Close()
}

// CopyAll copies the file or (recursively) the directory at src to dst.
// Permissions are preserved. dst must not already exist.
func CopyAll(dst, src string) error {
	return filepath.Walk(src, makeWalkFn(dst, src))
}

func makeWalkFn(dst, src string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, strings.TrimPrefix(path, src))
		if info.IsDir() {
			return os.Mkdir(dstPath, info.Mode())
		}
		return CopyFile(dstPath, path)
	}
}
