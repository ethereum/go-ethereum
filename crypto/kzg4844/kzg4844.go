// Copyright 2023 The go-ethereum Authors
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

// Package kzg4844 implements the KZG crypto for EIP-4844.
package kzg4844

import (
	"embed"
	"errors"
	"hash"
	"reflect"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:embed trusted_setup.json
var content embed.FS

var (
	blobT       = reflect.TypeOf(Blob{})
	commitmentT = reflect.TypeOf(Commitment{})
	proofT      = reflect.TypeOf(Proof{})
)

// Blob represents a 4844 data blob.
type Blob [131072]byte

// UnmarshalJSON parses a blob in hex syntax.
func (b *Blob) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(blobT, input, b[:])
}

// MarshalText returns the hex representation of b.
func (b Blob) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

// Commitment is a serialized commitment to a polynomial.
type Commitment [48]byte

// UnmarshalJSON parses a commitment in hex syntax.
func (c *Commitment) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(commitmentT, input, c[:])
}

// MarshalText returns the hex representation of c.
func (c Commitment) MarshalText() ([]byte, error) {
	return hexutil.Bytes(c[:]).MarshalText()
}

// Proof is a serialized commitment to the quotient polynomial.
type Proof [48]byte

// UnmarshalJSON parses a proof in hex syntax.
func (p *Proof) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(proofT, input, p[:])
}

// MarshalText returns the hex representation of p.
func (p Proof) MarshalText() ([]byte, error) {
	return hexutil.Bytes(p[:]).MarshalText()
}

// Point is a BLS field element.
type Point [32]byte

// Claim is a claimed evaluation value in a specific point.
type Claim [32]byte

// useCKZG controls whether the cryptography should use the Go or C backend.
var useCKZG atomic.Bool

// UseCKZG can be called to switch the default Go implementation of KZG to the C
// library if fo some reason the user wishes to do so (e.g. consensus bug in one
// or the other).
func UseCKZG(use bool) error {
	if use && !ckzgAvailable {
		return errors.New("CKZG unavailable on your platform")
	}
	useCKZG.Store(use)

	// Initializing the library can take 2-4 seconds - and can potentially crash
	// on CKZG and non-ADX CPUs - so might as well do it now and don't wait until
	// a crypto operation is actually needed live.
	if use {
		ckzgIniter.Do(ckzgInit)
	} else {
		gokzgIniter.Do(gokzgInit)
	}
	return nil
}

// BlobToCommitment creates a small commitment out of a data blob.
func BlobToCommitment(blob Blob) (Commitment, error) {
	if useCKZG.Load() {
		return ckzgBlobToCommitment(blob)
	}
	return gokzgBlobToCommitment(blob)
}

// ComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func ComputeProof(blob Blob, point Point) (Proof, Claim, error) {
	if useCKZG.Load() {
		return ckzgComputeProof(blob, point)
	}
	return gokzgComputeProof(blob, point)
}

// VerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
func VerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	if useCKZG.Load() {
		return ckzgVerifyProof(commitment, point, claim, proof)
	}
	return gokzgVerifyProof(commitment, point, claim, proof)
}

// ComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func ComputeBlobProof(blob Blob, commitment Commitment) (Proof, error) {
	if useCKZG.Load() {
		return ckzgComputeBlobProof(blob, commitment)
	}
	return gokzgComputeBlobProof(blob, commitment)
}

// VerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func VerifyBlobProof(blob Blob, commitment Commitment, proof Proof) error {
	if useCKZG.Load() {
		return ckzgVerifyBlobProof(blob, commitment, proof)
	}
	return gokzgVerifyBlobProof(blob, commitment, proof)
}

// CalcBlobHashV1 calculates the 'versioned blob hash' of a commitment.
// The given hasher must be a sha256 hash instance, otherwise the result will be invalid!
func CalcBlobHashV1(hasher hash.Hash, commit *Commitment) (vh [32]byte) {
	if hasher.Size() != 32 {
		panic("wrong hash size")
	}
	hasher.Reset()
	hasher.Write(commit[:])
	hasher.Sum(vh[:0])
	vh[0] = 0x01 // version
	return vh
}

// IsValidVersionedHash checks that h is a structurally-valid versioned blob hash.
func IsValidVersionedHash(h []byte) bool {
	return len(h) == 32 && h[0] == 0x01
}
