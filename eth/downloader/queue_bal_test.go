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

package downloader

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestBlockAccessListStaleReservation covers the lifecycle behind the logging
// change in reserveHeaders: a reserved access list request is out of reach of
// pruneBALTasks, so it can outlive the delivery of its block and is only
// dropped on the next reservation. The test pins those state transitions; the
// level the drop is reported at is not observed here.
func TestBlockAccessListStaleReservation(t *testing.T) {
	// Create an access list along with its header commitment
	list := bal.BlockAccessList{{Address: common.Address{0x01}}}
	enc, err := rlp.EncodeToBytes(&list)
	if err != nil {
		t.Fatal(err)
	}
	balHash := crypto.Keccak256Hash(enc)

	// Assemble a chain of empty-body headers committing to the access list
	var (
		headers = make([]*types.Header, 10)
		hashes  = make([]common.Hash, 10)
		parent  common.Hash
	)
	for i := range headers {
		headers[i] = &types.Header{
			ParentHash:          parent,
			Number:              big.NewInt(int64(i + 1)),
			Difficulty:          big.NewInt(1),
			TxHash:              types.EmptyTxsHash,
			UncleHash:           types.EmptyUncleHash,
			ReceiptHash:         types.EmptyReceiptsHash,
			BlockAccessListHash: &balHash,
		}
		hashes[i] = headers[i].Hash()
		parent = hashes[i]
	}
	q := newQueue(16, 16)
	q.Prepare(1, FullSync)
	q.Schedule(headers, hashes, 1)

	// All the bodies are empty, so reserving them completes the fetch results
	peer := dummyPeer("peer-1")
	if req, _, _ := q.ReserveBodies(peer, 10); req != nil {
		t.Fatal("there should be no body fetch tasks remaining")
	}
	// Put an access list request in flight and deliver the blocks upstream
	// underneath it. Pruning cannot reach the reserved tasks.
	req, _, _ := q.ReserveBALs(peer, 2)
	if req == nil {
		t.Fatal("expected access list fetch task")
	}
	if got, exp := len(q.Results(false)), 10; got != exp {
		t.Fatalf("wrong result count, got %d, exp %d", got, exp)
	}
	if got, exp := q.PendingBALs(), 0; got != exp {
		t.Fatalf("wrong pending access list count after delivery, got %d, exp %d", got, exp)
	}
	// The peer turns out not to possess the lists, handing the tasks back
	if _, err := q.DeliverBALs(peer.id, []rlp.RawValue{rlp.EmptyString, rlp.EmptyString}, []common.Hash{{}, {}}); err != nil {
		t.Fatalf("failed to deliver access lists: %v", err)
	}
	if got, exp := q.PendingBALs(), 2; got != exp {
		t.Fatalf("wrong pending access list count after hand back, got %d, exp %d", got, exp)
	}
	// The next reservation must drop them without producing a request
	peer2 := dummyPeer("peer-2")
	if req, _, _ := q.ReserveBALs(peer2, 10); req != nil {
		t.Fatalf("expected no access list fetch task, got %d", len(req.Headers))
	}
	if got, exp := q.PendingBALs(), 0; got != exp {
		t.Errorf("wrong pending access list count after reservation, got %d, exp %d", got, exp)
	}
	if len(q.balTaskPool) != 0 {
		t.Errorf("access list task pool not drained, %d left", len(q.balTaskPool))
	}
}
