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

package utils

import (
	"math/rand"
	"testing"
)

type testWrsItem struct {
	idx  int
	widx *int
}

func testWeight(i interface{}) uint64 {
	t := i.(*testWrsItem)
	w := *t.widx
	if w == -1 || w == t.idx {
		return uint64(t.idx + 1)
	}
	return 0
}

func TestWeightedRandomSelect(t *testing.T) {
	testFn := func(cnt int) {
		s := NewWeightedRandomSelect(testWeight)
		w := -1
		list := make([]testWrsItem, cnt)
		for i := range list {
			list[i] = testWrsItem{idx: i, widx: &w}
			s.Update(&list[i])
		}
		w = rand.Intn(cnt)
		c := s.Choose()
		if c == nil {
			t.Errorf("expected item, got nil")
		} else {
			if c.(*testWrsItem).idx != w {
				t.Errorf("expected another item")
			}
		}
		w = -2
		if s.Choose() != nil {
			t.Errorf("expected nil, got item")
		}
	}
	testFn(1)
	testFn(10)
	testFn(100)
	testFn(1000)
	testFn(10000)
	testFn(100000)
	testFn(1000000)
}

// TestOOB tests values which doesn't fit in int64
func TestOOB(t *testing.T) {
	s := NewWeightedRandomSelect(func(i interface{}) uint64 {
		// Dummy weight function to return a very large weight
		return uint64(0xffffffffffffffff)
	})
	s.Update(testWrsItem{idx: 0, widx: nil})
	// int64 conversion should make the sumweight negative
	if int64(s.root.sumWeight) >= 0 {
		t.Fatalf("test is dysfunctional, sumweight not negative: %d", int64(s.root.sumWeight))
	}
	s.Choose()
}
