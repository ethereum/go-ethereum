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
// mirroring the unconditional registration done in cmd/geth.
func startNode(t *testing.T, genesis *core.Genesis, blocks []*types.Block, withSyncer bool) (*node.Node, *eth.Ethereum) {
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
	if withSyncer {
		if _, err := Register(n, ethservice, Config{}); err != nil {
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
	return n, ethservice
}

// Tests that the sync override service does not invent finalized and safe
// block markers on its own. Those markers must only ever reflect a value
// supplied by the consensus client via the engine API; deriving them from
// head-2*EpochLength / head-EpochLength serves blocks over the "finalized"
// and "safe" RPC labels that were never actually finalized and can still be
// reorged.
func TestSyncerDoesNotInventFinalityMarkers(t *testing.T) {
	// The chain must be long enough for head-2*EpochLength to exist after
	// the node under test has caught up.
	n := int(2*bparams.EpochLength + 8)
	genesis, blocks := generatePostMergeChain(n)

	// The full node owns the complete chain and acts as the sync peer.
	fullNode, _ := startNode(t, genesis, blocks, false)
	defer fullNode.Close()

	// The node under test comes up a few blocks behind the chain head, as
	// after a restart, with no finality information from a consensus client.
	laggingNode, laggingEth := startNode(t, genesis, blocks[:n-8], true)
	defer laggingNode.Close()

	for fullNode.Server().NodeInfo().Ports.Listener == 0 {
		time.Sleep(50 * time.Millisecond)
	}
	laggingNode.Server().AddPeer(fullNode.Server().Self())
	fullNode.Server().AddPeer(laggingNode.Server().Self())
	for laggingNode.Server().PeerCount() == 0 {
		time.Sleep(50 * time.Millisecond)
	}

	events := make(chan downloader.SyncEvent, 16)
	sub := laggingEth.Downloader().SubscribeSyncEvents(events)
	defer sub.Unsubscribe()

	// Let the node catch up to the chain head, as the consensus client
	// would instruct it to via the engine API.
	head := blocks[len(blocks)-1].Header()
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
