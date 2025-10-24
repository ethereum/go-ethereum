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
	"crypto/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-eth-kzg"
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

func randBlob() *Blob {
	var blob Blob
	for i := 0; i < len(blob); i += gokzg4844.SerializedScalarSize {
		fieldElementBytes := randFieldElement()
		copy(blob[i:i+gokzg4844.SerializedScalarSize], fieldElementBytes[:])
	}
	return &blob
}

func TestCKZGWithPoint(t *testing.T)  { testKZGWithPoint(t, true) }
func TestGoKZGWithPoint(t *testing.T) { testKZGWithPoint(t, false) }
func testKZGWithPoint(t *testing.T, ckzg bool) {
	if ckzg && !ckzgAvailable {
		t.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

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

func TestCKZGWithBlob(t *testing.T)  { testKZGWithBlob(t, true) }
func TestGoKZGWithBlob(t *testing.T) { testKZGWithBlob(t, false) }
func testKZGWithBlob(t *testing.T, ckzg bool) {
	if ckzg && !ckzgAvailable {
		t.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

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

func BenchmarkCKZGBlobToCommitment(b *testing.B)  { benchmarkBlobToCommitment(b, true) }
func BenchmarkGoKZGBlobToCommitment(b *testing.B) { benchmarkBlobToCommitment(b, false) }
func benchmarkBlobToCommitment(b *testing.B, ckzg bool) {
	if ckzg && !ckzgAvailable {
		b.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	blob := randBlob()

	for b.Loop() {
		BlobToCommitment(blob)
	}
}

func BenchmarkCKZGComputeProof(b *testing.B)  { benchmarkComputeProof(b, true) }
func BenchmarkGoKZGComputeProof(b *testing.B) { benchmarkComputeProof(b, false) }
func benchmarkComputeProof(b *testing.B, ckzg bool) {
	if ckzg && !ckzgAvailable {
		b.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	var (
		blob  = randBlob()
		point = randFieldElement()
	)

	for b.Loop() {
		ComputeProof(blob, point)
	}
}

func BenchmarkCKZGVerifyProof(b *testing.B)  { benchmarkVerifyProof(b, true) }
func BenchmarkGoKZGVerifyProof(b *testing.B) { benchmarkVerifyProof(b, false) }
func benchmarkVerifyProof(b *testing.B, ckzg bool) {
	if ckzg && !ckzgAvailable {
		b.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	var (
		blob            = randBlob()
		point           = randFieldElement()
		commitment, _   = BlobToCommitment(blob)
		proof, claim, _ = ComputeProof(blob, point)
	)

	for b.Loop() {
		VerifyProof(commitment, point, claim, proof)
	}
}

func BenchmarkCKZGComputeBlobProof(b *testing.B)  { benchmarkComputeBlobProof(b, true) }
func BenchmarkGoKZGComputeBlobProof(b *testing.B) { benchmarkComputeBlobProof(b, false) }
func benchmarkComputeBlobProof(b *testing.B, ckzg bool) {
	if ckzg && !ckzgAvailable {
		b.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	var (
		blob          = randBlob()
		commitment, _ = BlobToCommitment(blob)
	)

	for b.Loop() {
		ComputeBlobProof(blob, commitment)
	}
}

func BenchmarkCKZGVerifyBlobProof(b *testing.B)  { benchmarkVerifyBlobProof(b, true) }
func BenchmarkGoKZGVerifyBlobProof(b *testing.B) { benchmarkVerifyBlobProof(b, false) }
func benchmarkVerifyBlobProof(b *testing.B, ckzg bool) {
	if ckzg && !ckzgAvailable {
		b.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	var (
		blob          = randBlob()
		commitment, _ = BlobToCommitment(blob)
		proof, _      = ComputeBlobProof(blob, commitment)
	)

	for b.Loop() {
		VerifyBlobProof(blob, commitment, proof)
	}
}

func TestCKZGCells(t *testing.T)  { testKZGCells(t, true) }
func TestGoKZGCells(t *testing.T) { testKZGCells(t, false) }
func testKZGCells(t *testing.T, ckzg bool) {
	if ckzg && !ckzgAvailable {
		t.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	blob1 := randBlob()
	blob2 := randBlob()

	commitment1, err := BlobToCommitment(blob1)
	if err != nil {
		t.Fatalf("failed to create KZG commitment from blob: %v", err)
	}
	commitment2, err := BlobToCommitment(blob2)
	if err != nil {
		t.Fatalf("failed to create KZG commitment from blob: %v", err)
	}

	proofs1, err := ComputeCellProofs(blob1)
	if err != nil {
		t.Fatalf("failed to create KZG proof at point: %v", err)
	}

	proofs2, err := ComputeCellProofs(blob2)
	if err != nil {
		t.Fatalf("failed to create KZG proof at point: %v", err)
	}
	proofs := append(proofs1, proofs2...)
	blobs := []Blob{*blob1, *blob2}
	if err := VerifyCellProofs(blobs, []Commitment{commitment1, commitment2}, proofs); err != nil {
		t.Fatalf("failed to verify KZG proof at point: %v", err)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/crypto/kzg4844
// cpu: Apple M1 Pro
// BenchmarkGOKZGComputeCellProofs
// BenchmarkGOKZGComputeCellProofs-8   	       8	 139012286 ns/op
func BenchmarkGOKZGComputeCellProofs(b *testing.B) { benchmarkComputeCellProofs(b, false) }
func BenchmarkCKZGComputeCellProofs(b *testing.B)  { benchmarkComputeCellProofs(b, true) }

func benchmarkComputeCellProofs(b *testing.B, ckzg bool) {
	if ckzg && !ckzgAvailable {
		b.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	blob := randBlob()
	_, _ = ComputeCellProofs(blob) // for kzg initialization
	b.ResetTimer()

	for b.Loop() {
		_, err := ComputeCellProofs(blob)
		if err != nil {
			b.Fatalf("failed to create KZG proof at point: %v", err)
		}
	}
}

func TestCKZGVerifyPartialCells(t *testing.T)  { testVerifyPartialCells(t, true) }
func TestGoKZGVerifyPartialCells(t *testing.T) { testVerifyPartialCells(t, false) }

func testVerifyPartialCells(t *testing.T, ckzg bool) {
	if ckzg && !ckzgAvailable {
		t.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	const blobCount = 3
	var blobs []*Blob
	var commitments []Commitment
	for range blobCount {
		blob := randBlob()
		commitment, err := BlobToCommitment(blob)
		if err != nil {
			t.Fatalf("failed to commit blob: %v", err)
		}
		blobs = append(blobs, blob)
		commitments = append(commitments, commitment)
	}

	var (
		partialCells  []Cell
		partialProofs []Proof
		commits       []Commitment
		indices       []uint64
	)

	for bi, blob := range blobs {
		proofs, err := ComputeCellProofs(blob)
		if err != nil {
			t.Fatalf("failed to compute cell proofs: %v", err)
		}
		cells, err := ComputeCells([]Blob{*blob})
		if err != nil {
			t.Fatalf("failed to compute cells: %v", err)
		}
		commits = append(commits, commitments[bi])

		// sample 0, 31, 63, 95 cells
		step := len(cells) / 4

		indices = []uint64{0, uint64(step - 1), uint64(2*step - 1), uint64(3*step - 1)}
		for _, idx := range indices {
			partialCells = append(partialCells, cells[idx])
			partialProofs = append(partialProofs, proofs[idx])
		}
	}
	// t.Fatalf("length: %d %d %d %d", len(partialCells), len(commits), len(partialProofs), len(indices))

	if err := VerifyCells(partialCells, commits, partialProofs, indices); err != nil {
		t.Fatalf("failed to verify partial cell proofs: %v", err)
	}
}

func TestCKZGRecoverBlob(t *testing.T)  { testRecoverBlob(t, true) }
func TestGoKZGRecoverBlob(t *testing.T) { testRecoverBlob(t, false) }

func testRecoverBlob(t *testing.T, ckzg bool) {
	if ckzg && !ckzgAvailable {
		t.Skip("CKZG unavailable in this test build")
	}
	defer func(old bool) { useCKZG.Store(old) }(useCKZG.Load())
	useCKZG.Store(ckzg)

	blobs := []Blob{}
	blobs = append(blobs, *randBlob())
	blobs = append(blobs, *randBlob())
	blobs = append(blobs, *randBlob())

	cells, err := ComputeCells(blobs)
	if err != nil {
		t.Fatalf("failed to compute cells: %v", err)
	}
	proofs := make([]Proof, 0)
	commitments := make([]Commitment, len(blobs))
	for i, blob := range blobs {
		proof, err := ComputeCellProofs(&blob)
		if err != nil {
			t.Fatalf("failed to compute proof: %v", err)
		}
		proofs = append(proofs, proof...)

		commitment, err := BlobToCommitment(&blob)
		if err != nil {
			t.Fatalf("failed to compute commitment: %v", err)
		}
		commitments[i] = commitment
	}

	var (
		partialCells []Cell
		indices      []uint64
	)

	for ci := 64; ci < 128; ci++ {
		indices = append(indices, uint64(ci))
	}

	for i := 0; i < len(cells); i += 128 {
		start := i + 64
		end := i + 128
		partialCells = append(partialCells, cells[start:end]...)
	}

	recoverBlobs, err := RecoverBlobs(partialCells, indices)

	if err != nil {
		t.Fatalf("failed to recover blob: %v", err)
	}

	if err := VerifyCellProofs(recoverBlobs, commitments, proofs); err != nil {
		t.Fatalf("failed to verify recovered blob: %v", err)
	}
}
