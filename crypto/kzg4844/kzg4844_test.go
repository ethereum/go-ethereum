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
	mrand "math/rand"
	"slices"
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

func switchBackend(t testing.TB, ckzg bool) (switchBack func()) {
	t.Helper()
	if ckzg && !ckzgAvailable {
		t.Skip("CKZG unavailable in this test build")
	}
	prev := useCKZG.Load()
	useCKZG.Store(ckzg)
	return func() { useCKZG.Store(prev) }
}

func TestCKZGWithPoint(t *testing.T)  { testKZGWithPoint(t, true) }
func TestGoKZGWithPoint(t *testing.T) { testKZGWithPoint(t, false) }
func testKZGWithPoint(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

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
	defer switchBackend(t, ckzg)()

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
	defer switchBackend(b, ckzg)()

	blob := randBlob()

	for b.Loop() {
		BlobToCommitment(blob)
	}
}

func BenchmarkCKZGComputeProof(b *testing.B)  { benchmarkComputeProof(b, true) }
func BenchmarkGoKZGComputeProof(b *testing.B) { benchmarkComputeProof(b, false) }
func benchmarkComputeProof(b *testing.B, ckzg bool) {
	defer switchBackend(b, ckzg)()

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
	defer switchBackend(b, ckzg)()

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
	defer switchBackend(b, ckzg)()

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
	defer switchBackend(b, ckzg)()

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
	defer switchBackend(t, ckzg)()

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
	defer switchBackend(b, ckzg)()

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

// randCellIndices picks n random unique indices from [0, CellsPerBlob) in sorted order.
func randCellIndices(rng *mrand.Rand, n int) []uint64 {
	perm := rng.Perm(CellsPerBlob)
	indices := make([]uint64, n)
	for i := 0; i < n; i++ {
		indices[i] = uint64(perm[i])
	}
	slices.Sort(indices)
	return indices
}

// randBlobAndProofs generates random blobs and precomputes their cells, proofs, and commitments.
type randBlobAndProofs struct {
	blobs       []Blob
	commitments []Commitment
	cells       []Cell // flat: blobs[i] cells at [i*CellsPerBlob : (i+1)*CellsPerBlob]
	proofs      []Proof
}

func newBlobs(t *testing.T, blobCount int) *randBlobAndProofs {
	d := &randBlobAndProofs{
		blobs:       make([]Blob, blobCount),
		commitments: make([]Commitment, blobCount),
	}
	for i := range blobCount {
		d.blobs[i] = *randBlob()
		commitment, err := BlobToCommitment(&d.blobs[i])
		if err != nil {
			t.Fatalf("failed to compute commitment: %v", err)
		}
		d.commitments[i] = commitment
		proofs, err := ComputeCellProofs(&d.blobs[i])
		if err != nil {
			t.Fatalf("failed to compute cell proofs: %v", err)
		}
		d.proofs = append(d.proofs, proofs...)
	}
	cells, err := ComputeCells(d.blobs)
	if err != nil {
		t.Fatalf("failed to compute cells: %v", err)
	}
	d.cells = cells
	return d
}

func TestCKZGVerifyPartialCells(t *testing.T)  { testVerifyPartialCells(t, true) }
func TestGoKZGVerifyPartialCells(t *testing.T) { testVerifyPartialCells(t, false) }

func testVerifyPartialCells(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const (
		iterations = 50
		blobCount  = 3
		cellsCount = 8
	)
	// Precompute blobs once, vary only cell indices per iteration
	d := newBlobs(t, blobCount)

	for iter := range iterations {
		rng := mrand.New(mrand.NewSource(int64(iter)))
		indices := randCellIndices(rng, cellsCount)

		var partialCells []Cell
		var partialProofs []Proof
		for i := range blobCount {
			for _, idx := range indices {
				partialCells = append(partialCells, d.cells[i*CellsPerBlob+int(idx)])
				partialProofs = append(partialProofs, d.proofs[i*CellProofsPerBlob+int(idx)])
			}
		}
		if err := VerifyCells(partialCells, d.commitments, partialProofs, indices); err != nil {
			t.Fatalf("iter %d: failed to verify partial cells: %v", iter, err)
		}
	}
}

func TestCKZGVerifyCellsWithCorruptedCells(t *testing.T) {
	testVerifyCellsWithCorruptedCells(t, true)
}
func TestGoKZGVerifyCellsWithCorruptedCells(t *testing.T) {
	testVerifyCellsWithCorruptedCells(t, false)
}

func testVerifyCellsWithCorruptedCells(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const blobCount = 3
	d := newBlobs(t, blobCount)
	indices := []uint64{0, 15, 63, 64, 95, 100, 120, 127}

	var partialCells []Cell
	var partialProofs []Proof
	for i := range blobCount {
		for _, idx := range indices {
			partialCells = append(partialCells, d.cells[i*CellsPerBlob+int(idx)])
			partialProofs = append(partialProofs, d.proofs[i*CellProofsPerBlob+int(idx)])
		}
	}
	// Corrupt the first cell
	corruptedCells := make([]Cell, len(partialCells))
	copy(corruptedCells, partialCells)
	corruptedCells[0][0] ^= 0xff

	if err := VerifyCells(corruptedCells, d.commitments, partialProofs, indices); err == nil {
		t.Fatal("expected verification failure with corrupted cell")
	}
}

func TestCKZGVerifyCellsWithCorruptedProofs(t *testing.T) {
	testVerifyCellsWithCorruptedProofs(t, true)
}
func TestGoKZGVerifyCellsWithCorruptedProofs(t *testing.T) {
	testVerifyCellsWithCorruptedProofs(t, false)
}

func testVerifyCellsWithCorruptedProofs(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const blobCount = 3
	d := newBlobs(t, blobCount)
	indices := []uint64{0, 15, 63, 64, 95, 100, 120, 127}

	var partialCells []Cell
	var partialProofs []Proof
	for i := range blobCount {
		for _, idx := range indices {
			partialCells = append(partialCells, d.cells[i*CellsPerBlob+int(idx)])
			partialProofs = append(partialProofs, d.proofs[i*CellProofsPerBlob+int(idx)])
		}
	}
	// Swap first and last proof
	wrongProofs := make([]Proof, len(partialProofs))
	copy(wrongProofs, partialProofs)
	wrongProofs[0], wrongProofs[len(wrongProofs)-1] = wrongProofs[len(wrongProofs)-1], wrongProofs[0]

	if err := VerifyCells(partialCells, d.commitments, wrongProofs, indices); err == nil {
		t.Fatal("expected verification failure with swapped proofs")
	}
}

func TestCKZGRecoverBlob(t *testing.T)  { testRecoverBlob(t, true) }
func TestGoKZGRecoverBlob(t *testing.T) { testRecoverBlob(t, false) }

func testRecoverBlob(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	// Precompute blobs once, vary only cell indices per iteration
	d := newBlobs(t, 3)

	for iter := range 50 {
		rng := mrand.New(mrand.NewSource(int64(iter)))
		numCells := DataPerBlob + rng.Intn(CellsPerBlob-DataPerBlob+1)
		indices := randCellIndices(rng, numCells)

		var partialCells []Cell
		for bi := range 3 {
			for _, idx := range indices {
				partialCells = append(partialCells, d.cells[bi*CellsPerBlob+int(idx)])
			}
		}
		recovered, err := RecoverBlobs(partialCells, indices)
		if err != nil {
			t.Fatalf("iter %d: failed to recover blob with %d cells: %v", iter, numCells, err)
		}
		if err := VerifyCellProofs(recovered, d.commitments, d.proofs); err != nil {
			t.Fatalf("iter %d: recovered blobs failed verification: %v", iter, err)
		}
		for i := range d.blobs {
			if recovered[i] != d.blobs[i] {
				t.Fatalf("iter %d: recovered blob %d does not match original", iter, i)
			}
		}
	}
}

func TestCKZGRecoverBlobWithInsufficientCells(t *testing.T) {
	testRecoverBlobWithInsufficientCells(t, true)
}
func TestGoKZGRecoverBlobWithInsufficientCells(t *testing.T) {
	testRecoverBlobWithInsufficientCells(t, false)
}

func testRecoverBlobWithInsufficientCells(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const blobCount = 3
	d := newBlobs(t, blobCount)

	// Use DataPerBlob-1 cells (one short of minimum required)
	indices := make([]uint64, DataPerBlob-1)
	for i := range indices {
		indices[i] = uint64(i)
	}
	var partialCells []Cell
	for bi := range blobCount {
		for _, idx := range indices {
			partialCells = append(partialCells, d.cells[bi*CellsPerBlob+int(idx)])
		}
	}
	if _, err := RecoverBlobs(partialCells, indices); err == nil {
		t.Fatalf("expected error with only %d cells, got none", len(indices))
	}
}
