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
	if _, ok := ReadShadowStateRoot(db, hash, 7); ok {
		t.Fatal("unwritten shadow root reported present")
	}
	WriteShadowStateRoot(db, hash, 7, root)
	if got, ok := ReadShadowStateRoot(db, hash, 7); !ok || got != root {
		t.Fatalf("shadow root = %x (ok=%v), want %x", got, ok, root)
	}
	// The empty binary root is the zero hash; presence must survive it.
	WriteShadowStateRoot(db, hash, 9, common.Hash{})
	if got, ok := ReadShadowStateRoot(db, hash, 9); !ok || got != (common.Hash{}) {
		t.Fatalf("zero shadow root = %x (ok=%v), want present zero", got, ok)
	}
}

// TestMigrationCursorStorage pins the tri-state both directions rely on:
// absent is a virgin database, unreadable is a hard error, never the same.
func TestMigrationCursorStorage(t *testing.T) {
	for _, pbt := range []bool{true, false} {
		name := "mpt"
		key := mptMigrationCursorKey
		if pbt {
			name, key = "pbt", pbtMigrationCursorKey
		}
		t.Run(name, func(t *testing.T) {
			db := NewMemoryDatabase()
			if _, ok, err := ReadMigrationCursor(db, pbt); ok || err != nil {
				t.Fatalf("empty database: ok=%v err=%v, want a virgin read", ok, err)
			}
			want := MigrationCursor{Number: 42, Hash: common.Hash{0x02}, Root: common.Hash{0xbb}}
			WriteMigrationCursor(db, pbt, want)
			got, ok, err := ReadMigrationCursor(db, pbt)
			if !ok || err != nil || got != want {
				t.Fatalf("cursor = %+v (ok=%v err=%v), want %+v", got, ok, err, want)
			}
			if _, ok, _ := ReadMigrationCursor(db, !pbt); ok {
				t.Fatal("the other direction read this cursor")
			}
			if err := db.Put(key, []byte{0x01, 0x02}); err != nil {
				t.Fatal(err)
			}
			if _, ok, err := ReadMigrationCursor(db, pbt); ok || err == nil {
				t.Fatal("truncated cursor read as virgin: the seeding fallbacks would run")
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
	WriteMigrationCursor(db, true, MigrationCursor{Number: 5, Hash: common.Hash{0x01}, Root: common.Hash{0x02}})
	WriteMigrationCursor(db, false, MigrationCursor{Number: 6, Hash: common.Hash{0x03}, Root: common.Hash{0x04}})
	WritePBTMigrationDone(db)
	WriteShadowStateRoot(db, common.Hash{0x01}, 5, common.Hash{0x02})

	if err := WipeMigrationState(db); err != nil {
		t.Fatal(err)
	}
	_, pbtOK, pbtErr := ReadMigrationCursor(db, true)
	_, mptOK, mptErr := ReadMigrationCursor(db, false)
	if pbtOK || mptOK || pbtErr != nil || mptErr != nil || ReadPBTMigrationDone(db) {
		t.Fatal("migration keys survived the wipe")
	}
	if _, ok := ReadShadowStateRoot(db, common.Hash{0x01}, 5); ok {
		t.Fatal("shadow-root record survived the wipe")
	}
}
