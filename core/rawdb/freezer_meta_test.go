// Copyright 2022 The go-ethereum Authors
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
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

func TestReadWriteFreezerTableMeta(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*")
	if err != nil {
		t.Fatalf("Failed to create file %v", err)
	}
	defer f.Close()

	meta, err := newMetadata(f)
	if err != nil {
		t.Fatalf("Failed to new metadata %v", err)
	}
	meta.setVirtualTail(100, false)

	meta, err = newMetadata(f)
	if err != nil {
		t.Fatalf("Failed to reload metadata %v", err)
	}
	if meta.version != freezerTableV2 {
		t.Fatalf("Unexpected version field")
	}
	if meta.virtualTail != uint64(100) {
		t.Fatalf("Unexpected virtual tail field")
	}
}

func TestUpgradeMetadata(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*")
	if err != nil {
		t.Fatalf("Failed to create file %v", err)
	}
	defer f.Close()

	// Write legacy metadata into file
	type obj struct {
		Version uint16
		Tail    uint64
	}
	var o obj
	o.Version = freezerTableV1
	o.Tail = 100

	if err := rlp.Encode(f, &o); err != nil {
		t.Fatalf("Failed to encode %v", err)
	}

	// Reload the metadata, a silent upgrade is expected
	meta, err := newMetadata(f)
	if err != nil {
		t.Fatalf("Failed to read metadata %v", err)
	}
	if meta.version != freezerTableV1 {
		t.Fatal("Unexpected version field")
	}
	if meta.virtualTail != uint64(100) {
		t.Fatal("Unexpected virtual tail field")
	}
	if meta.flushOffset != 0 {
		t.Fatal("Unexpected flush offset field")
	}

	meta.setFlushOffset(100, true)

	meta, err = newMetadata(f)
	if err != nil {
		t.Fatalf("Failed to read metadata %v", err)
	}
	if meta.version != freezerTableV2 {
		t.Fatal("Unexpected version field")
	}
	if meta.virtualTail != uint64(100) {
		t.Fatal("Unexpected virtual tail field")
	}
	if meta.flushOffset != 100 {
		t.Fatal("Unexpected flush offset field")
	}
}

func TestInvalidMetadata(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*")
	if err != nil {
		t.Fatalf("Failed to create file %v", err)
	}
	defer f.Close()

	// Write invalid legacy metadata into file
	type obj struct {
		Version uint16
		Tail    uint64
	}
	var o obj
	o.Version = freezerTableV2 // -> invalid version tag
	o.Tail = 100

	if err := rlp.Encode(f, &o); err != nil {
		t.Fatalf("Failed to encode %v", err)
	}
	_, err = newMetadata(f)
	if err == nil {
		t.Fatal("Unexpected success")
	}
}
