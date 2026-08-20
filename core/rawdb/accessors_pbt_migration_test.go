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

// TestShadowStateRootStorage exercises the shadow-root table.
func TestShadowStateRootStorage(t *testing.T) {
	db := NewMemoryDatabase()
	var (
		hash = common.Hash{0x01}
		root = common.Hash{0xaa}
	)
	if _, ok := ReadShadowStateRoot(db, 7, hash); ok {
		t.Fatal("unwritten shadow root reported present")
	}
	WriteShadowStateRoot(db, 7, hash, root)
	if got, ok := ReadShadowStateRoot(db, 7, hash); !ok || got != root {
		t.Fatalf("shadow root = %x (ok=%v), want %x", got, ok, root)
	}
	// The empty binary root is the zero hash; presence must survive it.
	WriteShadowStateRoot(db, 9, hash, common.Hash{})
	if got, ok := ReadShadowStateRoot(db, 9, hash); !ok || got != (common.Hash{}) {
		t.Fatalf("zero shadow root = %x (ok=%v), want present zero", got, ok)
	}
}

// TestPBTMigrationCursorStorage pins the cursor roundtrip.
func TestPBTMigrationCursorStorage(t *testing.T) {
	db := NewMemoryDatabase()
	if _, _, _, ok := ReadPBTMigrationCursor(db); ok {
		t.Fatal("cursor reported present on an empty database")
	}
	if HasPBTMigrationCursor(db) {
		t.Fatal("cursor key present on an empty database")
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
	if !HasPBTMigrationCursor(db) {
		t.Fatal("truncated cursor not detected as present: the seeding fallbacks would run")
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

// TestMPTMigrationCursorStorage pins the merkle window cursor's roundtrip.
func TestMPTMigrationCursorStorage(t *testing.T) {
	db := NewMemoryDatabase()
	if _, _, _, ok := ReadMPTMigrationCursor(db); ok {
		t.Fatal("cursor reported present on an empty database")
	}
	if HasMPTMigrationCursor(db) {
		t.Fatal("cursor key present on an empty database")
	}
	WriteMPTMigrationCursor(db, 7, common.Hash{0x03}, common.Hash{0xcc})
	num, hash, root, ok := ReadMPTMigrationCursor(db)
	if !ok || num != 7 || hash != (common.Hash{0x03}) || root != (common.Hash{0xcc}) {
		t.Fatalf("cursor = %d %x %x %v, want 7, 0x03.., 0xcc.., true", num, hash, root, ok)
	}
	if err := db.Put(mptMigrationCursorKey, []byte{0x01}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := ReadMPTMigrationCursor(db); ok {
		t.Fatal("truncated cursor reported present")
	}
	if !HasMPTMigrationCursor(db) {
		t.Fatal("truncated cursor not detected as present")
	}
}

// TestWipeMigrationState pins that a re-anchor clears every raw-namespace
// migration key: a stale cursor would shadow the fresh anchor.
func TestWipeMigrationState(t *testing.T) {
	db := NewMemoryDatabase()
	WritePBTMigrationCursor(db, 5, common.Hash{0x01}, common.Hash{0x02})
	WriteMPTMigrationCursor(db, 6, common.Hash{0x03}, common.Hash{0x04})
	WritePBTMigrationDone(db)
	WriteShadowStateRoot(db, 5, common.Hash{0x01}, common.Hash{0x02})

	if err := WipeMigrationState(db); err != nil {
		t.Fatal(err)
	}
	if HasPBTMigrationCursor(db) || HasMPTMigrationCursor(db) || ReadPBTMigrationDone(db) {
		t.Fatal("migration keys survived the wipe")
	}
	if _, ok := ReadShadowStateRoot(db, 5, common.Hash{0x01}); ok {
		t.Fatal("shadow-root record survived the wipe")
	}
}
