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
	"encoding/binary"
	"math/big"
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

func TestCKZGRecoverCells(t *testing.T)  { testRecoverCells(t, true) }
func TestGoKZGRecoverCells(t *testing.T) { testRecoverCells(t, false) }

// testRecoverCells checks that RecoverCells reconstructs the complete 128-cell
// set for each blob, via both the fast systematic path (data cells present) and
// the erasure-recovery slow path (non-data subset), byte-identical to the
// original cells.
func testRecoverCells(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const blobCount = 2
	d := newBlobs(t, blobCount)

	collect := func(indices []uint64) []Cell {
		var cs []Cell
		for bi := range blobCount {
			for _, idx := range indices {
				cs = append(cs, d.cells[bi*CellsPerBlob+int(idx)])
			}
		}
		return cs
	}
	seq := func(start, n int) []uint64 {
		idx := make([]uint64, n)
		for i := range idx {
			idx[i] = uint64(start + i)
		}
		return idx
	}
	assertRecoversAll := func(name string, indices []uint64) {
		t.Helper()
		got, err := RecoverCells(collect(indices), indices)
		if err != nil {
			t.Fatalf("%s: RecoverCells failed: %v", name, err)
		}
		if len(got) != blobCount*CellsPerBlob {
			t.Fatalf("%s: got %d cells, want %d", name, len(got), blobCount*CellsPerBlob)
		}
		for i := range d.cells {
			if got[i] != d.cells[i] {
				t.Fatalf("%s: cell %d does not match original", name, i)
			}
		}
	}

	// Fast path: exactly the data cells.
	assertRecoversAll("fast/data-0..63", seq(0, DataPerBlob))
	// Slow path: a non-data 64-cell subset (indices 32..95).
	assertRecoversAll("slow/non-data-32..95", seq(32, DataPerBlob))
	// Full custody: all cells present (still takes the fast path).
	assertRecoversAll("full-0..127", seq(0, CellsPerBlob))

	// Fewer than DataPerBlob cells cannot be recovered.
	short := seq(0, DataPerBlob-1)
	if _, err := RecoverCells(collect(short), short); err == nil {
		t.Fatalf("expected error with only %d cells", DataPerBlob-1)
	}

	// Malformed indices (duplicate tail): the fast path declines and the
	// erasure path rejects, so both agree on refusal.
	dup := append(seq(0, DataPerBlob), DataPerBlob-1)
	if _, err := RecoverCells(collect(dup), dup); err == nil {
		t.Fatalf("expected error for duplicate index")
	}

	// Randomized subsets exercise the erasure path with arbitrary shapes
	// (kept to a few iterations: each is a full erasure decode).
	for iter := range 3 {
		rng := mrand.New(mrand.NewSource(int64(iter)))
		n := DataPerBlob + rng.Intn(CellsPerBlob-DataPerBlob)
		assertRecoversAll("random", randCellIndices(rng, n))
	}
}

// erasureRecoverBlobs runs the KZG erasure recovery directly, bypassing the
// concat fast path RecoverBlobs takes, so the two can be compared.
func erasureRecoverBlobs(cells []Cell, cellIndices []uint64) ([]Blob, error) {
	if useCKZG.Load() {
		return ckzgRecoverBlobs(cells, cellIndices)
	}
	return gokzgRecoverBlobs(cells, cellIndices)
}

func TestCKZGBlobsFromDataCells(t *testing.T)  { testBlobsFromDataCells(t, true) }
func TestGoKZGBlobsFromDataCells(t *testing.T) { testBlobsFromDataCells(t, false) }

// testBlobsFromDataCells checks that the KZG-free fast path reconstructs the
// original blobs whenever the data cells are present, agrees byte-for-byte with
// RecoverBlobs, and declines (ok=false) when a data cell is missing.
func testBlobsFromDataCells(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const blobCount = 2
	d := newBlobs(t, blobCount)

	// collect gathers the cells for the given per-blob indices across all blobs.
	collect := func(indices []uint64) []Cell {
		var cells []Cell
		for bi := range blobCount {
			for _, idx := range indices {
				cells = append(cells, d.cells[bi*CellsPerBlob+int(idx)])
			}
		}
		return cells
	}
	// assertRecovers checks the fast path succeeds and matches both the original
	// blobs and the erasure recovery it stands in for.
	assertRecovers := func(name string, indices []uint64) {
		t.Helper()
		cells := collect(indices)
		fast, ok := blobsFromDataCells(cells, indices)
		if !ok {
			t.Fatalf("%s: fast path declined, expected success", name)
		}
		slow, err := erasureRecoverBlobs(cells, indices)
		if err != nil {
			t.Fatalf("%s: erasure recovery failed: %v", name, err)
		}
		for i := range d.blobs {
			if fast[i] != d.blobs[i] {
				t.Fatalf("%s: fast blob %d does not match original", name, i)
			}
			if fast[i] != slow[i] {
				t.Fatalf("%s: fast blob %d does not match the erasure recovery", name, i)
			}
		}
	}

	// Exactly the data cells, in canonical order.
	dataIndices := make([]uint64, DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	assertRecovers("data-only", dataIndices)

	// Full custody: all cells present, data cells plus extension cells.
	allIndices := make([]uint64, CellsPerBlob)
	for i := range allIndices {
		allIndices[i] = uint64(i)
	}
	assertRecovers("full-custody", allIndices)

	// Data cells present but out of order: must decline, as RecoverBlobs
	// rejects non-ascending indices.
	unordered := slices.Clone(dataIndices)
	unordered[0], unordered[1] = unordered[1], unordered[0]
	if _, ok := blobsFromDataCells(collect(unordered), unordered); ok {
		t.Fatalf("unordered-data: fast path succeeded, expected decline")
	}

	// A data cell missing (index 63 replaced by an extension cell): the fast
	// path must decline, while RecoverBlobs can still reconstruct.
	missing := slices.Clone(dataIndices)
	missing[DataPerBlob-1] = DataPerBlob // drop data cell 63, add extension cell 64
	if _, ok := blobsFromDataCells(collect(missing), missing); ok {
		t.Fatalf("missing-data: fast path succeeded, expected decline")
	}
	if _, err := RecoverBlobs(collect(missing), missing); err != nil {
		t.Fatalf("missing-data: RecoverBlobs failed: %v", err)
	}

	// Too few cells for recovery at all: fast path declines.
	short := dataIndices[:DataPerBlob-1]
	if _, ok := blobsFromDataCells(collect(short), short); ok {
		t.Fatalf("insufficient: fast path succeeded, expected decline")
	}

	// Malformed extension tails: inputs RecoverBlobs would reject, which the
	// fast path must decline rather than accept.
	duplicate := append(slices.Clone(dataIndices), DataPerBlob-1) // 63 repeated
	if _, ok := blobsFromDataCells(collect(duplicate), duplicate); ok {
		t.Fatalf("duplicate-tail: fast path succeeded, expected decline")
	}
	if _, err := RecoverBlobs(collect(duplicate), duplicate); err == nil {
		t.Fatalf("duplicate-tail: RecoverBlobs succeeded, expected error")
	}
	unorderedTail := append(slices.Clone(dataIndices), 65, 64)
	if _, ok := blobsFromDataCells(collect(unorderedTail), unorderedTail); ok {
		t.Fatalf("unordered-tail: fast path succeeded, expected decline")
	}
	outOfRange := append(slices.Clone(dataIndices), CellsPerBlob)
	cellsOOR := append(slices.Clone(collect(dataIndices)[:DataPerBlob]), Cell{}) // one blob
	if _, ok := blobsFromDataCells(cellsOOR, outOfRange); ok {
		t.Fatalf("out-of-range-tail: fast path succeeded, expected decline")
	}

	// Non-canonical field elements: the fast path bypasses the KZG library, so
	// it has to reject what that library would reject while deserializing. The
	// offending element goes in the last slot of the last cell of the last blob,
	// so a check that only looked at the first element, cell or blob would still
	// be caught, and it is the modulus itself, the tightest non-canonical value.
	var modulus [32]byte
	fr.Modulus().FillBytes(modulus[:])
	poison := func(cells []Cell) {
		last := &cells[len(cells)-1]
		copy(last[len(last)-32:], modulus[:])
	}
	// In a data cell, which the concatenation reads:
	badData := slices.Clone(collect(dataIndices))
	poison(badData)
	if _, ok := blobsFromDataCells(badData, dataIndices); ok {
		t.Fatalf("non-canonical-data: fast path succeeded, expected decline")
	}
	if _, err := RecoverBlobs(badData, dataIndices); err == nil {
		t.Fatalf("non-canonical-data: RecoverBlobs succeeded, expected error")
	}
	// And in a tail cell, which it ignores: declining keeps RecoverBlobs
	// rejecting exactly what the erasure path rejects.
	withTail := append(slices.Clone(dataIndices), DataPerBlob)
	badTail := collect(withTail)
	poison(badTail)
	if _, ok := blobsFromDataCells(badTail, withTail); ok {
		t.Fatalf("non-canonical-tail: fast path succeeded, expected decline")
	}
	if _, err := RecoverBlobs(badTail, withTail); err == nil {
		t.Fatalf("non-canonical-tail: RecoverBlobs succeeded, expected error")
	}

	// Single blob: the slicing math must hold for blobCount == 1 too.
	d1 := newBlobs(t, 1)
	single, ok := blobsFromDataCells(d1.cells[:DataPerBlob], dataIndices)
	if !ok {
		t.Fatalf("single-blob: fast path declined, expected success")
	}
	if single[0] != d1.blobs[0] {
		t.Fatalf("single-blob: reconstructed blob does not match original")
	}

	// Randomized well-formed tails: the data cells plus a random sorted subset
	// of the extension indices must be accepted and agree with RecoverBlobs.
	for iter := range 5 {
		rng := mrand.New(mrand.NewSource(int64(iter)))
		perm := rng.Perm(CellsPerBlob - DataPerBlob)
		tail := make([]uint64, rng.Intn(CellsPerBlob-DataPerBlob+1))
		for i := range tail {
			tail[i] = uint64(DataPerBlob + perm[i])
		}
		slices.Sort(tail)
		assertRecovers("random-tail", append(slices.Clone(dataIndices), tail...))
	}
}

func TestCKZGRecoverBlobsFastPath(t *testing.T)  { testRecoverBlobsFastPath(t, true) }
func TestGoKZGRecoverBlobsFastPath(t *testing.T) { testRecoverBlobsFastPath(t, false) }

// testRecoverBlobsFastPath checks that RecoverBlobs takes the KZG-free fast
// path when the data cells are present and falls back to full erasure recovery
// otherwise, matching the original blobs in both cases.
func testRecoverBlobsFastPath(t *testing.T, ckzg bool) {
	defer switchBackend(t, ckzg)()

	const blobCount = 2
	d := newBlobs(t, blobCount)

	// collect gathers the cells for the given per-blob indices across all blobs.
	collect := func(indices []uint64) []Cell {
		var cells []Cell
		for bi := range blobCount {
			for _, idx := range indices {
				cells = append(cells, d.cells[bi*CellsPerBlob+int(idx)])
			}
		}
		return cells
	}
	// assertRecovers checks recovery succeeds, verifies against the cell proofs,
	// and matches the original blobs.
	assertRecovers := func(name string, indices []uint64) {
		t.Helper()
		blobs, err := RecoverBlobs(collect(indices), indices)
		if err != nil {
			t.Fatalf("%s: recovery failed: %v", name, err)
		}
		if err := VerifyCellProofs(blobs, d.commitments, d.proofs); err != nil {
			t.Fatalf("%s: recovered blobs failed verification: %v", name, err)
		}
		for i := range d.blobs {
			if blobs[i] != d.blobs[i] {
				t.Fatalf("%s: recovered blob %d does not match original", name, i)
			}
		}
	}

	// Fast path: exactly the data cells, in canonical order.
	dataIndices := make([]uint64, DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	assertRecovers("data-only (fast path)", dataIndices)

	// Fallback: a non-data subset (data cell 0 swapped for extension cell 64)
	// must route through the KZG erasure decode and still reconstruct.
	sparse := slices.Clone(dataIndices)
	sparse[0] = DataPerBlob // drop data cell 0, add extension cell 64
	slices.Sort(sparse)
	if _, ok := blobsFromDataCells(collect(sparse), sparse); ok {
		t.Fatalf("test setup: expected fast path to decline for the sparse subset")
	}
	assertRecovers("sparse (fallback)", sparse)

	// Insufficient cells: recovery must error on either path.
	short := dataIndices[:DataPerBlob-1]
	if _, err := RecoverBlobs(collect(short), short); err == nil {
		t.Fatalf("insufficient: expected error, got none")
	}

	// A redundant cell that is canonical but inconsistent with the data pins the
	// one intentional divergence between the paths: the data cells decide, so the
	// blobs come back correct, where the erasure recovery would have mixed the
	// conflicting cell into the polynomial and returned neither faithfully.
	conflicting := append(slices.Clone(dataIndices), DataPerBlob)
	cells := collect(conflicting)
	clear(cells[DataPerBlob][:]) // zero is canonical, and is not the real cell
	if !isCanonicalCell(&cells[DataPerBlob]) {
		t.Fatalf("conflicting-tail: test setup must leave the cell canonical")
	}
	blobs, err := RecoverBlobs(cells, conflicting)
	if err != nil {
		t.Fatalf("conflicting-tail: RecoverBlobs failed: %v", err)
	}
	for i := range d.blobs {
		if blobs[i] != d.blobs[i] {
			t.Fatalf("conflicting-tail: blob %d was not taken from the data cells", i)
		}
	}
}

// TestFieldModulusLimbs pins the hardcoded modulus limbs used by the
// canonicalness check against the field's own definition.
func TestFieldModulusLimbs(t *testing.T) {
	var want [32]byte
	fr.Modulus().FillBytes(want[:])

	var got [32]byte
	binary.BigEndian.PutUint64(got[0:8], frModulusW0)
	binary.BigEndian.PutUint64(got[8:16], frModulusW1)
	binary.BigEndian.PutUint64(got[16:24], frModulusW2)
	binary.BigEndian.PutUint64(got[24:32], frModulusW3)

	if got != want {
		t.Fatalf("modulus limbs encode %x, field modulus is %x", got, want)
	}
}

// TestIsCanonicalFieldElement cross-checks the hand-rolled comparison against
// the field implementation whose deserialization it stands in for: the two
// extremes, uniform random inputs, and the boundary region where the limb
// comparison chain has to walk past equal limbs.
func TestIsCanonicalFieldElement(t *testing.T) {
	var (
		e   fr.Element
		buf [32]byte
	)
	check := func() {
		t.Helper()
		want := e.SetBytesCanonical(buf[:]) == nil
		if got := isCanonicalFieldElement(buf[:]); got != want {
			t.Fatalf("isCanonicalFieldElement(%x) = %v, library says %v", buf, got, want)
		}
	}
	// The extremes, which random input never produces: all-zero (buf as it
	// stands) and all-ones.
	check()
	for i := range buf {
		buf[i] = 0xff
	}
	check()

	// Uniform 32-byte values: a little under half are canonical, so both
	// answers get exercised.
	rng := mrand.New(mrand.NewSource(1))
	for range 4096 {
		if _, err := rng.Read(buf[:]); err != nil {
			t.Fatal(err)
		}
		check()
	}
	// One step either side of each limb's modulus value: only these inputs
	// reach the comparison of the limb in question.
	mod := fr.Modulus()
	for limb := range 4 {
		unit := new(big.Int).Lsh(big.NewInt(1), uint(64*(3-limb)))
		for _, delta := range []int64{-2, -1, 0, 1, 2} {
			v := new(big.Int).Add(mod, new(big.Int).Mul(big.NewInt(delta), unit))
			if v.Sign() < 0 || v.BitLen() > 256 {
				continue
			}
			v.FillBytes(buf[:])
			check()
		}
	}
}
