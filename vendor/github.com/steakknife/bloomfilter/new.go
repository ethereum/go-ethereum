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
	"crypto/rand"
	"encoding/binary"
	"log"
)

const (
	// MMin is the minimum Bloom filter bits count
	MMin = 2
	// KMin is the minimum number of keys
	KMin = 1
	// Uint64Bytes is the number of bytes in type uint64
	Uint64Bytes = 8
)

// New Filter with CSPRNG keys
//
// m is the size of the Bloom filter, in bits, >= 2
//
// k is the number of random keys, >= 1
func New(m, k uint64) (*Filter, error) {
	return NewWithKeys(m, newRandKeys(k))
}

func newRandKeys(k uint64) []uint64 {
	keys := make([]uint64, k)
	err := binary.Read(rand.Reader, binary.LittleEndian, keys)
	if err != nil {
		log.Panicf(
			"Cannot read %d bytes from CSRPNG crypto/rand.Read (err=%v)",
			Uint64Bytes, err,
		)
	}
	return keys
}

// NewCompatible Filter compatible with f
func (f *Filter) NewCompatible() (*Filter, error) {
	return NewWithKeys(f.m, f.keys)
}

// NewOptimal Bloom filter with random CSPRNG keys
func NewOptimal(maxN uint64, p float64) (*Filter, error) {
	m := OptimalM(maxN, p)
	k := OptimalK(m, maxN)
	debug("New optimal bloom filter ::"+
		" requested max elements (n):%d,"+
		" probability of collision (p):%1.10f "+
		"-> recommends -> bits (m): %d (%f GiB), "+
		"number of keys (k): %d",
		maxN, p, m, float64(m)/(gigabitsPerGiB), k)
	return New(m, k)
}

// UniqueKeys is true if all keys are unique
func UniqueKeys(keys []uint64) bool {
	for j := 0; j < len(keys)-1; j++ {
		elem := keys[j]
		for i := 1; i < j; i++ {
			if keys[i] == elem {
				return false
			}
		}
	}
	return true
}

// NewWithKeys creates a new Filter from user-supplied origKeys
func NewWithKeys(m uint64, origKeys []uint64) (f *Filter, err error) {
	bits, err := newBits(m)
	if err != nil {
		return nil, err
	}
	keys, err := newKeysCopy(origKeys)
	if err != nil {
		return nil, err
	}
	return &Filter{
		m:    m,
		n:    0,
		bits: bits,
		keys: keys,
	}, nil
}

func newBits(m uint64) ([]uint64, error) {
	if m < MMin {
		return nil, errM()
	}
	return make([]uint64, (m+63)/64), nil
}

func newKeysBlank(k uint64) ([]uint64, error) {
	if k < KMin {
		return nil, errK()
	}
	return make([]uint64, k), nil
}

func newKeysCopy(origKeys []uint64) (keys []uint64, err error) {
	if !UniqueKeys(origKeys) {
		return nil, errUniqueKeys()
	}
	keys, err = newKeysBlank(uint64(len(origKeys)))
	if err != nil {
		return keys, err
	}
	copy(keys, origKeys)
	return keys, err
}

func newWithKeysAndBits(m uint64, keys []uint64, bits []uint64, n uint64) (
	f *Filter, err error,
) {
	f, err = NewWithKeys(m, keys)
	if err != nil {
		return nil, err
	}
	copy(f.bits, bits)
	f.n = n
	return f, nil
}
