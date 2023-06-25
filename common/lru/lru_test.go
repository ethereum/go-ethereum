// Copyright 2023 The go-ethereum Authors
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

package lru

import (
	"testing"
)

func TestLRUAdd(t *testing.T) {
	cache := NewCache[int, int](128)
	for i := 0; i < 256; i++ {
		evicted := cache.Add(i, i)
		if i < 128 && evicted == true {
			t.Fatalf("%d should not be evicted", i)
		} else if i >= 128 && evicted == false {
			t.Fatalf("%d should be evicted", i)
		}
	}
}

func TestLRUContains(t *testing.T) {
	cache := NewCache[int, int](2)
	cache.Add(1, 1)
	cache.Add(2, 2)
	if !cache.Contains(1) {
		t.Errorf("1 should be in the cache")
	}
	cache.Add(3, 3)
	if cache.Contains(1) {
		t.Errorf("Contains should not have updated recency of 1")
	}
}

func TestLRUGet(t *testing.T) {
	cache := NewCache[int, int](2)
	cache.Add(1, 1)
	cache.Add(2, 2)
	if v, ok := cache.Get(1); !ok || v != 1 {
		t.Errorf("1 should be in the cache")
	}
	cache.Add(3, 3)
	if v, ok := cache.Get(1); !ok || v != 1 {
		t.Errorf("Get should have updated recency of 1")
	}
	if _, ok := cache.Get(2); ok {
		t.Errorf("2 shold be removed by recency policy")
	}
}

func TestLRULen(t *testing.T) {
	cache := NewCache[int, int](2)
	cache.Add(1, 1)
	if cache.Len() != 1 {
		t.Fatalf("bad len: %v", cache.Len())
	}
	cache.Add(2, 2)
	if cache.Len() != 2 {
		t.Fatalf("bad len: %v", cache.Len())
	}
	cache.Add(3, 3)
	if cache.Len() != 2 {
		t.Fatalf("bad len: %v", cache.Len())
	}
}

func TestLRUPeek(t *testing.T) {
	cache := NewCache[int, int](2)
	cache.Add(1, 1)
	cache.Add(2, 2)
	if v, ok := cache.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1")
	}
	cache.Add(3, 3)
	if cache.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

func TestLRUPurge(t *testing.T) {
	cache := NewCache[int, int](2)
	cache.Add(1, 1)
	cache.Add(2, 2)
	if cache.Len() != 2 {
		t.Fatalf("bad len: %v", cache.Len())
	}
	cache.Purge()
	if cache.Len() != 0 {
		t.Fatalf("bad len: %v", cache.Len())
	}
	if cache.Contains(1) {
		t.Fatalf("should not have 1")
	}
	if cache.Contains(2) {
		t.Fatalf("should not have 2")
	}
}

func TestLRURemove(t *testing.T) {
	cache := NewCache[int, int](2)
	cache.Add(1, 1)
	cache.Add(2, 2)
	if cache.Remove(3) {
		t.Fatalf("should not be able to remove 3")
	}
	if !cache.Remove(2) {
		t.Fatalf("should be able to remove 2")
	}
	if cache.Contains(2) {
		t.Fatalf("should not have 2")
	}
	if cache.Len() != 1 {
		t.Fatalf("bad len: %v", cache.Len())
	}
}

func TestLRUKeys(t *testing.T) {
	cache := NewCache[int, int](128)
	for i := 0; i < 256; i++ {
		cache.Add(i, i)
	}
	keys := cache.Keys()
	if len(keys) != 128 {
		t.Fatal("wrong Keys() length", len(keys))
	}
	for i, k := range keys {
		v, ok := cache.Peek(k)
		if !ok {
			t.Fatalf("expected key %d be present", i)
		}
		if v != k {
			t.Fatalf("expected %d == %d", k, v)
		}
		if v != i+128 {
			t.Fatalf("wrong value at key %d: %d, want %d", i, v, i+128)
		}
	}
}
