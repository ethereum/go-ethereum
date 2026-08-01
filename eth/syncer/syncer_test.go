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

package syncer

import (
	"math/big"
	"testing"
	"time"

	bparams "github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

// generatePostMergeChain creates a post-merge chain of empty blocks.
func generatePostMergeChain(n int) (*core.Genesis, []*types.Block) {
	config := *params.AllEthashProtocolChanges
	config.TerminalTotalDifficulty = common.Big0
	config.MergeNetsplitBlock = common.Big0

	genesis := &core.Genesis{
		Config:     &config,
		ExtraData:  []byte("test genesis"),
		Timestamp:  9000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: big.NewInt(0),
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, beacon.New(ethash.NewFaker()), n, func(i int, g *core.BlockGen) {
		g.SetExtra([]byte("test"))
		g.OffsetTime(5)
	})
	return genesis, blocks
}

// startNode creates a full node with the given chain imported. If withSyncer
// is set, the sync override service is registered on it with an empty config,
// mirroring the unconditional registration done in cmd/geth, and returned so
// a test can drive it; otherwise the returned syncer is nil.
func startNode(t *testing.T, genesis *core.Genesis, blocks []*types.Block, withSyncer bool) (*node.Node, *eth.Ethereum, *Syncer) {
	t.Helper()

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			MaxPeers:    25,
		},
	})
	if err != nil {
		t.Fatal("can't create node:", err)
	}
	ethcfg := &ethconfig.Config{
		Genesis:        genesis,
		SyncMode:       ethconfig.FullSync,
		TrieTimeout:    time.Minute,
		TrieDirtyCache: 256,
		TrieCleanCache: 256,
		Miner:          miner.DefaultConfig,
	}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	var syncer *Syncer
	if withSyncer {
		syncer, err = Register(n, ethservice, Config{})
		if err != nil {
			t.Fatal("can't register syncer:", err)
		}
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks); err != nil {
		n.Close()
		t.Fatal("can't import test blocks:", err)
	}
	return n, ethservice, syncer
}

// newSyncPair brings up a full node holding the complete chain plus a second
// node a few blocks behind it, peers them, and returns the lagging node's eth
// service, the syncer registered on it, and the chain head it has yet to reach.
func newSyncPair(t *testing.T) (*eth.Ethereum, *Syncer, *types.Header) {
	t.Helper()

	// The chain must be long enough for head-2*EpochLength to exist after
	// the node under test has caught up.
	n := int(2*bparams.EpochLength + 8)
	genesis, blocks := generatePostMergeChain(n)

	// The full node owns the complete chain and acts as the sync peer.
	fullNode, _, _ := startNode(t, genesis, blocks, false)
	t.Cleanup(func() { fullNode.Close() })

	// The node under test comes up a few blocks behind the chain head, as
	// after a restart, with no finality information from a consensus client.
	laggingNode, laggingEth, syncer := startNode(t, genesis, blocks[:n-8], true)
	t.Cleanup(func() { laggingNode.Close() })

	for fullNode.Server().NodeInfo().Ports.Listener == 0 {
		time.Sleep(50 * time.Millisecond)
	}
	// Only the lagging node dials. Dialing from both sides at once lets the
	// two connections collide and both get dropped, and the retry is slow
	// enough to make the test drag.
	laggingNode.Server().AddPeer(fullNode.Server().Self())
	for laggingNode.Server().PeerCount() == 0 {
		time.Sleep(50 * time.Millisecond)
	}
	return laggingEth, syncer, blocks[len(blocks)-1].Header()
}

// Tests that the sync override service does not invent finalized and safe
// block markers on its own. Those markers must only ever reflect a value
// supplied by the consensus client via the engine API; deriving them from
// head-2*EpochLength / head-EpochLength serves blocks over the "finalized"
// and "safe" RPC labels that were never actually finalized and can still be
// reorged.
func TestSyncerDoesNotInventFinalityMarkers(t *testing.T) {
	laggingEth, _, head := newSyncPair(t)

	events := make(chan downloader.SyncEvent, 16)
	sub := laggingEth.Downloader().SubscribeSyncEvents(events)
	defer sub.Unsubscribe()

	// Let the node catch up to the chain head, as the consensus client would
	// instruct it to via the engine API. The syncer's own Sync is deliberately
	// not used here: no sync target has been set through it, which is exactly
	// the condition under which it must not synthesize anything.
	if err := laggingEth.Downloader().BeaconSync(head, nil); err != nil {
		t.Fatal("can't trigger beacon sync:", err)
	}
	timeout := time.After(60 * time.Second)
	for synced := false; !synced; {
		select {
		case ev := <-events:
			if ev.Type == downloader.SyncCompleted {
				synced = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for sync to complete")
		}
	}
	if number := laggingEth.BlockChain().CurrentBlock().Number.Uint64(); number != head.Number.Uint64() {
		t.Fatalf("node not synced to head, got %d, want %d", number, head.Number)
	}
	// No consensus client has supplied any finality information, so the node
	// must not expose finalized or safe blocks. Poll for a while to let the
	// syncer service process the sync completion event.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if final := laggingEth.BlockChain().CurrentFinalBlock(); final != nil {
			t.Fatalf("finalized block %d invented without consensus client, only the CL may set the finalized marker", final.Number)
		}
		if safe := laggingEth.BlockChain().CurrentSafeBlock(); safe != nil {
			t.Fatalf("safe block %d invented without consensus client, only the CL may set the safe marker", safe.Number)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Tests the positive case: once a sync target has been set through the syncer
// itself, no consensus client is attached to supply finality, so the syncer is
// the only thing that can give the "finalized" and "safe" RPC labels a value
// and must still do so.
func TestSyncerSetsFinalityMarkersForExplicitTarget(t *testing.T) {
	laggingEth, syncer, head := newSyncPair(t)

	events := make(chan downloader.SyncEvent, 16)
	sub := laggingEth.Downloader().SubscribeSyncEvents(events)
	defer sub.Unsubscribe()

	// Sync is what geth's --synctarget path calls. It only sets the target and
	// kicks off the beacon sync, so wait for the sync to actually complete.
	if err := syncer.Sync(head.Hash()); err != nil {
		t.Fatal("can't sync to target:", err)
	}
	timeout := time.After(60 * time.Second)
	for synced := false; !synced; {
		select {
		case ev := <-events:
			if ev.Type == downloader.SyncCompleted {
				synced = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for sync to complete")
		}
	}
	if number := laggingEth.BlockChain().CurrentBlock().Number.Uint64(); number != head.Number.Uint64() {
		t.Fatalf("node not synced to head, got %d, want %d", number, head.Number)
	}

	// The markers are synthesized relative to the head when the syncer
	// processes the sync completion event, which it does after Sync returns.
	wantFinal := head.Number.Uint64() - bparams.EpochLength*2
	wantSafe := head.Number.Uint64() - bparams.EpochLength

	deadline := time.Now().Add(30 * time.Second)
	for {
		final, safe := laggingEth.BlockChain().CurrentFinalBlock(), laggingEth.BlockChain().CurrentSafeBlock()
		if final != nil && safe != nil {
			if got := final.Number.Uint64(); got != wantFinal {
				t.Fatalf("wrong finalized block, got %d, want %d", got, wantFinal)
			}
			if got := safe.Number.Uint64(); got != wantSafe {
				t.Fatalf("wrong safe block, got %d, want %d", got, wantSafe)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for finality markers with a sync target set, finalized %v, safe %v", final, safe)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
