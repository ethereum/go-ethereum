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

package kzg4844

import (
	"math/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
)

func randFieldElement() [32]byte {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic("failed to get random field element")
	}
	var r fr.Element
	r.SetBytes(bytes)

	return gokzg4844.SerializeScalar(r)
}

func randBlob() Blob {
	var blob Blob
	for i := 0; i < len(blob); i += gokzg4844.SerializedScalarSize {
		fieldElementBytes := randFieldElement()
		copy(blob[i:i+gokzg4844.SerializedScalarSize], fieldElementBytes[:])
	}
	return blob
}

func TestKZGWithPoint(t *testing.T) {
	blob := randBlob()

	commitment, err := BlobToCommitment(blob)
	if err != nil {
		t.Fatalf("failed to create KZG commitment from blob: %v", err)
	}
	point := randFieldElement()
	proof, claim, err := ComputeProof(blob, point)
	if err != nil {
		t.Fatalf("failed to create KZG proof at point: %v", err)
	}
	if err := VerifyProof(commitment, point, claim, proof); err != nil {
		t.Fatalf("failed to verify KZG proof at point: %v", err)
	}
}

func TestKZGWithBlob(t *testing.T) {
	blob := randBlob()

	commitment, err := BlobToCommitment(blob)
	if err != nil {
		t.Fatalf("failed to create KZG commitment from blob: %v", err)
	}
	proof, err := ComputeBlobProof(blob, commitment)
	if err != nil {
		t.Fatalf("failed to create KZG proof for blob: %v", err)
	}
	if err := VerifyBlobProof(blob, commitment, proof); err != nil {
		t.Fatalf("failed to verify KZG proof for blob: %v", err)
	}
}
