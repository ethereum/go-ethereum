// Copyright 2026 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestShadowStateRootStorage exercises the shadow-root table: absent reads are
// zero, roundtrips return what was written, and deletes clear.
func TestShadowStateRootStorage(t *testing.T) {
	db := NewMemoryDatabase()
	var (
		hash = common.Hash{0x01}
		root = common.Hash{0xaa}
	)
	if got := ReadShadowStateRoot(db, 7, hash); got != (common.Hash{}) {
		t.Fatalf("unwritten shadow root = %x, want zero", got)
	}
	WriteShadowStateRoot(db, 7, hash, root)
	if got := ReadShadowStateRoot(db, 7, hash); got != root {
		t.Fatalf("shadow root = %x, want %x", got, root)
	}
	if got := ReadShadowStateRoot(db, 8, hash); got != (common.Hash{}) {
		t.Fatalf("wrong-number read = %x, want zero", got)
	}
	DeleteShadowStateRoot(db, 7, hash)
	if got := ReadShadowStateRoot(db, 7, hash); got != (common.Hash{}) {
		t.Fatalf("deleted shadow root = %x, want zero", got)
	}
}

// TestPBTMigrationCursorStorage pins the cursor roundtrip and that a missing
// or truncated record reports not-ok rather than garbage.
func TestPBTMigrationCursorStorage(t *testing.T) {
	db := NewMemoryDatabase()
	if _, _, _, ok := ReadPBTMigrationCursor(db); ok {
		t.Fatal("cursor reported present on an empty database")
	}
	WritePBTMigrationCursor(db, 42, common.Hash{0x02}, common.Hash{0xbb})
	num, hash, root, ok := ReadPBTMigrationCursor(db)
	if !ok || num != 42 || hash != (common.Hash{0x02}) || root != (common.Hash{0xbb}) {
		t.Fatalf("cursor = %d %x %x %v, want 42, 0x02.., 0xbb.., true", num, hash, root, ok)
	}
	if err := db.Put(pbtMigrationCursorKey, []byte{0x01, 0x02}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := ReadPBTMigrationCursor(db); ok {
		t.Fatal("truncated cursor reported present")
	}
}

// TestPBTMigrationDoneFlag pins the terminal marker: absent means running.
func TestPBTMigrationDoneFlag(t *testing.T) {
	db := NewMemoryDatabase()
	if ReadPBTMigrationDone(db) {
		t.Fatal("migration done on an empty database")
	}
	WritePBTMigrationDone(db)
	if !ReadPBTMigrationDone(db) {
		t.Fatal("migration not done after writing the marker")
	}
}
