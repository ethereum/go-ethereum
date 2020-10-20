// Copyright 2019 The go-ethereum Authors
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

package merkletree

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"testing/quick"
)

type merkleTreeTest struct {
	err     error
	entries []*Entry
}

type entryRange struct {
	pos   uint64
	level uint64
}

// entryRanges implements the sort interface to allow sorting a list of entries
// range by the start point.
type entryRanges []entryRange

func (s entryRanges) Len() int { return len(s) }
func (s entryRanges) Less(i, j int) bool {
	d1, d2 := 1<<s[i].level, 1<<s[j].level
	return float64(s[i].pos)/float64(d1) < float64(s[j].pos)/float64(d2)
}
func (s entryRanges) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (t *merkleTreeTest) run() bool {
	tree, dropped := NewMerkleTree(t.entries)
	var ranges entryRanges
	for _, entry := range t.entries {
		if _, ok := dropped[string(entry.Value)]; ok {
			continue
		}
		proof, err := tree.Prove(entry)
		if err != nil {
			t.err = err
			return false
		}
		pos, err := VerifyProof(tree.Root.Hash(), proof)
		if err != nil {
			t.err = err
			return false
		}
		ranges = append(ranges, entryRange{pos, entry.level})
	}
	sort.Sort(ranges)
	position := float64(0)
	for i := 0; i < len(ranges); i++ {
		d := 1 << ranges[i].level
		if float64(ranges[i].pos)/float64(d) != position {
			t.err = errors.New("invalid probability range")
			return false
		}
		position = float64(ranges[i].pos+1) / float64(d)
	}
	return true
}

// Generate returns a new merkletree test of the given size. All randomness is
// derived from r.
func (*merkleTreeTest) Generate(r *rand.Rand, size int) reflect.Value {
	var entries []*Entry
	length := r.Intn(100) + 1
	for i := 0; i < length; i++ {
		value := make([]byte, 20)
		r.Read(value)
		entries = append(entries, &Entry{
			Value:  value,
			Weight: uint64(r.Intn(100) + 1),
		})
	}
	return reflect.ValueOf(&merkleTreeTest{entries: entries})
}

func (t *merkleTreeTest) String() string {
	var ret string
	for index, entry := range t.entries {
		ret += fmt.Sprintf("%d => (%d:%x)\n", index, entry.Weight, entry.Value)
	}
	return ret
}

func TestMerkleTree(t *testing.T) {
	config := &quick.Config{MaxCount: 10000}
	err := quick.Check((*merkleTreeTest).run, config)
	if cerr, ok := err.(*quick.CheckError); ok {
		test := cerr.In[0].(*merkleTreeTest)
		t.Errorf("%v:\n%s", test.err, test)
	} else if err != nil {
		t.Error(err)
	}
}

func TestEmptyWeightTree(t *testing.T) {
	for i := 0; i < 10; i++ {
		len := rand.Intn(100) + 100
		var entries []*Entry
		for i := 0; i < len; i++ {
			value := make([]byte, 20)
			rand.Read(value)
			entries = append(entries, &Entry{
				Value:  value,
				Weight: 0,
			})
		}
		tree, dropped := NewMerkleTree(entries)
		for _, entry := range entries {
			if _, ok := dropped[string(entry.Value)]; ok {
				continue
			}
			proof, err := tree.Prove(entry)
			if err != nil {
				t.Fatalf("Failed to generate proof")
			}
			_, err = VerifyProof(tree.Root.Hash(), proof)
			if err != nil {
				t.Fatalf("Failed to prove proof")
			}
		}
	}
}
