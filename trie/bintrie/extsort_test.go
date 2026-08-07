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

package bintrie

import (
	"bytes"
	"io"
	"math/rand"
	"path/filepath"
	"testing"
)

// drainSorted adds every record, sorts, and returns the drained stream.
func drainSorted(t *testing.T, budget int, recs []leafRecord) []leafRecord {
	t.Helper()
	s := NewLeafSorter(t.TempDir(), budget)
	defer s.Close()
	for _, r := range recs {
		if err := s.Add(r.key, r.value); err != nil {
			t.Fatalf("Add(%x): %v", r.key, err)
		}
	}
	stream, err := s.Sort()
	if err != nil {
		t.Fatal(err)
	}
	var out []leafRecord
	for {
		k, v, err := stream.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, leafRecord{key: k, value: v})
	}
}

// randomLeafRecords builds n records with distinct conformant keys.
func randomLeafRecords(t *testing.T, rng *rand.Rand, n int) []leafRecord {
	t.Helper()
	seen := make(map[string]struct{}, n)
	recs := make([]leafRecord, 0, n)
	for len(recs) < n {
		key := randomConformantKey(rng)
		if _, dup := seen[string(key)]; dup {
			continue
		}
		seen[string(key)] = struct{}{}
		var value [32]byte
		rng.Read(value[:])
		if isZeroValue(value[:]) {
			value[0] = 1
		}
		recs = append(recs, leafRecord{key: key, value: value[:]})
	}
	return recs
}

// TestLeafSorterOrders pins that the output is ascending and complete, both
// on the in-memory fast path and across spilled runs. A tiny budget forces a
// spill every few records, so the merge is exercised with many runs rather
// than a token two.
func TestLeafSorterOrders(t *testing.T) {
	rng := rand.New(rand.NewSource(8347))
	recs := randomLeafRecords(t, rng, 500)

	for _, tc := range []struct {
		name   string
		budget int
	}{
		{"in-memory", 0},
		{"many runs", 512},
		{"few runs", 16 * 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := drainSorted(t, tc.budget, recs)
			if len(out) != len(recs) {
				t.Fatalf("drained %d records, put in %d", len(out), len(recs))
			}
			want := make(map[string][]byte, len(recs))
			for _, r := range recs {
				want[string(r.key)] = r.value
			}
			for i, r := range out {
				if i > 0 && bytes.Compare(out[i-1].key, r.key) >= 0 {
					t.Fatalf("record %d out of order: %x after %x", i, r.key, out[i-1].key)
				}
				if v, ok := want[string(r.key)]; !ok || !bytes.Equal(v, r.value) {
					t.Fatalf("record %x came out with the wrong value", r.key)
				}
			}
		})
	}
}

// TestLeafSorterRejects covers the input validation: malformed keys, wrong
// value sizes, zero values, and duplicates - both inside one run and across
// runs, which fail at different stages.
func TestLeafSorterRejects(t *testing.T) {
	key := HeaderKey(commonAddress(1), 0)
	value := bytes.Repeat([]byte{1}, 32)

	t.Run("malformed key", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 0)
		defer s.Close()
		if err := s.Add([]byte{0x02, 0xff}, value); err == nil {
			t.Fatal("a malformed key was accepted")
		}
	})
	t.Run("wrong value size", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 0)
		defer s.Close()
		if err := s.Add(key, []byte{1, 2, 3}); err == nil {
			t.Fatal("a short value was accepted")
		}
	})
	t.Run("zero value", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 0)
		defer s.Close()
		if err := s.Add(key, make([]byte, 32)); err == nil {
			t.Fatal("a zero value was accepted; the state layer resolves those to absence")
		}
	})
	t.Run("duplicate in one run", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 0)
		defer s.Close()
		if err := s.Add(key, value); err != nil {
			t.Fatal(err)
		}
		if err := s.Add(key, value); err != nil {
			t.Fatal(err) // buffered; the sort surfaces it
		}
		if _, err := s.Sort(); err == nil {
			t.Fatal("a duplicate key within one run survived the sort")
		}
	})
	t.Run("duplicate across runs", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 1) // spill after every record
		defer s.Close()
		if err := s.Add(key, value); err != nil {
			t.Fatal(err)
		}
		if err := s.Add(key, value); err != nil {
			t.Fatal(err)
		}
		stream, err := s.Sort()
		if err != nil {
			t.Fatal(err)
		}
		var streamErr error
		for {
			_, _, err := stream.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				streamErr = err
				break
			}
		}
		if streamErr == nil {
			t.Fatal("a duplicate key across runs survived the merge")
		}
	})
}

// TestLeafSorterEdges covers empty input, a single record, and temp-file
// cleanup on Close.
func TestLeafSorterEdges(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 0)
		defer s.Close()
		stream, err := s.Sort()
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := stream.Next(); err != io.EOF {
			t.Fatalf("empty sort yielded %v, want EOF", err)
		}
	})
	t.Run("single", func(t *testing.T) {
		out := drainSorted(t, 1, []leafRecord{{key: HeaderKey(commonAddress(7), 1), value: bytes.Repeat([]byte{9}, 32)}})
		if len(out) != 1 {
			t.Fatalf("drained %d records, want 1", len(out))
		}
	})
	t.Run("close removes spills", func(t *testing.T) {
		dir := t.TempDir()
		s := NewLeafSorter(dir, 1)
		rng := rand.New(rand.NewSource(1))
		for _, r := range randomLeafRecords(t, rng, 20) {
			if err := s.Add(r.key, r.value); err != nil {
				t.Fatal(err)
			}
		}
		stream, err := s.Sort()
		if err != nil {
			t.Fatal(err)
		}
		for {
			if _, _, err := stream.Next(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatal(err)
			}
		}
		s.Close()
		leftovers, err := filepath.Glob(filepath.Join(dir, "bintrie-sort-*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(leftovers) != 0 {
			t.Fatalf("%d spill files survived Close", len(leftovers))
		}
	})
	t.Run("add after sort refused", func(t *testing.T) {
		s := NewLeafSorter(t.TempDir(), 0)
		defer s.Close()
		if _, err := s.Sort(); err != nil {
			t.Fatal(err)
		}
		if err := s.Add(HeaderKey(commonAddress(1), 0), bytes.Repeat([]byte{1}, 32)); err == nil {
			t.Fatal("Add after Sort was accepted")
		}
	})
}

// commonAddress builds a deterministic address for tests.
func commonAddress(i byte) (a [20]byte) {
	a[0] = i
	return a
}

// TestLeafSorterMatchesStackBuilder ties the two halves of the pipeline: a
// shuffled record set, externally sorted with a spill-heavy budget and fed to
// the stack builder, must reproduce the incremental engine's root.
func TestLeafSorterMatchesStackBuilder(t *testing.T) {
	rng := rand.New(rand.NewSource(20260807))
	recs := randomLeafRecords(t, rng, 300)

	inc := newTestTrie()
	for _, r := range recs {
		setKey(t, inc, r.key, r.value)
	}
	want := inc.Hash()

	s := NewLeafSorter(t.TempDir(), 256)
	defer s.Close()
	for _, r := range recs {
		if err := s.Add(r.key, r.value); err != nil {
			t.Fatal(err)
		}
	}
	stream, err := s.Sort()
	if err != nil {
		t.Fatal(err)
	}
	b := NewStackBuilder(nil)
	for {
		k, v, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Add(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if got := b.Finish(); got != want {
		t.Fatalf("sorted-and-built root %x, incremental %x", got, want)
	}
}
