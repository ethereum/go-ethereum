// Copyright 2014 Will Fitzgerald. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file tests bit sets

package bitset

import (
	"math/rand"
	"testing"
)

func BenchmarkSet(b *testing.B) {
	b.StopTimer()
	r := rand.New(rand.NewSource(0))
	sz := 100000
	s := New(uint(sz))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Set(uint(r.Int31n(int32(sz))))
	}
}

func BenchmarkGetTest(b *testing.B) {
	b.StopTimer()
	r := rand.New(rand.NewSource(0))
	sz := 100000
	s := New(uint(sz))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Test(uint(r.Int31n(int32(sz))))
	}
}

func BenchmarkSetExpand(b *testing.B) {
	b.StopTimer()
	sz := uint(100000)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var s BitSet
		s.Set(sz)
	}
}

// go test -bench=Count
func BenchmarkCount(b *testing.B) {
	b.StopTimer()
	s := New(100000)
	for i := 0; i < 100000; i += 100 {
		s.Set(uint(i))
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Count()
	}
}

// go test -bench=Iterate
func BenchmarkIterate(b *testing.B) {
	b.StopTimer()
	s := New(10000)
	for i := 0; i < 10000; i += 3 {
		s.Set(uint(i))
	}
	b.StartTimer()
	for j := 0; j < b.N; j++ {
		c := uint(0)
		for i, e := s.NextSet(0); e; i, e = s.NextSet(i + 1) {
			c++
		}
	}
}

// go test -bench=SparseIterate
func BenchmarkSparseIterate(b *testing.B) {
	b.StopTimer()
	s := New(100000)
	for i := 0; i < 100000; i += 30 {
		s.Set(uint(i))
	}
	b.StartTimer()
	for j := 0; j < b.N; j++ {
		c := uint(0)
		for i, e := s.NextSet(0); e; i, e = s.NextSet(i + 1) {
			c++
		}
	}
}

// go test -bench=LemireCreate
// see http://lemire.me/blog/2016/09/22/swift-versus-java-the-bitset-performance-test/
func BenchmarkLemireCreate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bitmap := New(0) // we force dynamic memory allocation
		for v := uint(0); v <= 100000000; v += 100 {
			bitmap.Set(v)
		}
	}
}

// go test -bench=LemireCount
// see http://lemire.me/blog/2016/09/22/swift-versus-java-the-bitset-performance-test/
func BenchmarkLemireCount(b *testing.B) {
	bitmap := New(100000000)
	for v := uint(0); v <= 100000000; v += 100 {
		bitmap.Set(v)
	}
	b.ResetTimer()
	sum := uint(0)
	for i := 0; i < b.N; i++ {
		sum += bitmap.Count()
	}
	if sum == 0 { // added just to fool ineffassign
		return
	}
}

// go test -bench=LemireIterate
// see http://lemire.me/blog/2016/09/22/swift-versus-java-the-bitset-performance-test/
func BenchmarkLemireIterate(b *testing.B) {
	bitmap := New(100000000)
	for v := uint(0); v <= 100000000; v += 100 {
		bitmap.Set(v)
	}
	b.ResetTimer()
	sum := uint(0)
	for i := 0; i < b.N; i++ {
		for i, e := bitmap.NextSet(0); e; i, e = bitmap.NextSet(i + 1) {
			sum++
		}
	}
	if sum == 0 { // added just to fool ineffassign
		return
	}
}
