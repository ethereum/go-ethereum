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

package catalyst

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

// migrationTestGenesis schedules the fork at t=48: the fourth twelve-second
// block.
func migrationTestGenesis() *core.Genesis {
	genesis := pbtGenesis()
	config := *genesis.Config
	forkTime := uint64(48)
	config.BinaryTrieTime = &forkTime
	genesis.Config = &config
	return genesis
}

// buildBlock seals one block on the given parent through the engine API and
// makes it the head.
func buildBlock(t *testing.T, api *ConsensusAPI, parent *types.Header, slot uint64, random common.Hash) *types.Header {
	t.Helper()
	chain := api.eth.BlockChain()
	// Building on a merkle parent needs the shadow caught up to it; the
	// consensus layer's retries hide this in production, the test waits.
	if !chain.Config().IsBinaryTrie(parent.Number, parent.Time) {
		awaitShadowReady(t, chain, parent)
	}
	targetGasLimit := parent.GasLimit
	attrs := &engine.PayloadAttributes{
		Timestamp:             parent.Time + 12,
		Random:                random,
		SuggestedFeeRecipient: common.Address{},
		Withdrawals:           []*types.Withdrawal{},
		BeaconRoot:            &common.Hash{},
		SlotNumber:            &slot,
		TargetGasLimit:        &targetGasLimit,
	}
	fcState := engine.ForkchoiceStateV1{HeadBlockHash: parent.Hash()}
	resp, err := api.ForkchoiceUpdatedV4(context.Background(), fcState, attrs, nil)
	if err != nil {
		t.Fatalf("slot %d: forkchoice update failed: %v", slot, err)
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("slot %d: forkchoice update not valid: %v", slot, resp.PayloadStatus.Status)
	}
	payload, err := api.getPayload(*resp.PayloadID, true, nil, nil)
	if err != nil {
		t.Fatalf("slot %d: payload retrieval failed: %v", slot, err)
	}
	execData := payload.ExecutionPayload
	status, err := api.NewPayloadV5(context.Background(), *execData, []common.Hash{}, &common.Hash{}, []hexutil.Bytes{})
	if err != nil {
		t.Fatalf("slot %d: payload import failed: %v", slot, err)
	}
	if status.Status != engine.VALID {
		t.Fatalf("slot %d: imported payload not valid: %v (%s)", slot, status.Status, derefErr(status.ValidationError))
	}
	fcState = engine.ForkchoiceStateV1{HeadBlockHash: execData.BlockHash}
	if _, err := api.ForkchoiceUpdatedV4(context.Background(), fcState, nil, nil); err != nil {
		t.Fatalf("slot %d: setting head failed: %v", slot, err)
	}
	head := chain.CurrentBlock()
	if head.Hash() != execData.BlockHash {
		t.Fatalf("slot %d: head is %x, expected %x", slot, head.Hash(), execData.BlockHash)
	}
	return head
}

// waitFor polls cond until it holds or the timeout fails the test.
func waitFor(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	for start := time.Now(); !cond(); time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > timeout {
			t.Fatal(msg)
		}
	}
}

// awaitShadowReady waits for the follower to record the given block.
func awaitShadowReady(t *testing.T, chain *core.BlockChain, header *types.Header) {
	t.Helper()
	for start := time.Now(); !chain.ShadowReady(header.Hash(), header.Number.Uint64()); {
		if time.Since(start) > 10*time.Second {
			t.Fatalf("shadow never reached block %d (%+v)", header.Number, chain.MigrationProgress())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestFullMigrationLifecycle is the acceptance run: a merkle-born node
// replays the shadow from genesis, swaps at the fork, keeps the old tree live
// through the window - across a reboot - and retires it at finality.
func TestFullMigrationLifecycle(t *testing.T) {
	genesis := migrationTestGenesis()
	datadir := t.TempDir()
	n, ethservice := startEthServiceAt(t, datadir, genesis, nil)
	closed := false
	defer func() {
		if !closed {
			n.Close()
		}
	}()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	if chain.TrieDB().IsPBT() {
		t.Fatal("migrating node opened on the binary tree")
	}
	var (
		signer    = types.LatestSigner(genesis.Config)
		recipient = common.Address{0xba, 0x17}
		pay       = func(pool *txpool.TxPool, nonce uint64) {
			pool.Add([]*types.Transaction{types.MustSignNewTx(testKey, signer, &types.LegacyTx{
				Nonce: nonce, Gas: 1_000_000, GasPrice: big.NewInt(2 * params.InitialBaseFee), To: &recipient, Value: big.NewInt(1000),
			})}, true)
		}
	)
	// Pre-fork only the binary direction runs; block 1 carries a transfer.
	pay(ethservice.TxPool(), 0)
	parent := chain.CurrentBlock()
	for i := 0; i < 2; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	awaitShadowReady(t, chain, chain.CurrentBlock())
	if p := chain.MigrationProgress(); p.Phase != "running" || p.Binary == nil || p.Merkle != nil {
		t.Fatalf("pre-fork progress %+v, want running with only the binary direction", p)
	}
	for i := 2; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	// The boundary parent's record fed the swap; the window records merkle
	// roots for the binary blocks, proven against a reverse conversion.
	if !chain.ShadowReady(chain.GetHeaderByNumber(3).Hash(), 3) {
		t.Fatal("the boundary parent has no recorded shadow root")
	}
	for _, number := range []uint64{4, 5} {
		header := chain.GetHeaderByNumber(number)
		awaitShadowReady(t, chain, header)
		got, _ := rawdb.ReadShadowStateRoot(ethservice.ChainDb(), header.Hash(), number)
		if want := convertCanonicalMerkle(t, chain, ethservice.ChainDb(), genesis, header); got != want {
			t.Fatalf("block %d window root %x, reverse conversion says %x", number, got, want)
		}
	}
	if rawdb.ReadPBTMigrationDone(ethservice.ChainDb()) {
		t.Fatal("window closed without finality")
	}
	// Both sides of the boundary stay readable, transfer included.
	for _, number := range []uint64{2, 5} {
		header := chain.GetHeaderByNumber(number)
		statedb, err := chain.StateAt(header)
		if err != nil {
			t.Fatalf("state at block %d: %v", number, err)
		}
		if got := statedb.GetBalance(recipient).Uint64(); got != 1000 {
			t.Fatalf("recipient balance at block %d = %d, want 1000", number, got)
		}
	}
	if p := chain.MigrationProgress(); p.Binary == nil || p.Binary.Phase != "parked" ||
		p.Merkle == nil || (p.Merkle.Phase != "following" && p.Merkle.Phase != "synced") {
		t.Fatalf("post-fork progress %+v, want binary parked and the window working", p)
	}
	// Reboot mid-window; the deleted lists make any re-replay a loud stall.
	head := chain.GetHeaderByNumber(5)
	want, _ := rawdb.ReadShadowStateRoot(ethservice.ChainDb(), head.Hash(), 5)
	for _, number := range []uint64{4, 5} {
		rawdb.DeleteAccessList(ethservice.ChainDb(), chain.GetHeaderByNumber(number).Hash(), number)
	}
	n.Close()
	closed = true

	n2, eth2 := startEthServiceAt(t, datadir, genesis, nil)
	closed2 := false
	defer func() {
		if !closed2 {
			n2.Close()
		}
	}()
	api2 := NewConsensusAPI(eth2)
	chain2 := eth2.BlockChain()
	if got := chain2.CurrentBlock(); got.Number.Uint64() != 5 {
		t.Fatalf("rebooted head %d, want 5: binary state lost on shutdown", got.Number)
	}
	if !chain2.TrieDB().IsPBT() {
		t.Fatal("rebooted node did not open on the binary tree")
	}
	waitFor(t, 10*time.Second, "window never resumed from its cursor", func() bool {
		m := chain2.MigrationProgress().Merkle
		if m != nil && m.Phase == "stalled" {
			t.Fatalf("window re-replayed instead of resuming: %s", m.Error)
		}
		return m != nil && m.Phase == "synced" && m.ShadowRoot == want
	})
	// The window replays a transaction-bearing binary block after the reboot.
	pay(eth2.TxPool(), 1)
	newHead := buildBlock(t, api2, chain2.CurrentBlock(), 6, common.Hash{})
	awaitShadowReady(t, chain2, newHead)
	if st, err := chain2.StateAt(newHead); err != nil {
		t.Fatalf("state at the new head: %v", err)
	} else if got := st.GetBalance(recipient).Uint64(); got != 2000 {
		t.Fatalf("recipient balance after the boundary transfer = %d, want 2000", got)
	}
	if p := chain2.MigrationProgress(); p.Binary != nil {
		t.Fatalf("binary direction revived without a pre-fork head: %+v", p.Binary)
	}
	// Finality closes the window; a finished node boots without a follower.
	fin := engine.ForkchoiceStateV1{HeadBlockHash: newHead.Hash(), SafeBlockHash: newHead.Hash(), FinalizedBlockHash: newHead.Hash()}
	if _, err := api2.ForkchoiceUpdatedV4(context.Background(), fin, nil, nil); err != nil {
		t.Fatalf("finalizing: %v", err)
	}
	buildBlock(t, api2, newHead, 7, common.Hash{})
	waitFor(t, 5*time.Second, "progress never reached done", func() bool {
		return chain2.MigrationProgress().Phase == "done"
	})
	n2.Close()
	closed2 = true

	n3, eth3 := startEthServiceAt(t, datadir, genesis, nil)
	defer n3.Close()
	if p := eth3.BlockChain().MigrationProgress(); p.Phase != "done" {
		t.Fatalf("finished node progress %q, want done", p.Phase)
	}
	if _, err := eth3.BlockChain().State(); err != nil {
		t.Fatalf("finished node cannot open its state: %v", err)
	}
}

// TestFullMigrationLifecycleBlocksKnob is the same rehearsal closed by the
// block-count knob, never finalizing.
func TestFullMigrationLifecycleBlocksKnob(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil, func(cfg *ethconfig.Config) {
		cfg.MigrationWindowBlocks = 3
	})
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	parent := chain.CurrentBlock()
	for i := 0; i < 6; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	waitFor(t, 5*time.Second, "knob never closed the window", func() bool {
		return rawdb.ReadPBTMigrationDone(ethservice.ChainDb())
	})
	if p := chain.MigrationProgress(); p.Phase != "done" {
		t.Fatalf("progress %q, want done", p.Phase)
	}
}

// TestDirectionPRevivesOnReorg parks the binary direction past the fork,
// then reorgs to a pre-fork sibling: replay must resume on the new branch.
func TestDirectionPRevivesOnReorg(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()

	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	sibling := buildBlock(t, api, chain.GetHeaderByNumber(2), 3, common.Hash{1})
	if sibling.Number.Uint64() != 3 {
		t.Fatalf("sibling landed at height %d, want 3", sibling.Number)
	}
	awaitShadowReady(t, chain, sibling)
}

// TestMigrationSurvivesRestartPreFork reboots a migrating node before the
// fork: the shadow resumes from its cursor and the migration completes.
func TestMigrationSurvivesRestartPreFork(t *testing.T) {
	genesis := migrationTestGenesis()
	datadir := t.TempDir()
	n, ethservice := startEthServiceAt(t, datadir, genesis, nil)
	closed := false
	defer func() {
		if !closed {
			n.Close()
		}
	}()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	parent := chain.CurrentBlock()
	for i := 0; i < 2; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	awaitShadowReady(t, chain, chain.CurrentBlock())
	n.Close()
	closed = true

	n2, eth2 := startEthServiceAt(t, datadir, genesis, nil)
	defer n2.Close()
	api2 := NewConsensusAPI(eth2)
	chain2 := eth2.BlockChain()
	parent = chain2.CurrentBlock()
	if parent.Number.Uint64() != 2 {
		t.Fatalf("rebooted head %d, want 2", parent.Number)
	}
	if p := chain2.MigrationProgress(); p.Merkle != nil {
		t.Fatalf("merkle window opened before the fork: %+v", p.Merkle)
	}
	for i := 2; i < 5; i++ {
		parent = buildBlock(t, api2, parent, uint64(i+1), common.Hash{})
	}
	awaitShadowReady(t, chain2, chain2.CurrentBlock())

	head := chain2.CurrentBlock()
	fin := engine.ForkchoiceStateV1{HeadBlockHash: head.Hash(), SafeBlockHash: head.Hash(), FinalizedBlockHash: head.Hash()}
	if _, err := api2.ForkchoiceUpdatedV4(context.Background(), fin, nil, nil); err != nil {
		t.Fatalf("finalizing: %v", err)
	}
	buildBlock(t, api2, head, 6, common.Hash{})
	waitFor(t, 5*time.Second, "migration never finished after the reboot", func() bool {
		return rawdb.ReadPBTMigrationDone(eth2.ChainDb())
	})
}

// TestBatchImportAcrossTheFork pins full sync: one InsertChain spanning the
// boundary must drive the shadow itself - the head event only fires after
// the batch.
func TestBatchImportAcrossTheFork(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	parent := chain.CurrentBlock()
	var blocks []*types.Block
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
		blocks = append(blocks, chain.GetBlockByHash(parent.Hash()))
	}

	importer, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, beacon.New(ethash.NewFaker()), core.DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer importer.Stop()
	if _, err := importer.InsertChain(blocks); err != nil {
		t.Fatalf("batch import across the fork: %v", err)
	}
	if head := importer.CurrentBlock(); head.Number.Uint64() != 5 {
		t.Fatalf("imported head %d, want 5", head.Number)
	}
}

// TestMigrationBoundaryStraddleReorg is the straddle rehearsal: two branches
// fork at a shared pre-fork ancestor and each cross the activation boundary
// independently. The builder rewinds its head across the format swap and
// re-parks the binary direction on the new branch. A separate victim then
// receives the winning branch exactly the way the engine delivers a reorg -
// per-block sidechain inserts (newPayload) followed by one head switch
// (forkchoiceUpdated) - and must re-park and keep the window running without
// stalling. The head switch is SetCanonical on a non-canonical block, which
// takes no maxReorgDepth check (eth/catalyst/api.go, the non-canonical
// branch of forkchoiceUpdated; the depth limit only guards rewinds to
// canonical ancestors), so the heal is never depth-refused.
func TestMigrationBoundaryStraddleReorg(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()

	// Branch A crosses the fork: blocks 1-3 merkle (t=12..36), 4-5 binary.
	parent := chain.CurrentBlock()
	var branchA []*types.Block
	for i := range 5 {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
		branchA = append(branchA, chain.GetBlockByHash(parent.Hash()))
	}
	if p := chain.MigrationProgress(); p.Binary == nil || p.Binary.Phase != "parked" || p.Binary.CursorHash != branchA[2].Hash() {
		t.Fatalf("binary direction not parked at A's boundary predecessor: %+v", p.Binary)
	}

	// Branch B forks at block 1 - two blocks below the boundary - crosses
	// with different blocks and outgrows A by one.
	forkPoint := chain.GetHeaderByNumber(1)
	bParent := forkPoint
	var branchB []*types.Block
	for i := range 5 {
		bParent = buildBlock(t, api, bParent, uint64(i+2), common.Hash{0xbb})
		branchB = append(branchB, chain.GetBlockByHash(bParent.Hash()))
	}
	boundaryB := branchB[1] // block 3, B's boundary predecessor
	awaitShadowReady(t, chain, branchB[4].Header())
	if p := chain.MigrationProgress(); p.Binary == nil || p.Binary.Phase != "parked" || p.Binary.CursorHash != boundaryB.Hash() {
		t.Fatalf("builder binary direction did not re-park at B's boundary predecessor: %+v", p.Binary)
	}
	if !chain.Migrating() {
		t.Fatal("builder window closed by the straddle reorg")
	}

	// The victim follows branch A first, the way full sync would.
	victim, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, beacon.New(ethash.NewFaker()), core.DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer victim.Stop()
	if _, err := victim.InsertChain(branchA); err != nil {
		t.Fatalf("victim import of branch A: %v", err)
	}
	awaitShadowReady(t, victim, branchA[4].Header())
	if p := victim.MigrationProgress(); p.Binary == nil || p.Binary.Phase != "parked" || p.Binary.CursorHash != branchA[2].Hash() {
		t.Fatalf("victim binary direction not parked at A's boundary predecessor: %+v", p.Binary)
	}

	// The straddle: branch B arrives engine-style - sidechain inserts, then
	// one forkchoice switch across the swap.
	for _, block := range branchB {
		if _, err := victim.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			t.Fatalf("sidechain insert of B block %d: %v", block.NumberU64(), err)
		}
	}
	if _, err := victim.SetCanonical(branchB[4]); err != nil {
		t.Fatalf("straddle heal: %v", err)
	}

	// (a) canonical is branch B end to end, across the format swap.
	if head := victim.CurrentBlock(); head.Hash() != branchB[4].Hash() {
		t.Fatalf("victim head %d %x, want B tip %x", head.Number, head.Hash(), branchB[4].Hash())
	}
	if got := victim.GetCanonicalHash(1); got != forkPoint.Hash() {
		t.Fatalf("canonical block 1 = %x, want the shared fork point %x", got, forkPoint.Hash())
	}
	for _, block := range branchB {
		if got := victim.GetCanonicalHash(block.NumberU64()); got != block.Hash() {
			t.Fatalf("canonical block %d = %x, want B's %x", block.NumberU64(), got, block.Hash())
		}
	}
	// (c) the window is still open and follows B's post-fork blocks.
	if !victim.Migrating() {
		t.Fatal("victim window closed by the straddle reorg")
	}
	for _, block := range branchB[2:] {
		awaitShadowReady(t, victim, block.Header())
	}
	// (b) the binary direction re-parked at B's boundary predecessor, and
	// (d) neither direction stalled.
	p := victim.MigrationProgress()
	if p.Binary == nil || p.Binary.Phase != "parked" || p.Binary.CursorHash != boundaryB.Hash() {
		t.Fatalf("victim binary direction did not re-park at B's boundary predecessor: %+v", p.Binary)
	}
	if p.Merkle == nil || (p.Merkle.Phase != "synced" && p.Merkle.Phase != "following") || p.Merkle.Error != "" {
		t.Fatalf("victim merkle window not following B: %+v", p.Merkle)
	}
}

// shadowRootEvent mirrors the debug_shadowRoots stream payload.
type shadowRootEvent struct {
	BlockHash  common.Hash    `json:"blockHash"`
	Number     hexutil.Uint64 `json:"number"`
	ShadowRoot common.Hash    `json:"shadowRoot"`
}

// TestShadowRootSidecar pins the distributor surface: the getter answers by
// block hash and the subscription streams one record per head.
func TestShadowRootSidecar(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	client := n.Attach()
	defer client.Close()
	events := make(chan shadowRootEvent, 16)
	ctx := context.Background()
	sub, err := client.Subscribe(ctx, "debug", events, "shadowRoots")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}

	seen := make(map[uint64]common.Hash)
	timeout := time.After(15 * time.Second)
	for len(seen) < 5 {
		select {
		case ev := <-events:
			seen[uint64(ev.Number)] = ev.ShadowRoot
		case err := <-sub.Err():
			t.Fatalf("subscription died: %v", err)
		case <-timeout:
			t.Fatalf("only %d of 5 shadow roots streamed", len(seen))
		}
	}
	for number, root := range seen {
		header := chain.GetHeaderByNumber(number)
		if want, ok := rawdb.ReadShadowStateRoot(ethservice.ChainDb(), header.Hash(), number); !ok || root != want {
			t.Fatalf("streamed root %x for block %d, record says %x (ok=%v)", root, number, want, ok)
		}
	}

	var got *common.Hash
	boundary := chain.GetHeaderByNumber(3)
	if err := client.CallContext(ctx, &got, "debug_shadowStateRoot", boundary.Hash()); err != nil {
		t.Fatalf("getter: %v", err)
	}
	if want, _ := rawdb.ReadShadowStateRoot(ethservice.ChainDb(), boundary.Hash(), 3); got == nil || *got != want {
		t.Fatalf("getter said %v, record says %x", got, want)
	}
	var absent *common.Hash
	if err := client.CallContext(ctx, &absent, "debug_shadowStateRoot", common.Hash{0xde, 0xad}); err != nil {
		t.Fatalf("getter for unknown hash: %v", err)
	}
	if absent != nil {
		t.Fatalf("unknown block got a root: %x", *absent)
	}
}

// TestShadowRootStreamNeverBlocksImport pins the sidecar's decoupling: a
// closed migration stops producing records, and the subscription must not
// backpressure the head feed while it waits for ones that never come.
func TestShadowRootStreamNeverBlocksImport(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil, func(cfg *ethconfig.Config) {
		cfg.MigrationWindowBlocks = 2
	})
	defer n.Close()

	client := n.Attach()
	defer client.Close()
	events := make(chan shadowRootEvent, 64)
	sub, err := client.Subscribe(context.Background(), "debug", events, "shadowRoots")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	waitFor(t, 5*time.Second, "knob never closed the window", func() bool {
		return rawdb.ReadPBTMigrationDone(ethservice.ChainDb())
	})
	start := time.Now()
	for i := 5; i < 75; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Fatalf("70 recordless heads took %v: the stream backpressures imports", elapsed)
	}
}

// convertCanonicalMerkle rebuilds a merkle trie from the canonical binary
// state at the given header - the reverse of the offline conversion. The
// universe is the genesis allocation plus every account and slot a stored
// access list ever wrote.
func convertCanonicalMerkle(t *testing.T, chain *core.BlockChain, db ethdb.Database, genesis *core.Genesis, header *types.Header) common.Hash {
	t.Helper()
	src, err := chain.StateAt(header)
	if err != nil {
		t.Fatal(err)
	}
	universe := make(map[common.Address]map[common.Hash]bool)
	touch := func(addr common.Address) map[common.Hash]bool {
		if universe[addr] == nil {
			universe[addr] = make(map[common.Hash]bool)
		}
		return universe[addr]
	}
	for addr, account := range genesis.Alloc {
		slots := touch(addr)
		for key := range account.Storage {
			slots[key] = true
		}
	}
	for n := uint64(1); n <= header.Number.Uint64(); n++ {
		hash := rawdb.ReadCanonicalHash(db, n)
		list := rawdb.ReadAccessList(db, hash, n)
		if list == nil {
			t.Fatalf("no stored access list for block %d", n)
		}
		for _, acc := range *list {
			slots := touch(acc.Address)
			for _, sw := range acc.StorageChanges {
				slots[common.Hash(sw.Slot.Bytes32())] = true
			}
		}
	}
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.HashDefaults)
	defer tdb.Close()
	dst, err := state.New(types.EmptyRootHash, state.NewDatabase(tdb, nil))
	if err != nil {
		t.Fatal(err)
	}
	for addr, slots := range universe {
		if !src.Exist(addr) {
			continue
		}
		dst.AddBalance(addr, src.GetBalance(addr), tracing.BalanceIncreaseGenesisBalance)
		dst.SetCode(addr, src.GetCode(addr), tracing.CodeChangeGenesis)
		dst.SetNonce(addr, src.GetNonce(addr), tracing.NonceChangeGenesis)
		for slot := range slots {
			if value := src.GetState(addr, slot); value != (common.Hash{}) {
				dst.SetState(addr, slot, value)
			}
		}
	}
	root, err := dst.Commit(0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	return root
}
