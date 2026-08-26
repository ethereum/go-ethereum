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
	"crypto/sha256"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

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
