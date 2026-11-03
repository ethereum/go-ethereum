// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func testPoWMiner(t *testing.T, test func(*testing.T)) {
	t.Helper()
	// Geth's process-wide sender workers outlive individual tests and must not
	// belong to a synctest bubble, which requires its goroutines to terminate.
	core.SenderCacher()
	synctest.Test(t, test)
}

func newPoWMinerEnv(t *testing.T, config PoWMinerConfig) (*testEnv, *PoWMiner) {
	t.Helper()
	env, _, _ := newMiningEnv(t, 1)
	m := NewPoWMiner(NewShard(env.sc), config)
	t.Cleanup(m.StopLocalMining)
	return env, m
}

// servedBlock returns the served candidate with the given seal hash.
func (m *PoWMiner) servedBlock(sealHash common.Hash) *types.Block {
	m.mu.Lock()
	defer m.mu.Unlock()
	block, _ := m.works.Get(sealHash)
	return block
}

func getWork(t *testing.T, m *PoWMiner, coinbase common.Address) *MiningWork {
	t.Helper()
	work, err := m.GetWork(context.Background(), coinbase)
	if err != nil {
		t.Fatal(err)
	}
	return work
}

func waitForHead(t *testing.T, env *testEnv, number uint64) {
	t.Helper()
	for deadline := time.Now().Add(time.Hour); env.chain.CurrentBlock().Number.Uint64() < number; {
		if time.Now().After(deadline) {
			t.Fatalf("head #%d did not reach #%d", env.chain.CurrentBlock().Number.Uint64(), number)
		}
		time.Sleep(time.Second)
	}
}

func TestPoWMinerGetWork(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		env, m := newPoWMinerEnv(t, PoWMinerConfig{})
		a, b := common.Address{0xaa}, common.Address{0xbb}
		first := getWork(t, m, a)
		block := m.servedBlock(first.SealHash)
		if block == nil || block.Coinbase() != a || block.ParentHash() != env.genesis.Hash() {
			t.Fatal("work was not built on the head for the requested coinbase")
		}
		if sealHash, _ := block.Header().QKCSealHash(); sealHash != first.SealHash || first.Number != 1 || first.Time != block.Time() {
			t.Fatalf("work does not describe its candidate: %+v", first)
		}
		// Without staking, the work targets the candidate's base difficulty.
		if first.Difficulty.Sign() <= 0 || first.Difficulty.Cmp(block.Header().QKCDifficulty()) != 0 {
			t.Fatalf("work difficulty %v, want base difficulty %v", first.Difficulty, block.Header().QKCDifficulty())
		}
		time.Sleep(DefaultWorkRefresh - time.Second)
		if work := getWork(t, m, a); work.SealHash != first.SealHash {
			t.Fatal("current work was rebuilt")
		}
		other := getWork(t, m, b)
		if other.SealHash == first.SealHash || m.servedBlock(other.SealHash).Coinbase() != b {
			t.Fatal("coinbases share work")
		}
		time.Sleep(time.Second)
		refreshed := getWork(t, m, a)
		if refreshed.SealHash == first.SealHash || refreshed.Time <= first.Time {
			t.Fatal("expired work was not rebuilt")
		}
		if err := m.SubmitWork(other.SealHash, types.EncodeNonce(1), common.Hash{}); err != nil {
			t.Fatal(err)
		}
		next := getWork(t, m, a)
		if next.Number != 2 || m.servedBlock(next.SealHash).ParentHash() != env.chain.CurrentBlock().Hash() {
			t.Fatal("work was not rebuilt on the new head")
		}
	})
}

func TestPoWMinerGetWorkRootReorg(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		env, m := newPoWMinerEnv(t, PoWMinerConfig{})
		coinbase := common.Address{0xaa}
		first := getWork(t, m, coinbase)
		// Replace the root tip the work references, keeping the shard head.
		seed := uint64(1)
		fork := env.sc.RootBlock(env.sc.rootTip.ParentHash).Header().CreateBlockToAppend(nil, uint256.NewInt(1), nil, &seed, nil)
		env.insertRoot(fork, true)
		env.checkHead(env.genesis)
		work := getWork(t, m, coinbase)
		extra, _ := m.servedBlock(work.SealHash).Header().QKCExtra()
		if work.SealHash == first.SealHash || extra.PrevRootBlock != fork.Hash() {
			t.Fatal("work was not rebuilt on the new root tip")
		}
	})
}

func TestPoWMinerSubmitWork(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		env, m := newPoWMinerEnv(t, PoWMinerConfig{})
		a, b := common.Address{0xaa}, common.Address{0xbb}
		if err := m.SubmitWork(common.Hash{1}, types.BlockNonce{}, common.Hash{}); !errors.Is(err, ErrUnknownWork) {
			t.Fatalf("unknown work: %v", err)
		}
		first := getWork(t, m, a)
		time.Sleep(DefaultWorkRefresh)
		second := getWork(t, m, a)
		other := getWork(t, m, b)
		// Superseded work is accepted while its parent is the head.
		nonce, mixHash := types.EncodeNonce(7), common.Hash{7}
		if err := m.SubmitWork(first.SealHash, nonce, mixHash); err != nil {
			t.Fatal(err)
		}
		head := env.chain.CurrentBlock()
		extra, _ := head.QKCExtra()
		sealHash, _ := head.QKCSealHash()
		if head.Number.Uint64() != 1 || head.Coinbase != a || sealHash != first.SealHash || extra.Nonce != nonce || extra.MixHash != mixHash {
			t.Fatal("head is not the sealed candidate")
		}
		if err := m.SubmitWork(first.SealHash, nonce, mixHash); !errors.Is(err, ErrUnknownWork) {
			t.Fatalf("imported work: %v", err)
		}
		for _, work := range []*MiningWork{second, other} {
			if err := m.SubmitWork(work.SealHash, nonce, mixHash); !errors.Is(err, ErrStaleWork) {
				t.Fatalf("work on the old head: %v", err)
			}
			if err := m.SubmitWork(work.SealHash, nonce, mixHash); !errors.Is(err, ErrUnknownWork) {
				t.Fatalf("dropped stale work: %v", err)
			}
		}
		env.checkHead(env.chain.GetBlockByHash(head.Hash()))
	})
}

// failingShard is a shard layer whose imports fail while err is set.
type failingShard struct {
	*Shard
	err error
}

func (s *failingShard) InsertMinedBlock(block *types.Block) error {
	if s.err != nil {
		return s.err
	}
	return s.Shard.InsertMinedBlock(block)
}

func TestPoWMinerSubmitWorkInsertFailure(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		env, _, _ := newMiningEnv(t, 1)
		insertErr := errors.New("insert failed")
		backend := &failingShard{Shard: NewShard(env.sc), err: insertErr}
		m := NewPoWMiner(backend, PoWMinerConfig{})
		work := getWork(t, m, common.Address{0xaa})
		if err := m.SubmitWork(work.SealHash, types.EncodeNonce(1), common.Hash{}); !errors.Is(err, insertErr) {
			t.Fatalf("insert failure: %v", err)
		}
		env.checkHead(env.genesis)
		// The work stays available for another submission.
		backend.err = nil
		if err := m.SubmitWork(work.SealHash, types.EncodeNonce(1), common.Hash{}); err != nil {
			t.Fatal(err)
		}
		if env.chain.CurrentBlock().Number.Uint64() != 1 {
			t.Fatal("retried submission was not imported")
		}
	})
}

// sealing is a local search whose seals the test supplies.
type sealing struct {
	works chan *MiningWork
	seals chan types.BlockNonce
}

func newSealing() *sealing {
	return &sealing{works: make(chan *MiningWork), seals: make(chan types.BlockNonce)}
}

func (s *sealing) Search(ctx context.Context, work *MiningWork) (types.BlockNonce, common.Hash, bool) {
	select {
	case s.works <- work:
	case <-ctx.Done():
		return types.BlockNonce{}, common.Hash{}, false
	}
	select {
	case nonce := <-s.seals:
		return nonce, common.Hash{}, true
	case <-ctx.Done():
		return types.BlockNonce{}, common.Hash{}, false
	}
}

func TestPoWMinerLocal(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		coinbase := common.Address{0x42}
		search := newSealing()
		env, m := newPoWMinerEnv(t, PoWMinerConfig{Coinbase: coinbase, Search: search})
		if err := m.StartLocalMining(); err != nil {
			t.Fatal(err)
		}
		if err := m.StartLocalMining(); err != nil {
			t.Fatal(err)
		}
		if first := <-search.works; first.Number != 1 {
			t.Fatalf("local mining started at block #%d", first.Number)
		}
		search.seals <- types.EncodeNonce(1)
		second := <-search.works
		if head := env.chain.CurrentBlock(); head.Number.Uint64() != 1 || head.Coinbase != coinbase {
			t.Fatal("locally sealed block was not imported")
		}
		if second.Number != 2 {
			t.Fatalf("mining did not continue on the new head: block #%d", second.Number)
		}
		// A remote block changes the head and restarts the local search.
		remote := getWork(t, m, common.Address{0xbb})
		if err := m.SubmitWork(remote.SealHash, types.EncodeNonce(2), common.Hash{}); err != nil {
			t.Fatal(err)
		}
		third := <-search.works
		if third.Number != 3 || m.servedBlock(third.SealHash).ParentHash() != env.chain.CurrentBlock().Hash() {
			t.Fatal("local search did not follow the remote head")
		}
		// Expired work is replaced.
		refreshed := <-search.works
		if refreshed.SealHash == third.SealHash || refreshed.Number != 3 {
			t.Fatal("local search did not refresh expired work")
		}
		m.StopLocalMining()
		head := env.chain.CurrentBlock().Hash()
		time.Sleep(time.Minute)
		if env.chain.CurrentBlock().Hash() != head {
			t.Fatal("stopped miner imported a block")
		}
		if err := m.StartLocalMining(); err != nil {
			t.Fatal(err)
		}
		if work := <-search.works; work.Number != 3 {
			t.Fatal("restarted miner did not resume on the head")
		}
	})
}

func TestPoWMinerSimulated(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		coinbase := common.Address{0x42}
		env, m := newPoWMinerEnv(t, PoWMinerConfig{Coinbase: coinbase, Search: NewSimulatedSearch(5 * time.Second)})
		if err := m.StartLocalMining(); err != nil {
			t.Fatal(err)
		}
		waitForHead(t, env, 3)
		m.StopLocalMining()
		for n := uint64(1); n <= 3; n++ {
			block, parent := env.chain.GetBlockByNumber(n), env.chain.GetBlockByNumber(n-1)
			if block.Coinbase() != coinbase || block.Time() <= parent.Time() || block.Time() > uint64(time.Now().Unix()) {
				t.Fatalf("unexpected block #%d: coinbase %x, time %d after %d", n, block.Coinbase(), block.Time(), parent.Time())
			}
		}
	})
}

func TestPoWMinerStartWithoutSearch(t *testing.T) {
	_, m := newPoWMinerEnv(t, PoWMinerConfig{})
	if err := m.StartLocalMining(); err == nil {
		t.Fatal("started local mining without a search")
	}
}

func TestSimulatedSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const blockTime = 10 * time.Second
		search := NewSimulatedSearch(blockTime)
		// Seals are never earlier than the candidate's timestamp.
		work := &MiningWork{Time: uint64(time.Now().Unix()) + 5}
		if _, _, ok := search.Search(context.Background(), work); !ok || time.Now().Before(time.Unix(int64(work.Time), 0)) {
			t.Fatal("sealed before the candidate's timestamp")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, _, ok := search.Search(ctx, &MiningWork{Time: work.Time + 3600}); ok {
			t.Fatal("cancelled search sealed")
		}
		// Delays average the block time.
		const samples = 500
		var total time.Duration
		for range samples {
			start := time.Now()
			search.Search(context.Background(), &MiningWork{Time: uint64(start.Unix())})
			total += time.Since(time.Unix(start.Unix(), 0))
		}
		if mean := total / samples; mean < blockTime*8/10 || mean > blockTime*12/10 {
			t.Fatalf("mean delay %v, want about %v", mean, blockTime)
		}
	})
}
