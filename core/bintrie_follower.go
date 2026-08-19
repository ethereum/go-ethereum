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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

const (
	// followTrackWindow is how close to the head the follower replays block
	// by block with per-hash root records; further out it folds ranges.
	followTrackWindow = 128

	// followBatchBlocks caps how many blocks one fold coalesces.
	followBatchBlocks = 128
)

// bintrieFollower advances the shadow binary tree of a migrating chain by
// replaying block access lists, so the tree reaches the fork already at the
// tip. It follows the canonical chain only, and the shadow database itself is
// the truth for where it stands: the persisted cursor is a resume hint that a
// crash may leave ahead of the durable state.
type bintrieFollower struct {
	db     ethdb.Database
	config *params.ChainConfig
	cfg    *BlockChainConfig

	shadow *triedb.Database // opened lazily; nil while idle
	sdb    state.Database

	mu         sync.Mutex
	cursorNum  uint64
	cursorHash common.Hash
	cursorRoot common.Hash
	stall      error

	chain   *BlockChain // nil when driven synchronously in tests
	headCh  chan ChainHeadEvent
	headSub event.Subscription
	term    chan chan struct{}
	closed  chan struct{}
}

// newBintrieFollower constructs the follower and starts its event loop.
func newBintrieFollower(chain *BlockChain) *bintrieFollower {
	f := &bintrieFollower{
		db:     chain.db,
		config: chain.chainConfig,
		cfg:    chain.cfg,
		chain:  chain,
		headCh: make(chan ChainHeadEvent),
		term:   make(chan chan struct{}),
		closed: make(chan struct{}),
	}
	f.headSub = chain.SubscribeChainHeadEvent(f.headCh)
	go f.loop()
	return f
}

// loop runs one sync at a time, re-arming when the head moved past what the
// finished run saw - heads arriving mid-run must not be dropped, or the
// shadow parks one event behind the tip.
func (f *bintrieFollower) loop() {
	defer close(f.closed)
	defer f.headSub.Unsubscribe()

	var (
		latest *types.Header
		stop   chan struct{}
		done   chan struct{}
	)
	launch := func(head *types.Header) {
		stop = make(chan struct{})
		done = make(chan struct{})
		go func() {
			defer close(done)
			f.sync(head, stop)
		}()
	}
	if head := f.chain.CurrentBlock(); head != nil {
		latest = head
		launch(head)
	}
	for {
		select {
		case ev := <-f.headCh:
			latest = ev.Header
			if done == nil {
				launch(latest)
			}
		case <-done:
			stop, done = nil, nil
			f.mu.Lock()
			hash, stalled := f.cursorHash, f.stall
			f.mu.Unlock()
			if stalled == nil && latest != nil && hash != latest.Hash() {
				launch(latest)
			}
		case ch := <-f.term:
			if done != nil {
				close(stop)
				<-done
			}
			close(ch)
			return
		}
	}
}

// close terminates the event loop and waits for a running sync to bail out.
func (f *bintrieFollower) close() {
	ch := make(chan struct{})
	select {
	case f.term <- ch:
		<-ch
	case <-f.closed:
	}
}

// sync is one follow attempt with the stall latched for reporting; the next
// head event retries, since a missing access list may have arrived meanwhile.
func (f *bintrieFollower) sync(head *types.Header, stop chan struct{}) {
	f.mu.Lock()
	f.stall = nil
	f.mu.Unlock()

	if err := f.follow(head, stop); err != nil {
		f.mu.Lock()
		f.stall = err
		f.mu.Unlock()
		log.Warn("Binary tree shadow stalled", "head", head.Number, "err", err)
	}
}

// follow advances the shadow to the given head along the canonical chain:
// resolve where the shadow stands, walk back across a reorg, fold ranges
// while far behind, and replay block by block near the tip.
func (f *bintrieFollower) follow(head *types.Header, stop chan struct{}) error {
	// While the merkle side is snap-syncing there is nothing consistent to
	// replay from, and a shadow opened now would be disabled by the same
	// flag. Stay idle; a later head retries.
	if rawdb.ReadSnapSyncStatusFlag(f.db) == rawdb.StateSyncRunning {
		return nil
	}
	if err := f.ensureShadow(); err != nil {
		return err
	}
	num, hash, root := f.cursor()

	// The cursor must sit on the canonical chain; otherwise the canonical
	// branch switched and replay restarts from the fork point. Nothing is
	// unwound: the stale layers stay in the tree until flattened away.
	if rawdb.ReadCanonicalHash(f.db, num) != hash {
		var found bool
		for n := int64(num) - 1; n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r := rawdb.ReadShadowStateRoot(f.db, uint64(n), ch); r != (common.Hash{}) && f.hasState(r) {
				num, hash, root, found = uint64(n), ch, r, true
				break
			}
		}
		if !found {
			return errors.New("shadow reorged past its live states: re-anchor with a conversion artifact")
		}
		log.Info("Binary tree shadow following reorg", "forkpoint", num)
		f.setCursor(num, hash, root)
	}
	for n := num + 1; n <= head.Number.Uint64(); n++ {
		if interrupted(stop) {
			return nil
		}
		if behind := head.Number.Uint64() - n; behind > followTrackWindow {
			consumed, newRoot, err := f.replayBatch(n, head.Number.Uint64(), root)
			if err != nil {
				return err
			}
			n += consumed - 1
			root = newRoot
			continue
		}
		hash := rawdb.ReadCanonicalHash(f.db, n)
		if hash == (common.Hash{}) {
			return nil // canonical chain shorter than the head; retry later
		}
		header := rawdb.ReadHeader(f.db, hash, n)
		if header == nil {
			return fmt.Errorf("missing canonical header %d %x", n, hash)
		}
		list := rawdb.ReadAccessList(f.db, hash, n)
		if list == nil && header.BlockAccessListHash != nil {
			return fmt.Errorf("access list missing for block %d %x", n, hash)
		}
		newRoot, err := replayAccessList(f.sdb, f.config, root, header.Number, header.Time, list)
		if err != nil {
			return fmt.Errorf("replaying block %d %x: %w", n, hash, err)
		}
		root = newRoot
		rawdb.WriteShadowStateRoot(f.db, n, hash, root)
		rawdb.WritePBTMigrationCursor(f.db, n, hash, root)
		f.setCursor(n, hash, root)
	}
	return nil
}

// replayBatch folds up to followBatchBlocks starting at from into end-state
// commits, staying outside the tracking window; the layers are flattened so a
// long catch-up holds bounded memory. Only the batch end gets a root record.
func (f *bintrieFollower) replayBatch(from, head uint64, root common.Hash) (uint64, common.Hash, error) {
	var (
		blocks []replayBlock
		hashes []common.Hash
	)
	for n := from; n <= head && head-n > followTrackWindow && len(blocks) < followBatchBlocks; n++ {
		hash := rawdb.ReadCanonicalHash(f.db, n)
		if hash == (common.Hash{}) {
			break
		}
		header := rawdb.ReadHeader(f.db, hash, n)
		if header == nil {
			return 0, common.Hash{}, fmt.Errorf("missing canonical header %d %x", n, hash)
		}
		list := rawdb.ReadAccessList(f.db, hash, n)
		if list == nil && header.BlockAccessListHash != nil {
			return 0, common.Hash{}, fmt.Errorf("access list missing for block %d %x", n, hash)
		}
		blocks = append(blocks, replayBlock{number: header.Number, time: header.Time, list: list})
		hashes = append(hashes, hash)
	}
	if len(blocks) == 0 {
		return 0, common.Hash{}, fmt.Errorf("no canonical blocks to batch at %d", from)
	}
	var (
		start = root
		taken int
	)
	for rest := blocks; len(rest) > 0; rest = rest[taken:] {
		var (
			next common.Hash
			err  error
		)
		next, taken, err = replayRange(f.sdb, f.config, root, rest)
		if err != nil {
			return 0, common.Hash{}, err
		}
		root = next
	}
	// Flatten the batch into the disk layer; without this a deep catch-up
	// accumulates one layer per fold.
	if root != start {
		if err := f.shadow.Commit(root, false); err != nil {
			return 0, common.Hash{}, err
		}
	}
	var (
		end     = uint64(len(blocks) - 1)
		endNum  = from + end
		endHash = hashes[end]
	)
	rawdb.WriteShadowStateRoot(f.db, endNum, endHash, root)
	rawdb.WritePBTMigrationCursor(f.db, endNum, endHash, root)
	f.setCursor(endNum, endHash, root)
	return uint64(len(blocks)), root, nil
}

// ensureShadow opens the shadow database and resolves the replay position:
// the cursor hint if its state is live, the deepest recorded live root below
// it otherwise, an imported anchor, or a fresh genesis seed.
func (f *bintrieFollower) ensureShadow() error {
	if f.shadow != nil {
		return nil
	}
	tdbConfig, err := f.cfg.triedbConfig(true)
	if err != nil {
		return err
	}
	shadow := triedb.NewDatabase(f.db, tdbConfig)
	f.shadow = shadow
	f.sdb = state.NewPBTDatabase(shadow, nil)

	// The hint is only a hint: a crash loses the in-memory layers, and the
	// durable shadow may sit behind it. Walk the records back to a live root.
	if num, hash, root, ok := rawdb.ReadPBTMigrationCursor(f.db); ok {
		if f.hasState(root) {
			f.setCursor(num, hash, root)
			return nil
		}
		for n := int64(num) - 1; n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r := rawdb.ReadShadowStateRoot(f.db, uint64(n), ch); r != (common.Hash{}) && f.hasState(r) {
				f.setCursor(uint64(n), ch, r)
				return nil
			}
		}
	}
	pbtdb := rawdb.NewTable(f.db, string(rawdb.PBTPrefix))
	if num, hash, ok := rawdb.ReadPBTAnchor(pbtdb); ok {
		// Virgin import: the disk root is the anchor's state. Once replay has
		// run, the records above resolve first.
		if rawdb.ReadCanonicalHash(f.db, num) != hash {
			return errors.New("imported anchor is not canonical: re-import a conversion artifact")
		}
		f.setCursor(num, hash, rawdb.ReadSnapshotRoot(pbtdb))
		return nil
	}
	ghash := rawdb.ReadCanonicalHash(f.db, 0)
	if root := rawdb.ReadSnapshotRoot(pbtdb); root != (common.Hash{}) {
		// Seeded but no cursor: the crash window right after the genesis
		// flush. Only the seed writes the namespace without a cursor.
		f.setCursor(0, ghash, root)
		return nil
	}
	// Fresh namespace: seed from the genesis allocation. That only follows a
	// chain whose every block carries an access list.
	genesis := rawdb.ReadHeader(f.db, ghash, 0)
	if genesis == nil {
		return errors.New("missing genesis header")
	}
	if !f.config.IsAmsterdam(genesis.Number, genesis.Time) {
		return errors.New("chain predates access lists: seed the shadow from a conversion artifact (geth bintrie import)")
	}
	alloc, err := getGenesisState(f.db, ghash)
	if err != nil {
		return err
	}
	if alloc == nil {
		return errors.New("genesis allocation unavailable: seed the shadow from a conversion artifact")
	}
	root, err := flushAlloc(&alloc, f.shadow, nil)
	if err != nil {
		return err
	}
	rawdb.WriteShadowStateRoot(f.db, 0, ghash, root)
	rawdb.WritePBTMigrationCursor(f.db, 0, ghash, root)
	f.setCursor(0, ghash, root)
	log.Info("Seeded binary tree shadow from genesis", "root", root)
	return nil
}

// waitCaughtUp blocks until the follower has recorded the given block's
// shadow root, it stalls, or the timeout passes. Blocks folded over by a
// batch never get a record; callers wait on tracked blocks near the tip.
func (f *bintrieFollower) waitCaughtUp(number uint64, hash common.Hash, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if r := rawdb.ReadShadowStateRoot(f.db, number, hash); r != (common.Hash{}) {
			return nil
		}
		f.mu.Lock()
		stall := f.stall
		f.mu.Unlock()
		if stall != nil {
			return stall
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("shadow has not reached block %d %x", number, hash)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// journal persists the shadow's in-memory layers at the follower's own root -
// not the chain head's, which belongs to the other tree - and releases it.
func (f *bintrieFollower) journal() {
	f.mu.Lock()
	shadow, root := f.shadow, f.cursorRoot
	f.mu.Unlock()
	if shadow == nil {
		return
	}
	if err := shadow.Journal(root); err != nil {
		log.Info("Failed to journal shadow trie", "err", err)
	}
	if err := shadow.Close(); err != nil {
		log.Error("Failed to close shadow trie", "err", err)
	}
}

func (f *bintrieFollower) cursor() (uint64, common.Hash, common.Hash) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cursorNum, f.cursorHash, f.cursorRoot
}

func (f *bintrieFollower) setCursor(num uint64, hash common.Hash, root common.Hash) {
	f.mu.Lock()
	f.cursorNum, f.cursorHash, f.cursorRoot = num, hash, root
	f.mu.Unlock()
}

// hasState reports whether the shadow still holds the given root, in memory
// or on disk.
func (f *bintrieFollower) hasState(root common.Hash) bool {
	_, err := f.shadow.NodeReader(root)
	return err == nil
}

func interrupted(stop chan struct{}) bool {
	if stop == nil {
		return false
	}
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

// MigrationProgress reports where a migrating chain's shadow tree stands.
type MigrationProgress struct {
	Phase      string      // inactive, idle, following, synced or stalled
	Cursor     uint64      // last replayed block number
	CursorHash common.Hash // last replayed block hash
	ShadowRoot common.Hash // the shadow root of that block
	Error      string      // what stalled the follower, if anything
}

// MigrationProgress reports the shadow follower's position, or an inactive
// report when the chain is not migrating.
func (bc *BlockChain) MigrationProgress() MigrationProgress {
	if bc.follower == nil {
		return MigrationProgress{Phase: "inactive"}
	}
	var head uint64
	if h := bc.CurrentBlock(); h != nil {
		head = h.Number.Uint64()
	}
	return bc.follower.progress(head)
}

func (f *bintrieFollower) progress(head uint64) MigrationProgress {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := MigrationProgress{
		Phase:      "following",
		Cursor:     f.cursorNum,
		CursorHash: f.cursorHash,
		ShadowRoot: f.cursorRoot,
	}
	switch {
	case f.stall != nil:
		p.Phase, p.Error = "stalled", f.stall.Error()
	case f.shadow == nil:
		p.Phase = "idle"
	case f.cursorNum >= head:
		p.Phase = "synced"
	}
	return p
}
