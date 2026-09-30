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
	"github.com/ethereum/go-ethereum/core/tracing"
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

// writeProbe records the steps of the chain writer and holds the ones asked for.
type writeProbe struct {
	steps chan writeStep

	lock  sync.Mutex
	log   []writeStep
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
		p.log = append(p.log, s)
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

func (p *writeProbe) release(step string, block *types.Block) {
	p.lock.Lock()
	defer p.lock.Unlock()

	key := writeStep{step, block.Hash()}
	if hold := p.holds[key]; hold != nil {
		close(hold)
		delete(p.holds, key)
	}
}

func (p *writeProbe) releaseAll() {
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

// count returns how many times the step ran for the block.
func (p *writeProbe) count(step string, block *types.Block) int {
	p.lock.Lock()
	defer p.lock.Unlock()

	var n int
	for _, s := range p.log {
		if s == (writeStep{step, block.Hash()}) {
			n++
		}
	}
	return n
}

// index returns the position of the first time the step ran for the block.
func (p *writeProbe) index(step string, block *types.Block) int {
	p.lock.Lock()
	defer p.lock.Unlock()

	for i, s := range p.log {
		if s == (writeStep{step, block.Hash()}) {
			return i
		}
	}
	return -1
}

// checkBlockReadable checks that the block and all its parts read back through the chain.
func checkBlockReadable(t *testing.T, chain *BlockChain, block *types.Block) {
	t.Helper()

	hash, number := block.Hash(), block.NumberU64()
	if got := chain.GetBlockByHash(hash); got == nil || got.Hash() != hash {
		t.Fatalf("block #%d missing by hash", number)
	}
	if got := chain.GetBlock(hash, number); got == nil || got.Hash() != hash {
		t.Fatalf("block #%d missing", number)
	}
	if got := chain.GetHeaderByHash(hash); got == nil || got.Hash() != hash {
		t.Fatalf("header #%d missing by hash", number)
	}
	if got := chain.GetHeader(hash, number); got == nil || got.Hash() != hash {
		t.Fatalf("header #%d missing", number)
	}
	if n := chain.GetBlockNumber(hash); n == nil || *n != number {
		t.Fatalf("number of block #%d missing", number)
	}
	if !chain.HasHeader(hash, number) || !chain.HasBlock(hash, number) || !chain.HasFastBlock(hash, number) {
		t.Fatalf("block #%d reported missing", number)
	}
	if !chain.HasState(block.Root()) || !chain.HasBlockAndState(hash, number) {
		t.Fatalf("state of block #%d reported missing", number)
	}
	if body := chain.GetBody(hash); body == nil || types.DeriveSha(types.Transactions(body.Transactions), trie.NewStackTrie(nil)) != block.TxHash() {
		t.Fatalf("body of block #%d missing", number)
	}
	var body types.Body
	if err := rlp.DecodeBytes(chain.GetBodyRLP(hash), &body); err != nil || len(body.Transactions) != len(block.Transactions()) {
		t.Fatalf("body rlp of block #%d missing: %v", number, err)
	}
	receipts := chain.GetReceiptsByHash(hash)
	if len(receipts) != len(block.Transactions()) {
		t.Fatalf("receipts of block #%d: have %d, want %d", number, len(receipts), len(block.Transactions()))
	}
	for i, receipt := range receipts {
		if receipt.TxHash != block.Transactions()[i].Hash() || receipt.BlockHash != hash || receipt.GasUsed == 0 {
			t.Fatalf("receipt %d of block #%d misses its derived fields", i, number)
		}
	}
	if types.DeriveSha(receipts, trie.NewStackTrie(nil)) != block.ReceiptHash() {
		t.Fatalf("receipts of block #%d don't match the header", number)
	}
	if raw := chain.GetRawReceipts(hash, number); len(raw) != len(block.Transactions()) {
		t.Fatalf("raw receipts of block #%d missing", number)
	}
	// Log filters turn missing logs into an error, so there's one entry per transaction
	logs := chain.GetLogs(hash, number)
	if logs == nil || len(logs) != len(receipts) {
		t.Fatalf("logs of block #%d missing", number)
	}
	for i, receipt := range receipts {
		if len(logs[i]) != len(receipt.Logs) {
			t.Fatalf("logs of tx %d in block #%d don't match its receipt", i, number)
		}
	}
	if enc := chain.GetReceiptsRLP(hash); len(enc) == 0 {
		t.Fatalf("receipts rlp of block #%d missing", number)
	}
}

// Tests that a queued block reads as present while either phase of its write is held.
func TestDeferredWriteServesPendingBlock(t *testing.T) {
	t.Run("state", func(t *testing.T) { testDeferredWriteServesPendingBlock(t, "state") })
	t.Run("persist", func(t *testing.T) { testDeferredWriteServesPendingBlock(t, "persist") })
}

func testDeferredWriteServesPendingBlock(t *testing.T, step string) {
	db := rawdb.NewMemoryDatabase()
	chain, _, blocks, _ := newWriteTestChain(t, 3, db, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	block := blocks[2]
	probe := newWriteProbe(chain)
	probe.hold(step, block)
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	probe.wait(t, step, block.Hash())

	// Nothing of the block is on disk, it's all served from the queued write
	if rawdb.HasHeader(db, block.Hash(), block.NumberU64()) || rawdb.HasBody(db, block.Hash(), block.NumberU64()) || rawdb.HasReceipts(db, block.Hash(), block.NumberU64()) {
		t.Fatal("block data on disk while its write is held")
	}
	checkBlockReadable(t, chain, block)

	// Once the write lands the same reads come from disk
	probe.release(step, block)
	chain.WaitWrites()
	if rawdb.ReadBlock(db, block.Hash(), block.NumberU64()) == nil || rawdb.ReadRawReceipts(db, block.Hash(), block.NumberU64()) == nil {
		t.Fatal("block data not on disk after the write landed")
	}
	checkBlockReadable(t, chain, block)
}

// Tests that the head markers of a queued head only reach disk after its block data.
func TestDeferredWriteHeadAfterBlockData(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		config = DefaultConfig().WithStateScheme(rawdb.PathScheme).WithNoAsyncFlush(true)
	)
	chain, gspec, blocks, _ := newWriteTestChain(t, 4, db, config)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:3]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	// Persist the head state, so the reopened chain has no reason to rewind
	if err := chain.TrieDB().Commit(blocks[2].Root(), false); err != nil {
		t.Fatalf("failed to commit state: %v", err)
	}
	block := blocks[3]
	probe := newWriteProbe(chain)
	probe.hold("head", block)
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if !chain.QueueHead(block) {
		t.Fatal("head update not queued")
	}
	// The head update waits behind the block write, so the block data is on
	// disk now and the head markers are not.
	probe.wait(t, "head", block.Hash())
	if rawdb.ReadBlock(db, block.Hash(), block.NumberU64()) == nil {
		t.Fatal("block data missing before the head update")
	}
	if head := rawdb.ReadHeadBlockHash(db); head != blocks[2].Hash() {
		t.Fatalf("head marker written before the head update: %x", head)
	}
	// Reopen what's on disk at this point, as a restart after a crash would.
	// A head marker ahead of its block would make it reset the chain.
	crashed := rawdb.NewMemoryDatabase()
	it := db.NewIterator(nil, nil)
	for it.Next() {
		crashed.Put(common.CopyBytes(it.Key()), common.CopyBytes(it.Value()))
	}
	it.Release()

	reopened, err := NewBlockChain(crashed, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("failed to reopen chain: %v", err)
	}
	defer reopened.Stop()

	if head := reopened.CurrentBlock(); head.Hash() != blocks[2].Hash() {
		t.Fatalf("reopened at #%d, want #%d", head.Number, blocks[2].NumberU64())
	}
	for _, b := range blocks {
		if reopened.GetBlockByHash(b.Hash()) == nil {
			t.Fatalf("block #%d lost across the crash", b.NumberU64())
		}
	}
	// The original chain carries on with the head update
	probe.release("head", block)
	chain.WaitWrites()
	if head := chain.CurrentBlock(); head.Hash() != block.Hash() {
		t.Fatalf("head #%d, want #%d", head.Number, block.NumberU64())
	}
}

// Tests that the head event is sent once the lookups and receipts of the head are on disk.
func TestDeferredWriteHeadEvent(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	chain, _, blocks, _ := newWriteTestChain(t, 2, db, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	// Send blocks until every subscriber took the event. The first channel
	// tells the event fired, the second keeps the writer in the send while
	// the disk is checked.
	var (
		fired = make(chan ChainHeadEvent)
		held  = make(chan ChainHeadEvent)
	)
	defer chain.SubscribeChainHeadEvent(fired).Unsubscribe()
	defer chain.SubscribeChainHeadEvent(held).Unsubscribe()

	block := blocks[1]
	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if !chain.QueueHead(block) {
		t.Fatal("head update not queued")
	}
	var ev ChainHeadEvent
	select {
	case ev = <-fired:
	case <-time.After(time.Minute):
		t.Fatal("no head event")
	}
	if ev.Header.Hash() != block.Hash() {
		t.Fatalf("head event for #%d, want #%d", ev.Header.Number, block.NumberU64())
	}
	for _, tx := range block.Transactions() {
		if rawdb.ReadTxLookupEntry(db, tx.Hash()) == nil {
			t.Fatalf("lookup of tx %x missing at the head event", tx.Hash())
		}
		if _, found := chain.GetCanonicalTransaction(tx.Hash()); found == nil {
			t.Fatalf("tx %x not canonical at the head event", tx.Hash())
		}
	}
	if rawdb.ReadRawReceipts(db, block.Hash(), block.NumberU64()) == nil {
		t.Fatal("receipts missing at the head event")
	}
	if rawdb.ReadHeadBlockHash(db) != block.Hash() || chain.CurrentBlock().Hash() != block.Hash() {
		t.Fatal("head not updated at the head event")
	}
	// Let the writer finish the send
	<-held
}

// Tests that repeating the head update of a queued head doesn't queue it again.
func TestDeferredWriteRepeatedHead(t *testing.T) {
	chain, _, blocks, _ := newWriteTestChain(t, 2, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	block := blocks[1]
	probe := newWriteProbe(chain)
	probe.hold("head", block)
	defer probe.releaseAll()

	events := make(chan ChainHeadEvent, 10)
	defer chain.SubscribeChainHeadEvent(events).Unsubscribe()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	// Repeat the update while the first one is still held
	for i := 0; i < 3; i++ {
		if !chain.QueueHead(block) {
			t.Fatalf("head update %d not accepted", i)
		}
	}
	probe.release("head", block)
	chain.WaitWrites()

	if n := probe.count("head", block); n != 1 {
		t.Fatalf("head update ran %d times", n)
	}
	if len(events) != 1 {
		t.Fatalf("%d head events sent", len(events))
	}
}

// Tests that a sibling of the queued head can't be queued and reorgs the regular way.
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

	// Hold the queued head back, so the current head stays the parent of both
	probe := newWriteProbe(chain)
	probe.hold("persist", blocks[2])
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), blocks[2], false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if !chain.QueueHead(blocks[2]) {
		t.Fatal("head update not queued")
	}
	// The sibling doesn't build on the queued block, so its write waits for the
	// queued writes, which are held, and then runs synchronously
	inserted := make(chan error, 1)
	go func() {
		_, err := chain.InsertBlockWithoutSetHead(context.Background(), sibling, false)
		inserted <- err
	}()
	probe.wait(t, "drain", common.Hash{})
	probe.release("persist", blocks[2])
	if err := <-inserted; err != nil {
		t.Fatalf("failed to insert sibling: %v", err)
	}
	// It extends the current head, but was never handed over, so it can't be queued
	if chain.QueueHead(sibling) {
		t.Fatal("sibling head queued without a reorg")
	}
	// The synchronous update reorgs away from the queued head
	if _, err := chain.SetCanonical(sibling); err != nil {
		t.Fatalf("failed to set sibling head: %v", err)
	}
	if head := chain.CurrentBlock(); head.Hash() != sibling.Hash() {
		t.Fatalf("head %x, want sibling %x", head.Hash(), sibling.Hash())
	}
	if hash := chain.GetCanonicalHash(sibling.NumberU64()); hash != sibling.Hash() {
		t.Fatalf("canonical hash %x, want sibling %x", hash, sibling.Hash())
	}
	for _, tx := range blocks[2].Transactions() {
		if lookup, _ := chain.GetCanonicalTransaction(tx.Hash()); lookup != nil {
			t.Fatalf("tx %x of the reorged block is still canonical", tx.Hash())
		}
	}
}

// Tests that the second phases run in order, each after the first phase of its block.
func TestDeferredWritePersistOrder(t *testing.T) {
	const synced = 130
	chain, _, blocks, _ := newWriteTestChain(t, synced+5, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	// Fill the layer tree, the disk layer ends up at block 2 and the oldest
	// diff layer is block 3.
	if _, err := chain.InsertChain(blocks[:synced]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	queued := blocks[synced:]
	probe := newWriteProbe(chain)
	probe.hold("persist", queued[0])
	defer probe.releaseAll()

	for _, block := range queued {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			t.Fatalf("failed to insert block #%d: %v", block.NumberU64(), err)
		}
	}
	// The flattening doesn't wait for the held block data. Each one capped
	// below its own block already, so the disk layer is block 7 now.
	chain.writer.waitFlattened(queued[len(queued)-1].Root())
	for _, block := range blocks[2:6] {
		if _, err := chain.TrieDB().NodeReader(block.Root()); err == nil {
			t.Fatalf("layer of block #%d not flattened", block.NumberU64())
		}
	}
	if _, err := chain.TrieDB().NodeReader(blocks[6].Root()); err != nil {
		t.Fatalf("disk layer of block #%d missing: %v", blocks[6].NumberU64(), err)
	}
	probe.release("persist", queued[0])
	chain.WaitWrites()

	// The recorded steps show the order the phases ran in
	for i, block := range queued {
		persist := probe.index("persist", block)
		if state := probe.index("state", block); state < 0 || persist < state {
			t.Fatalf("block #%d persisted before its state commit", block.NumberU64())
		}
		if i > 0 && persist < probe.index("persist", queued[i-1]) {
			t.Fatalf("block #%d persisted before its parent", block.NumberU64())
		}
	}
}

// Tests that a block waits for the flattening below its parent before it runs,
// and reports that wait as commit time.
func TestDeferredWriteCommitWait(t *testing.T) {
	chain, _, blocks, _ := newWriteTestChain(t, 3, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	parent, block := blocks[1], blocks[2]
	probe := newWriteProbe(chain)
	probe.hold("flatten", parent)
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), parent, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	// The commit goes on next to the held flattening, so the parent state gets readable
	probe.wait(t, "flatten", parent.Hash())
	select {
	case <-chain.writer.job(parent.Hash()).committed:
	case <-time.After(time.Minute):
		t.Fatal("state commit waited for the flattening")
	}

	type result struct {
		res *blockProcessingResult
		err error
	}
	done := make(chan result, 1)
	go func() {
		res, err := chain.ProcessBlock(context.Background(), parent.Root(), block, ExecuteConfig{})
		done <- result{res, err}
	}()
	probe.wait(t, "wait", parent.Hash())
	select {
	case <-done:
		t.Fatal("block processed before the flattening below its parent")
	default:
	}
	probe.release("flatten", parent)
	r := <-done
	if r.err != nil {
		t.Fatalf("failed to process block: %v", r.err)
	}
	if r.res.stats.CommitWait <= 0 {
		t.Fatal("wait for the parent not reported as commit time")
	}
	// With the parent written there's nothing left to wait for
	chain.WaitWrites()
	res, err := chain.ProcessBlock(context.Background(), parent.Root(), block, ExecuteConfig{})
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}
	if res.stats.CommitWait != 0 {
		t.Fatalf("commit wait %v with the parent written", res.stats.CommitWait)
	}
}

// Tests that the flattening that makes room for a block runs while its state commit is held.
func TestDeferredWriteFlattenNextToCommit(t *testing.T) {
	const synced = 130
	chain, _, blocks, _ := newWriteTestChain(t, synced+1, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	// Fill the layer tree, the disk layer ends up at block 2
	if _, err := chain.InsertChain(blocks[:synced]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	block := blocks[synced]
	probe := newWriteProbe(chain)
	probe.hold("state", block)
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	// Block 3 gets flattened to make room, while the commit of the block is held
	probe.wait(t, "flatten", block.Hash())
	for timeout := time.After(time.Minute); ; {
		if _, err := chain.TrieDB().NodeReader(blocks[1].Root()); err != nil {
			break
		}
		select {
		case <-timeout:
			t.Fatal("flattening waited for the state commit")
		case <-time.After(time.Millisecond):
		}
	}
	if _, err := chain.TrieDB().NodeReader(block.Root()); err == nil {
		t.Fatal("layer of the block added while its commit is held")
	}
	probe.release("state", block)
	chain.WaitWrites()

	// The new layer sits on top of the usual number of diff layers
	if _, err := chain.TrieDB().NodeReader(block.Root()); err != nil {
		t.Fatalf("layer of the block missing: %v", err)
	}
	if _, err := chain.TrieDB().NodeReader(blocks[2].Root()); err != nil {
		t.Fatalf("disk layer of block #%d missing: %v", blocks[2].NumberU64(), err)
	}
}

// Tests that the state of the last block written counts as present without the
// database, until a synchronous update.
func TestDeferredWriteLastState(t *testing.T) {
	chain, _, blocks, _ := newWriteTestChain(t, 2, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), blocks[1], false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	chain.WaitWrites()

	// The writer vouches for the block it wrote last, and only for that one
	if !chain.writer.hasState(blocks[1].Root()) {
		t.Fatal("state of the last block written not present")
	}
	if chain.writer.hasState(blocks[0].Root()) {
		t.Fatal("state of a block written synchronously present in the writer")
	}
	// A rewind may drop that state, so the writer forgets it first
	if err := chain.SetHead(blocks[0].NumberU64()); err != nil {
		t.Fatalf("failed to rewind: %v", err)
	}
	if chain.writer.hasState(blocks[1].Root()) {
		t.Fatal("state of the last block written still present after a rewind")
	}
}

// Tests that a block on another branch than the queued ones is written synchronously,
// so its flattening can't drop a layer the writer still counts as present.
func TestDeferredWriteForkSynchronous(t *testing.T) {
	const synced = 130
	chain, gspec, blocks, genDb := newWriteTestChain(t, synced+1, rawdb.NewMemoryDatabase(), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	defer chain.Stop()

	// Fill the layer tree, the disk layer ends up at block 2
	if _, err := chain.InsertChain(blocks[:synced]); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	// Queue a side block right on top of the disk layer and hold its block data
	side, _ := GenerateChain(gspec.Config, blocks[1], beacon.New(ethash.NewFaker()), genDb, 1, func(i int, gen *BlockGen) {
		gen.AddTx(writeTestTx(gspec.Config, gen, common.Address{0x02}))
	})
	probe := newWriteProbe(chain)
	probe.hold("persist", side[0])
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), side[0], false); err != nil {
		t.Fatalf("failed to insert side block: %v", err)
	}
	// The next canonical block doesn't build on it, so its write waits for the
	// queued one and then runs synchronously
	block := blocks[synced]
	inserted := make(chan error, 1)
	go func() {
		_, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false)
		inserted <- err
	}()
	probe.wait(t, "drain", common.Hash{})
	probe.release("persist", side[0])
	if err := <-inserted; err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if probe.count("state", block) != 0 {
		t.Fatal("block on another branch queued")
	}
	// Its flattening dropped the side layer, and nothing counts it as present anymore
	if _, err := chain.TrieDB().NodeReader(side[0].Root()); err == nil {
		t.Fatal("side layer not dropped")
	}
	if chain.HasState(side[0].Root()) {
		t.Fatal("dropped side state still present")
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
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if !chain.QueueHead(block) {
		t.Fatal("head update not queued")
	}
	done := make(chan error, 1)
	go func() { done <- chain.SetHead(2) }()

	// The rewind waits for the held write, so it can't have returned yet
	probe.wait(t, "drain", common.Hash{})
	select {
	case <-done:
		t.Fatal("rewind ran before the queued writes landed")
	default:
	}
	probe.release("persist", block)
	if err := <-done; err != nil {
		t.Fatalf("failed to rewind: %v", err)
	}
	// The queued head landed first and got rewound, it didn't come back after
	if head := chain.CurrentBlock(); head.Number.Uint64() != 2 {
		t.Fatalf("head #%d after rewind, want #2", head.Number)
	}
	if chain.GetBlockByHash(block.Hash()) != nil || chain.GetBlockByHash(blocks[2].Hash()) != nil {
		t.Fatal("rewound blocks still present")
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
	defer probe.releaseAll()

	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	if !chain.QueueHead(block) {
		t.Fatal("head update not queued")
	}
	stopped := make(chan struct{})
	go func() {
		chain.Stop()
		close(stopped)
	}()
	// Stop waits for the held write, so it can't have returned yet
	probe.wait(t, "drain", common.Hash{})
	select {
	case <-stopped:
		t.Fatal("chain stopped before the queued writes landed")
	default:
	}
	probe.release("state", block)
	<-stopped

	// The head and its state made it into the journal
	reopened, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("failed to reopen chain: %v", err)
	}
	defer reopened.Stop()

	if head := reopened.CurrentBlock(); head.Hash() != block.Hash() {
		t.Fatalf("reopened at #%d, want #%d", head.Number, block.NumberU64())
	}
	if !reopened.HasState(block.Root()) {
		t.Fatal("head state lost across the restart")
	}
}

// Tests that imports needing the state commit before they return still write synchronously.
func TestDeferredWriteSyncPaths(t *testing.T) {
	stateHook := &tracing.Hooks{OnStateUpdate: func(*tracing.StateUpdate) {}}
	for _, tc := range []struct {
		name    string
		config  *BlockChainConfig
		witness bool
	}{
		{"hash", DefaultConfig().WithStateScheme(rawdb.HashScheme), false},
		{"statehook", func() *BlockChainConfig {
			config := DefaultConfig().WithStateScheme(rawdb.PathScheme)
			config.VmConfig.Tracer = stateHook
			return config
		}(), false},
		{"witness", DefaultConfig().WithStateScheme(rawdb.PathScheme), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			chain, _, blocks, _ := newWriteTestChain(t, 2, db, tc.config)
			defer chain.Stop()
			probe := newWriteProbe(chain)

			if _, err := chain.InsertChain(blocks[:1]); err != nil {
				t.Fatalf("failed to insert chain: %v", err)
			}
			block := blocks[1]
			if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, tc.witness); err != nil {
				t.Fatalf("failed to insert block: %v", err)
			}
			if rawdb.ReadBlock(db, block.Hash(), block.NumberU64()) == nil {
				t.Fatal("block not on disk when the import returned")
			}
			// The writer never saw a step
			probe.lock.Lock()
			defer probe.lock.Unlock()
			if len(probe.log) != 0 {
				t.Fatalf("chain writer used: %v", probe.log)
			}
		})
	}
}
