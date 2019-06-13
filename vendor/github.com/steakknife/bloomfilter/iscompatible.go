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

import "unsafe"

func uint64ToBool(x uint64) bool {
	return *(*bool)(unsafe.Pointer(&x)) // #nosec
}

// returns 0 if equal, does not compare len(b0) with len(b1)
func noBranchCompareUint64s(b0, b1 []uint64) uint64 {
	r := uint64(0)
	for i, b0i := range b0 {
		r |= b0i ^ b1[i]
	}
	return r
}

// IsCompatible is true if f and f2 can be Union()ed together
func (f *Filter) IsCompatible(f2 *Filter) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()

	f.lock.RLock()
	defer f2.lock.RUnlock()

	// 0 is true, non-0 is false
	compat := f.M() ^ f2.M()
	compat |= f.K() ^ f2.K()
	compat |= noBranchCompareUint64s(f.keys, f2.keys)
	return uint64ToBool(^compat)
}
