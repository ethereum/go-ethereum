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

// Package ethash implements the ethash proof-of-work consensus engine.
package ethash

import (
	"encoding/binary"
	"math/bits"

	"golang.org/x/crypto/sha3"
)

const (
	progpowCacheBytes   = 16 * 1024             // Total size 16*1024 bytes
	progpowCacheWords   = progpowCacheBytes / 4 // Total size 16*1024 bytes
	progpowLanes        = 16                    // The number of parallel lanes that coordinate to calculate a single hash instance.
	progpowRegs         = 32                    // The register file usage size
	progpowDagLoads     = 4                     // Number of uint32 loads from the DAG per lane
	progpowCntCache     = 11
	progpowCntMath      = 18
	progpowPeriodLength = 10           // Blocks per progpow epoch (N)
	progpowCntDag       = loopAccesses // Number of DAG accesses, same as ethash (64)
	progpowMixBytes     = 2 * mixBytes
)

func progpowLight(size uint64, cache []uint32, hash []byte, nonce uint64,
	blockNumber uint64, cDag []uint32) ([]byte, []byte) {
	keccak512 := makeHasher(sha3.NewLegacyKeccak512())
	lookup := func(index uint32) []byte {
		return generateDatasetItem(cache, index/16, keccak512)
	}
	return progpow(hash, nonce, size, blockNumber, cDag, lookup)
}

func progpowFull(dataset []uint32, hash []byte, nonce uint64, blockNumber uint64) ([]byte, []byte) {
	lookup := func(index uint32) []byte {
		mix := make([]byte, hashBytes)
		for i := uint32(0); i < hashWords; i++ {
			binary.LittleEndian.PutUint32(mix[i*4:], dataset[index+i])
		}
		return mix
	}
	cDag := make([]uint32, progpowCacheBytes/4)
	for i := uint32(0); i < progpowCacheBytes/4; i += 2 {
		cDag[i+0] = dataset[i+0]
		cDag[i+1] = dataset[i+1]
	}
	return progpow(hash, nonce, uint64(len(dataset))*4, blockNumber, cDag, lookup)
}

func rotl32(x uint32, n uint32) uint32 {
	return ((x) << (n % 32)) | ((x) >> (32 - (n % 32)))
}

func rotr32(x uint32, n uint32) uint32 {
	return ((x) >> (n % 32)) | ((x) << (32 - (n % 32)))
}

func lower32(in uint64) uint32 {
	return uint32(in)
}

func higher32(in uint64) uint32 {
	return uint32(in >> 32)
}

var keccakfRNDC = [24]uint32{
	0x00000001, 0x00008082, 0x0000808a, 0x80008000, 0x0000808b, 0x80000001,
	0x80008081, 0x00008009, 0x0000008a, 0x00000088, 0x80008009, 0x8000000a,
	0x8000808b, 0x0000008b, 0x00008089, 0x00008003, 0x00008002, 0x00000080,
	0x0000800a, 0x8000000a, 0x80008081, 0x00008080, 0x80000001, 0x80008008}

func keccakF800Round(st *[25]uint32, r int) {
	var keccakfROTC = [24]uint32{1, 3, 6, 10, 15, 21, 28, 36, 45, 55, 2,
		14, 27, 41, 56, 8, 25, 43, 62, 18, 39, 61,
		20, 44}
	var keccakfPILN = [24]uint32{10, 7, 11, 17, 18, 3, 5, 16, 8, 21, 24,
		4, 15, 23, 19, 13, 12, 2, 20, 14, 22, 9,
		6, 1}
	bc := make([]uint32, 5)
	// Theta
	for i := 0; i < 5; i++ {
		bc[i] = st[i] ^ st[i+5] ^ st[i+10] ^ st[i+15] ^ st[i+20]
	}

	for i := 0; i < 5; i++ {
		t := bc[(i+4)%5] ^ rotl32(bc[(i+1)%5], 1)
		for j := 0; j < 25; j += 5 {
			st[j+i] ^= t
		}
	}

	// Rho Pi
	t := st[1]
	for i, j := range keccakfPILN {
		bc[0] = st[j]
		st[j] = rotl32(t, keccakfROTC[i])
		t = bc[0]
	}

	//  Chi
	for j := 0; j < 25; j += 5 {
		bc[0] = st[j+0]
		bc[1] = st[j+1]
		bc[2] = st[j+2]
		bc[3] = st[j+3]
		bc[4] = st[j+4]
		st[j+0] ^= ^bc[1] & bc[2]
		st[j+1] ^= ^bc[2] & bc[3]
		st[j+2] ^= ^bc[3] & bc[4]
		st[j+3] ^= ^bc[4] & bc[0]
		st[j+4] ^= ^bc[0] & bc[1]
	}

	//  Iota
	st[0] ^= keccakfRNDC[r]
	//return st
}

func keccakF800Short(headerHash []byte, nonce uint64, result []uint32) uint64 {
	var st [25]uint32

	for i := 0; i < 8; i++ {
		st[i] = (uint32(headerHash[4*i])) +
			(uint32(headerHash[4*i+1]) << 8) +
			(uint32(headerHash[4*i+2]) << 16) +
			(uint32(headerHash[4*i+3]) << 24)
	}

	st[8] = lower32(nonce)
	st[9] = higher32(nonce)
	for i := 0; i < 8; i++ {
		st[10+i] = result[i]
	}

	for r := 0; r < 21; r++ {
		keccakF800Round(&st, r)
	}
	keccakF800Round(&st, 21)
	ret := make([]byte, 8)
	binary.BigEndian.PutUint32(ret[4:], st[0])
	binary.BigEndian.PutUint32(ret, st[1])
	return binary.LittleEndian.Uint64(ret)
}

func keccakF800Long(headerHash []byte, nonce uint64, result []uint32) []byte {
	var st [25]uint32

	for i := 0; i < 8; i++ {
		st[i] = (uint32(headerHash[4*i])) +
			(uint32(headerHash[4*i+1]) << 8) +
			(uint32(headerHash[4*i+2]) << 16) +
			(uint32(headerHash[4*i+3]) << 24)
	}

	st[8] = lower32(nonce)
	st[9] = higher32(nonce)
	for i := 0; i < 8; i++ {
		st[10+i] = result[i]
	}

	for r := 0; r <= 21; r++ {
		keccakF800Round(&st, r)
	}
	ret := make([]byte, 32)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint32(ret[i*4:], st[i])
	}
	return ret
}

func fnv1a(h *uint32, d uint32) uint32 {
	*h = (*h ^ d) * uint32(0x1000193)
	return *h
}

type kiss99State struct {
	z     uint32
	w     uint32
	jsr   uint32
	jcong uint32
}

func kiss99(st *kiss99State) uint32 {
	var MWC uint32
	st.z = 36969*(st.z&65535) + (st.z >> 16)
	st.w = 18000*(st.w&65535) + (st.w >> 16)
	MWC = ((st.z << 16) + st.w)
	st.jsr ^= (st.jsr << 17)
	st.jsr ^= (st.jsr >> 13)
	st.jsr ^= (st.jsr << 5)
	st.jcong = 69069*st.jcong + 1234567
	return ((MWC ^ st.jcong) + st.jsr)
}

func fillMix(seed uint64, laneId uint32) [progpowRegs]uint32 {
	var st kiss99State
	var mix [progpowRegs]uint32

	fnvHash := uint32(0x811c9dc5)

	st.z = fnv1a(&fnvHash, lower32(seed))
	st.w = fnv1a(&fnvHash, higher32(seed))
	st.jsr = fnv1a(&fnvHash, laneId)
	st.jcong = fnv1a(&fnvHash, laneId)

	for i := 0; i < progpowRegs; i++ {
		mix[i] = kiss99(&st)
	}
	return mix
}

// Merge new data from b into the value in a
// Assuming A has high entropy only do ops that retain entropy
// even if B is low entropy
// (IE don't do A&B)
func merge(a *uint32, b uint32, r uint32) {
	switch r % 4 {
	case 0:
		*a = (*a * 33) + b
	case 1:
		*a = (*a ^ b) * 33
	case 2:
		*a = rotl32(*a, ((r>>16)%31)+1) ^ b
	default:
		*a = rotr32(*a, ((r>>16)%31)+1) ^ b
	}
}

func progpowInit(seed uint64) (kiss99State, [progpowRegs]uint32, [progpowRegs]uint32) {
	var randState kiss99State

	fnvHash := uint32(0x811c9dc5)

	randState.z = fnv1a(&fnvHash, lower32(seed))
	randState.w = fnv1a(&fnvHash, higher32(seed))
	randState.jsr = fnv1a(&fnvHash, lower32(seed))
	randState.jcong = fnv1a(&fnvHash, higher32(seed))

	// Create a random sequence of mix destinations for merge()
	// and mix sources for cache reads
	// guarantees every destination merged once
	// guarantees no duplicate cache reads, which could be optimized away
	// Uses Fisher-Yates shuffle
	var dstSeq = [32]uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	var srcSeq = [32]uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}

	for i := uint32(progpowRegs - 1); i > 0; i-- {
		j := kiss99(&randState) % (i + 1)
		dstSeq[i], dstSeq[j] = dstSeq[j], dstSeq[i]
		j = kiss99(&randState) % (i + 1)
		srcSeq[i], srcSeq[j] = srcSeq[j], srcSeq[i]
	}
	return randState, dstSeq, srcSeq
}

// Random math between two input values
func progpowMath(a uint32, b uint32, r uint32) uint32 {
	switch r % 11 {
	case 0:
		return a + b
	case 1:
		return a * b
	case 2:
		return higher32(uint64(a) * uint64(b))
	case 3:
		if a < b {
			return a
		}
		return b
	case 4:
		return rotl32(a, b)
	case 5:
		return rotr32(a, b)
	case 6:
		return a & b
	case 7:
		return a | b
	case 8:
		return a ^ b
	case 9:
		return uint32(bits.LeadingZeros32(a) + bits.LeadingZeros32(b))
	case 10:
		return uint32(bits.OnesCount32(a) + bits.OnesCount32(b))

	default:
		return 0
	}
}

func progpowLoop(seed uint64, loop uint32, mix *[progpowLanes][progpowRegs]uint32,
	lookup func(index uint32) []byte,
	cDag []uint32, datasetSize uint32) {
	// All lanes share a base address for the global load
	// Global offset uses mix[0] to guarantee it depends on the load result
	gOffset := mix[loop%progpowLanes][0] % (64 * datasetSize / (progpowLanes * progpowDagLoads))

	var (
		srcCounter uint32
		dstCounter uint32
		randState  kiss99State
		srcSeq     [32]uint32
		dstSeq     [32]uint32
		rnd        = kiss99
		index      uint32
		data_g     []uint32 = make([]uint32, progpowDagLoads)
	)
	// 256 bytes of dag data
	dag_item := make([]byte, 256)
	// The lookup returns 64, so we'll fetch four items
	copy(dag_item, lookup((gOffset*progpowLanes)*progpowDagLoads))
	copy(dag_item[64:], lookup((gOffset*progpowLanes)*progpowDagLoads+16))
	copy(dag_item[128:], lookup((gOffset*progpowLanes)*progpowDagLoads+32))
	copy(dag_item[192:], lookup((gOffset*progpowLanes)*progpowDagLoads+48))

	// Lanes can execute in parallel and will be convergent
	for l := uint32(0); l < progpowLanes; l++ {

		// initialize the seed and mix destination sequence
		randState, dstSeq, srcSeq = progpowInit(seed)
		srcCounter = uint32(0)
		dstCounter = uint32(0)

		//if progpowCntCache > progpowCntMath {
		//	iMax = progpowCntCache
		//} else {
		//	iMax = progpowCntMath
		//}

		for i := uint32(0); i < progpowCntMath; i++ {
			if i < progpowCntCache {
				// Cached memory access
				// lanes access random location

				src := srcSeq[(srcCounter)%progpowRegs]
				srcCounter++

				offset := mix[l][src] % progpowCacheWords
				data32 := cDag[offset]

				dst := dstSeq[(dstCounter)%progpowRegs]
				dstCounter++

				r := kiss99(&randState)
				merge(&mix[l][dst], data32, r)
			}

			//if i < progpowCntMath
			{
				// Random Math
				srcRnd := rnd(&randState) % (progpowRegs * (progpowRegs - 1))
				src1 := srcRnd % progpowRegs
				src2 := srcRnd / progpowRegs
				if src2 >= src1 {
					src2++
				}
				data32 := progpowMath(mix[l][src1], mix[l][src2], rnd(&randState))

				dst := dstSeq[(dstCounter)%progpowRegs]
				dstCounter++

				merge(&mix[l][dst], data32, rnd(&randState))
			}
		}
		index = ((l ^ loop) % progpowLanes) * progpowDagLoads

		data_g[0] = binary.LittleEndian.Uint32(dag_item[4*index:])
		data_g[1] = binary.LittleEndian.Uint32(dag_item[4*(index+1):])
		data_g[2] = binary.LittleEndian.Uint32(dag_item[4*(index+2):])
		data_g[3] = binary.LittleEndian.Uint32(dag_item[4*(index+3):])

		merge(&mix[l][0], data_g[0], rnd(&randState))

		for i := 1; i < progpowDagLoads; i++ {
			dst := dstSeq[(dstCounter)%progpowRegs]
			dstCounter++
			merge(&mix[l][dst], data_g[i], rnd(&randState))
		}
	}
}

func progpow(hash []byte, nonce uint64, size uint64, blockNumber uint64, cDag []uint32,
	lookup func(index uint32) []byte) ([]byte, []byte) {
	var (
		mix         [progpowLanes][progpowRegs]uint32
		laneResults [progpowLanes]uint32
	)
	result := make([]uint32, 8)
	seed := keccakF800Short(hash, nonce, result)
	for lane := uint32(0); lane < progpowLanes; lane++ {
		mix[lane] = fillMix(seed, lane)
	}
	period := (blockNumber / progpowPeriodLength)
	for l := uint32(0); l < progpowCntDag; l++ {
		progpowLoop(period, l, &mix, lookup, cDag, uint32(size/progpowMixBytes))
	}

	// Reduce mix data to a single per-lane result
	for lane := uint32(0); lane < progpowLanes; lane++ {
		laneResults[lane] = 0x811c9dc5
		for i := uint32(0); i < progpowRegs; i++ {
			fnv1a(&laneResults[lane], mix[lane][i])
		}
	}
	for i := uint32(0); i < 8; i++ {
		result[i] = 0x811c9dc5
	}
	for lane := uint32(0); lane < progpowLanes; lane++ {
		fnv1a(&result[lane%8], laneResults[lane])
	}
	finalHash := keccakF800Long(hash, seed, result[:])
	mixHash := make([]byte, 8*4)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint32(mixHash[i*4:], result[i])
	}
	return mixHash[:], finalHash[:]
}
