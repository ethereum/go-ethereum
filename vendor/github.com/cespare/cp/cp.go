// Package cp offers simple file and directory copying for Go.
package cp

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

var errCopyFileWithDir = errors.New("dir argument to CopyFile")

const (
	flagDefault   = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	flagOverwrite = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
)

// CopyFile copies the file at src to dst. The new file must not exist.
// It is created with the same permissions as src.
func CopyFile(dst, src string) error {
	return copyFile(dst, src, flagDefault)
}

// CopyFileOverwrite is like CopyFile except that it overwrites dst
// if it already exists.
func CopyFileOverwrite(dst, src string) error {
	return copyFile(dst, src, flagOverwrite)
}

func copyFile(dst, src string, flag int) error {
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

	wf, err := os.OpenFile(dst, flag, rstat.Mode())
	if err != nil {
		return err
	}
	defer wf.Close()
	if flag&os.O_EXCL == 0 {
		// We may be overwriting an existing file.
		// Ensure the file mode matches.
		stat, err := wf.Stat()
		if err != nil {
			return err
		}
		if stat.Mode() != rstat.Mode() {
			if err := wf.Chmod(rstat.Mode()); err != nil {
				return err
			}
		}
	}
	if _, err := io.Copy(wf, rf); err != nil {
		return err
	}
	return wf.Close()
}

// CopyAll copies the file or (recursively) the directory at src to dst.
// Permissions are preserved. The target directory must not already exist.
func CopyAll(dst, src string) error {
	return filepath.Walk(src, makeWalkFn(dst, src, flagDefault))
}

// CopyAllOverwrite is like CopyAll except that it recursively overwrites
// any existing directories or files.
func CopyAllOverwrite(dst, src string) error {
	return filepath.Walk(src, makeWalkFn(dst, src, flagOverwrite))
}

func makeWalkFn(dst, src string, flag int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			// Given the Walk contract, Rel must succeed.
			panic("shouldn't happen")
		}
		dstPath := filepath.Join(dst, rel)
		if info.IsDir() {
			err := os.Mkdir(dstPath, info.Mode())
			// In overwrite mode, allow the directory to already exist
			// (but make sure the permissions match).
			if os.IsExist(err) && flag&os.O_EXCL == 0 {
				return os.Chmod(dstPath, info.Mode())
			}
			return err
		}
		return copyFile(dstPath, path, flag)
	}
}
