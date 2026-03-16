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

package pebble

import (
	"fmt"
	"runtime"

	pebblev1 "github.com/cockroachdb/pebble"
	v1bloom "github.com/cockroachdb/pebble/bloom"
	pebblev2 "github.com/cockroachdb/pebble/v2"
	v2vfs "github.com/cockroachdb/pebble/v2/vfs"
	v1vfs "github.com/cockroachdb/pebble/vfs"
	"github.com/ethereum/go-ethereum/log"
)

// formatMinV2 is the minimum FormatMajorVersion supported by pebble v2.
// Databases with a lower format version must be opened with pebble v1.
const formatMinV2 = pebblev2.FormatFlushableIngest

// PeekFormatVersion reads the format version of an existing pebble database
// without opening it. It returns 0 if the database does not exist.
func PeekFormatVersion(file string) (bool, uint64, error) {
	desc, err := pebblev2.Peek(file, v2vfs.Default)
	if err != nil {
		// Pebble v2 Peek may fail on very old databases that don't have
		// the format version marker. Try v1 Peek as fallback.
		desc1, err1 := pebblev1.Peek(file, v1vfs.Default)
		if err1 != nil {
			return false, 0, err // Return original v2 error
		}
		if !desc1.Exists {
			return false, 0, nil
		}
		return true, uint64(desc1.FormatMajorVersion), nil
	}
	if !desc.Exists {
		return false, 0, nil
	}
	return true, uint64(desc.FormatMajorVersion), nil
}

// NeedsV1 returns true if the database at the given path requires pebble v1
// to open (format version too old for pebble v2).
func NeedsV1(file string) bool {
	exists, ver, err := PeekFormatVersion(file)
	if err != nil || !exists {
		return false // New database or error; use v2
	}
	return pebblev2.FormatMajorVersion(ver) < formatMinV2
}

// Upgrade upgrades an existing pebble v1 database to be compatible with pebble v2.
// It opens the database with pebble v1 at its current format version, then uses
// RatchetFormatMajorVersion to migrate to FormatFlushableIngest (the minimum format
// version that pebble v2 supports).
//
// Notably, it's not an irreversible upgrade, the database can still be opened with
// legacy Geth binary.
func Upgrade(file string) error {
	exists, ver, err := PeekFormatVersion(file)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("pebble database not found at %s", file)
	}
	if pebblev2.FormatMajorVersion(ver) >= formatMinV2 {
		log.Info("Database format already compatible with pebble v2", "version", ver)
		return nil
	}
	v1Target := pebblev1.FormatFlushableIngest
	log.Info("Upgrading pebble database format via v1", "from", ver, "to", v1Target)

	numCPU := runtime.NumCPU()
	opt := &pebblev1.Options{
		// Open at the current on-disk format version; do not request a
		// higher version here so that the upgrade happens explicitly via
		// RatchetFormatMajorVersion below.
		MaxConcurrentCompactions: func() int { return numCPU },
		Levels: []pebblev1.LevelOptions{
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: v1bloom.FilterPolicy(10)},
			{TargetFileSize: 4 * 1024 * 1024, FilterPolicy: v1bloom.FilterPolicy(10)},
			{TargetFileSize: 8 * 1024 * 1024, FilterPolicy: v1bloom.FilterPolicy(10)},
			{TargetFileSize: 16 * 1024 * 1024, FilterPolicy: v1bloom.FilterPolicy(10)},
			{TargetFileSize: 32 * 1024 * 1024, FilterPolicy: v1bloom.FilterPolicy(10)},
			{TargetFileSize: 64 * 1024 * 1024, FilterPolicy: v1bloom.FilterPolicy(10)},
			{TargetFileSize: 128 * 1024 * 1024},
		},
		Logger: panicLogger{},
	}
	db, err := pebblev1.Open(file, opt)
	if err != nil {
		return fmt.Errorf("failed to open database with pebble v1 for upgrade: %w", err)
	}
	if err := db.RatchetFormatMajorVersion(v1Target); err != nil {
		db.Close()
		return fmt.Errorf("failed to ratchet format version to %d: %w", v1Target, err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database after v1 upgrade: %w", err)
	}
	log.Info("Pebble v1 format upgrade complete, verifying v2 compatibility")

	// Verify that pebble v2 can open the upgraded database.
	opt2 := &pebblev2.Options{
		Logger: panicLogger{},
	}
	db2, err := pebblev2.Open(file, opt2)
	if err != nil {
		return fmt.Errorf("failed to open database with pebble v2 after upgrade: %w", err)
	}
	if err := db2.Close(); err != nil {
		return fmt.Errorf("failed to close database after v2 verification: %w", err)
	}
	log.Info("Pebble database format upgrade complete")
	return nil
}
