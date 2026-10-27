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

package eth

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// newClonePauseTester creates a chain on the path scheme with the first of the
// given number of blocks imported, and a pauser of its writes.
func newClonePauseTester(t *testing.T, n int) (*clonePauser, *core.BlockChain, []*types.Block) {
	t.Helper()

	gspec := &core.Genesis{Config: params.TestChainConfig, BaseFee: common.Big1}
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), n, func(i int, block *core.BlockGen) {
		block.SetCoinbase(common.Address{byte(i)})
	})
	db, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	config := core.DefaultConfig().WithStateScheme(rawdb.PathScheme)
	config.TrieJournalDirectory = t.TempDir()
	chain, err := core.NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	t.Cleanup(chain.Stop)

	if _, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("Failed to import: %v", err)
	}
	return newClonePauser(chain, clonePaths("chaindata", db, config.TrieJournalDirectory, "")), chain, blocks
}

// waitingContext is a context that tells when it is first waited on.
type waitingContext struct {
	context.Context
	once    sync.Once
	waiting chan struct{}
}

func (ctx *waitingContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.waiting) })
	return ctx.Context.Done()
}

// importer imports the given blocks in background, reporting the result.
func importer(chain *core.BlockChain, blocks []*types.Block) chan error {
	done := make(chan error, 1)
	go func() {
		_, err := chain.InsertChain(blocks)
		done <- err
	}()
	return done
}

// held checks that the chain mutex is held: a pause with a done context waits
// for a held mutex until it sees the context done, while it takes a free one
// without looking at the context.
func held(t *testing.T, chain *core.BlockChain) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waiting := &waitingContext{Context: ctx, waiting: make(chan struct{})}
	if _, err := chain.PauseForClone(waiting); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error of a pause with its context done: %v", err)
	}
	select {
	case <-waiting.waiting:
	default:
		t.Fatal("The chain mutex isn't held")
	}
}

// released checks that the import waiting for a pause ends, now that the pause
// has.
func released(t *testing.T, done chan error) {
	t.Helper()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Failed to import after the pause: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("The pause holds the chain writes")
	}
}

func TestClonePauseLease(t *testing.T) {
	cp, chain, blocks := newClonePauseTester(t, 3)

	res, err := cp.pause(context.Background(), time.Minute, time.Minute, false)
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	if head := chain.CurrentBlock(); uint64(res.Number) != head.Number.Uint64() || res.Hash != head.Hash() || res.Root != head.Root {
		t.Fatalf("Unexpected head of the pause: %d %x", res.Number, res.Hash)
	}
	if res.JournalSize == 0 || res.Paths["ancient"] == "" || res.Paths["journal"] == "" {
		t.Fatalf("Missing journal size or paths: %d %v", res.JournalSize, res.Paths)
	}
	if expires := time.UnixMilli(res.Expires); time.Until(expires) <= 0 || time.Until(expires) > time.Minute {
		t.Fatalf("Unexpected expiry: %v", expires)
	}
	// One pause at a time, and only its id resumes it.
	if _, err := cp.pause(context.Background(), time.Minute, time.Minute, false); !errors.Is(err, errClonePauseHeld) {
		t.Fatalf("Unexpected error pausing during a pause: %v", err)
	}
	if _, err := cp.resume("wrong"); !errors.Is(err, errCloneUnknown) {
		t.Fatalf("Unexpected error resuming an unknown pause: %v", err)
	}
	held(t, chain)

	done := importer(chain, blocks[1:2])
	resumed, err := cp.resume(res.ID)
	if err != nil {
		t.Fatalf("Failed to resume: %v", err)
	}
	if !resumed.Intact {
		t.Fatal("A pause resumed before its timeout isn't intact")
	}
	released(t, done)
	// A pause resumed twice is reported the same, and another one can be taken.
	if again, err := cp.resume(res.ID); err != nil || *again != *resumed {
		t.Fatalf("Unexpected second resume: %v %v", again, err)
	}
	next, err := cp.pause(context.Background(), time.Minute, time.Minute, false)
	if err != nil {
		t.Fatalf("Failed to pause again: %v", err)
	}
	if next.ID == res.ID || uint64(next.Number) != blocks[1].NumberU64() {
		t.Fatalf("Unexpected second pause: %s at %d", next.ID, next.Number)
	}
	// The id of the pause before doesn't end the new one.
	if again, err := cp.resume(res.ID); err != nil || *again != *resumed {
		t.Fatalf("Unexpected resume of the pause before: %v %v", again, err)
	}
	held(t, chain)
	done = importer(chain, blocks[2:])
	if _, err := cp.resume(next.ID); err != nil {
		t.Fatalf("Failed to resume: %v", err)
	}
	released(t, done)
}

func TestClonePauseExpiry(t *testing.T) {
	cp, chain, blocks := newClonePauseTester(t, 2)

	// The pause has a minute to get ready, and expires at once.
	res, err := cp.pause(context.Background(), time.Minute, time.Nanosecond, false)
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	// The import goes on once the pause expires, which then isn't intact.
	released(t, importer(chain, blocks[1:]))
	resumed, err := cp.resume(res.ID)
	if err != nil {
		t.Fatalf("Failed to resume an expired pause: %v", err)
	}
	if resumed.Intact {
		t.Fatal("An expired pause is intact")
	}
}

func TestClonePauseClose(t *testing.T) {
	cp, chain, blocks := newClonePauseTester(t, 2)

	res, err := cp.pause(context.Background(), time.Minute, time.Minute, false)
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	held(t, chain)

	done := importer(chain, blocks[1:])
	cp.close()
	released(t, done)
	if resumed, err := cp.resume(res.ID); err != nil || resumed.Intact {
		t.Fatalf("Unexpected resume after the shutdown: %v %v", resumed, err)
	}
	if _, err := cp.pause(context.Background(), time.Minute, time.Minute, false); !errors.Is(err, errCloneShutdown) {
		t.Fatalf("Unexpected error pausing at shutdown: %v", err)
	}
}

func TestClonePauseAfterHead(t *testing.T) {
	cp, chain, blocks := newClonePauseTester(t, 2)

	// Without a new head, the wait ends with the timeout.
	if _, err := cp.pause(context.Background(), time.Nanosecond, time.Minute, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Unexpected error waiting for a head: %v", err)
	}
	// The pause is taken at the next head. The wait for it reaches its context
	// once it is subscribed to the heads.
	parent, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var (
		ctx   = &waitingContext{Context: parent, waiting: make(chan struct{})}
		taken = make(chan error, 1)
		pause *core.ClonePause
	)
	go func() {
		var err error
		pause, err = cp.take(ctx, true)
		taken <- err
	}()
	select {
	case <-ctx.waiting:
	case err := <-taken:
		t.Fatalf("Paused without waiting for a head, err: %v", err)
	}
	if _, err := chain.InsertChain(blocks[1:]); err != nil {
		t.Fatalf("Failed to import: %v", err)
	}
	if err := <-taken; err != nil {
		t.Fatalf("Failed to pause after a head: %v", err)
	}
	defer pause.Resume()
	if pause.Head.Hash() != blocks[1].Hash() {
		t.Fatalf("Unexpected head of the pause: %d, want %d", pause.Head.Number, blocks[1].NumberU64())
	}
}

func TestClonePauseIPCOnly(t *testing.T) {
	cp, _, _ := newClonePauseTester(t, 1)

	server := rpc.NewServer()
	defer server.Stop()
	if err := server.RegisterName("admin", NewAdminAPI(&Ethereum{clonePauser: cp})); err != nil {
		t.Fatalf("Failed to register the API: %v", err)
	}
	// Refused over HTTP.
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	client, err := rpc.Dial(httpServer.URL)
	if err != nil {
		t.Fatalf("Failed to dial over HTTP: %v", err)
	}
	defer client.Close()
	var res ClonePauseResult
	if err := client.Call(&res, "admin_pauseForClone", nil); err == nil || !strings.Contains(err.Error(), "only served over IPC") {
		t.Fatalf("Unexpected pause over HTTP: %v", err)
	}
	// Served in process, as over IPC.
	inproc := rpc.DialInProc(server)
	defer inproc.Close()
	if err := inproc.Call(&res, "admin_pauseForClone", map[string]any{"timeout": "1m"}); err != nil {
		t.Fatalf("Failed to pause in process: %v", err)
	}
	if err := inproc.Call(&res, "admin_pauseForClone", map[string]any{"timeout": "3m"}); err == nil {
		t.Fatal("A timeout above the limit is accepted")
	}
	var resumed CloneResumeResult
	if err := client.Call(&resumed, "admin_resumeAfterClone", res.ID); err == nil || !strings.Contains(err.Error(), "only served over IPC") {
		t.Fatalf("Unexpected resume over HTTP: %v", err)
	}
	if err := inproc.Call(&resumed, "admin_resumeAfterClone", res.ID); err != nil || !resumed.Intact {
		t.Fatalf("Failed to resume in process: %v %v", resumed, err)
	}
}

func TestClonePaths(t *testing.T) {
	db, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	defer db.Close()

	// The era store is reported where the chain freezer opens it, if configured.
	var (
		ancient, _ = db.AncientDatadir()
		chain      = filepath.Join(ancient, rawdb.ChainFreezerName)
		elsewhere  = filepath.Join(t.TempDir(), "era")
		escaping   = filepath.Join("..", "..", "era")
	)
	for era, want := range map[string]string{"": "", "era": filepath.Join(chain, "era"), elsewhere: elsewhere, escaping: filepath.Join(chain, escaping)} {
		paths := clonePaths("chaindata", db, "triedb", era)
		if paths["chaindata"] != "chaindata" || paths["journal"] != "triedb" || paths["ancient"] != ancient || paths["era"] != want {
			t.Fatalf("Unexpected paths with the era store at %q: %v", era, paths)
		}
	}
}

func TestClonePauseNoDatadir(t *testing.T) {
	_, chain, _ := newClonePauseTester(t, 1)

	db, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	defer db.Close()
	cp := newClonePauser(chain, clonePaths("", db, "", ""))
	if _, err := cp.pause(context.Background(), time.Minute, time.Minute, false); !errors.Is(err, errCloneNoDatadir) {
		t.Fatalf("Unexpected error pausing a node without a data directory: %v", err)
	}
}
