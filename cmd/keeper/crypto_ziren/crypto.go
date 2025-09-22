// Copyright 2025 The go-ethereum Authors
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

package crypto

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	originalcrypto "github.com/ethereum/go-ethereum/crypto"
)

// Ziren zkVM system call numbers
const (
	// SYS_KECCAK_SPONGE is the system call number for keccak sponge compression in Ziren zkVM
	// This performs the keccak-f[1600] permutation on a 1600-bit (200-byte) state
	SYS_KECCAK_SPONGE = 0x010109
)

// Keccak256 constants
const (
	keccakRate     = 136 // 1088 bits = 136 bytes for keccak256
	keccakCapacity = 64  // 512 bits = 64 bytes
	keccakStateSize = 200 // 1600 bits = 200 bytes
)

// zirenKeccakSponge calls the Ziren zkVM keccak sponge compression function
// This performs the keccak-f[1600] permutation on the 200-byte state
func zirenKeccakSponge(state *[keccakStateSize]byte) error {
	_, _, errno := syscall.Syscall(
		SYS_KECCAK_SPONGE,
		uintptr(unsafe.Pointer(state)), // State pointer (input/output)
		0, 0, // Unused parameters
	)
	
	if errno != 0 {
		return errors.New("keccak sponge syscall failed")
	}
	
	return nil
}

// zirenKeccak256 implements full keccak256 using the Ziren sponge syscall
func zirenKeccak256(data []byte) []byte {
	// Initialize state to zeros
	var state [keccakStateSize]byte
	
	// Pad input according to keccak256 specification
	// Padding: append 0x01, then zero or more 0x00 bytes, then 0x80
	padded := make([]byte, len(data))
	copy(padded, data)
	padded = append(padded, 0x01) // Domain separator for keccak256
	
	// Pad to multiple of rate (136 bytes for keccak256)
	for len(padded)%keccakRate != (keccakRate - 1) {
		padded = append(padded, 0x00)
	}
	padded = append(padded, 0x80) // Final padding bit
	
	// Absorb phase: process input in chunks of rate size
	for i := 0; i < len(padded); i += keccakRate {
		// XOR current chunk with state
		for j := 0; j < keccakRate && i+j < len(padded); j++ {
			state[j] ^= padded[i+j]
		}
		
		// Apply keccak-f[1600] permutation via syscall
		if err := zirenKeccakSponge(&state); err != nil {
			// Fallback to standard implementation on error
			return originalcrypto.Keccak256(data)
		}
	}
	
	// Squeeze phase: extract 32 bytes (256 bits) for keccak256
	result := make([]byte, 32)
	copy(result, state[:32])
	
	return result
}

// Re-export everything from original crypto package except the parts we're overriding
var (
	S256                     = originalcrypto.S256
	PubkeyToAddress         = originalcrypto.PubkeyToAddress
	Ecrecover               = originalcrypto.Ecrecover
	SigToPub                = originalcrypto.SigToPub
	Sign                    = originalcrypto.Sign
	VerifySignature         = originalcrypto.VerifySignature
	DecompressPubkey        = originalcrypto.DecompressPubkey
	CompressPubkey          = originalcrypto.CompressPubkey
	HexToECDSA              = originalcrypto.HexToECDSA
	LoadECDSA               = originalcrypto.LoadECDSA
	SaveECDSA               = originalcrypto.SaveECDSA
	GenerateKey             = originalcrypto.GenerateKey
	ValidateSignatureValues = originalcrypto.ValidateSignatureValues
	Keccak512               = originalcrypto.Keccak512
)

// Re-export types
type (
	KeccakState = originalcrypto.KeccakState
)

// zirenKeccakState implements crypto.KeccakState using the Ziren sponge precompile
type zirenKeccakState struct {
	state    [keccakStateSize]byte // 200-byte keccak state
	absorbed int                   // Number of bytes absorbed into current block
	buffer   [keccakRate]byte      // Rate-sized buffer for current block
	finalized bool                 // Whether absorption is complete
}

func (k *zirenKeccakState) Reset() {
	for i := range k.state {
		k.state[i] = 0
	}
	for i := range k.buffer {
		k.buffer[i] = 0
	}
	k.absorbed = 0
	k.finalized = false
}

func (k *zirenKeccakState) Clone() KeccakState {
	clone := &zirenKeccakState{
		absorbed:  k.absorbed,
		finalized: k.finalized,
	}
	copy(clone.state[:], k.state[:])
	copy(clone.buffer[:], k.buffer[:])
	return clone
}

func (k *zirenKeccakState) Write(data []byte) (int, error) {
	if k.finalized {
		panic("write to finalized keccak state")
	}
	
	written := 0
	for len(data) > 0 {
		// Fill current block
		canWrite := keccakRate - k.absorbed
		if canWrite > len(data) {
			canWrite = len(data)
		}
		
		copy(k.buffer[k.absorbed:], data[:canWrite])
		k.absorbed += canWrite
		data = data[canWrite:]
		written += canWrite
		
		// If block is full, absorb it
		if k.absorbed == keccakRate {
			k.absorbBlock()
		}
	}
	
	return written, nil
}

// absorbBlock XORs the current buffer into state and applies the sponge permutation
func (k *zirenKeccakState) absorbBlock() {
	// XOR buffer into state
	for i := 0; i < keccakRate; i++ {
		k.state[i] ^= k.buffer[i]
	}
	
	// Apply keccak-f[1600] permutation via Ziren syscall
	if err := zirenKeccakSponge(&k.state); err != nil {
		// On error, fallback to standard Go implementation
		// This shouldn't happen in production but provides safety
		fallbackState := originalcrypto.NewKeccakState()
		fallbackState.Reset()
		fallbackState.Write(k.buffer[:k.absorbed])
		fallbackState.Read(k.state[:32])
	}
	
	// Reset buffer
	k.absorbed = 0
	for i := range k.buffer {
		k.buffer[i] = 0
	}
}

func (k *zirenKeccakState) Read(hash []byte) (int, error) {
	if len(hash) < 32 {
		return 0, errors.New("hash slice too short")
	}
	
	if !k.finalized {
		k.finalize()
	}
	
	copy(hash[:32], k.state[:32])
	return 32, nil
}

// finalize completes the absorption phase with padding
func (k *zirenKeccakState) finalize() {
	// Add keccak256 padding: 0x01, then zeros, then 0x80
	k.buffer[k.absorbed] = 0x01
	k.absorbed++
	
	// Pad with zeros until we have room for final bit
	for k.absorbed < keccakRate-1 {
		k.buffer[k.absorbed] = 0x00
		k.absorbed++
	}
	
	// Add final padding bit
	k.buffer[keccakRate-1] = 0x80
	k.absorbed = keccakRate
	
	// Absorb final block
	k.absorbBlock()
	k.finalized = true
}

func (k *zirenKeccakState) Sum(data []byte) []byte {
	hash := make([]byte, 32)
	k.Read(hash)
	return append(data, hash...)
}

func (k *zirenKeccakState) Size() int {
	return 32
}

func (k *zirenKeccakState) BlockSize() int {
	return 136 // keccak256 block size
}

// Keccak256 calculates and returns the Keccak256 hash using the ziren platform precompile.
func Keccak256(data ...[]byte) []byte {
	// For multiple data chunks, concatenate them
	if len(data) == 0 {
		return zirenKeccak256(nil)
	}
	if len(data) == 1 {
		return zirenKeccak256(data[0])
	}
	
	// Concatenate multiple data chunks
	var totalLen int
	for _, d := range data {
		totalLen += len(d)
	}
	
	combined := make([]byte, 0, totalLen)
	for _, d := range data {
		combined = append(combined, d...)
	}
	
	return zirenKeccak256(combined)
}

// Keccak256Hash calculates and returns the Keccak256 hash as a Hash using the ziren platform precompile.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	hash := Keccak256(data...)
	copy(h[:], hash)
	return h
}

// NewKeccakState returns a new keccak state hasher using the ziren platform precompile.
func NewKeccakState() KeccakState {
	return &zirenKeccakState{}
}
