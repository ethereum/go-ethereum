//go:build arm64 && !purego

package keccak

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

// Apple Silicon always has Armv8.2-A SHA3 extensions (VEOR3, VRAX1, VXAR, VBCAX).
// On other ARM64 platforms, detect at runtime via CPU feature flags.
// When SHA3 is unavailable, falls back to x/crypto/sha3.
func init() {
	useASM = runtime.GOOS == "darwin" || runtime.GOOS == "ios" || cpu.ARM64.HasSHA3
}

// keccakF1600Sha3 permutes state. When buf != nil, it first XORs rate bytes
// of buf into state, saving one full memory pass.
//
//go:noescape
func keccakF1600Sha3(a *[200]byte, buf *byte)

func keccakF1600(a *[200]byte) {
	keccakF1600Sha3(a, nil)
}

func xorAndPermute(state *[200]byte, buf *byte) {
	keccakF1600Sha3(state, buf)
}
