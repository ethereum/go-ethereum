package blobpool

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// makeV1Tx creates a V1 blob transaction with cell proofs, then strips blobs
// (simulating what ETH/72 peers send).
func makeV1Tx(t *testing.T, nonce uint64, blobCount int, blobOffset int, key *ecdsa.PrivateKey) *types.Transaction {
	t.Helper()
	tx := makeMultiBlobTx(nonce, 1, 1, 1, blobCount, blobOffset, key, types.BlobSidecarVersion1)
	return tx.WithoutBlob()
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
	return NewBlobBuffer(
		func(ptx *PooledBlobTx) error { return nil },
		func(peer string) {},
	)
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
		custody: &custody,
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

	if err := buf.AddTx(tx, "peerA"); err != nil {
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

	if err := buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, &custody); err != nil {
		t.Fatal(err)
	}
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

	if err := buf.AddCells(hash, map[string]*PeerDelivery{"peerB": delivery}, &custody); err != nil {
		t.Fatal(err)
	}
	if !buf.HasCells(hash) {
		t.Fatal("cells should be buffered")
	}

	if err := buf.AddTx(tx, "peerA"); err != nil {
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
	buf.AddTx(tx, "peerA")

	indicesA := []uint64{0, 2, 4, 6}
	indicesB := []uint64{1, 3, 5, 7}
	deliveryA := makePeerDelivery(t, 0, blobCount, indicesA)
	deliveryB := makePeerDelivery(t, 0, blobCount, indicesB)

	allIndices := append(indicesA, indicesB...)
	custody := types.NewCustodyBitmap(allIndices)

	if err := buf.AddCells(hash, map[string]*PeerDelivery{
		"peerB": deliveryA,
		"peerC": deliveryB,
	}, &custody); err != nil {
		t.Fatal(err)
	}
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after add")
	}
}

func TestBadCell(t *testing.T) {
	key, _ := crypto.GenerateKey()
	blobCount := 1

	var dropped []string
	buf := NewBlobBuffer(
		func(ptx *PooledBlobTx) error { return nil },
		func(peer string) { dropped = append(dropped, peer) },
	)

	tx := makeV1Tx(t, 0, blobCount, 0, key)
	hash := tx.Hash()
	buf.AddTx(tx, "peerA")

	goodDelivery := makePeerDelivery(t, 0, blobCount, []uint64{0, 1, 2, 3})
	badDelivery := makePeerDelivery(t, 0, blobCount, []uint64{4, 5, 6, 7})
	for i := range badDelivery.Cells {
		for j := range badDelivery.Cells[i] {
			badDelivery.Cells[i][j] ^= 0xFF
		}
	}

	allIndices := []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	custody := types.NewCustodyBitmap(allIndices)

	err := buf.AddCells(hash, map[string]*PeerDelivery{
		"peerB": goodDelivery,
		"peerC": badDelivery,
	}, &custody)
	if err == nil {
		t.Fatal("expected error from bad cells")
	}

	if len(dropped) != 1 || dropped[0] != "peerC" {
		t.Fatalf("only peerC should have been dropped, got: %v", dropped)
	}
	if buf.HasTx(hash) || buf.HasCells(hash) {
		t.Fatal("buffer should be empty after bad cell drop")
	}
}

func TestBadTx(t *testing.T) {
	key, _ := crypto.GenerateKey()

	var dropped []string
	buf := NewBlobBuffer(
		func(ptx *PooledBlobTx) error { return nil },
		func(peer string) { dropped = append(dropped, peer) },
	)

	blobtx := &types.BlobTx{
		ChainID:    uint256.MustFromBig(params.MainnetChainConfig.ChainID),
		Nonce:      0,
		GasTipCap:  uint256.NewInt(1),
		GasFeeCap:  uint256.NewInt(1),
		Gas:        21000,
		BlobFeeCap: uint256.NewInt(1),
		BlobHashes: []common.Hash{testBlobVHashes[0]},
		Value:      uint256.NewInt(100),
		Sidecar: types.NewBlobTxSidecar(types.BlobSidecarVersion1,
			nil,
			[]kzg4844.Commitment{testBlobCommits[1]},
			testBlobCellProofs[1],
		),
	}
	tx := types.MustSignNewTx(key, types.LatestSigner(params.MainnetChainConfig), blobtx)

	err := buf.AddTx(tx, "peerA")
	if err == nil {
		t.Fatal("expected error from commitment mismatch")
	}
	if len(dropped) != 1 || dropped[0] != "peerA" {
		t.Fatalf("only peerA should have been dropped, got: %v", dropped)
	}
	if buf.HasTx(tx.Hash()) {
		t.Fatal("tx should not be buffered")
	}
}
