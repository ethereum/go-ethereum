package blobpool

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

// TestBufferByteCap checks that transactions whose cells never arrive cannot
// grow the buffer past its byte limit, and that the oldest ones give way.
func TestBufferByteCap(t *testing.T) {
	key, _ := crypto.GenerateKey()
	buf := newTestBuffer(t)

	// Size the buffer so that it fits exactly three transactions.
	probe := makeV1Tx(t, 0, 1, 0, key)
	buf.maxBytes = 3 * probe.Size()

	var hashes []common.Hash
	for nonce := uint64(0); nonce < 6; nonce++ {
		tx := makeV1Tx(t, nonce, 1, 0, key)
		hashes = append(hashes, tx.Hash())
		if err := buf.AddTx([]*types.Transaction{tx}, "peerA")[0]; err != nil {
			t.Fatalf("tx %d: %v", nonce, err)
		}
		// Entries are stamped with wall clock time, so make sure consecutive
		// transactions are distinguishable by age.
		time.Sleep(time.Millisecond)
	}
	if buf.txBytes > buf.maxBytes {
		t.Errorf("buffer over its limit: %d > %d", buf.txBytes, buf.maxBytes)
	}
	if len(buf.txs) != 3 {
		t.Errorf("expected 3 buffered txs, got %d", len(buf.txs))
	}
	// The three most recent transactions should have displaced the earlier ones.
	for i, hash := range hashes {
		if want := i >= 3; buf.HasTx(hash) != want {
			t.Errorf("tx %d: buffered %v, want %v", i, buf.HasTx(hash), want)
		}
	}
}

// TestBufferByteAccounting checks that the accounted size tracks the buffer
// content across every path that adds or removes an entry.
func TestBufferByteAccounting(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 1
	buf := newTestBuffer(t)

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()

	// A repeated delivery must not be counted twice.
	buf.AddTx([]*types.Transaction{tx}, "peerA")
	buf.AddTx([]*types.Transaction{tx}, "peerB")
	if buf.txBytes != tx.Size() {
		t.Fatalf("after duplicate delivery: accounted %d, want %d", buf.txBytes, tx.Size())
	}

	// Completing the transaction must release its space.
	indices := make([]uint64, kzg4844.DataPerBlob)
	for i := range indices {
		indices[i] = uint64(i)
	}
	buf.AddCells(hash, map[string]*PeerDelivery{
		"peerC": makePeerDelivery(t, 0, blobCount, indices),
	}, types.NewCustodyBitmap(indices))
	if buf.txBytes != 0 {
		t.Fatalf("after completion: accounted %d, want 0", buf.txBytes)
	}

	// So must expiry.
	buf.AddTx([]*types.Transaction{makeV1Tx(t, 1, blobCount, 0, key)}, "peerA")
	for _, entry := range buf.txs {
		entry.added = time.Now().Add(-2 * bufferLifetime)
	}
	buf.AddTx([]*types.Transaction{makeV1Tx(t, 2, blobCount, 0, key)}, "peerA")
	if want := makeV1Tx(t, 2, blobCount, 0, key).Size(); buf.txBytes != want {
		t.Fatalf("after expiry: accounted %d, want %d", buf.txBytes, want)
	}
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
	sorted, resultCustody, err := sortCells(entry, blobCount)
	if err != nil {
		t.Fatal(err)
	}

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
