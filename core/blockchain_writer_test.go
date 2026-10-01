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

package core

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	writeTestKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	writeTestAddr   = crypto.PubkeyToAddress(writeTestKey.PublicKey)
)

// newWriteTestChain opens a chain on db and generates n blocks with a transfer each.
func newWriteTestChain(t *testing.T, n int, db ethdb.Database, config *BlockChainConfig) (*BlockChain, *Genesis, []*types.Block, ethdb.Database) {
	t.Helper()

	gspec := &Genesis{
		Config:  params.MergedTestChainConfig,
		Alloc:   withSystemContracts(types.GenesisAlloc{writeTestAddr: {Balance: big.NewInt(params.Ether)}}),
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	genDb, blocks, _ := GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), n, func(i int, gen *BlockGen) {
		gen.AddTx(writeTestTx(gspec.Config, gen, common.Address{0x01}))
	})
	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	// The generator database is handed back for building side blocks
	return chain, gspec, blocks, genDb
}

func writeTestTx(config *params.ChainConfig, gen *BlockGen, to common.Address) *types.Transaction {
	tx, err := types.SignTx(types.NewTransaction(gen.TxNonce(writeTestAddr), to, big.NewInt(1), params.TxGas, gen.header.BaseFee, nil), types.LatestSigner(config), writeTestKey)
	if err != nil {
		panic(err)
	}
	return tx
}

type writeStep struct {
	step  string
	block common.Hash
}

// writeProbe reports the steps of the chain writer and holds the ones asked for.
type writeProbe struct {
	steps chan writeStep

	lock  sync.Mutex
	holds map[writeStep]chan struct{}
}

func newWriteProbe(chain *BlockChain) *writeProbe {
	p := &writeProbe{
		steps: make(chan writeStep, 1024),
		holds: make(map[writeStep]chan struct{}),
	}
	chain.SetWriteHookForTesting(func(step string, block *types.Block) {
		s := writeStep{step: step}
		if block != nil {
			s.block = block.Hash()
		}
		p.lock.Lock()
		hold := p.holds[s]
		p.lock.Unlock()

		p.steps <- s
		if hold != nil {
			<-hold
		}
	})
	return p
}

// hold makes the writer wait before the step for the block until release.
func (p *writeProbe) hold(step string, block *types.Block) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.holds[writeStep{step, block.Hash()}] = make(chan struct{})
}

// release lets every held step go on.
func (p *writeProbe) release() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for key, hold := range p.holds {
		close(hold)
		delete(p.holds, key)
	}
}

// wait blocks until the writer reaches the step for the block, zero for no block.
func (p *writeProbe) wait(t *testing.T, step string, block common.Hash) {
	t.Helper()

	// The timeout only turns a hang into a failure
	timeout := time.After(time.Minute)
	for {
		select {
		case s := <-p.steps:
			if s.step == step && s.block == block {
				return
			}
		case <-timeout:
			t.Fatalf("chain writer never reached step %q", step)
		}
	}
}

// queueBlock hands the block to the chain writer and queues it as the head.
func queueBlock(t *testing.T, chain *BlockChain, block *types.Block) {
	t.Helper()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if !chain.QueueHead(block) {
		t.Fatal("head update not queued")
	}
}

// waitsForWrites runs fn and checks that it only returns once the held write is released.
func waitsForWrites(t *testing.T, probe *writeProbe, fn func() error) error {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- fn() }()

	// It drains the writer first, so it can't return while the write is held
	probe.wait(t, "drain", common.Hash{})
	select {
	case <-done:
		t.Fatal("returned before the queued writes landed")
	default:
	}
	probe.release()
	return <-done
}

// checkBlockReadable checks that the block and all its parts read back through the chain.
func checkBlockReadable(t *testing.T, chain *BlockChain, block *types.Block) {
	t.Helper()

	hash, number := block.Hash(), block.NumberU64()
	if got := chain.GetBlockByHash(hash); got == nil || got.Hash() != hash || chain.GetBlock(hash, number) == nil {
		t.Fatalf("block #%d missing", number)
	}
	if got := chain.GetHeaderByHash(hash); got == nil || got.Hash() != hash || chain.GetHeader(hash, number) == nil {
		t.Fatalf("header #%d missing", number)
	}
	if n := chain.GetBlockNumber(hash); n == nil || *n != number || !chain.HasHeader(hash, number) || !chain.HasBlock(hash, number) || !chain.HasFastBlock(hash, number) {
		t.Fatalf("block #%d reported missing", number)
	}
	if !chain.HasState(block.Root()) || !chain.HasBlockAndState(hash, number) {
		t.Fatalf("state of block #%d reported missing", number)
	}
	var body types.Body
	if got := chain.GetBody(hash); got == nil || types.DeriveSha(types.Transactions(got.Transactions), trie.NewStackTrie(nil)) != block.TxHash() {
		t.Fatalf("body of block #%d missing", number)
	}
	if err := rlp.DecodeBytes(chain.GetBodyRLP(hash), &body); err != nil || len(body.Transactions) != len(block.Transactions()) {
		t.Fatalf("body rlp of block #%d missing: %v", number, err)
	}
	receipts := chain.GetReceiptsByHash(hash)
	if len(receipts) != len(block.Transactions()) || types.DeriveSha(receipts, trie.NewStackTrie(nil)) != block.ReceiptHash() {
		t.Fatalf("receipts of block #%d don't match the header", number)
	}
	// Log filters turn missing logs into an error, so there's an entry per transaction
	logs := chain.GetLogs(hash, number)
	if logs == nil || len(logs) != len(receipts) || len(chain.GetRawReceipts(hash, number)) != len(receipts) || len(chain.GetReceiptsRLP(hash)) == 0 {
		t.Fatalf("raw receipts or logs of block #%d missing", number)
	}
	for i, receipt := range receipts {
		if receipt.TxHash != block.Transactions()[i].Hash() || receipt.BlockHash != hash || receipt.GasUsed == 0 || len(logs[i]) != len(receipt.Logs) {
			t.Fatalf("receipt %d of block #%d misses derived fields or logs", i, number)
		}
	}
}

// checkHeadReadable checks that the block reads back as the chain head, by number and through its transactions.
func checkHeadReadable(t *testing.T, chain *BlockChain, block *types.Block) {
	t.Helper()

	hash, number := block.Hash(), block.NumberU64()
	if chain.CurrentBlock().Hash() != hash || chain.CurrentHeader().Hash() != hash || chain.CurrentSnapBlock().Hash() != hash {
		t.Fatalf("block #%d not the current head", number)
	}
	header, got := chain.GetHeaderByNumber(number), chain.GetBlockByNumber(number)
	if chain.GetCanonicalHash(number) != hash || header == nil || header.Hash() != hash || got == nil || got.Hash() != hash {
		t.Fatalf("block #%d not canonical", number)
	}
	// Peers syncing down from the head get it along with its parent
	var head, parent types.Header
	headers := chain.GetHeadersFrom(number, 2)
	if len(headers) != 2 || rlp.DecodeBytes(headers[0], &head) != nil || head.Hash() != hash || rlp.DecodeBytes(headers[1], &parent) != nil || parent.Hash() != block.ParentHash() {
		t.Fatalf("headers down from #%d served wrong", number)
	}
	for i, tx := range block.Transactions() {
		lookup, found := chain.GetCanonicalTransaction(tx.Hash())
		if found == nil || lookup.BlockHash != hash || lookup.BlockIndex != number || lookup.Index != uint64(i) {
			t.Fatalf("tx %d of block #%d not found", i, number)
		}
		if receipt, err := chain.GetCanonicalReceipt(found, hash, number, uint64(i)); err != nil || receipt.TxHash != tx.Hash() || receipt.BlockHash != hash {
			t.Fatalf("receipt of tx %d in block #%d missing: %v", i, number, err)
		}
	}
}

// Tests that a queued block and head read back from memory while each step of
// their write is held, and from disk once it lands. The head markers reach disk
// after the block data, and the head event goes out after the markers.
func TestDeferredWriteServesQueuedBlock(t *testing.T) {
	for _, step := range []string{"state", "persist", "head"} {
		t.Run(step, func(t *testing.T) { testDeferredWriteServesQueuedBlock(t, step) })
	}
}

func testDeferredWriteServesQueuedBlock(t *testing.T, step string) {
	db := rawdb.NewMemoryDatabase()
	chain, _, blocks, _ := newWriteTestChain(t, 3, db, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	block, hash, number := blocks[2], blocks[2].Hash(), blocks[2].NumberU64()
	probe := newWriteProbe(chain)
	probe.hold(step, block)
	defer probe.release()

	// Sending blocks until every subscriber took the event. The first channel
	// tells the event fired, the second keeps the writer in the send while the
	// disk is checked.
	fired, held := make(chan ChainHeadEvent), make(chan ChainHeadEvent)
	defer chain.SubscribeChainHeadEvent(fired).Unsubscribe()
	defer chain.SubscribeChainHeadEvent(held).Unsubscribe()

	queueBlock(t, chain, block)
	probe.wait(t, step, hash)

	// The head markers aren't on disk yet. The block data is only there once
	// the head step runs, so the markers never point at a missing block.
	if rawdb.ReadHeadBlockHash(db) != blocks[1].Hash() || rawdb.ReadCanonicalHash(db, number) != (common.Hash{}) {
		t.Fatal("head markers on disk while the write is held")
	}
	if (rawdb.ReadBlock(db, hash, number) != nil) != (step == "head") {
		t.Fatal("block data on disk at the wrong step")
	}
	checkBlockReadable(t, chain, block)
	checkHeadReadable(t, chain, block)

	// The event goes out once the markers and lookups of the head are on disk
	probe.release()
	select {
	case <-fired:
	case <-time.After(time.Minute):
		t.Fatal("no head event")
	}
	if rawdb.ReadHeadBlockHash(db) != hash || rawdb.ReadTxLookupEntry(db, block.Transactions()[0].Hash()) == nil {
		t.Fatal("head markers missing at the head event")
	}
	// Let the writer finish the send
	<-held

	// Once the write lands the same reads come from disk
	chain.WaitWrites()
	chain.writer.lock.RLock()
	queued := len(chain.writer.jobs) + len(chain.writer.heads) + len(chain.writer.headTxs)
	chain.writer.lock.RUnlock()
	if queued != 0 {
		t.Fatal("block still served from memory after the write landed")
	}
	checkBlockReadable(t, chain, block)
	checkHeadReadable(t, chain, block)
}

// Tests that a sibling of the queued head is written synchronously.
func TestDeferredWriteSiblingHead(t *testing.T) {
	chain, gspec, blocks, genDb := newWriteTestChain(t, 3, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	side, _ := GenerateChain(gspec.Config, blocks[1], beacon.New(ethash.NewFaker()), genDb, 1, func(i int, gen *BlockGen) {
		gen.AddTx(writeTestTx(gspec.Config, gen, common.Address{0x02}))
	})
	sibling := side[0]

	probe := newWriteProbe(chain)
	probe.hold("persist", blocks[2])
	defer probe.release()

	queueBlock(t, chain, blocks[2])

	// The sibling doesn't build on the queued block, so its write waits for the
	// queued writes, which are held, and then runs synchronously
	if err := waitsForWrites(t, probe, func() error {
		_, err := chain.InsertBlockWithoutSetHead(context.Background(), sibling, false)
		return err
	}); err != nil {
		t.Fatalf("failed to insert sibling: %v", err)
	}
}

// Tests that a rewind waits for the queued writes before it touches the chain.
func TestDeferredWriteRewindDrains(t *testing.T) {
	chain, _, blocks, _ := newWriteTestChain(t, 4, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:3]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	block := blocks[3]
	probe := newWriteProbe(chain)
	probe.hold("persist", block)
	defer probe.release()

	queueBlock(t, chain, block)
	if err := waitsForWrites(t, probe, func() error { return chain.SetHead(2) }); err != nil {
		t.Fatalf("failed to rewind: %v", err)
	}
	// The queued head landed first and got rewound, so it didn't come back
	// after, and the writer forgot the state it wrote last
	if chain.CurrentBlock().Number.Uint64() != 2 || chain.GetBlockByHash(block.Hash()) != nil || chain.writer.hasState(block.Root()) {
		t.Fatal("queued head outlived the rewind")
	}
}

// Tests that stopping the chain lets the queued writes land before the journal.
func TestDeferredWriteStopDrains(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		config = DefaultConfig().WithStateScheme(rawdb.PathScheme)
	)
	chain, gspec, blocks, _ := newWriteTestChain(t, 3, db, config)

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	block := blocks[2]
	probe := newWriteProbe(chain)
	probe.hold("state", block)
	defer probe.release()

	queueBlock(t, chain, block)
	waitsForWrites(t, probe, func() error {
		chain.Stop()
		return nil
	})

	// The head and its state made it into the journal
	reopened, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("failed to reopen chain: %v", err)
	}
	defer reopened.Stop()

	if head := reopened.CurrentBlock(); head.Hash() != block.Hash() || !reopened.HasState(block.Root()) {
		t.Fatalf("reopened at #%d, want #%d with its state", head.Number, block.NumberU64())
	}
}
