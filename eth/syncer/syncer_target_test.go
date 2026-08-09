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

package syncer_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	ethproto "github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/eth/syncer"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

// headerPeer is a stub downloader peer serving exactly one header by hash. It
// implements just enough of the downloader.Peer interface for the syncer's
// target resolution (Downloader.GetHeader); every other retrieval fails. The
// zero-valued eth.Request it hands out makes Request.Close a no-op, which is
// the documented escape hatch for tests mocking out the dispatcher.
type headerPeer struct {
	header *types.Header
}

func (p *headerPeer) RequestHeadersByHash(hash common.Hash, amount int, skip int, reverse bool, sink chan *ethproto.Response) (*ethproto.Request, error) {
	if hash != p.header.Hash() {
		return nil, errors.New("unknown header requested")
	}
	req := new(ethproto.Request)
	res := &ethproto.Response{
		Req:  req,
		Res:  &ethproto.BlockHeadersRequest{p.header},
		Meta: []common.Hash{p.header.Hash()},
		Time: time.Millisecond,
		Done: make(chan error, 1),
	}
	go func() {
		// Deliver the canned response unless the requester timed out.
		select {
		case sink <- res:
		case <-time.After(10 * time.Second):
		}
	}()
	return req, nil
}

func (p *headerPeer) RequestHeadersByNumber(uint64, int, int, bool, chan *ethproto.Response) (*ethproto.Request, error) {
	return nil, errors.New("headers by number not supported")
}

func (p *headerPeer) RequestBodies([]common.Hash, chan *ethproto.Response) (*ethproto.Request, error) {
	return nil, errors.New("bodies not supported")
}

func (p *headerPeer) RequestReceipts([]common.Hash, []uint64, []uint64, chan *ethproto.Response) (*ethproto.Request, error) {
	return nil, errors.New("receipts not supported")
}

func (p *headerPeer) RequestBALs([]common.Hash, chan *ethproto.Response) (*ethproto.Request, error) {
	return nil, errors.New("block access lists not supported")
}

// newTestSyncer spins up an in-process node at genesis with the given sync
// mode, registers a syncer on it and injects a stub downloader peer serving
// the header of block #2 of a chain generated on the same genesis. It returns
// the syncer and the served target header.
//
// Block #2 is served rather than #1 on purpose: the node only has the genesis,
// so an accepted target cannot link up with the local chain and the skeleton
// filler stays suspended for the duration of the test.
func newTestSyncer(t *testing.T, mode ethconfig.SyncMode) (*syncer.Syncer, *types.Header) {
	t.Helper()

	// Post-merge chain configuration, so the beacon-mode downloader is used.
	config := *params.AllEthashProtocolChanges
	config.TerminalTotalDifficulty = common.Big0
	config.MergeNetsplitBlock = common.Big0

	genesis := &core.Genesis{
		Config:     &config,
		Timestamp:  9000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: common.Big0,
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, beacon.New(ethash.NewFaker()), 2, nil)

	stack, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			NoDial:      true,
		},
	})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	backend, err := eth.New(stack, &ethconfig.Config{
		Genesis:        genesis,
		SyncMode:       mode,
		TrieTimeout:    time.Minute,
		TrieDirtyCache: 256,
		TrieCleanCache: 256,
		Miner:          miner.DefaultConfig,
	})
	if err != nil {
		t.Fatalf("failed to create eth service: %v", err)
	}
	s, err := syncer.Register(stack, backend, syncer.Config{})
	if err != nil {
		t.Fatalf("failed to register syncer: %v", err)
	}
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	t.Cleanup(func() { stack.Close() })

	head := blocks[1].Header()
	if err := backend.Downloader().RegisterPeer("stub", ethproto.ETH70, &headerPeer{header: head}); err != nil {
		t.Fatalf("failed to register stub peer: %v", err)
	}
	return s, head
}

// TestRejectedTargetNotInstalled verifies that a sync request rejected by the
// syncmode check does not install its target: retrying the same hash must
// keep reporting the actual rejection reason instead of tripping the
// stale-target check against a target that never started syncing.
func TestRejectedTargetNotInstalled(t *testing.T) {
	s, head := newTestSyncer(t, ethconfig.SnapSync)

	for i := 1; i <= 2; i++ {
		err := s.Sync(head.Hash())
		if err == nil {
			t.Fatalf("attempt %d: expected an error under snap syncmode, got none", i)
		}
		if !strings.Contains(err.Error(), "unsupported syncmode") {
			t.Fatalf("attempt %d: expected syncmode rejection, got: %v", i, err)
		}
	}
}

// TestStaleTargetRejected verifies that once a sync target has been accepted,
// re-requesting a target that is not strictly newer is still rejected.
func TestStaleTargetRejected(t *testing.T) {
	s, head := newTestSyncer(t, ethconfig.FullSync)

	if err := s.Sync(head.Hash()); err != nil {
		t.Fatalf("initial sync request failed: %v", err)
	}
	err := s.Sync(head.Hash())
	if err == nil || !strings.Contains(err.Error(), "stale sync target") {
		t.Fatalf("expected stale target rejection, got: %v", err)
	}
}
