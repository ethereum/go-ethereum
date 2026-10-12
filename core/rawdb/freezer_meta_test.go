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

	meta, err := newMetadata(f, false)
	if err != nil {
		t.Fatalf("Failed to new metadata %v", err)
	}
	meta.setVirtualTail(100, false)

	meta, err = newMetadata(f, false)
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
	meta, err := newMetadata(f, false)
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

	meta, err = newMetadata(f, false)
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
	_, err = newMetadata(f, false)
	if err == nil {
		t.Fatal("Unexpected success")
	}
}

// TestMetadataFixedSize checks that the metadata file keeps its size when its
// encoding grows, so that a crash can't keep a longer encoding at the old
// length. A file an earlier version wrote, shorter, gets that size at its first
// rewrite.
func TestMetadataFixedSize(t *testing.T) {
	for _, tt := range []struct {
		name string
		blob []byte // the file before the first write
	}{
		{"new", nil},
		// [2, 0, 61452]
		{"v2 written by an earlier version", []byte{0xc5, 0x02, 0x80, 0x82, 0xf0, 0x0c}},
		// [1, 0]
		{"v1", []byte{0xc2, 0x01, 0x80}},
		// [2, 0, 6] written by an earlier version over a padded [2, 0, 73740]
		{"padded, rewritten by an earlier version", append([]byte{0xc3, 0x02, 0x80, 0x06, 0x01, 0x20, 0x0c}, make([]byte, 25)...)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "*")
			if err != nil {
				t.Fatalf("Failed to create file %v", err)
			}
			defer f.Close()
			if _, err := f.Write(tt.blob); err != nil {
				t.Fatal(err)
			}
			meta, err := newMetadata(f, false)
			if err != nil {
				t.Fatalf("Failed to new metadata %v", err)
			}
			for _, v := range []uint64{0x10, 0x1000, 0x100000, 0x10000000, 0x1000000000} {
				if err := meta.setVirtualTail(v, false); err != nil {
					t.Fatal(err)
				}
				if err := meta.setFlushOffset(int64(v)*2, false); err != nil {
					t.Fatal(err)
				}
				stat, err := f.Stat()
				if err != nil {
					t.Fatal(err)
				}
				if stat.Size() != freezerMetaSize {
					t.Fatalf("Metadata file size %d, want %d", stat.Size(), freezerMetaSize)
				}
				reloaded, err := newMetadata(f, false)
				if err != nil {
					t.Fatalf("Failed to reload metadata %v", err)
				}
				if reloaded.version != freezerTableV2 || reloaded.virtualTail != v || reloaded.flushOffset != int64(v)*2 {
					t.Fatalf("Reloaded version %d, tail %d, offset %d; want 2, %d, %d", reloaded.version, reloaded.virtualTail, reloaded.flushOffset, v, v*2)
				}
			}
		})
	}
}

// TestMetadataPadding checks that metadata decodes with whatever follows its
// encoding in the file: the padding, part of it (an extension that a crash cut
// short), or the end of a longer encoding from before.
func TestMetadataPadding(t *testing.T) {
	for _, tt := range []struct {
		name    string
		enc     []byte
		version uint16
		tail    uint64
		offset  int64
	}{
		{"v1", []byte{0xc2, 0x01, 0x80}, freezerTableV1, 0, 0},                       // [1, 0]
		{"v2", []byte{0xc5, 0x02, 0x80, 0x82, 0xf0, 0x0c}, freezerTableV2, 0, 61452}, // [2, 0, 61452]
	} {
		for _, suffix := range []struct {
			name  string
			bytes []byte
		}{
			{"nothing", nil},
			{"some zeros", make([]byte, 3)},
			{"padding", make([]byte, freezerMetaSize-len(tt.enc))},
			{"old bytes", []byte{0x01, 0x20, 0x0c}},
		} {
			t.Run(tt.name+", "+suffix.name, func(t *testing.T) {
				f, err := os.CreateTemp(t.TempDir(), "*")
				if err != nil {
					t.Fatalf("Failed to create file %v", err)
				}
				defer f.Close()
				if _, err := f.Write(append(tt.enc, suffix.bytes...)); err != nil {
					t.Fatal(err)
				}
				m, err := newMetadata(f, false)
				if err != nil {
					t.Fatalf("Failed to read metadata %v", err)
				}
				if m.version != tt.version || m.virtualTail != tt.tail || m.flushOffset != tt.offset {
					t.Fatalf("Read version %d, tail %d, offset %d; want %d, %d, %d", m.version, m.virtualTail, m.flushOffset, tt.version, tt.tail, tt.offset)
				}
			})
		}
	}
}
