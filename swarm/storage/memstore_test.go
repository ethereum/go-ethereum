// Copyright 2016 The go-ethereum Authors
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

package storage

import (
	"testing"
)

func testMemStore(l int64, branches int64, t *testing.T) {
	m := NewMemStore(nil, defaultCacheCapacity)
	testStore(m, l, branches, t)
}

func TestMemStore128_10000(t *testing.T) {
	testMemStore(10000, 128, t)
}

func TestMemStore128_1000(t *testing.T) {
	testMemStore(1000, 128, t)
}

func TestMemStore128_100(t *testing.T) {
	testMemStore(100, 128, t)
}

func TestMemStore2_100(t *testing.T) {
	testMemStore(100, 2, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := NewMemStore(nil, defaultCacheCapacity)
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}
