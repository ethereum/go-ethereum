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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package rawdb

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

func TestStoreLoadFreezerTableMeta(t *testing.T) {
	var cases = []struct {
		version   uint16
		deleted   uint64
		hidden    uint64
		expectErr error
	}{
		{
			freezerVersion, 100, 200, nil,
		},
		{
			0, 100, 200, errIncompatibleVersion, // legacy version
		},
	}
	for _, c := range cases {
		f, err := ioutil.TempFile(os.TempDir(), "*")
		if err != nil {
			t.Fatalf("Failed to create file %v", err)
		}
		err = storeMetadata(f, &freezerTableMeta{
			version: c.version,
			deleted: c.deleted,
			hidden:  c.hidden,
		})
		if err != nil {
			t.Fatalf("Failed to store metadata %v", err)
		}
		meta, err := loadMetadata(f)
		if !errors.Is(err, c.expectErr) {
			t.Fatalf("Unexpected error %v", err)
		}
		if c.expectErr == nil {
			if meta.version != c.version {
				t.Fatalf("Unexpected version field")
			}
			if meta.deleted != c.deleted {
				t.Fatalf("Unexpected deleted field")
			}
			if meta.hidden != c.hidden {
				t.Fatalf("Unexpected hidden field")
			}
		}
	}
}
