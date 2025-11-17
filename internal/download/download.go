// Copyright 2019 The go-ethereum Authors
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

// Package download implements checksum-verified file downloads.
package download

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ChecksumDB keeps file checksums and tool versions.
type ChecksumDB struct {
	hashes   []hashEntry
	versions []versionEntry
}

type versionEntry struct {
	name    string
	version string
}

type hashEntry struct {
	hash string
	file string
	url  *url.URL
}

// MustLoadChecksums loads a file containing checksums.
func MustLoadChecksums(file string) *ChecksumDB {
	content, err := os.ReadFile(file)
	if err != nil {
		panic("can't load checksum file: " + err.Error())
	}
	db, err := ParseChecksums(content)
	if err != nil {
		panic(fmt.Sprintf("invalid checksums in %s: %v", file, err))
	}
	return db
}

// ParseChecksums parses a checksum database.
func ParseChecksums(input []byte) (*ChecksumDB, error) {
	var (
		csdb    = new(ChecksumDB)
		rd      = bytes.NewBuffer(input)
		lastURL *url.URL
	)
	for lineNum := 1; ; lineNum++ {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			// Blank lines are allowed, and they reset the current urlEntry.
			lastURL = nil

		case strings.HasPrefix(line, "#"):
			// It's a comment. Some comments have special meaning.
			content := strings.TrimLeft(line, "# ")
			switch {
			case strings.HasPrefix(content, "version:"):
				// Version comments define the version of a tool.
				v := strings.Split(content, ":")[1]
				parts := strings.Split(v, " ")
				if len(parts) != 2 {
					return nil, fmt.Errorf("line %d: invalid version string: %q", lineNum, v)
				}
				csdb.versions = append(csdb.versions, versionEntry{parts[0], parts[1]})

			case strings.HasPrefix(content, "https://") || strings.HasPrefix(content, "http://"):
				// URL comments define the URL where the following files are found. Here
				// we keep track of the last found urlEntry and attach it to each file later.
				u, err := url.Parse(content)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid URL: %v", lineNum, err)
				}
				lastURL = u
			}

		default:
			// It's a file hash entry.
			fields := strings.Fields(line)
			if len(fields) != 2 {
				return nil, fmt.Errorf("line %d: invalid number of space-separated fields (%d)", lineNum, len(fields))
			}
			csdb.hashes = append(csdb.hashes, hashEntry{fields[0], fields[1], lastURL})
		}
	}
	return csdb, nil
}

// Files returns an iterator over all file names.
func (db *ChecksumDB) Files() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, e := range db.hashes {
			if !yield(e.file) {
				return
			}
		}
	}
}

// DownloadAndVerifyAll downloads all files and checks that they match the checksum given in
// the database. This task can be used to sanity-check new checksums.
func (db *ChecksumDB) DownloadAndVerifyAll() {
	var tmp = os.TempDir()
	for _, e := range db.hashes {
		if e.url == nil {
			fmt.Printf("Skipping verification of %s: no URL defined in checksum database", e.file)
			continue
		}
		url := e.url.JoinPath(e.file).String()
		dst := filepath.Join(tmp, e.file)
		if err := db.DownloadFile(url, dst); err != nil {
			fmt.Println("error:", err)
		}
	}
}

// verifyHash checks that the file at 'path' has the expected hash.
func verifyHash(path, expectedHash string) error {
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	h := sha256.New()
	if _, err := io.Copy(h, bufio.NewReader(fd)); err != nil {
		return err
	}
	fileHash := hex.EncodeToString(h.Sum(nil))
	if fileHash != expectedHash {
		return fmt.Errorf("invalid file hash: %s %s", fileHash, filepath.Base(path))
	}
	return nil
}

// DownloadFileFromKnownURL downloads a file from the URL defined in the checksum database.
func (db *ChecksumDB) DownloadFileFromKnownURL(dstPath string) error {
	base := filepath.Base(dstPath)
	url, err := db.FindURL(base)
	if err != nil {
		return err
	}
	return db.DownloadFile(url, dstPath)
}

// DownloadFile downloads a file and verifies its checksum.
func (db *ChecksumDB) DownloadFile(url, dstPath string) error {
	basename := filepath.Base(dstPath)
	hash := db.findHash(basename)
	if hash == "" {
		return fmt.Errorf("no known hash for file %q", basename)
	}
	// Shortcut if already downloaded.
	if verifyHash(dstPath, hash) == nil {
		fmt.Printf("%s is up-to-date\n", dstPath)
		return nil
	}

	fmt.Printf("%s is stale\n", dstPath)
	fmt.Printf("downloading from %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download error: status %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	// Download to a temporary file.
	tmpfile := dstPath + ".tmp"
	fd, err := os.OpenFile(tmpfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	dst := newDownloadWriter(fd, resp.ContentLength)
	_, err = io.Copy(dst, resp.Body)
	dst.Close()
	if err != nil {
		os.Remove(tmpfile)
		return err
	}
	if err := verifyHash(tmpfile, hash); err != nil {
		os.Remove(tmpfile)
		return err
	}
	// It's valid, rename to dstPath to complete the download.
	return os.Rename(tmpfile, dstPath)
}

// findHash returns the known hash of a file.
func (db *ChecksumDB) findHash(basename string) string {
	for _, e := range db.hashes {
		if e.file == basename {
			return e.hash
		}
	}
	return ""
}

// FindVersion returns the current known version of a tool, if it is defined in the file.
func (db *ChecksumDB) FindVersion(tool string) (string, error) {
	for _, e := range db.versions {
		if e.name == tool {
			return e.version, nil
		}
	}
	return "", fmt.Errorf("tool version %q not defined in checksum database", tool)
}

// FindURL gets the URL for a file.
func (db *ChecksumDB) FindURL(basename string) (string, error) {
	for _, e := range db.hashes {
		if e.file == basename {
			if e.url == nil {
				return "", fmt.Errorf("file %q has no URL defined", e.file)
			}
			return e.url.JoinPath(e.file).String(), nil
		}
	}
	return "", fmt.Errorf("file %q does not exist in checksum database", basename)
}

type downloadWriter struct {
	file    *os.File
	dstBuf  *bufio.Writer
	size    int64
	written int64
	lastpct int64
}

func newDownloadWriter(dst *os.File, size int64) *downloadWriter {
	return &downloadWriter{
		file:   dst,
		dstBuf: bufio.NewWriter(dst),
		size:   size,
	}
}

func (w *downloadWriter) Write(buf []byte) (int, error) {
	n, err := w.dstBuf.Write(buf)

	// Report progress.
	w.written += int64(n)
	pct := w.written * 10 / w.size * 10
	if pct != w.lastpct {
		if w.lastpct != 0 {
			fmt.Print("...")
		}
		fmt.Print(pct, "%")
		w.lastpct = pct
	}
	return n, err
}

func (w *downloadWriter) Close() error {
	if w.lastpct > 0 {
		fmt.Println() // Finish the progress line.
	}
	flushErr := w.dstBuf.Flush()
	closeErr := w.file.Close()
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}
