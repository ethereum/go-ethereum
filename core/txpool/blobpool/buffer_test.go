package blobpool

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// makeV1Tx creates a V1 blob transaction with cell proofs, then strips blobs
// (simulating what ETH/72 peers send).
func makeV1Tx(t *testing.T, nonce uint64, blobCount int, blobOffset int, key *ecdsa.PrivateKey) *types.Transaction {
	t.Helper()
	tx := makeMultiBlobTx(nonce, 1, 1, 1, blobCount, blobOffset, key)
	return removeBlobs(tx)
}

// makePeerDelivery creates a PeerDelivery for given cell indices from a set of blobs.
func makePeerDelivery(t *testing.T, blobOffset, blobCount int, indices []uint64) *PeerDelivery {
	t.Helper()
	var allCells []kzg4844.Cell
	for i := 0; i < blobCount; i++ {
		cells, err := kzg4844.ComputeCells([]kzg4844.Blob{*testBlobs[blobOffset+i]})
		if err != nil {
			t.Fatal(err)
		}
		allCells = append(allCells, cells...)
	}
	var deliveryCells []kzg4844.Cell
	for b := 0; b < blobCount; b++ {
		for _, idx := range indices {
			deliveryCells = append(deliveryCells, allCells[b*kzg4844.CellsPerBlob+int(idx)])
		}
	}
	return &PeerDelivery{Cells: deliveryCells, Indices: indices}
}

func newTestBuffer(t *testing.T) *BlobBuffer {
	t.Helper()
	return NewBlobBuffer(BlobBufferFunctions{
		ValidateTx: func(tx *types.Transaction) error { return nil },
		AddToPool:  func(ptx *BlobTxForPool) error { return nil },
		DropPeer:   func(peer string) {},
	})
}

func TestSortCells(t *testing.T) {
	blobCount := 2
	blobOffset := 0

	peerA := makePeerDelivery(t, blobOffset, blobCount, []uint64{5, 3})
	peerB := makePeerDelivery(t, blobOffset, blobCount, []uint64{1, 7})

	custody := types.NewCustodyBitmap([]uint64{1, 3, 5, 7})
	entry := &cellEntry{
		deliveries: map[string]*PeerDelivery{
			"peerA": peerA,
			"peerB": peerB,
		},
		custody: custody,
	}
	sorted, resultCustody := sortCells(entry, blobCount)

	resultIndices := resultCustody.Indices()
	if len(resultIndices) != 4 {
		t.Fatalf("expected 4 indices, got %d", len(resultIndices))
	}
	for i, expected := range []uint64{1, 3, 5, 7} {
		if resultIndices[i] != expected {
			t.Errorf("index %d: expected %d, got %d", i, expected, resultIndices[i])
		}
	}

	expected := makePeerDelivery(t, blobOffset, blobCount, []uint64{1, 3, 5, 7})
	if len(sorted) != len(expected.Cells) {
		t.Fatalf("sorted length %d != expected %d", len(sorted), len(expected.Cells))
	}
	for i := range sorted {
		if sorted[i] != expected.Cells[i] {
			t.Errorf("cell %d mismatch", i)
		}
	}
}

func TestAddTxThenCells(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 2
	buf := newTestBuffer(t)

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()

	if err := buf.AddTx([]*types.Transaction{tx}, "peerA")[0]; err != nil {
		t.Fatal(err)
	}
	if !buf.HasTx(hash) {
		t.Fatal("tx should be buffered")
	}

	dataIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	delivery := makePeerDelivery(t, 0, blobCount, dataIndices)
	custody := types.NewCustodyBitmap(dataIndices)

	buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, custody)
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after add")
	}
}

func TestAddCellsThenTx(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 2
	buf := newTestBuffer(t)

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()

	dataIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	delivery := makePeerDelivery(t, 0, blobCount, dataIndices)
	custody := types.NewCustodyBitmap(dataIndices)

	buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, custody)
	if !buf.HasCells(hash) {
		t.Fatal("cells should be buffered")
	}

	if err := buf.AddTx([]*types.Transaction{tx}, "peerA")[0]; err != nil {
		t.Fatal(err)
	}
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after add")
	}
}

func TestMultiPeerDelivery(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 2
	buf := newTestBuffer(t)

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()
	buf.AddTx([]*types.Transaction{tx}, "peerA")

	indicesA := []uint64{0, 2, 4, 6}
	indicesB := []uint64{1, 3, 5, 7}
	deliveryA := makePeerDelivery(t, 0, blobCount, indicesA)
	deliveryB := makePeerDelivery(t, 0, blobCount, indicesB)

	allIndices := append(indicesA, indicesB...)
	custody := types.NewCustodyBitmap(allIndices)

	buf.AddCells(hash, map[string]*PeerDelivery{
		"peerB": deliveryA,
		"peerC": deliveryB,
	}, custody)
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after add")
	}
}

func TestBadCell(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 1

	var dropped []string
	buf := NewBlobBuffer(BlobBufferFunctions{
		ValidateTx: func(tx *types.Transaction) error { return nil },
		AddToPool:  func(ptx *BlobTxForPool) error { return nil },
		DropPeer:   func(peer string) { dropped = append(dropped, peer) },
	})

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()
	buf.AddTx([]*types.Transaction{tx}, "peerA")

	goodDelivery := makePeerDelivery(t, 0, blobCount, []uint64{0, 1, 2, 3})
	badDelivery := makePeerDelivery(t, 0, blobCount, []uint64{4, 5, 6, 7})
	for i := range badDelivery.Cells {
		for j := range badDelivery.Cells[i] {
			badDelivery.Cells[i][j] ^= 0xFF
		}
	}

	allIndices := []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	custody := types.NewCustodyBitmap(allIndices)

	buf.AddCells(hash, map[string]*PeerDelivery{
		"peerB": goodDelivery,
		"peerC": badDelivery,
	}, custody)

	if len(dropped) != 1 || dropped[0] != "peerC" {
		t.Fatalf("only peerC should have been dropped, got: %v", dropped)
	}
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after bad cell drop")
	}
}

// TestProviderExtension checks that a transaction completed with the data cells
// only (a full fetch) is extended to the full cell set before being handed to
// the pool, with all-ones custody and cells matching the ground truth.
func TestProviderExtension(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 2

	var stored []*BlobTxForPool
	buf := NewBlobBuffer(BlobBufferFunctions{
		ValidateTx: func(tx *types.Transaction) error { return nil },
		AddToPool:  func(ptx *BlobTxForPool) error { stored = append(stored, ptx); return nil },
		DropPeer:   func(peer string) {},
	})

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()
	buf.AddTx([]*types.Transaction{tx}, "peerA")

	dataIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	delivery := makePeerDelivery(t, 0, blobCount, dataIndices)
	buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, types.NewCustodyBitmap(dataIndices))

	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after completion")
	}
	buf.Flush()
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored tx, got %d", len(stored))
	}
	cs := stored[0].CellSidecar
	if cs.Custody != types.CustodyBitmapAll {
		t.Fatalf("custody not extended to all-ones: %d cells", cs.Custody.OneCount())
	}
	if len(cs.Cells) != blobCount*kzg4844.CellsPerBlob {
		t.Fatalf("expected %d cells, got %d", blobCount*kzg4844.CellsPerBlob, len(cs.Cells))
	}
	// Compare every cell (data and extension) against the ground truth.
	for b := 0; b < blobCount; b++ {
		truth, err := kzg4844.ComputeCells([]kzg4844.Blob{*testBlobs[b]})
		if err != nil {
			t.Fatal(err)
		}
		for i := range truth {
			if cs.Cells[b*kzg4844.CellsPerBlob+i] != truth[i] {
				t.Fatalf("blob %d cell %d does not match ground truth", b, i)
			}
		}
	}
}

// TestProviderExtensionBadProof checks that a transaction whose shipped
// extension proofs don't match its (verified) data is discarded rather than
// stored: the extension cells are correct by construction, so a proof mismatch
// means the transaction's own sidecar is inconsistent.
func TestProviderExtensionBadProof(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 1

	var (
		stored  []*BlobTxForPool
		dropped []string
	)
	buf := NewBlobBuffer(BlobBufferFunctions{
		ValidateTx: func(tx *types.Transaction) error { return nil },
		AddToPool:  func(ptx *BlobTxForPool) error { stored = append(stored, ptx); return nil },
		DropPeer:   func(peer string) { dropped = append(dropped, peer) },
	})

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	// Corrupt one extension proof (an index outside the delivered custody, so
	// per-peer verification cannot catch it).
	tx.BlobTxSidecar().Proofs[kzg4844.DataPerBlob][0] ^= 0xff
	hash := tx.Hash()
	buf.AddTx([]*types.Transaction{tx}, "peerA")

	dataIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	delivery := makePeerDelivery(t, 0, blobCount, dataIndices)
	buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, types.NewCustodyBitmap(dataIndices))

	buf.Flush()
	if len(stored) != 0 {
		t.Fatalf("tx with bad extension proof must not be stored, got %d", len(stored))
	}
	if len(dropped) != 0 {
		t.Fatalf("cell-delivering peers must not be dropped for a bad tx sidecar, got: %v", dropped)
	}
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after drop")
	}
}

// TestNoExtensionBelowThreshold checks that a partial (sampler) transaction
// with fewer than DataPerBlob cells is stored as delivered, unextended.
func TestNoExtensionBelowThreshold(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 2

	var stored []*BlobTxForPool
	buf := NewBlobBuffer(BlobBufferFunctions{
		ValidateTx: func(tx *types.Transaction) error { return nil },
		AddToPool:  func(ptx *BlobTxForPool) error { stored = append(stored, ptx); return nil },
		DropPeer:   func(peer string) {},
	})

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()
	buf.AddTx([]*types.Transaction{tx}, "peerA")

	indices := []uint64{3, 17, 64, 100} // sampler custody, incl. extension columns
	delivery := makePeerDelivery(t, 0, blobCount, indices)
	buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, types.NewCustodyBitmap(indices))

	buf.Flush()
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored tx, got %d", len(stored))
	}
	cs := stored[0].CellSidecar
	if cs.Custody != types.NewCustodyBitmap(indices) {
		t.Fatalf("sampler custody must be stored as delivered")
	}
	if len(cs.Cells) != blobCount*len(indices) {
		t.Fatalf("expected %d cells, got %d", blobCount*len(indices), len(cs.Cells))
	}
}
