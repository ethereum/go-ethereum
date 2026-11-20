// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.7 && amd64 && !gccgo && !appengine
// +build go1.7,amd64,!gccgo,!appengine

package blake2b

import "golang.org/x/sys/cpu"

func init() {
	useAVX2 = cpu.X86.HasAVX2
	useAVX = cpu.X86.HasAVX
	useSSE4 = cpu.X86.HasSSE41
}

//go:noescape
func fAVX2(h *[8]uint64, m *[16]uint64, c0, c1 uint64, flag uint64, rounds uint64)

//go:noescape
func fAVX(h *[8]uint64, m *[16]uint64, c0, c1 uint64, flag uint64, rounds uint64)

//go:noescape
func fSSE4(h *[8]uint64, m *[16]uint64, c0, c1 uint64, flag uint64, rounds uint64)

func f(h *[8]uint64, m *[16]uint64, c0, c1 uint64, flag uint64, rounds uint64) {
	switch {
	case useAVX2:
		fAVX2(h, m, c0, c1, flag, rounds)
	case useAVX:
		fAVX(h, m, c0, c1, flag, rounds)
	case useSSE4:
		fSSE4(h, m, c0, c1, flag, rounds)
	default:
		fGeneric(h, m, c0, c1, flag, rounds)
	}
}

//go:noescape
func fAVX2Rounds(v *[16]uint64, m *[16]uint64, rounds uint64)

// fRounds must not be inlined: its prologue carries the stack-growth check that
// is the only preemption point of the chunk loop in fLong. Inlined, the loop
// would call NOSPLIT assembly directly and become unpreemptible again.
//
//go:noinline
func fRounds(v *[16]uint64, m *[16]uint64, rounds uint64) {
	fAVX2Rounds(v, m, rounds)
}

func fLong(h *[8]uint64, m *[16]uint64, c0, c1 uint64, flag uint64, rounds uint64) {
	if !useAVX2 {
		fGeneric(h, m, c0, c1, flag, rounds)
		return
	}
	var v [16]uint64
	copy(v[:8], h[:])
	copy(v[8:], iv[:])
	v[12] ^= c0
	v[13] ^= c1
	v[14] ^= flag
	for rounds > 0 {
		n := min(rounds, maxAsmRounds)
		fRounds(&v, m, n)
		rounds -= n
	}
	for i := range h {
		h[i] ^= v[i] ^ v[i+8]
	}
}
