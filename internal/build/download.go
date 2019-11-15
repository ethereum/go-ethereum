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

package build

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ChecksumDB keeps file checksums.
type ChecksumDB struct {
	allChecksums []string
}

// MustLoadChecksums loads a file containing checksums.
func MustLoadChecksums(file string) *ChecksumDB {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal("can't load checksum file: " + err.Error())
	}
	return &ChecksumDB{strings.Split(string(content), "\n")}
}

// Verify checks whether the given file is valid according to the checksum database.
func (db *ChecksumDB) Verify(path string) error {
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
	if !db.findHash(filepath.Base(path), fileHash) {
		return fmt.Errorf("invalid file hash %s", fileHash)
	}
	return nil
}

func (db *ChecksumDB) findHash(basename, hash string) bool {
	want := hash + "  " + basename
	for _, line := range db.allChecksums {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

// DownloadFile downloads a file and verifies its checksum.
func (db *ChecksumDB) DownloadFile(url, dstPath string) error {
	if err := db.Verify(dstPath); err == nil {
		fmt.Printf("%s is up-to-date\n", dstPath)
		return nil
	}
	fmt.Printf("%s is stale\n", dstPath)
	fmt.Printf("downloading from %s\n", url)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download error: code %d, err %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	fd, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	dst := bufio.NewWriter(io.MultiWriter(fd, &dots{length: resp.ContentLength}))
	_, copyErr := io.Copy(dst, resp.Body)
	flushErr := dst.Flush()
	fd.Close()
	if copyErr != nil {
		return copyErr
	} else if flushErr != nil {
		return flushErr
	}

	return db.Verify(dstPath)
}

type dots struct {
	c       int64
	length  int64
	lastpct int64
}

func (d *dots) Write(buf []byte) (int, error) {
	d.c += int64(len(buf))
	pct := d.c * 10 / d.length * 10
	if pct != d.lastpct {
		if d.lastpct != 0 {
			fmt.Print("...")
		}
		fmt.Print(pct, "%")
		d.lastpct = pct
	}
	if pct == 100 {
		fmt.Println()
	}
	return len(buf), nil
}
