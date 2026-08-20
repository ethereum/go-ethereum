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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
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

// TestMigrationNodeCrossesTheFork drives the rehearsal through the engine
// API: merkle blocks, the swap on the shadow's recorded root, native binary
// blocks with the merkle window live, and the finality close.
func TestMigrationNodeCrossesTheFork(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	if chain.TrieDB().IsPBT() {
		t.Fatal("migrating node opened on the binary tree")
	}

	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	// The boundary parent's shadow root fed the activation block, and past
	// the boundary the roles flip: the window records merkle roots for the
	// binary blocks until the close.
	boundaryParent := chain.GetHeaderByNumber(3)
	if !chain.ShadowReady(boundaryParent.Hash(), 3) {
		t.Fatal("the boundary parent has no recorded shadow root")
	}
	awaitShadowReady(t, chain, chain.GetHeaderByNumber(4))
	awaitShadowReady(t, chain, chain.GetHeaderByNumber(5))

	// Both sides of the boundary stay readable.
	for _, number := range []uint64{2, 5} {
		header := chain.GetHeaderByNumber(number)
		statedb, err := chain.StateAt(header)
		if err != nil {
			t.Fatalf("state at block %d: %v", number, err)
		}
		if got := statedb.GetBalance(testAddr).ToBig(); got.Cmp(testBalance) != 0 {
			t.Fatalf("balance at block %d = %v, want %v", number, got, testBalance)
		}
	}

	// Finality closes the window on the next head.
	head := chain.CurrentBlock()
	fin := engine.ForkchoiceStateV1{HeadBlockHash: head.Hash(), SafeBlockHash: head.Hash(), FinalizedBlockHash: head.Hash()}
	if _, err := api.ForkchoiceUpdatedV4(context.Background(), fin, nil, nil); err != nil {
		t.Fatalf("finalizing: %v", err)
	}
	buildBlock(t, api, head, 6, common.Hash{})
	for start := time.Now(); !rawdb.ReadPBTMigrationDone(ethservice.ChainDb()); {
		if time.Since(start) > 5*time.Second {
			t.Fatal("migration never marked itself done after finality")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestMerkleWindowAdvancesLive proves the window's records: each post-fork
// block's merkle root matches an independent reverse conversion of its
// binary state.
func TestMerkleWindowAdvancesLive(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()

	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	for _, number := range []uint64{4, 5} {
		header := chain.GetHeaderByNumber(number)
		awaitShadowReady(t, chain, header)
		got, _ := rawdb.ReadShadowStateRoot(ethservice.ChainDb(), number, header.Hash())
		if want := convertCanonicalMerkle(t, chain, ethservice.ChainDb(), genesis, header); got != want {
			t.Fatalf("block %d window root %x, reverse conversion says %x", number, got, want)
		}
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

// startPersistentEthService is startEthService over a real data directory,
// so a test can stop the node and reboot it on the same chain data.
func startPersistentEthService(t testing.TB, datadir string, genesis *core.Genesis, blocks []*types.Block, mods ...func(*ethconfig.Config)) (*node.Node, *eth.Ethereum) {
	t.Helper()
	n, err := node.New(&node.Config{
		DataDir: datadir,
		Name:    "geth",
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		}})
	if err != nil {
		t.Fatal("can't create node:", err)
	}
	ethcfg := &ethconfig.Config{
		Genesis:         genesis,
		SyncMode:        ethconfig.FullSync,
		TrieTimeout:     time.Minute,
		TrieDirtyCache:  256,
		TrieCleanCache:  256,
		DatabaseCache:   64,
		DatabaseHandles: 64,
		Miner:           miner.DefaultConfig,
	}
	for _, mod := range mods {
		mod(ethcfg)
	}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if len(blocks) > 0 {
		if _, err := ethservice.BlockChain().InsertChain(blocks); err != nil {
			n.Close()
			t.Fatal("can't import test blocks:", err)
		}
	}
	if err := ethservice.TxPool().Sync(); err != nil {
		t.Fatal("failed to sync txpool after initial blockchain import:", err)
	}
	ethservice.SetSynced()
	return n, ethservice
}

// TestMerkleWindowResumesAfterRestart pins the shutdown contract: both
// handles journal at their newest roots, so a rebooted window resumes from
// its cursor. The deleted lists make any re-replay a loud stall.
func TestMerkleWindowResumesAfterRestart(t *testing.T) {
	genesis := migrationTestGenesis()
	datadir := t.TempDir()
	n, ethservice := startPersistentEthService(t, datadir, genesis, nil)
	closed := false
	defer func() {
		if !closed {
			n.Close()
		}
	}()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	head := chain.GetHeaderByNumber(5)
	awaitShadowReady(t, chain, head)
	want, _ := rawdb.ReadShadowStateRoot(ethservice.ChainDb(), 5, head.Hash())
	for _, number := range []uint64{4, 5} {
		rawdb.DeleteAccessList(ethservice.ChainDb(), chain.GetHeaderByNumber(number).Hash(), number)
	}
	n.Close()
	closed = true

	n2, eth2 := startPersistentEthService(t, datadir, genesis, nil)
	defer n2.Close()
	reopened := eth2.BlockChain()
	if got := reopened.CurrentBlock(); got.Hash() != head.Hash() {
		t.Fatalf("rebooted head %d %x, want %d: binary state lost on shutdown", got.Number, got.Hash(), head.Number)
	}
	for start := time.Now(); ; time.Sleep(10 * time.Millisecond) {
		if m := reopened.MigrationProgress().Merkle; m != nil {
			if m.Phase == "synced" && m.ShadowRoot == want {
				return
			}
			if m.Phase == "stalled" {
				t.Fatalf("window re-replayed instead of resuming: %s", m.Error)
			}
		}
		if time.Since(start) > 10*time.Second {
			t.Fatalf("window never resumed: %+v", reopened.MigrationProgress())
		}
	}
}

// TestMigrationProgressPhases pins the debug surface: one direction per
// flavour, parked or working by which side of the boundary the head is on,
// and a terminal done.
func TestMigrationProgressPhases(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()

	parent := chain.CurrentBlock()
	for i := 0; i < 2; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	awaitShadowReady(t, chain, chain.CurrentBlock())
	p := chain.MigrationProgress()
	if p.Phase != "running" || p.Binary == nil || p.Merkle != nil {
		t.Fatalf("pre-fork progress %+v, want running with only the binary direction", p)
	}
	if got := p.Binary.Phase; got != "following" && got != "synced" {
		t.Fatalf("pre-fork binary phase %q", got)
	}

	for i := 2; i < 5; i++ {
		parent = buildBlock(t, api, parent, uint64(i+1), common.Hash{})
	}
	awaitShadowReady(t, chain, chain.CurrentBlock())
	p = chain.MigrationProgress()
	if p.Binary == nil || p.Binary.Phase != "parked" {
		t.Fatalf("post-fork binary progress %+v, want parked", p.Binary)
	}
	if p.Merkle == nil || (p.Merkle.Phase != "following" && p.Merkle.Phase != "synced") {
		t.Fatalf("post-fork merkle progress %+v, want it working", p.Merkle)
	}

	head := chain.CurrentBlock()
	fin := engine.ForkchoiceStateV1{HeadBlockHash: head.Hash(), SafeBlockHash: head.Hash(), FinalizedBlockHash: head.Hash()}
	if _, err := api.ForkchoiceUpdatedV4(context.Background(), fin, nil, nil); err != nil {
		t.Fatalf("finalizing: %v", err)
	}
	buildBlock(t, api, head, 6, common.Hash{})
	for start := time.Now(); chain.MigrationProgress().Phase != "done"; {
		if time.Since(start) > 5*time.Second {
			t.Fatalf("progress never reached done: %+v", chain.MigrationProgress())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
