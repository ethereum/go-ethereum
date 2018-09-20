// Written in 2012-2014 by Dmitry Chestnykh.
//
// To the extent possible under law, the author have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
// http://creativecommons.org/publicdomain/zero/1.0/

// Package siphash implements SipHash-2-4, a fast short-input PRF
// created by Jean-Philippe Aumasson and Daniel J. Bernstein.
package siphash

import "hash"

const (
	// BlockSize is the block size of hash algorithm in bytes.
	BlockSize = 8

	// Size is the size of hash output in bytes.
	Size = 8

	// Size128 is the size of 128-bit hash output in bytes.
	Size128 = 16
)

type digest struct {
	v0, v1, v2, v3 uint64  // state
	k0, k1         uint64  // two parts of key
	x              [8]byte // buffer for unprocessed bytes
	nx             int     // number of bytes in buffer x
	size           int     // output size in bytes (8 or 16)
	t              uint8   // message bytes counter (mod 256)
}

// newDigest returns a new digest with the given output size in bytes (must be 8 or 16).
func newDigest(size int, key []byte) *digest {
	if size != Size && size != Size128 {
		panic("size must be 8 or 16")
	}
	d := new(digest)
	d.k0 = uint64(key[0]) | uint64(key[1])<<8 | uint64(key[2])<<16 | uint64(key[3])<<24 |
		uint64(key[4])<<32 | uint64(key[5])<<40 | uint64(key[6])<<48 | uint64(key[7])<<56
	d.k1 = uint64(key[8]) | uint64(key[9])<<8 | uint64(key[10])<<16 | uint64(key[11])<<24 |
		uint64(key[12])<<32 | uint64(key[13])<<40 | uint64(key[14])<<48 | uint64(key[15])<<56
	d.size = size
	d.Reset()
	return d
}

// New returns a new hash.Hash64 computing SipHash-2-4 with 16-byte key and 8-byte output.
func New(key []byte) hash.Hash64 {
	return newDigest(Size, key)
}

// New128 returns a new hash.Hash computing SipHash-2-4 with 16-byte key and 16-byte output.
//
// Note that 16-byte output is considered experimental by SipHash authors at this time.
func New128(key []byte) hash.Hash {
	return newDigest(Size128, key)
}

func (d *digest) Reset() {
	d.v0 = d.k0 ^ 0x736f6d6570736575
	d.v1 = d.k1 ^ 0x646f72616e646f6d
	d.v2 = d.k0 ^ 0x6c7967656e657261
	d.v3 = d.k1 ^ 0x7465646279746573
	d.t = 0
	d.nx = 0
	if d.size == Size128 {
		d.v1 ^= 0xee
	}
}

func (d *digest) Size() int { return d.size }

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Write(p []byte) (nn int, err error) {
	nn = len(p)
	d.t += uint8(nn)
	if d.nx > 0 {
		n := len(p)
		if n > BlockSize-d.nx {
			n = BlockSize - d.nx
		}
		d.nx += copy(d.x[d.nx:], p)
		if d.nx == BlockSize {
			once(d)
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		blocks(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d *digest) Sum64() uint64 {
	for i := d.nx; i < BlockSize-1; i++ {
		d.x[i] = 0
	}
	d.x[7] = d.t
	return finalize(d)
}

func (d0 *digest) sum128() (r0, r1 uint64) {
	// Make a copy of d0 so that caller can keep writing and summing.
	d := *d0

	for i := d.nx; i < BlockSize-1; i++ {
		d.x[i] = 0
	}
	d.x[7] = d.t
	blocks(&d, d.x[:])

	v0, v1, v2, v3 := d.v0, d.v1, d.v2, d.v3
	v2 ^= 0xee

	// Round 1.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 2.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 3.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 4.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	r0 = v0 ^ v1 ^ v2 ^ v3

	v1 ^= 0xdd

	// Round 1.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 2.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 3.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 4.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	r1 = v0 ^ v1 ^ v2 ^ v3

	return r0, r1
}

func (d *digest) Sum(in []byte) []byte {
	if d.size == Size {
		r := d.Sum64()
		in = append(in,
			byte(r),
			byte(r>>8),
			byte(r>>16),
			byte(r>>24),
			byte(r>>32),
			byte(r>>40),
			byte(r>>48),
			byte(r>>56))
	} else {
		r0, r1 := d.sum128()
		in = append(in,
			byte(r0),
			byte(r0>>8),
			byte(r0>>16),
			byte(r0>>24),
			byte(r0>>32),
			byte(r0>>40),
			byte(r0>>48),
			byte(r0>>56),
			byte(r1),
			byte(r1>>8),
			byte(r1>>16),
			byte(r1>>24),
			byte(r1>>32),
			byte(r1>>40),
			byte(r1>>48),
			byte(r1>>56))
	}
	return in
}
