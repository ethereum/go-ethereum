// Copyright 2025 The go-ethereum Authors
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

// Package eradl implements downloading of era1 files.
package eradl

import (
	_ "embed"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/ethereum/go-ethereum/internal/download"
	"github.com/ethereum/go-ethereum/internal/era"
)

//go:embed checksums_mainnet.txt
var mainnetDB []byte

//go:embed checksums_sepolia.txt
var sepoliaDB []byte

type Loader struct {
	csdb    *download.ChecksumDB
	network string
	baseURL *url.URL
}

// New creates an era1 loader for the given server URL and network name.
func New(baseURL string, network string) (*Loader, error) {
	var checksums []byte
	switch network {
	case "mainnet":
		checksums = mainnetDB
	case "sepolia":
		checksums = sepoliaDB
	default:
		return nil, fmt.Errorf("missing era1 checksum definitions for network %q", network)
	}

	csdb, err := download.ParseChecksums(checksums)
	if err != nil {
		return nil, fmt.Errorf("invalid checksums: %v", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL %q: %v", baseURL, err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("invalid base URL scheme, expected http(s): %q", baseURL)
	}

	l := &Loader{
		network: network,
		csdb:    csdb,
		baseURL: base,
	}
	return l, nil
}

// DownloadAll downloads all known era1 files to the given directory.
func (l *Loader) DownloadAll(destDir string) error {
	for file := range l.csdb.Files() {
		if err := l.download(file, destDir); err != nil {
			return err
		}
	}
	return nil
}

// DownloadBlockRange fetches the era1 files for the given block range.
func (l *Loader) DownloadBlockRange(start, end uint64, destDir string) error {
	startEpoch := start / uint64(era.MaxEra1Size)
	endEpoch := end / uint64(era.MaxEra1Size)
	return l.DownloadEpochRange(startEpoch, endEpoch, destDir)
}

// DownloadEpochRange fetches the era1 files in the given epoch range.
func (l *Loader) DownloadEpochRange(start, end uint64, destDir string) error {
	pat := regexp.MustCompile(regexp.QuoteMeta(l.network) + "-([0-9]+)-[0-9a-f]+\\.era1")
	for file := range l.csdb.Files() {
		m := pat.FindStringSubmatch(file)
		if len(m) == 2 {
			fileEpoch, _ := strconv.Atoi(m[1])
			if uint64(fileEpoch) >= start && uint64(fileEpoch) <= end {
				if err := l.download(file, destDir); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (l *Loader) download(file, destDir string) error {
	url := l.baseURL.JoinPath(file).String()
	dest := filepath.Join(destDir, file)
	return l.csdb.DownloadFile(url, dest)
}
