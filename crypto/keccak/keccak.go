// Package keccak provides Keccak-256 hashing with platform-specific acceleration.
package keccak

import "hash"

// KeccakState wraps the keccak hasher. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

const rate = 136 // sponge rate for Keccak-256: (1600 - 2*256) / 8

var _ KeccakState = (*Hasher)(nil)

func NewLegacyKeccak256() *Hasher {
	return &Hasher{}
}
