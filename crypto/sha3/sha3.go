// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sha3 implements the SHA3 hash algorithm (formerly called Keccak) chosen by NIST in 2012.
// This file provides a SHA3 implementation which implements the standard hash.Hash interface.
// Writing input data, including padding, and reading output data are computed in this file.
// Note that the current implementation can compute the hash of an integral number of bytes only.
// This is a consequence of the hash interface in which a buffer of bytes is passed in.
// The internals of the Keccak-f function are computed in keccakf.go.
// For the detailed specification, refer to the Keccak web site (http://keccak.noekeon.org/).
package sha3

import (
	"encoding/binary"
	"hash"
)

// laneSize is the size in bytes of each "lane" of the internal state of SHA3 (5 * 5 * 8).
// Note that changing this size would requires using a type other than uint64 to store each lane.
const laneSize = 8

// sliceSize represents the dimensions of the internal state, a square matrix of
// sliceSize ** 2 lanes. This is the size of both the "rows" and "columns" dimensions in the
// terminology of the SHA3 specification.
const sliceSize = 5

// numLanes represents the total number of lanes in the state.
const numLanes = sliceSize * sliceSize

// stateSize is the size in bytes of the internal state of SHA3 (5 * 5 * WSize).
const stateSize = laneSize * numLanes

// digest represents the partial evaluation of a checksum.
// Note that capacity, and not outputSize, is the critical security parameter, as SHA3 can output
// an arbitrary number of bytes for any given capacity. The Keccak proposal recommends that
// capacity = 2*outputSize to ensure that finding a collision of size outputSize requires
// O(2^{outputSize/2}) computations (the birthday lower bound). Future standards may modify the
// capacity/outputSize ratio to allow for more output with lower cryptographic security.
type digest struct {
	a          [numLanes]uint64 // main state of the hash
	outputSize int              // desired output size in bytes
	capacity   int              // number of bytes to leave untouched during squeeze/absorb
	absorbed   int              // number of bytes absorbed thus far
}

// minInt returns the lesser of two integer arguments, to simplify the absorption routine.
func minInt(v1, v2 int) int {
	if v1 <= v2 {
		return v1
	}
	return v2
}

// rate returns the number of bytes of the internal state which can be absorbed or squeezed
// in between calls to the permutation function.
func (d *digest) rate() int {
	return stateSize - d.capacity
}

// Reset clears the internal state by zeroing bytes in the state buffer.
// This can be skipped for a newly-created hash state; the default zero-allocated state is correct.
func (d *digest) Reset() {
	d.absorbed = 0
	for i := range d.a {
		d.a[i] = 0
	}
}

// BlockSize, required by the hash.Hash interface, does not have a standard intepretation
// for a sponge-based construction like SHA3. We return the data rate: the number of bytes which
// can be absorbed per invocation of the permutation function. For Merkle-Damgård based hashes
// (ie SHA1, SHA2, MD5) the output size of the internal compression function is returned.
// We consider this to be roughly equivalent because it represents the number of bytes of output
// produced per cryptographic operation.
func (d *digest) BlockSize() int { return d.rate() }

// Size returns the output size of the hash function in bytes.
func (d *digest) Size() int {
	return d.outputSize
}

// unalignedAbsorb is a helper function for Write, which absorbs data that isn't aligned with an
// 8-byte lane. This requires shifting the individual bytes into position in a uint64.
func (d *digest) unalignedAbsorb(p []byte) {
	var t uint64
	for i := len(p) - 1; i >= 0; i-- {
		t <<= 8
		t |= uint64(p[i])
	}
	offset := (d.absorbed) % d.rate()
	t <<= 8 * uint(offset%laneSize)
	d.a[offset/laneSize] ^= t
	d.absorbed += len(p)
}

// Write "absorbs" bytes into the state of the SHA3 hash, updating as needed when the sponge
// "fills up" with rate() bytes. Since lanes are stored internally as type uint64, this requires
// converting the incoming bytes into uint64s using a little endian interpretation. This
// implementation is optimized for large, aligned writes of multiples of 8 bytes (laneSize).
// Non-aligned or uneven numbers of bytes require shifting and are slower.
func (d *digest) Write(p []byte) (int, error) {
	// An initial offset is needed if the we aren't absorbing to the first lane initially.
	offset := d.absorbed % d.rate()
	toWrite := len(p)

	// The first lane may need to absorb unaligned and/or incomplete data.
	if (offset%laneSize != 0 || len(p) < 8) && len(p) > 0 {
		toAbsorb := minInt(laneSize-(offset%laneSize), len(p))
		d.unalignedAbsorb(p[:toAbsorb])
		p = p[toAbsorb:]
		offset = (d.absorbed) % d.rate()

		// For every rate() bytes absorbed, the state must be permuted via the F Function.
		if (d.absorbed)%d.rate() == 0 {
			keccakF1600(&d.a)
		}
	}

	// This loop should absorb the bulk of the data into full, aligned lanes.
	// It will call the update function as necessary.
	for len(p) > 7 {
		firstLane := offset / laneSize
		lastLane := minInt(d.rate()/laneSize, firstLane+len(p)/laneSize)

		// This inner loop absorbs input bytes into the state in groups of 8, converted to uint64s.
		for lane := firstLane; lane < lastLane; lane++ {
			d.a[lane] ^= binary.LittleEndian.Uint64(p[:laneSize])
			p = p[laneSize:]
		}
		d.absorbed += (lastLane - firstLane) * laneSize
		// For every rate() bytes absorbed, the state must be permuted via the F Function.
		if (d.absorbed)%d.rate() == 0 {
			keccakF1600(&d.a)
		}

		offset = 0
	}

	// If there are insufficient bytes to fill the final lane, an unaligned absorption.
	// This should always start at a correct lane boundary though, or else it would be caught
	// by the uneven opening lane case above.
	if len(p) > 0 {
		d.unalignedAbsorb(p)
	}

	return toWrite, nil
}

// pad computes the SHA3 padding scheme based on the number of bytes absorbed.
// The padding is a 1 bit, followed by an arbitrary number of 0s and then a final 1 bit, such that
// the input bits plus padding bits are a multiple of rate(). Adding the padding simply requires
// xoring an opening and closing bit into the appropriate lanes.
func (d *digest) pad() {
	offset := d.absorbed % d.rate()
	// The opening pad bit must be shifted into position based on the number of bytes absorbed
	padOpenLane := offset / laneSize
	d.a[padOpenLane] ^= 0x0000000000000001 << uint(8*(offset%laneSize))
	// The closing padding bit is always in the last position
	padCloseLane := (d.rate() / laneSize) - 1
	d.a[padCloseLane] ^= 0x8000000000000000
}

// finalize prepares the hash to output data by padding and one final permutation of the state.
func (d *digest) finalize() {
	d.pad()
	keccakF1600(&d.a)
}

// squeeze outputs an arbitrary number of bytes from the hash state.
// Squeezing can require multiple calls to the F function (one per rate() bytes squeezed),
// although this is not the case for standard SHA3 parameters. This implementation only supports
// squeezing a single time, subsequent squeezes may lose alignment. Future implementations
// may wish to support multiple squeeze calls, for example to support use as a PRNG.
func (d *digest) squeeze(in []byte, toSqueeze int) []byte {
	// Because we read in blocks of laneSize, we need enough room to read
	// an integral number of lanes
	needed := toSqueeze + (laneSize-toSqueeze%laneSize)%laneSize
	if cap(in)-len(in) < needed {
		newIn := make([]byte, len(in), len(in)+needed)
		copy(newIn, in)
		in = newIn
	}
	out := in[len(in) : len(in)+needed]

	for len(out) > 0 {
		for i := 0; i < d.rate() && len(out) > 0; i += laneSize {
			binary.LittleEndian.PutUint64(out[:], d.a[i/laneSize])
			out = out[laneSize:]
		}
		if len(out) > 0 {
			keccakF1600(&d.a)
		}
	}
	return in[:len(in)+toSqueeze] // Re-slice in case we wrote extra data.
}

// Sum applies padding to the hash state and then squeezes out the desired nubmer of output bytes.
func (d *digest) Sum(in []byte) []byte {
	// Make a copy of the original hash so that caller can keep writing and summing.
	dup := *d
	dup.finalize()
	return dup.squeeze(in, dup.outputSize)
}

// The NewKeccakX constructors enable initializing a hash in any of the four recommend sizes
// from the Keccak specification, all of which set capacity=2*outputSize. Note that the final
// NIST standard for SHA3 may specify different input/output lengths.
// The output size is indicated in bits but converted into bytes internally.
func NewKeccak224() hash.Hash { return &digest{outputSize: 224 / 8, capacity: 2 * 224 / 8} }
func NewKeccak256() hash.Hash { return &digest{outputSize: 256 / 8, capacity: 2 * 256 / 8} }
func NewKeccak384() hash.Hash { return &digest{outputSize: 384 / 8, capacity: 2 * 384 / 8} }
func NewKeccak512() hash.Hash { return &digest{outputSize: 512 / 8, capacity: 2 * 512 / 8} }
