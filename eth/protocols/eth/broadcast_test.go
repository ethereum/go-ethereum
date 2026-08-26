// Copyright 2026 The go-ethereum Authors
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

package eth

import (
	"crypto/rand"
	"crypto/sha256"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// announceTestPool is a TxPool/BlobPool stub with mutable custody, so that a
// transaction can become reconstructable while it sits in the announce queue.
type announceTestPool struct {
	lock    sync.RWMutex
	custody map[common.Hash]*types.CustodyBitmap
}

func newAnnounceTestPool() *announceTestPool {
	return &announceTestPool{custody: make(map[common.Hash]*types.CustodyBitmap)}
}

func (p *announceTestPool) setCustody(hash common.Hash, indices []uint64) {
	bitmap := types.NewCustodyBitmap(indices)
	p.lock.Lock()
	defer p.lock.Unlock()
	p.custody[hash] = &bitmap
}

func (p *announceTestPool) GetCustody(hash common.Hash) *types.CustodyBitmap {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.custody[hash]
}

func (p *announceTestPool) GetBlobHashes(hash common.Hash) []common.Hash { return nil }

func (p *announceTestPool) GetBlobCells([]common.Hash, types.CustodyBitmap) ([][]*kzg4844.Cell, [][]*kzg4844.Proof, error) {
	return nil, nil, nil
}

func (p *announceTestPool) Has(hash common.Hash) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	_, ok := p.custody[hash]
	return ok
}

func (p *announceTestPool) Get(hash common.Hash) *types.Transaction { return nil }

func (p *announceTestPool) GetRLP(hash common.Hash, version uint) []byte { return nil }

func (p *announceTestPool) GetMetadata(hash common.Hash) *txpool.TxMetadata {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if _, ok := p.custody[hash]; !ok {
		return nil
	}
	return &txpool.TxMetadata{Type: types.BlobTxType, Size: 100}
}

func fullCustodyIndices() []uint64 {
	indices := make([]uint64, kzg4844.DataPerBlob)
	for i := range indices {
		indices[i] = uint64(i)
	}
	return indices
}

// TestSparseBlobAnnouncementRetriedAfterCustodyGrows checks that withholding an
// announcement from a legacy peer is not permanent.
func TestSparseBlobAnnouncementRetriedAfterCustodyGrows(t *testing.T) {
	pool := newAnnounceTestPool()

	sparseTx := common.Hash{0x01}
	servedTx := common.Hash{0x02}
	unrelatedTx := common.Hash{0x03}
	pool.setCustody(sparseTx, []uint64{0, 1, 2, 3, 4, 5, 6, 7})
	pool.setCustody(servedTx, fullCustodyIndices())
	pool.setCustody(unrelatedTx, fullCustodyIndices())

	app, net := p2p.MsgPipe()
	defer app.Close()

	var id enode.ID
	rand.Read(id[:])
	peer := NewPeer(ETH71, p2p.NewPeer(id, "legacy", nil), net, pool, pool, nil)
	defer peer.Close()

	announced := announcementStream(app)

	peer.AsyncSendPooledTransactionHashes([]common.Hash{sparseTx, servedTx})
	hashes := nextAnnouncement(t, announced)
	if len(hashes) != 1 || hashes[0] != servedTx {
		t.Fatalf("first announcement = %v, want only the reconstructable transaction %v", hashes, servedTx)
	}
	pool.setCustody(sparseTx, fullCustodyIndices())

	// Wake the loop with an unrelated transaction rather than re-announcing
	// sparseTx, which would pass even if the withheld round had dropped it.
	peer.AsyncSendPooledTransactionHashes([]common.Hash{unrelatedTx})

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("transaction was never announced after becoming reconstructable")
		case hashes, ok := <-announced:
			if !ok {
				t.Fatal("announcement stream closed before the transaction was announced")
			}
			for _, h := range hashes {
				if h == sparseTx {
					return
				}
			}
		}
	}
}

// announcementStream drains announcements into a channel, so that a missing one
// fails on a deadline instead of blocking forever.
func announcementStream(app *p2p.MsgPipeRW) <-chan []common.Hash {
	stream := make(chan []common.Hash, 16)
	go func() {
		defer close(stream)
		for {
			msg, err := app.ReadMsg()
			if err != nil {
				return
			}
			if msg.Code != NewPooledTransactionHashesMsg {
				continue
			}
			var announcement NewPooledTransactionHashesPacket71
			if err := msg.Decode(&announcement); err != nil {
				return
			}
			stream <- announcement.Hashes
		}
	}()
	return stream
}

func nextAnnouncement(t *testing.T, stream <-chan []common.Hash) []common.Hash {
	t.Helper()
	select {
	case hashes, ok := <-stream:
		if !ok {
			t.Fatal("announcement stream closed unexpectedly")
		}
		return hashes
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for an announcement")
		return nil
	}
}

func TestCanServeTransaction(t *testing.T) {
	sparse := types.NewCustodyBitmap([]uint64{0, 1, 2, 3, 4, 5, 6, 7})
	recoverableIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range recoverableIndices {
		recoverableIndices[i] = uint64(i)
	}
	recoverable := types.NewCustodyBitmap(recoverableIndices)

	tests := []struct {
		name    string
		version uint
		custody *types.CustodyBitmap
		want    bool
	}{
		{"legacy transaction to eth/71", ETH71, nil, true},
		{"sparse blob transaction to eth/71", ETH71, &sparse, false},
		{"recoverable blob transaction to eth/71", ETH71, &recoverable, true},
		{"sparse blob transaction to eth/72", ETH72, &sparse, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := canServeTransaction(test.version, test.custody); got != test.want {
				t.Fatalf("canServeTransaction(%d, %v) = %v, want %v", test.version, test.custody, got, test.want)
			}
		})
	}
}

func TestLegacyPeerDoesNotReceiveSparseBlobAnnouncement(t *testing.T) {
	backend := newTestBackendWithGenerator(0, true, true, nil)
	defer backend.close()

	var blob kzg4844.Blob
	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := kzg4844.ComputeCellProofs(&blob)
	if err != nil {
		t.Fatal(err)
	}
	cells, err := kzg4844.ComputeCells([]kzg4844.Blob{blob})
	if err != nil {
		t.Fatal(err)
	}
	blobHash := kzg4844.CalcBlobHashV1(sha256.New(), &commitment)
	makeTx := func(nonce uint64) *types.Transaction {
		tx, err := types.SignNewTx(testKey, types.NewCancunSigner(params.TestChainConfig.ChainID), &types.BlobTx{
			ChainID:    uint256.MustFromBig(params.TestChainConfig.ChainID),
			Nonce:      nonce,
			GasTipCap:  uint256.NewInt(20_000_000_000),
			GasFeeCap:  uint256.NewInt(21_000_000_000),
			Gas:        params.TxGas,
			To:         testAddr,
			BlobHashes: []common.Hash{blobHash},
			BlobFeeCap: uint256.MustFromBig(common.Big1),
		})
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	sparseTx := makeTx(0)
	fullTx := makeTx(1)
	custody := types.NewCustodyBitmap([]uint64{0, 1, 2, 3, 4, 5, 6, 7})
	sparse := &blobpool.BlobTxForPool{
		Tx: sparseTx,
		CellSidecar: &types.BlobTxCellSidecar{
			Version:     types.BlobSidecarVersion1,
			Cells:       cells[:custody.OneCount()],
			Commitments: []kzg4844.Commitment{commitment},
			Proofs:      proofs,
			Custody:     custody,
		},
	}
	full := &blobpool.BlobTxForPool{
		Tx: fullTx,
		CellSidecar: &types.BlobTxCellSidecar{
			Version:     types.BlobSidecarVersion1,
			Cells:       cells,
			Commitments: []kzg4844.Commitment{commitment},
			Proofs:      proofs,
			Custody:     types.CustodyBitmapAll,
		},
	}
	if err := backend.blobpool.AddPooledTx(sparse); err != nil {
		t.Fatal(err)
	}
	if err := backend.blobpool.AddPooledTx(full); err != nil {
		t.Fatal(err)
	}
	if encoded := backend.txpool.GetRLP(sparseTx.Hash(), ETH71); encoded != nil {
		t.Fatalf("legacy encoding unexpectedly succeeded with %d bytes", len(encoded))
	}
	if encoded := backend.txpool.GetRLP(fullTx.Hash(), ETH71); encoded == nil {
		t.Fatal("legacy encoding unexpectedly failed for a reconstructable transaction")
	}

	peer, _ := newTestPeer("legacy", ETH71, backend)
	defer peer.close()
	peer.AsyncSendPooledTransactionHashes([]common.Hash{sparseTx.Hash(), fullTx.Hash()})

	msg, err := peer.app.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Code != NewPooledTransactionHashesMsg {
		t.Fatalf("message code %d, want %d", msg.Code, NewPooledTransactionHashesMsg)
	}
	var announcement NewPooledTransactionHashesPacket71
	if err := msg.Decode(&announcement); err != nil {
		t.Fatal(err)
	}
	if len(announcement.Hashes) != 1 || announcement.Hashes[0] != fullTx.Hash() {
		t.Fatalf("announced hashes %v, want only reconstructable transaction %v", announcement.Hashes, fullTx.Hash())
	}
}
