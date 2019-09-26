// Package bloomfilter is face-meltingly fast, thread-safe,
// marshalable, unionable, probability- and
// optimal-size-calculating Bloom filter in go
//
// https://github.com/steakknife/bloomfilter
//
// Copyright © 2014, 2015, 2018 Barry Allard
//
// MIT license
//
package bloomfilter

import (
	"hash"
	"sync"
)

// Filter is an opaque Bloom filter type
type Filter struct {
	lock sync.RWMutex
	bits []uint64
	keys []uint64
	m    uint64 // number of bits the "bits" field should recognize
	n    uint64 // number of inserted elements
}

// Hashable -> hashes
func (f *Filter) hash(v hash.Hash64) []uint64 {
	rawHash := v.Sum64()
	n := len(f.keys)
	hashes := make([]uint64, n)
	for i := 0; i < n; i++ {
		hashes[i] = rawHash ^ f.keys[i]
	}
	return hashes
}

// M is the size of Bloom filter, in bits
func (f *Filter) M() uint64 {
	return f.m
}

// K is the count of keys
func (f *Filter) K() uint64 {
	return uint64(len(f.keys))
}

// Add a hashable item, v, to the filter
func (f *Filter) Add(v hash.Hash64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for _, i := range f.hash(v) {
		// f.setBit(i)
		i %= f.m
		f.bits[i>>6] |= 1 << uint(i&0x3f)
	}
	f.n++
}

// Contains tests if f contains v
// false: f definitely does not contain value v
// true:  f maybe contains value v
func (f *Filter) Contains(v hash.Hash64) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()

	r := uint64(1)
	for _, i := range f.hash(v) {
		// r |= f.getBit(k)
		i %= f.m
		r &= (f.bits[i>>6] >> uint(i&0x3f)) & 1
	}
	return uint64ToBool(r)
}

// Copy f to a new Bloom filter
func (f *Filter) Copy() (*Filter, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	out, err := f.NewCompatible()
	if err != nil {
		return nil, err
	}
	copy(out.bits, f.bits)
	out.n = f.n
	return out, nil
}

// UnionInPlace merges Bloom filter f2 into f
func (f *Filter) UnionInPlace(f2 *Filter) error {
	if !f.IsCompatible(f2) {
		return errIncompatibleBloomFilters()
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	for i, bitword := range f2.bits {
		f.bits[i] |= bitword
	}
	return nil
}

// Union merges f2 and f2 into a new Filter out
func (f *Filter) Union(f2 *Filter) (out *Filter, err error) {
	if !f.IsCompatible(f2) {
		return nil, errIncompatibleBloomFilters()
	}

	f.lock.RLock()
	defer f.lock.RUnlock()

	out, err = f.NewCompatible()
	if err != nil {
		return nil, err
	}
	for i, bitword := range f2.bits {
		out.bits[i] = f.bits[i] | bitword
	}
	return out, nil
}
