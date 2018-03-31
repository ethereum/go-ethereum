// Package keccak implements the Keccak (SHA-3) hash algorithm.
// http://keccak.noekeon.org.
package keccakpg

import (
	_ "fmt"
	"hash"
)

const stdRounds = 24

var roundConstants = []uint64{
	0x0000000000000001, 0x0000000000008082,
	0x800000000000808A, 0x8000000080008000,
	0x000000000000808B, 0x0000000080000001,
	0x8000000080008081, 0x8000000000008009,
	0x000000000000008A, 0x0000000000000088,
	0x0000000080008009, 0x000000008000000A,
	0x000000008000808B, 0x800000000000008B,
	0x8000000000008089, 0x8000000000008003,
	0x8000000000008002, 0x8000000000000080,
	0x000000000000800A, 0x800000008000000A,
	0x8000000080008081, 0x8000000000008080,
	0x0000000080000001, 0x8000000080008008,
}

var rotationConstants = [24]uint{
	1, 3, 6, 10, 15, 21, 28, 36,
	45, 55, 2, 14, 27, 41, 56, 8,
	25, 43, 62, 18, 39, 61, 20, 44,
}

var piLane = [24]uint{
	10, 7, 11, 17, 18, 3, 5, 16,
	8, 21, 24, 4, 15, 23, 19, 13,
	12, 2, 20, 14, 22, 9, 6, 1,
}

type keccak struct {
	S         [25]uint64
	size      int
	blockSize int
	rounds    int
	buf       []byte
}

func newKeccak(bitlen, rounds int) hash.Hash {
	var h keccak
	h.size = bitlen / 8
	h.blockSize = (200 - 2*h.size)
	h.rounds = rounds
	if rounds != stdRounds {
		//fmt.Printf("keccak: warning non standard number of rounds %d vs %d\n", rounds, stdRounds)
	}
	return &h
}

func NewCustom(bits, rounds int) hash.Hash {
	return newKeccak(bits, rounds)
}

func New160() hash.Hash {
	return newKeccak(160, stdRounds)
}

func New224() hash.Hash {
	return newKeccak(224, stdRounds)
}

func New256() hash.Hash {
	return newKeccak(256, stdRounds)
}

func New384() hash.Hash {
	return newKeccak(384, stdRounds)
}

func New512() hash.Hash {
	return newKeccak(512, stdRounds)
}

func (k *keccak) Write(b []byte) (int, error) {
	n := len(b)

	if len(k.buf) > 0 {
		x := k.blockSize - len(k.buf)
		if x > len(b) {
			x = len(b)
		}
		k.buf = append(k.buf, b[:x]...)
		b = b[x:]

		if len(k.buf) < k.blockSize {
			return n, nil
		}

		k.f(k.buf)
		k.buf = nil
	}

	for len(b) >= k.blockSize {
		k.f(b[:k.blockSize])
		b = b[k.blockSize:]
	}

	k.buf = b

	return n, nil
}

func (k0 *keccak) Sum(b []byte) []byte {

	k := *k0

	last := k.pad(k.buf)
	k.f(last)

	buf := make([]byte, len(k.S)*8)
	for i := range k.S {
		putUint64le(buf[i*8:], k.S[i])
	}
	return append(b, buf[:k.size]...)
}

func (k *keccak) Reset() {
	for i := range k.S {
		k.S[i] = 0
	}
	k.buf = nil
}

func (k *keccak) Size() int {
	return k.size
}

func (k *keccak) BlockSize() int {
	return k.blockSize
}

func rotl64(x uint64, n uint) uint64 {
	return (x << n) | (x >> (64 - n))
}

func (k *keccak) f(block []byte) {

	if len(block) != k.blockSize {
		panic("f() called with invalid block size")
	}

	for i := 0; i < k.blockSize/8; i++ {
		k.S[i] ^= uint64le(block[i*8:])
	}

	for r := 0; r < k.rounds; r++ {
		var bc [5]uint64

		// theta
		for i := range bc {
			bc[i] = k.S[i] ^ k.S[5+i] ^ k.S[10+i] ^ k.S[15+i] ^ k.S[20+i]
		}
		for i := range bc {
			t := bc[(i+4)%5] ^ rotl64(bc[(i+1)%5], 1)
			for j := 0; j < len(k.S); j += 5 {
				k.S[i+j] ^= t
			}
		}

		// rho phi
		temp := k.S[1]
		for i := range piLane {
			j := piLane[i]
			temp2 := k.S[j]
			k.S[j] = rotl64(temp, rotationConstants[i])
			temp = temp2
		}

		// chi
		for j := 0; j < len(k.S); j += 5 {
			for i := range bc {
				bc[i] = k.S[j+i]
			}
			for i := range bc {
				k.S[j+i] ^= (^bc[(i+1)%5]) & bc[(i+2)%5]
			}
		}

		// iota
		k.S[0] ^= roundConstants[r]
	}
}

func (k *keccak) pad(block []byte) []byte {

	padded := make([]byte, k.blockSize)

	copy(padded, k.buf)
	padded[len(k.buf)] = 0x01
	padded[len(padded)-1] |= 0x80

	return padded
}

func uint64le(v []byte) uint64 {
	return uint64(v[0]) |
		uint64(v[1])<<8 |
		uint64(v[2])<<16 |
		uint64(v[3])<<24 |
		uint64(v[4])<<32 |
		uint64(v[5])<<40 |
		uint64(v[6])<<48 |
		uint64(v[7])<<56

}

func putUint64le(v []byte, x uint64) {
	v[0] = byte(x)
	v[1] = byte(x >> 8)
	v[2] = byte(x >> 16)
	v[3] = byte(x >> 24)
	v[4] = byte(x >> 32)
	v[5] = byte(x >> 40)
	v[6] = byte(x >> 48)
	v[7] = byte(x >> 56)
}
