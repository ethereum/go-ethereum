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
	"github.com/ethereum/go-ethereum/ethdb"
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

// TestMigrationCursorStorage pins both directions' cursor roundtrips and
// the present-but-corrupt probe the resolvers rely on.
func TestMigrationCursorStorage(t *testing.T) {
	cases := []struct {
		name  string
		key   []byte
		has   func(ethdb.KeyValueReader) bool
		read  func(ethdb.KeyValueReader) (uint64, common.Hash, common.Hash, bool)
		write func(ethdb.KeyValueWriter, uint64, common.Hash, common.Hash)
	}{
		{"pbt", pbtMigrationCursorKey, HasPBTMigrationCursor, ReadPBTMigrationCursor, WritePBTMigrationCursor},
		{"mpt", mptMigrationCursorKey, HasMPTMigrationCursor, ReadMPTMigrationCursor, WriteMPTMigrationCursor},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := NewMemoryDatabase()
			if _, _, _, ok := tc.read(db); ok || tc.has(db) {
				t.Fatal("cursor present on an empty database")
			}
			tc.write(db, 42, common.Hash{0x02}, common.Hash{0xbb})
			num, hash, root, ok := tc.read(db)
			if !ok || num != 42 || hash != (common.Hash{0x02}) || root != (common.Hash{0xbb}) {
				t.Fatalf("cursor = %d %x %x %v, want 42, 0x02.., 0xbb.., true", num, hash, root, ok)
			}
			if err := db.Put(tc.key, []byte{0x01, 0x02}); err != nil {
				t.Fatal(err)
			}
			if _, _, _, ok := tc.read(db); ok {
				t.Fatal("truncated cursor reported present")
			}
			if !tc.has(db) {
				t.Fatal("truncated cursor not detected: the seeding fallbacks would run")
			}
		})
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
