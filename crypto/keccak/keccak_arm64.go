//go:build arm64 && !purego

package keccak

import (
	"runtime"

	"golang.org/x/crypto/sha3"
	"golang.org/x/sys/cpu"
)

// Apple Silicon always has Armv8.2-A SHA3 extensions (VEOR3, VRAX1, VXAR, VBCAX).
// On other ARM64 platforms, detect at runtime via CPU feature flags.
// When SHA3 is unavailable, falls back to x/crypto/sha3.
var useSHA3 = runtime.GOOS == "darwin" || runtime.GOOS == "ios" || cpu.ARM64.HasSHA3

//go:noescape
func keccakF1600(a *[200]byte)

//go:noescape
func xorAndPermute(state *[200]byte, buf *byte)

// Sum256 computes the Keccak-256 hash of data. Zero heap allocations when SHA3 is available.
func Sum256(data []byte) [32]byte {
	if !useSHA3 {
		return sum256XCrypto(data)
	}
	return sum256Sponge(data)
}

func sum256XCrypto(data []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// Hasher is a streaming Keccak-256 hasher.
// Uses NEON SHA3 assembly when available, x/crypto/sha3 otherwise.
type Hasher struct {
	sponge
	xc KeccakState // x/crypto fallback
}

// Reset resets the hasher to its initial state.
func (h *Hasher) Reset() {
	if useSHA3 {
		h.sponge.Reset()
	} else {
		if h.xc == nil {
			h.xc = sha3.NewLegacyKeccak256().(KeccakState)
		} else {
			h.xc.Reset()
		}
	}
}

// Write absorbs data into the hasher.
// Panics if called after Read.
func (h *Hasher) Write(p []byte) (int, error) {
	if !useSHA3 {
		if h.xc == nil {
			h.xc = sha3.NewLegacyKeccak256().(KeccakState)
		}
		return h.xc.Write(p)
	}
	return h.sponge.Write(p)
}

// Sum256 finalizes and returns the 32-byte Keccak-256 digest.
// Does not modify the hasher state.
func (h *Hasher) Sum256() [32]byte {
	if !useSHA3 {
		if h.xc == nil {
			return Sum256(nil)
		}
		var out [32]byte
		h.xc.Sum(out[:0])
		return out
	}
	return h.sponge.Sum256()
}

// Sum appends the current Keccak-256 digest to b and returns the resulting slice.
// Does not modify the hasher state.
func (h *Hasher) Sum(b []byte) []byte {
	if !useSHA3 {
		if h.xc == nil {
			d := Sum256(nil)
			return append(b, d[:]...)
		}
		return h.xc.Sum(b)
	}
	return h.sponge.Sum(b)
}

// Read squeezes an arbitrary number of bytes from the sponge.
// On the first call, it pads and permutes, transitioning from absorbing to squeezing.
// Subsequent calls to Write will panic. It never returns an error.
func (h *Hasher) Read(out []byte) (int, error) {
	if !useSHA3 {
		if h.xc == nil {
			h.xc = sha3.NewLegacyKeccak256().(KeccakState)
		}
		return h.xc.Read(out)
	}
	return h.sponge.Read(out)
}