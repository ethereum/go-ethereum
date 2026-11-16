// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package bloomfilter stands in for github.com/holiman/bloomfilter/v2 in the
// keeper (see the replace directive in its go.mod). The real package imports
// encoding/gob for its serialisation, and gob's initialisation costs a zkVM
// guest millions of instructions; the keeper, which executes a block from a
// witness, never builds the state snapshots the filter accelerates. This
// Filter is a bloom filter that never filters: Contains always reports a
// possible member, which is a valid (if useless) answer, so the snapshot
// code stays correct should it run.
package bloomfilter

import "hash"

// Filter is a bloom filter with every bit set.
type Filter struct {
	m, k uint64
}

// New returns a filter of m bits and k hash functions.
func New(m, k uint64) (*Filter, error) {
	return &Filter{m: m, k: k}, nil
}

func (f *Filter) M() uint64                   { return f.m }
func (f *Filter) K() uint64                   { return f.k }
func (f *Filter) N() uint64                   { return 0 }
func (f *Filter) Add(v hash.Hash64)           {}
func (f *Filter) AddHash(hash uint64)         {}
func (f *Filter) Contains(v hash.Hash64) bool { return true }
func (f *Filter) ContainsHash(hash uint64) bool {
	return true
}
func (f *Filter) Copy() (*Filter, error)        { return &Filter{m: f.m, k: f.k}, nil }
func (f *Filter) UnionInPlace(f2 *Filter) error { return nil }
func (f *Filter) Union(f2 *Filter) (*Filter, error) {
	return f.Copy()
}
