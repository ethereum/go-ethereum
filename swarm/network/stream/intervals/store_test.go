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

package intervals

import "testing"

// TestMemStore tests basic functionality of MemStore.
func TestMemStore(t *testing.T) {
	testStore(t, NewMemStore())
}

// testStore is a helper function to test various Store implementations.
func testStore(t *testing.T, s Store) {
	key1 := "key1"
	i1 := NewIntervals(0)
	i1.Add(10, 20)
	if err := s.Put(key1, i1); err != nil {
		t.Fatal(err)
	}
	g, err := s.Get(key1)
	if err != nil {
		t.Fatal(err)
	}
	if g.String() != i1.String() {
		t.Errorf("expected interval %s, got %s", i1, g)
	}

	key2 := "key2"
	i2 := NewIntervals(0)
	i2.Add(10, 20)
	if err := s.Put(key2, i2); err != nil {
		t.Fatal(err)
	}
	g, err = s.Get(key2)
	if err != nil {
		t.Fatal(err)
	}
	if g.String() != i2.String() {
		t.Errorf("expected interval %s, got %s", i2, g)
	}

	if err := s.Delete(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(key1); err != ErrNotFound {
		t.Errorf("expected error %v, got %s", ErrNotFound, err)
	}
	if _, err := s.Get(key2); err != nil {
		t.Errorf("expected error %v, got %s", nil, err)
	}

	if err := s.Delete(key2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(key2); err != ErrNotFound {
		t.Errorf("expected error %v, got %s", ErrNotFound, err)
	}
}
