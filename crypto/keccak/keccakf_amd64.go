//go:build amd64 && !purego

package keccak

import "unsafe"

//go:noescape
func keccakF1600(a *[200]byte)

// Sum256 computes the Keccak-256 hash of data. Zero heap allocations.
func Sum256(data []byte) [32]byte { return sum256Sponge(data) }

// Hasher is a streaming Keccak-256 hasher. Designed for stack allocation.
type Hasher struct{ sponge }

func xorAndPermute(state *[200]byte, buf *byte) {
	xorIn(state, unsafe.Slice(buf, rate))
	keccakF1600(state)
}