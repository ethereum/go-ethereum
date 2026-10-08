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

package snap

import (
	"bytes"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

var errCommittedBatch = errors.New("injected error after durable batch write")

type committedErrorDB struct {
	ethdb.Database
	armed atomic.Bool
}

func (db *committedErrorDB) NewBatch() ethdb.Batch {
	return &committedErrorBatch{Batch: db.Database.NewBatch(), db: db}
}

func (db *committedErrorDB) NewBatchWithSize(size int) ethdb.Batch {
	return &committedErrorBatch{Batch: db.Database.NewBatchWithSize(size), db: db}
}

type committedErrorBatch struct {
	ethdb.Batch
	db      *committedErrorDB
	journal bool
}

func (b *committedErrorBatch) Put(key, value []byte) error {
	if bytes.Equal(key, []byte("SnapshotSyncStatus")) {
		b.journal = true
	}
	return b.Batch.Put(key, value)
}

func (b *committedErrorBatch) Write() error {
	if !b.journal || !b.db.armed.CompareAndSwap(true, false) {
		return b.Batch.Write()
	}
	if err := b.Batch.Write(); err != nil {
		return err
	}
	return errCommittedBatch
}

func TestCatchUpCommittedWriteErrorPreservesJournal(t *testing.T) {
	nodeScheme, sourceTrie, elems, addrs := makeAccountTrieWithAddresses(100, rawdb.PathScheme)
	rootA := sourceTrie.Hash()
	numA := uint64(100)
	targetAddr := addrs[0]
	targetHash := crypto.Keccak256Hash(targetAddr[:])

	db := &committedErrorDB{Database: rawdb.NewMemoryDatabase()}
	emptyHash := common.Hash{}
	zero := uint64(0)

	pivotA := &types.Header{
		Number: new(big.Int).SetUint64(numA), Root: rootA, Difficulty: common.Big0,
		BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
		BlobGasUsed: &zero, ExcessBlobGas: &zero,
		ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
	}
	rawdb.WriteHeader(db, pivotA)
	rawdb.WriteCanonicalHash(db, pivotA.Hash(), numA)

	wantBalance := uint256.NewInt(424242)
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, targetAddr, wantBalance)
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	var decoded bal.BlockAccessList
	if err := rlp.DecodeBytes(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	balHash := decoded.Hash()
	nextLeaves := withLeafBalance(t, elems, targetHash, wantBalance.Uint64())
	pivotB := &types.Header{
		ParentHash: pivotA.Hash(), Root: accountTrieRoot(nextLeaves),
		Number: new(big.Int).SetUint64(numA + 1), Difficulty: common.Big0,
		BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
		BlobGasUsed: &zero, ExcessBlobGas: &zero,
		ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
		BlockAccessListHash: &balHash,
	}
	rawdb.WriteHeader(db, pivotB)
	rawdb.WriteCanonicalHash(db, pivotB.Hash(), numA+1)
	bals := map[common.Hash]rlp.RawValue{pivotB.Hash(): bytes.Clone(buf.Bytes())}

	// Seed a complete path-scheme sync at A.
	{
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := newSyncerV2(db, nodeScheme)
		src := newTestPeerV2("seed", t, term)
		src.accountTrie = sourceTrie.Copy()
		src.accountValues = elems
		syncer.Register(src)
		src.remote = syncer
		if err := syncer.Sync(pivotA, cancel); err != nil {
			t.Fatalf("seed sync failed: %v", err)
		}
	}

	// Simulate the ambiguous storage-engine outcome: the catch-up batch,
	// including the next-pivot journal, is durable but Write returns an error.
	db.armed.Store(true)
	{
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := newSyncerV2(db, nodeScheme)
		src := newTestPeerV2("fault", t, term)
		src.accountTrie = sourceTrie.Copy()
		src.accountValues = elems
		src.accessLists = bals
		syncer.Register(src)
		src.remote = syncer
		if err := syncer.Sync(pivotB, cancel); !errors.Is(err, errCommittedBatch) {
			t.Fatalf("faulted sync error = %v, want %v", err, errCommittedBatch)
		}
	}

	// A new process must trust the atomic journal already on disk. Before the
	// fix, Sync's defer overwrote it with pivot A; path-scheme catch-up then
	// tried to reopen A's root after those path nodes had been replaced.
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	restarted := newSyncerV2(db, nodeScheme)
	src := newTestPeerV2("restart", t, term)
	src.accountTrie = sourceTrie.Copy()
	src.accountValues = elems
	src.accessLists = bals
	restarted.Register(src)
	src.remote = restarted
	if err := restarted.Sync(pivotB, cancel); err != nil {
		t.Fatalf("restart sync failed: %v", err)
	}
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.pivot == nil || loader.pivot.Hash() != pivotB.Hash() {
		t.Fatalf("persisted pivot after restart = %v, want %v", loader.pivot, pivotB.Hash())
	}
	data := rawdb.ReadAccountSnapshot(db, targetHash)
	account, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("decode target account: %v", err)
	}
	if account.Balance.Cmp(wantBalance) != 0 {
		t.Fatalf("balance after restart = %v, want %v", account.Balance, wantBalance)
	}
	verifyTrie(rawdb.PathScheme, db, pivotB.Root, t)
}
