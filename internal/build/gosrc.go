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
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// EnsureGoSources ensures that path contains a file with the given SHA256 hash,
// and if not, it downloads a fresh Go source package from upstream and replaces
// path with it (if the hash matches).
func EnsureGoSources(version string, hash []byte, path string) error {
	// Sanity check the destination path to ensure we don't do weird things
	if !strings.HasSuffix(path, ".tar.gz") {
		return fmt.Errorf("destination path (%s) must end with .tar.gz", path)
	}
	// If the file exists, validate it's hash
	if archive, err := ioutil.ReadFile(path); err == nil { // Go sources are ~20MB, it's fine to read all
		hasher := sha256.New()
		hasher.Write(archive)
		have := hasher.Sum(nil)

		if bytes.Equal(have, hash) {
			fmt.Printf("Go %s [%x] available at %s\n", version, hash, path)
			return nil
		}
		fmt.Printf("Go %s hash mismatch (have %x, want %x) at %s, deleting old archive\n", version, have, hash, path)
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	// Archive missing or bad hash, download a new one
	fmt.Printf("Downloading Go %s [want %x] into %s\n", version, hash, path)

	res, err := http.Get(fmt.Sprintf("https://dl.google.com/go/go%s.src.tar.gz", version))
	if err != nil || res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to access Go sources: code %d, err %v", res.StatusCode, err)
	}
	defer res.Body.Close()

	archive, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	// Sanity check the downloaded archive, save if checks out
	hasher := sha256.New()
	hasher.Write(archive)

	if have := hasher.Sum(nil); !bytes.Equal(have, hash) {
		return fmt.Errorf("downloaded Go %s hash mismatch (have %x, want %x)", version, have, hash)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, archive, 0644); err != nil {
		return err
	}
	fmt.Printf("Downloaded Go %s [%x] into %s\n", version, hash, path)
	return nil
}
