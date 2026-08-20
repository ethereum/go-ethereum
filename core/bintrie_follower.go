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
	"github.com/ethereum/go-ethereum/core/types/bal"
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
	pbt    bool // the tree this follower maintains: the canonical one's opposite

	mu         sync.Mutex
	shadow     *triedb.Database // opened lazily; nil while idle
	sdb        state.Database
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
func newBintrieFollower(chain *BlockChain, pbt bool) *bintrieFollower {
	f := &bintrieFollower{
		db:     chain.db,
		config: chain.chainConfig,
		cfg:    chain.cfg,
		pbt:    pbt,
		chain:  chain,
		headCh: make(chan ChainHeadEvent),
		term:   make(chan chan struct{}),
		closed: make(chan struct{}),
	}
	f.headSub = chain.SubscribeChainHeadEvent(f.headCh)
	go f.loop()
	return f
}

// loop runs one sync at a time. A head arriving mid-run re-arms the sync on
// completion - the shadow must not park one event behind the tip - but a run
// that could not progress does not re-arm itself, or an idle wait (snap sync,
// canonical shorter than the head) becomes a busy loop.
func (f *bintrieFollower) loop() {
	defer close(f.closed)
	defer f.headSub.Unsubscribe()

	var (
		latest   *types.Header
		launched *types.Header
		stop     chan struct{}
		done     chan struct{}
	)
	launch := func(head *types.Header) {
		launched = head
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
			// Once a post-fork block finalizes, the window closes: the other
			// tree stops being maintained and may be disposed of.
			if final := f.chain.CurrentFinalBlock(); final != nil && f.config.IsBinaryTrie(final.Number, final.Time) {
				if done != nil {
					close(stop)
					<-done
				}
				rawdb.WritePBTMigrationDone(f.db)
				log.Info("State migration finished", "finalized", final.Number)
				return
			}
			latest = ev.Header
			if done == nil {
				launch(latest)
			}
		case <-done:
			stop, done = nil, nil
			f.mu.Lock()
			stalled := f.stall
			f.mu.Unlock()
			if stalled == nil && latest != nil && latest.Hash() != launched.Hash() {
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

// sync is one follow attempt; the outcome is latched for reporting. A stall
// retries on the next head event - a missing access list may have arrived.
func (f *bintrieFollower) sync(head *types.Header, stop chan struct{}) {
	err := f.follow(head, stop)
	f.mu.Lock()
	f.stall = err
	f.mu.Unlock()
	if err != nil {
		log.Warn("Binary tree shadow stalled", "head", head.Number, "err", err)
	}
}

// follow advances the shadow to the given head along the canonical chain:
// resolve where the shadow stands, walk back across a reorg, fold ranges
// while far behind, and replay block by block near the tip. Every replayed
// block is proven to chain onto the previous one and its access list is
// proven against the header's commitment - a wrong shadow root would
// otherwise surface only at the fork.
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
			if r, ok := f.replayedRoot(uint64(n), ch); ok {
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
	prev := hash
	for n := num + 1; n <= head.Number.Uint64(); n++ {
		if interrupted(stop) {
			return nil
		}
		// A snap sync starting mid-run rewrites the chain under the replay.
		if rawdb.ReadSnapSyncStatusFlag(f.db) == rawdb.StateSyncRunning {
			return nil
		}
		if behind := head.Number.Uint64() - n; behind > followTrackWindow {
			consumed, newRoot, newPrev, err := f.replayBatch(n, head.Number.Uint64(), root, prev)
			if err != nil {
				return err
			}
			if consumed == 0 {
				return nil // canonical moved under the run; retry on the next head
			}
			n += consumed - 1
			root, prev = newRoot, newPrev
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
		// The canonical index is rewritten under a mid-run reorg; splicing
		// two branches must stop the walk, never replay across it.
		if header.ParentHash != prev {
			return nil
		}
		if f.config.IsBinaryTrie(header.Number, header.Time) == f.pbt {
			if f.pbt {
				// Activation: execution commits this tree from here on; the
				// follower's forward duty ends at the boundary.
				return nil
			}
			// The header commits this tree already; carry its root along.
			root, prev = header.Root, hash
			f.setCursor(n, hash, header.Root)
			continue
		}
		list, err := f.readVerifiedList(header)
		if err != nil {
			return err
		}
		newRoot, err := replayAccessList(f.sdb, f.config, root, header.Number, header.Time, list)
		if err != nil {
			return fmt.Errorf("replaying block %d %x: %w", n, hash, err)
		}
		root, prev = newRoot, hash
		rawdb.WriteShadowStateRoot(f.db, n, hash, root)
		rawdb.WritePBTMigrationCursor(f.db, n, hash, root)
		f.setCursor(n, hash, root)
	}
	return nil
}

// readVerifiedList loads a block's access list and proves it against the
// header's commitment - the only integrity check the shadow gets before the
// fork validates its lineage.
func (f *bintrieFollower) readVerifiedList(header *types.Header) (*bal.BlockAccessList, error) {
	number := header.Number.Uint64()
	if header.BlockAccessListHash == nil {
		return nil, fmt.Errorf("block %d %x commits no access list: anchor past their activation", number, header.Hash())
	}
	list := rawdb.ReadAccessList(f.db, header.Hash(), number)
	if list == nil {
		return nil, fmt.Errorf("access list missing for block %d %x", number, header.Hash())
	}
	if h := list.Hash(); h != *header.BlockAccessListHash {
		return nil, fmt.Errorf("access list mismatch for block %d %x: have %x, header commits %x", number, header.Hash(), h, *header.BlockAccessListHash)
	}
	return list, nil
}

// replayBatch folds up to followBatchBlocks starting at from into end-state
// commits, staying outside the tracking window; the layers are flattened so a
// long catch-up holds bounded memory. Only the batch end gets a root record,
// written before the flatten: a crash between them re-replays, the other
// order strands a live root no record names.
func (f *bintrieFollower) replayBatch(from, head uint64, root common.Hash, prev common.Hash) (uint64, common.Hash, common.Hash, error) {
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
			return 0, common.Hash{}, common.Hash{}, fmt.Errorf("missing canonical header %d %x", n, hash)
		}
		if header.ParentHash != prev {
			break // reorged mid-collection; the next run re-walks
		}
		if f.config.IsBinaryTrie(header.Number, header.Time) == f.pbt {
			break // the per-block loop owns flavour boundaries
		}
		list, err := f.readVerifiedList(header)
		if err != nil {
			return 0, common.Hash{}, common.Hash{}, err
		}
		blocks = append(blocks, replayBlock{number: header.Number, time: header.Time, list: list})
		hashes = append(hashes, hash)
		prev = hash
	}
	if len(blocks) == 0 {
		return 0, root, prev, nil // canonical moved under the run; retry later
	}
	start := root
	for rest, taken := blocks, 0; len(rest) > 0; rest = rest[taken:] {
		var (
			next common.Hash
			err  error
		)
		next, taken, err = replayRange(f.sdb, f.config, root, rest)
		if err != nil {
			return 0, common.Hash{}, common.Hash{}, err
		}
		if taken == 0 {
			return 0, common.Hash{}, common.Hash{}, errors.New("fold made no progress")
		}
		root = next
	}
	var (
		end     = uint64(len(blocks) - 1)
		endNum  = from + end
		endHash = hashes[end]
	)
	rawdb.WriteShadowStateRoot(f.db, endNum, endHash, root)
	rawdb.WritePBTMigrationCursor(f.db, endNum, endHash, root)
	f.setCursor(endNum, endHash, root)
	if root != start {
		if err := f.shadow.Commit(root, false); err != nil {
			return 0, common.Hash{}, common.Hash{}, err
		}
	}
	return uint64(len(blocks)), root, endHash, nil
}

// openTree opens the follower's tree handle if it is not open yet, and
// returns it - the chain's flavour dispatch reaches the other tree this way.
func (f *bintrieFollower) openTree() (*triedb.Database, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.shadow != nil {
		return f.shadow, nil
	}
	tdbConfig, err := f.cfg.triedbConfig(f.pbt)
	if err != nil {
		return nil, err
	}
	shadow := triedb.NewDatabase(f.db, tdbConfig)
	f.shadow = shadow
	if f.pbt {
		f.sdb = state.NewPBTDatabase(shadow, nil)
	} else {
		f.sdb = state.NewMPTDatabase(shadow, nil)
	}
	return shadow, nil
}

// ensureShadow opens the shadow database and resolves the replay position.
// With a cursor ever written, the position must resolve through it or the
// records - the seeding fallbacks read pathdb's current disk root, and past
// the first replay that root's block is a guess; binding it wrongly poisons
// every record from there on.
func (f *bintrieFollower) ensureShadow() error {
	f.mu.Lock()
	resolved := f.shadow != nil && (f.cursorHash != common.Hash{})
	f.mu.Unlock()
	if resolved {
		return nil
	}
	shadow, err := f.openTree()
	if err != nil {
		return err
	}

	if rawdb.HasPBTMigrationCursor(f.db) {
		num, hash, root, ok := rawdb.ReadPBTMigrationCursor(f.db)
		if !ok {
			return errors.New("migration cursor is corrupt: re-anchor with a conversion artifact")
		}
		if f.hasState(root) {
			f.setCursor(num, hash, root)
			return nil
		}
		// A batch crashing between its record and the cursor write leaves
		// the durable root recorded above the hint; scan one batch up first,
		// then walk down to the deepest live record.
		for n := num + 1; n <= num+followBatchBlocks; n++ {
			ch := rawdb.ReadCanonicalHash(f.db, n)
			if r, ok := rawdb.ReadShadowStateRoot(f.db, n, ch); ok && f.hasState(r) {
				f.setCursor(n, ch, r)
				return nil
			}
		}
		for n := int64(num) - 1; n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r, ok := f.replayedRoot(uint64(n), ch); ok {
				f.setCursor(uint64(n), ch, r)
				return nil
			}
		}
		return errors.New("shadow position unresolvable: re-anchor with a conversion artifact")
	}
	if !f.pbt {
		// The merkle window follower has no artifacts: its floor is the
		// newest block whose header commits its tree with the state alive.
		head := rawdb.ReadHeadHeader(f.db)
		if head == nil {
			return errors.New("missing head header")
		}
		for n := int64(head.Number.Uint64()); n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r, ok := f.replayedRoot(uint64(n), ch); ok {
				f.setCursor(uint64(n), ch, r)
				return nil
			}
		}
		return errors.New("merkle window position unresolvable: no live merkle state")
	}
	pbtdb := rawdb.NewTable(f.db, string(rawdb.PBTPrefix))
	if num, hash, ok := rawdb.ReadPBTAnchor(pbtdb); ok {
		// Virgin import: the follower never ran, so the disk root is the
		// anchor's state, and both are proven before use.
		if rawdb.ReadCanonicalHash(f.db, num) != hash {
			return errors.New("imported anchor is not canonical: re-import a conversion artifact")
		}
		root := rawdb.ReadSnapshotRoot(pbtdb)
		if !f.hasState(root) {
			return errors.New("imported anchor state is gone: re-import a conversion artifact")
		}
		f.setCursor(num, hash, root)
		return nil
	}
	ghash := rawdb.ReadCanonicalHash(f.db, 0)
	if root := rawdb.ReadSnapshotRoot(pbtdb); root != (common.Hash{}) {
		// Seeded, then crashed before the first record: only the genesis
		// seed writes the namespace with no cursor ever recorded.
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
	root, err := flushAlloc(&alloc, shadow, nil)
	if err != nil {
		return err
	}
	rawdb.WriteShadowStateRoot(f.db, 0, ghash, root)
	rawdb.WritePBTMigrationCursor(f.db, 0, ghash, root)
	f.setCursor(0, ghash, root)
	log.Info("Seeded binary tree shadow from genesis", "root", root)
	return nil
}

// replayedRoot returns the follower-tree root of the given canonical block if
// the tree still holds it: the recorded shadow root, or - for a block whose
// header commits this tree - the header root itself.
func (f *bintrieFollower) replayedRoot(number uint64, hash common.Hash) (common.Hash, bool) {
	if r, ok := rawdb.ReadShadowStateRoot(f.db, number, hash); ok && f.hasState(r) {
		return r, true
	}
	if header := rawdb.ReadHeader(f.db, hash, number); header != nil {
		if f.config.IsBinaryTrie(header.Number, header.Time) == f.pbt && f.hasState(header.Root) {
			return header.Root, true
		}
	}
	return common.Hash{}, false
}

// waitCaughtUp blocks until the follower has recorded the given block's
// shadow root, it stalls, or the timeout passes. Blocks folded over by a
// batch never get a record; callers wait on tracked blocks near the tip.
func (f *bintrieFollower) waitCaughtUp(number uint64, hash common.Hash, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, ok := rawdb.ReadShadowStateRoot(f.db, number, hash); ok {
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
// not the chain head's, which belongs to the other tree - and releases it. A
// failed journal is survivable: the next start walks the records back to the
// disk layer and re-replays.
func (f *bintrieFollower) journal() {
	f.mu.Lock()
	shadow, root := f.shadow, f.cursorRoot
	f.mu.Unlock()
	if shadow == nil {
		return
	}
	if err := shadow.Journal(root); err != nil {
		log.Warn("Failed to journal shadow trie; restart will re-replay", "err", err)
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
	f.mu.Lock()
	shadow := f.shadow
	f.mu.Unlock()
	_, err := shadow.NodeReader(root)
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
