//go:build (!arm64 && !amd64) || purego

package keccak

import (
	"golang.org/x/crypto/sha3"
)

// Sum256 computes the Keccak-256 hash of data.
// On non-arm64 platforms, delegates to x/crypto/sha3.NewLegacyKeccak256().
func Sum256(data []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// Hasher is a streaming Keccak-256 hasher wrapping x/crypto/sha3.
type Hasher struct {
	h KeccakState
}

func (h *Hasher) init() {
	if h.h == nil {
		h.h = sha3.NewLegacyKeccak256().(KeccakState)
	}
}

// Reset resets the hasher to its initial state.
func (h *Hasher) Reset() {
	h.init()
	h.h.Reset()
}

// Write absorbs data into the hasher.
// Panics if called after Read.
func (h *Hasher) Write(p []byte) (int, error) {
	h.init()
	return h.h.Write(p)
}

// Sum256 finalizes and returns the 32-byte Keccak-256 digest.
// Does not modify the hasher state.
func (h *Hasher) Sum256() [32]byte {
	h.init()
	var out [32]byte
	h.h.Sum(out[:0])
	return out
}

// Sum appends the current Keccak-256 digest to b and returns the resulting slice.
// Does not modify the hasher state.
func (h *Hasher) Sum(b []byte) []byte {
	h.init()
	return h.h.Sum(b)
}

// Size returns the number of bytes Sum will produce (32).
func (h *Hasher) Size() int { return 32 }

// BlockSize returns the sponge rate in bytes (136).
func (h *Hasher) BlockSize() int { return rate }

// Read squeezes an arbitrary number of bytes from the sponge.
// On the first call, it pads and permutes, transitioning from absorbing to squeezing.
// Subsequent calls to Write will panic. It never returns an error.
func (h *Hasher) Read(out []byte) (int, error) {
	h.init()
	return h.h.Read(out)
}