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
	"testing"

	pebblev1 "github.com/cockroachdb/pebble"
	pebblev2 "github.com/cockroachdb/pebble/v2"
	v2vfs "github.com/cockroachdb/pebble/v2/vfs"
	v1vfs "github.com/cockroachdb/pebble/vfs"
)

// TestPeekFormatVersionLegacyV1 verifies that PeekFormatVersion correctly
// detects a legacy pebble v1 database written in the FormatMostCompatible
// layout. Older Geth never set FormatMajorVersion, so pebble v1 defaulted to
// FormatMostCompatible, which uses the CURRENT file rather than a manifest
// marker. Pebble v2's Peek does not understand this layout and reports
// Exists=false with a nil error, so PeekFormatVersion must fall back to v1.
func TestPeekFormatVersionLegacyV1(t *testing.T) {
	dir := t.TempDir()

	// Create a v1 database with default options (no FormatMajorVersion set),
	// which yields FormatMostCompatible, exactly as legacy Geth would.
	db, err := pebblev1.Open(dir, &pebblev1.Options{})
	if err != nil {
		t.Fatalf("failed to create v1 database: %v", err)
	}
	if got := db.FormatMajorVersion(); got != pebblev1.FormatMostCompatible {
		db.Close()
		t.Fatalf("unexpected on-disk format version: have %d, want %d", got, pebblev1.FormatMostCompatible)
	}
	if err := db.Set([]byte("foo"), []byte("bar"), pebblev1.Sync); err != nil {
		db.Close()
		t.Fatalf("failed to write to v1 database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close v1 database: %v", err)
	}

	// Document the underlying pebble v2 behavior that motivates the v1
	// fallback: v2's Peek silently fails to recognize this database.
	if desc, err := pebblev2.Peek(dir, v2vfs.Default); err == nil && desc.Exists {
		t.Fatal("expected pebble v2 Peek to not recognize a FormatMostCompatible database")
	}

	exists, ver, err := PeekFormatVersion(dir)
	if err != nil {
		t.Fatalf("PeekFormatVersion returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected legacy v1 database to be detected, got exists=false")
	}
	if ver != uint64(pebblev1.FormatMostCompatible) {
		t.Fatalf("unexpected format version: have %d, want %d", ver, pebblev1.FormatMostCompatible)
	}
	// The database is too old for pebble v2, so it must be routed through v1.
	if !NeedsV1(dir) {
		t.Fatal("expected NeedsV1 to be true for a FormatMostCompatible database")
	}
}

// TestPeekFormatVersionV2 verifies that PeekFormatVersion detects a database
// written at a pebble v2 compatible format version directly via v2's Peek.
func TestPeekFormatVersionV2(t *testing.T) {
	dir := t.TempDir()

	db, err := pebblev2.Open(dir, &pebblev2.Options{
		FormatMajorVersion: formatMinV2,
	})
	if err != nil {
		t.Fatalf("failed to create v2 database: %v", err)
	}
	if err := db.Set([]byte("foo"), []byte("bar"), pebblev2.Sync); err != nil {
		db.Close()
		t.Fatalf("failed to write to v2 database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close v2 database: %v", err)
	}

	exists, ver, err := PeekFormatVersion(dir)
	if err != nil {
		t.Fatalf("PeekFormatVersion returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected v2 database to be detected, got exists=false")
	}
	if ver != uint64(formatMinV2) {
		t.Fatalf("unexpected format version: have %d, want %d", ver, formatMinV2)
	}
	if NeedsV1(dir) {
		t.Fatal("expected NeedsV1 to be false for a v2 database")
	}
}

// TestPeekFormatVersionEmpty verifies that an empty directory (a new database
// location) is reported as non-existent by both v2 and v1 Peek, rather than
// being misreported.
func TestPeekFormatVersionEmpty(t *testing.T) {
	dir := t.TempDir()

	// Sanity check that v1's Peek also reports a non-existent database here,
	// so the test exercises the "neither version found a database" branch.
	if desc, err := pebblev1.Peek(dir, v1vfs.Default); err != nil {
		t.Fatalf("v1 Peek on empty directory returned error: %v", err)
	} else if desc.Exists {
		t.Fatal("expected v1 Peek to report no database in an empty directory")
	}

	exists, _, err := PeekFormatVersion(dir)
	if err != nil {
		t.Fatalf("PeekFormatVersion returned error: %v", err)
	}
	if exists {
		t.Fatal("expected no database in an empty directory, got exists=true")
	}
	if NeedsV1(dir) {
		t.Fatal("expected NeedsV1 to be false for an empty directory")
	}
}
