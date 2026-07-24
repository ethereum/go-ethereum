// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
)

// Tests for eth/72 (sparse blobpool).

func (s *Suite) TestBlobTxAvailabilityFailure(t *utesting.T) {
	t.Log(`This test announces 10 blob txs from a single peer. With fetchProbability 0.15, 
there will be at least one partial fetch (1-0.15^10). When only 1 peer announced availability, 
partial fetch GetCells should never arrive. Any GetCells that does arrive must be a full fetch.`)

	custody := types.NewCustodyBitmap([]uint64{7, 19, 33, 52, 68, 90, 111, 126})
	if err := s.engine.sendForkchoiceUpdated(&custody); err != nil {
		t.Fatalf("send fcu failed: %v", err)
	}

	txs, _ := s.makeBlobTxs(10, 4, 0x30)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// Announce all transactions from a single peer.
	hashes := make([]common.Hash, len(txs))
	txTypes := make([]byte, len(txs))
	sizes := make([]uint32, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
		txTypes[i] = types.BlobTxType
		sizes[i] = uint32(tx.Size())
	}
	ann := eth.NewPooledTransactionHashesPacket72{
		Types:  txTypes,
		Sizes:  sizes,
		Hashes: hashes,
		Mask:   types.CustodyBitmapAll,
	}
	if err := conn.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
		t.Fatalf("announce failed: %v", err)
	}

	// Read messages for a short period. Any GetCells that arrives must be
	// a full fetch request (mask >= DataPerBlob), not a partial fetch.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := conn.ReadEth()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return // timeout, test passed
			}
			t.Fatalf("unexpected error: %v", err)
		}
		switch req := msg.(type) {
		case *eth.GetCellsRequestPacket:
			if req.Mask.OneCount() < kzg4844.DataPerBlob {
				t.Fatalf("received partial GetCells request with only %d cells from single peer announcement", req.Mask.OneCount())
			}
		case *eth.GetPooledTransactionsPacket:
			encTxs, _ := rlp.EncodeToRawList(txs)
			conn.Write(ethProto, eth.PooledTransactionsMsg, eth.PooledTransactionsPacket{
				RequestId: req.RequestId,
				List:      encTxs,
			})
		}
	}
}

func (s *Suite) TestGetCells(t *utesting.T) {
	t.Log(`This test checks that blob tx announcements trigger GetCells requests,
and that providing valid cells causes the tx to enter the pool.`)

	if err := s.engine.sendForkchoiceUpdated(nil); err != nil {
		t.Fatalf("send fcu failed: %v", err)
	}

	txs, blobs := s.makeBlobTxs(1, 1, 0x31)
	tx := txs[0]
	blob := blobs[0]

	// Create two peers to ensure GetCells arrives regardless of full/partial fetch path.
	// As per the specification, the tested node must fetch cells when the transaction has
	// two providers.
	conn1, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn1.Close()
	if err := conn1.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	conn2, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn2.Close()
	if err := conn2.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	ann := eth.NewPooledTransactionHashesPacket72{
		Types:  []byte{types.BlobTxType},
		Sizes:  []uint32{uint32(tx.Size())},
		Hashes: []common.Hash{tx.Hash()},
		Mask:   types.CustodyBitmapAll,
	}
	if err := conn1.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
		t.Fatalf("conn1 announce failed: %v", err)
	}
	if err := conn2.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
		t.Fatalf("conn2 announce failed: %v", err)
	}

	// Wait for GetPooledTransactions on either conn, respond with tx (without blobs).
	pooledReq, pc, err := readAnyFrom[eth.GetPooledTransactionsPacket](conn1, conn2)
	if err != nil {
		t.Fatalf("failed to read GetPooledTransactions: %v", err)
	}
	encTxs, _ := rlp.EncodeToRawList([]*types.Transaction{tx})
	resp := eth.PooledTransactionsPacket{RequestId: pooledReq.RequestId, List: encTxs}
	if err := pc.Write(ethProto, eth.PooledTransactionsMsg, resp); err != nil {
		t.Fatalf("writing pooled tx response failed: %v", err)
	}

	// Wait for GetCells request on either conn.
	cellsReq, cc, err := readAnyFrom[eth.GetCellsRequestPacket](conn1, conn2)
	if err != nil {
		t.Fatalf("failed to read GetCells: %v", err)
	}
	if len(cellsReq.Hashes) == 0 || cellsReq.Hashes[0] != tx.Hash() {
		t.Fatalf("GetCells for wrong hash: %v", cellsReq.Hashes)
	}

	// Respond with valid cells matching the requested mask.
	cells := buildCells(blob, cellsReq.Mask)
	cellsResp := eth.CellsPacket{
		RequestId: cellsReq.RequestId,
		CellsResponse: eth.CellsResponse{
			Hashes: []common.Hash{tx.Hash()},
			Cells:  encodeCells([][]kzg4844.Cell{cells}),
			Mask:   cellsReq.Mask,
		},
	}
	if err := cc.Write(ethProto, eth.CellsMsg, cellsResp); err != nil {
		t.Fatalf("writing cells response failed: %v", err)
	}

	// Peers should not be disconnected after providing valid data.
	if readUntilDisconnect(cc) {
		t.Fatalf("unexpected disconnect on cells-providing peer")
	}
}

func (s *Suite) TestBlobTxWithInvalidCells(t *utesting.T) {
	t.Log(`This test checks that a peer responding to GetCells with invalid cells is disconnected, 
while the other peer is not.`)

	if err := s.engine.sendForkchoiceUpdated(nil); err != nil {
		t.Fatalf("send fcu failed: %v", err)
	}

	txs, blobs := s.makeBlobTxs(1, 1, 0x32)
	tx := txs[0]
	blob := blobs[0]

	// Create two peers to make the tested node send GetCells (tx has two providers).
	conn1, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn1.Close()
	if err := conn1.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	conn2, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn2.Close()
	if err := conn2.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// The announcement is sent by both peers.
	ann := eth.NewPooledTransactionHashesPacket72{
		Types:  []byte{types.BlobTxType},
		Sizes:  []uint32{uint32(tx.Size())},
		Hashes: []common.Hash{tx.Hash()},
		Mask:   types.CustodyBitmapAll,
	}
	if err := conn1.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
		t.Fatalf("conn1 announce failed: %v", err)
	}
	if err := conn2.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
		t.Fatalf("conn2 announce failed: %v", err)
	}

	// The tested node should request the transaction and cells from any of the peers.
	pooledReq, pc, err := readAnyFrom[eth.GetPooledTransactionsPacket](conn1, conn2)
	if err != nil {
		t.Fatalf("failed to read GetPooledTransactions: %v", err)
	}
	encTxs, _ := rlp.EncodeToRawList([]*types.Transaction{tx})
	if err := pc.Write(ethProto, eth.PooledTransactionsMsg,
		eth.PooledTransactionsPacket{RequestId: pooledReq.RequestId, List: encTxs}); err != nil {
		t.Fatalf("writing pooled tx response failed: %v", err)
	}
	cellsReq, cc, err := readAnyFrom[eth.GetCellsRequestPacket](conn1, conn2)
	if err != nil {
		t.Fatalf("failed to read GetCells: %v", err)
	}

	// Respond with corrupted cells (all zero bytes).
	blobCount := len(blob)
	corrupted := make([]kzg4844.Cell, blobCount*cellsReq.Mask.OneCount())
	badResp := eth.CellsPacket{
		RequestId: cellsReq.RequestId,
		CellsResponse: eth.CellsResponse{
			Hashes: []common.Hash{tx.Hash()},
			Cells:  encodeCells([][]kzg4844.Cell{corrupted}),
			Mask:   cellsReq.Mask,
		},
	}
	if err := cc.Write(ethProto, eth.CellsMsg, badResp); err != nil {
		t.Fatalf("writing bad cells response failed: %v", err)
	}

	// The peer that sent corrupted cells must be disconnected.
	if !readUntilDisconnect(cc) {
		t.Fatalf("expected peer to be disconnected after invalid cells")
	}

	// The innocent peer must stay connected.
	otherConn := conn1
	if cc == conn1 {
		otherConn = conn2
	}
	if readUntilDisconnect(otherConn) {
		t.Fatalf("innocent peer should not be disconnected")
	}
}

// buildCells extracts cells at mask indices from the original tx's blobs
func buildCells(blobs []kzg4844.Blob, mask types.CustodyBitmap) []kzg4844.Cell {
	allCells, _ := kzg4844.ComputeCells(blobs)
	indices := mask.Indices()
	result := make([]kzg4844.Cell, 0, len(blobs)*len(indices))
	for b := 0; b < len(blobs); b++ {
		for _, idx := range indices {
			result = append(result, allCells[b*kzg4844.CellsPerBlob+int(idx)])
		}
	}
	return result
}

func encodeCells(cells [][]kzg4844.Cell) rlp.RawList[rlp.RawList[kzg4844.Cell]] {
	inner := make([]rlp.RawList[kzg4844.Cell], len(cells))
	for i, c := range cells {
		inner[i], _ = rlp.EncodeToRawList(c)
	}
	out, _ := rlp.EncodeToRawList(inner)
	return out
}
