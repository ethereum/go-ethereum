// Copyright 2018 The go-ethereum Authors
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

package memorydb

import (
	"bytes"
	"testing"
)

// Tests that key-value iteration on top of a memory database works.
func TestMemoryDBIterator(t *testing.T) {
	tests := []struct {
		content map[string]string
		prefix  string
		order   []string
	}{
		// Empty databases should be iterable
		{map[string]string{}, "", nil},
		{map[string]string{}, "non-existent-prefix", nil},

		// Single-item databases should be iterable
		{map[string]string{"key": "val"}, "", []string{"key"}},
		{map[string]string{"key": "val"}, "k", []string{"key"}},
		{map[string]string{"key": "val"}, "l", nil},

		// Multi-item databases should be fully iterable
		{
			map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
			"",
			[]string{"k1", "k2", "k3", "k4", "k5"},
		},
		{
			map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
			"k",
			[]string{"k1", "k2", "k3", "k4", "k5"},
		},
		{
			map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
			"l",
			nil,
		},
		// Multi-item databases should be prefix-iterable
		{
			map[string]string{
				"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
				"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
			},
			"ka",
			[]string{"ka1", "ka2", "ka3", "ka4", "ka5"},
		},
		{
			map[string]string{
				"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
				"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
			},
			"kc",
			nil,
		},
	}
	for i, tt := range tests {
		// Create the key-value data store
		db := New()
		for key, val := range tt.content {
			if err := db.Put([]byte(key), []byte(val)); err != nil {
				t.Fatalf("test %d: failed to insert item %s:%s into database: %v", i, key, val, err)
			}
		}
		// Iterate over the database with the given configs and verify the results
		it, idx := db.NewIteratorWithPrefix([]byte(tt.prefix)), 0
		for it.Next() {
			if !bytes.Equal(it.Key(), []byte(tt.order[idx])) {
				t.Errorf("test %d: item %d: key mismatch: have %s, want %s", i, idx, string(it.Key()), tt.order[idx])
			}
			if !bytes.Equal(it.Value(), []byte(tt.content[tt.order[idx]])) {
				t.Errorf("test %d: item %d: value mismatch: have %s, want %s", i, idx, string(it.Value()), tt.content[tt.order[idx]])
			}
			idx++
		}
		if err := it.Error(); err != nil {
			t.Errorf("test %d: iteration failed: %v", i, err)
		}
		if idx != len(tt.order) {
			t.Errorf("test %d: iteration terminated prematurely: have %d, want %d", i, idx, len(tt.order))
		}
	}
}
