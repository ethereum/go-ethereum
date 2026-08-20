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

// MissingAccessListError names a canonical block whose access list is not
// stored; replay stalls on it until the list arrives.
type MissingAccessListError struct {
	Number uint64
	Hash   common.Hash
}

func (e *MissingAccessListError) Error() string {
	return fmt.Sprintf("access list missing for block %d %x", e.Number, e.Hash)
}

// BALRequest identifies a block whose access list the migration needs.
type BALRequest struct {
	Number uint64
	Hash   common.Hash
}

// bintrieFollower replays block access lists onto the shadow trees: the
// binary one ahead of the fork, the merkle one through the window after it.
// Each tree is the truth for where its replay stands; the persisted cursors
// are hints a crash may leave ahead of them.
type bintrieFollower struct {
	db     ethdb.Database
	config *params.ChainConfig
	cfg    *BlockChainConfig

	mu        sync.Mutex
	dirs      [2]*followerTree // binary then merkle; created on first use
	requester func([]BALRequest)

	chain   *BlockChain // nil when driven synchronously in tests
	headCh  chan ChainHeadEvent
	kickCh  chan struct{}
	headSub event.Subscription
	term    chan chan struct{}
	closed  chan struct{}
}

// followerTree is one direction of the migration: the tree it maintains and
// the replay position it stands at. Its fields are guarded by the follower's
// mutex.
type followerTree struct {
	f     *bintrieFollower
	pbt   bool
	owned bool // opened by the follower: journaled, closed and batched on

	handle     *triedb.Database // opened lazily; nil while idle
	sdb        state.Database
	cursorNum  uint64
	cursorHash common.Hash
	cursorRoot common.Hash
	stall      error
}

func newBintrieFollower(chain *BlockChain) *bintrieFollower {
	f := &bintrieFollower{
		db:     chain.db,
		config: chain.chainConfig,
		cfg:    chain.cfg,
		chain:  chain,
		headCh: make(chan ChainHeadEvent),
		kickCh: make(chan struct{}, 1),
		term:   make(chan chan struct{}),
		closed: make(chan struct{}),
	}
	f.headSub = chain.SubscribeChainHeadEvent(f.headCh)
	go f.loop()
	return f
}

func dirIndex(pbt bool) int {
	if pbt {
		return 0
	}
	return 1
}

// direction returns the tree of the given flavour, creating it on first use.
func (f *bintrieFollower) direction(pbt bool) *followerTree {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i := dirIndex(pbt); f.dirs[i] != nil {
		return f.dirs[i]
	}
	t := &followerTree{f: f, pbt: pbt}
	f.dirs[dirIndex(pbt)] = t
	return t
}

// peek returns the tree of the given flavour if it exists.
func (f *bintrieFollower) peek(pbt bool) *followerTree {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dirs[dirIndex(pbt)]
}

// live returns the directions created so far, binary first.
func (f *bintrieFollower) live() []*followerTree {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*followerTree
	for _, t := range f.dirs {
		if t != nil {
			out = append(out, t)
		}
	}
	return out
}

// loop runs one sync at a time. A head arriving mid-run re-arms on
// completion; a run that could not progress must not, or an idle wait spins.
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
		kickCh := f.kickCh
		if done != nil {
			kickCh = nil // the running sync's completion re-arms
		}
		select {
		case <-kickCh:
			if latest != nil {
				launch(latest)
			}
		case ev := <-f.headCh:
			// A closed window stops maintaining the merkle tree for good; it
			// may be disposed of, and a deeper reorg afterwards means
			// re-anchoring.
			if closer := f.windowClose(ev.Header); closer != nil {
				if done != nil {
					close(stop)
					<-done
				}
				if m := f.peek(false); m == nil {
					log.Warn("Merkle window closing without ever running", "closed", closer.Number)
				} else if num, _, _ := m.cursor(); num < closer.Number.Uint64() {
					log.Warn("Merkle window closing behind", "cursor", num, "closed", closer.Number)
				}
				rawdb.WritePBTMigrationDone(f.db)
				log.Info("State migration finished", "closed", closer.Number)
				return
			}
			latest = ev.Header
			if done == nil {
				launch(latest)
			}
		case <-done:
			stop, done = nil, nil
			if !f.allStalled() && latest != nil && latest.Hash() != launched.Hash() {
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

// windowClose returns the block closing the window at this head, if any: a
// finalized post-fork block, or the head once MigrationWindowBlocks post-fork
// blocks stand under it.
func (f *bintrieFollower) windowClose(head *types.Header) *types.Header {
	if final := f.chain.CurrentFinalBlock(); final != nil && f.config.IsBinaryTrie(final.Number, final.Time) {
		return final
	}
	n := f.cfg.MigrationWindowBlocks
	if n == 0 || !f.config.IsBinaryTrie(head.Number, head.Time) {
		return nil
	}
	if head.Number.Uint64()-f.firstPostFork(head)+1 >= n {
		return head
	}
	return nil
}

// firstPostFork walks the canonical headers down to the activation boundary.
// The walk is bounded: the window closes before it exceeds the knob.
func (f *bintrieFollower) firstPostFork(head *types.Header) uint64 {
	h := head
	for h.Number.Uint64() > 0 {
		parent := f.chain.GetHeader(h.ParentHash, h.Number.Uint64()-1)
		if parent == nil || !f.config.IsBinaryTrie(parent.Number, parent.Time) {
			break
		}
		h = parent
	}
	return h.Number.Uint64()
}

func (f *bintrieFollower) close() {
	ch := make(chan struct{})
	select {
	case f.term <- ch:
		<-ch
	case <-f.closed:
	}
}

// sync runs every live direction once, ensuring the one the head calls for
// exists: the direction opposite the head's flavour does the replay work,
// its sibling only parks or rewinds. Outcomes latch; the next head retries.
func (f *bintrieFollower) sync(head *types.Header, stop chan struct{}) {
	f.direction(!f.config.IsBinaryTrie(head.Number, head.Time))
	for _, t := range f.live() {
		err := t.follow(head, stop)
		f.mu.Lock()
		t.stall = err
		f.mu.Unlock()
		if err != nil {
			log.Warn("Migration direction stalled", "pbt", t.pbt, "head", head.Number, "err", err)
			var missing *MissingAccessListError
			if errors.As(err, &missing) {
				f.requestLists(missing.Number, head.Number.Uint64())
			}
		}
	}
}

// requestLists hands the requester the absent lists from the given block up
// to the head, at most one batch; the requester must not block.
func (f *bintrieFollower) requestLists(from, head uint64) {
	f.mu.Lock()
	req := f.requester
	f.mu.Unlock()
	if req == nil {
		return
	}
	var reqs []BALRequest
	for n := from; n <= head && len(reqs) < followBatchBlocks; n++ {
		hash := rawdb.ReadCanonicalHash(f.db, n)
		if hash == (common.Hash{}) {
			break
		}
		if !rawdb.HasAccessList(f.db, hash, n) {
			reqs = append(reqs, BALRequest{Number: n, Hash: hash})
		}
	}
	if len(reqs) > 0 {
		req(reqs)
	}
}

func (f *bintrieFollower) setRequester(req func([]BALRequest)) {
	f.mu.Lock()
	f.requester = req
	f.mu.Unlock()
}

// kick re-arms a stalled sync, typically after new lists landed.
func (f *bintrieFollower) kick() {
	select {
	case f.kickCh <- struct{}{}:
	default:
	}
}

// allStalled reports whether every live direction stalled; an idle follower
// has not.
func (f *bintrieFollower) allStalled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	stalled := false
	for _, t := range f.dirs {
		if t == nil {
			continue
		}
		if t.stall == nil {
			return false
		}
		stalled = true
	}
	return stalled
}

// follow advances the direction the head calls for: the one opposite its
// flavour.
func (f *bintrieFollower) follow(head *types.Header, stop chan struct{}) error {
	return f.direction(!f.config.IsBinaryTrie(head.Number, head.Time)).follow(head, stop)
}

// follow advances the tree to the head along the canonical chain. Every
// replayed block is proven to chain onto the previous one and its access list
// against the header's commitment: a wrong root surfaces only at the fork.
func (t *followerTree) follow(head *types.Header, stop chan struct{}) error {
	f := t.f
	// While the merkle side snap-syncs there is nothing consistent to replay
	// from, and a shadow opened now would be disabled by the same flag.
	if rawdb.ReadSnapSyncStatusFlag(f.db) == rawdb.StateSyncRunning {
		return nil
	}
	if err := t.ensure(); err != nil {
		return err
	}
	num, hash, root := t.cursor()

	// The cursor must sit on the canonical chain; otherwise the canonical
	// branch switched and replay restarts from the fork point. Nothing is
	// unwound: the stale layers stay in the tree until flattened away.
	if rawdb.ReadCanonicalHash(f.db, num) != hash {
		var found bool
		for n := int64(num) - 1; n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r, ok := t.replayedRoot(uint64(n), ch); ok {
				num, hash, root, found = uint64(n), ch, r, true
				break
			}
		}
		if !found {
			return errors.New("shadow reorged past its live states: re-anchor")
		}
		log.Info("Shadow tree following reorg", "pbt", t.pbt, "forkpoint", num)
		t.setCursor(num, hash, root)
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
		if behind := head.Number.Uint64() - n; behind > followTrackWindow && t.owned {
			consumed, newRoot, newPrev, err := t.replayBatch(n, head.Number.Uint64(), root, prev)
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
		if f.config.IsBinaryTrie(header.Number, header.Time) == t.pbt {
			if t.pbt {
				// Activation: execution commits this tree from here on.
				return nil
			}
			root, prev = header.Root, hash
			t.setCursor(n, hash, header.Root)
			continue
		}
		list, err := f.readVerifiedList(header)
		if err != nil {
			return err
		}
		newRoot, err := replayAccessList(t.sdb, f.config, root, header.Number, header.Time, list)
		if err != nil {
			return fmt.Errorf("replaying block %d %x: %w", n, hash, err)
		}
		root, prev = newRoot, hash
		rawdb.WriteShadowStateRoot(f.db, n, hash, root)
		t.persistCursor(n, hash, root)
		t.setCursor(n, hash, root)
	}
	return nil
}

// readVerifiedList loads a block's access list and proves it against the
// header's commitment.
func (f *bintrieFollower) readVerifiedList(header *types.Header) (*bal.BlockAccessList, error) {
	number := header.Number.Uint64()
	if header.BlockAccessListHash == nil {
		return nil, fmt.Errorf("block %d %x commits no access list", number, header.Hash())
	}
	list := rawdb.ReadAccessList(f.db, header.Hash(), number)
	if list == nil {
		return nil, &MissingAccessListError{Number: number, Hash: header.Hash()}
	}
	if h := list.Hash(); h != *header.BlockAccessListHash {
		return nil, fmt.Errorf("access list mismatch for block %d %x: have %x, header commits %x", number, header.Hash(), h, *header.BlockAccessListHash)
	}
	return list, nil
}

// replayBatch folds ranges outside the tracking window and flattens them.
// Only the batch end gets a record, written before the flatten: a crash
// between them re-replays; the other order strands a root no record names.
func (t *followerTree) replayBatch(from, head uint64, root common.Hash, prev common.Hash) (uint64, common.Hash, common.Hash, error) {
	f := t.f
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
		if f.config.IsBinaryTrie(header.Number, header.Time) == t.pbt {
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
		next, taken, err = replayRange(t.sdb, f.config, root, rest)
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
	t.persistCursor(endNum, endHash, root)
	t.setCursor(endNum, endHash, root)
	if root != start {
		if err := t.handle.Commit(root, false); err != nil {
			return 0, common.Hash{}, common.Hash{}, err
		}
	}
	return uint64(len(blocks)), root, endHash, nil
}

// tree opens and returns the handle of the given flavour.
func (f *bintrieFollower) tree(pbt bool) (*triedb.Database, error) {
	return f.direction(pbt).open()
}

// open resolves the tree handle on first use: the canonical handle when the
// flavours match - a second instance over one namespace corrupts it - and a
// follower-owned one otherwise.
func (t *followerTree) open() (*triedb.Database, error) {
	f := t.f
	// An unfinished snap sync disables - or kills - a handle opened over it;
	// refuse until the flag clears, covering every probe, not just replay.
	if rawdb.ReadSnapSyncStatusFlag(f.db) == rawdb.StateSyncRunning {
		return nil, errors.New("snap sync running: shadow trees wait")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.handle != nil {
		return t.handle, nil
	}
	if f.chain != nil && f.chain.triedb.IsPBT() == t.pbt {
		t.handle = f.chain.triedb
	} else {
		tdbConfig, err := f.cfg.triedbConfig(t.pbt)
		if err != nil {
			return nil, err
		}
		t.handle = triedb.NewDatabase(f.db, tdbConfig)
		t.owned = true
	}
	if t.pbt {
		t.sdb = state.NewPBTDatabase(t.handle, nil)
	} else {
		t.sdb = state.NewMPTDatabase(t.handle, nil)
	}
	return t.handle, nil
}

// hasCursor reports whether the direction's cursor was ever persisted.
func (t *followerTree) hasCursor() bool {
	if t.pbt {
		return rawdb.HasPBTMigrationCursor(t.f.db)
	}
	return rawdb.HasMPTMigrationCursor(t.f.db)
}

// readCursor loads the direction's persisted cursor.
func (t *followerTree) readCursor() (uint64, common.Hash, common.Hash, bool) {
	if t.pbt {
		return rawdb.ReadPBTMigrationCursor(t.f.db)
	}
	return rawdb.ReadMPTMigrationCursor(t.f.db)
}

// persistCursor stores the direction's cursor.
func (t *followerTree) persistCursor(num uint64, hash common.Hash, root common.Hash) {
	if t.pbt {
		rawdb.WritePBTMigrationCursor(t.f.db, num, hash, root)
	} else {
		rawdb.WriteMPTMigrationCursor(t.f.db, num, hash, root)
	}
}

// ensure opens the tree and resolves the replay position. With a cursor ever
// written it must resolve through it or the records: the seeding fallbacks
// bind pathdb's disk root to a guessed block, poisoning the rest.
func (t *followerTree) ensure() error {
	f := t.f
	f.mu.Lock()
	resolved := t.handle != nil && (t.cursorHash != common.Hash{})
	f.mu.Unlock()
	if resolved {
		return nil
	}
	handle, err := t.open()
	if err != nil {
		return err
	}

	if t.hasCursor() {
		num, hash, root, ok := t.readCursor()
		if !ok {
			return errors.New("migration cursor corrupt: re-anchor")
		}
		if t.hasState(root) {
			t.setCursor(num, hash, root)
			return nil
		}
		// A batch crashing between its record and the cursor write leaves
		// the durable root recorded above the hint; scan one batch up first,
		// then walk down to the deepest live record.
		for n := num + 1; n <= num+followBatchBlocks; n++ {
			ch := rawdb.ReadCanonicalHash(f.db, n)
			if r, ok := rawdb.ReadShadowStateRoot(f.db, n, ch); ok && t.hasState(r) {
				t.setCursor(n, ch, r)
				return nil
			}
		}
		for n := int64(num) - 1; n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r, ok := t.replayedRoot(uint64(n), ch); ok {
				t.setCursor(uint64(n), ch, r)
				return nil
			}
		}
		return errors.New("shadow position unresolvable: re-anchor")
	}
	if !t.pbt {
		// The merkle window has no artifacts: its floor is the newest block
		// whose header commits its tree with the state alive.
		head := rawdb.ReadHeadHeader(f.db)
		if head == nil {
			return errors.New("missing head header")
		}
		for n := int64(head.Number.Uint64()); n >= 0; n-- {
			ch := rawdb.ReadCanonicalHash(f.db, uint64(n))
			if r, ok := t.replayedRoot(uint64(n), ch); ok {
				t.setCursor(uint64(n), ch, r)
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
			return errors.New("imported anchor not canonical: re-import")
		}
		root := rawdb.ReadSnapshotRoot(pbtdb)
		if !t.hasState(root) {
			return errors.New("imported anchor state gone: re-import")
		}
		t.setCursor(num, hash, root)
		return nil
	}
	ghash := rawdb.ReadCanonicalHash(f.db, 0)
	if root := rawdb.ReadSnapshotRoot(pbtdb); root != (common.Hash{}) {
		// Seeded, then crashed before the first record: only the genesis
		// seed writes the namespace with no cursor ever recorded.
		t.setCursor(0, ghash, root)
		return nil
	}
	// Fresh namespace: seed from the genesis allocation.
	genesis := rawdb.ReadHeader(f.db, ghash, 0)
	if genesis == nil {
		return errors.New("missing genesis header")
	}
	if !f.config.IsAmsterdam(genesis.Number, genesis.Time) {
		return errors.New("chain predates access lists: seed with geth bintrie import")
	}
	alloc, err := getGenesisState(f.db, ghash)
	if err != nil {
		return err
	}
	if alloc == nil {
		return errors.New("genesis allocation unavailable")
	}
	root, err := flushAlloc(&alloc, handle, nil)
	if err != nil {
		return err
	}
	rawdb.WriteShadowStateRoot(f.db, 0, ghash, root)
	t.persistCursor(0, ghash, root)
	t.setCursor(0, ghash, root)
	log.Info("Seeded binary tree shadow from genesis", "root", root)
	return nil
}

// replayedRoot returns the tree's root for the given block if the tree still
// holds it: the record, or an own-flavour block's header root.
func (t *followerTree) replayedRoot(number uint64, hash common.Hash) (common.Hash, bool) {
	f := t.f
	if r, ok := rawdb.ReadShadowStateRoot(f.db, number, hash); ok && t.hasState(r) {
		return r, true
	}
	if header := rawdb.ReadHeader(f.db, hash, number); header != nil {
		if f.config.IsBinaryTrie(header.Number, header.Time) == t.pbt && t.hasState(header.Root) {
			return header.Root, true
		}
	}
	return common.Hash{}, false
}

// waitCaughtUp blocks until the given block's root is recorded, a stall in
// the direction that would replay it, or the timeout. Batched-over blocks
// never get a record; wait near the tip.
func (f *bintrieFollower) waitCaughtUp(number uint64, hash common.Hash, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, ok := rawdb.ReadShadowStateRoot(f.db, number, hash); ok {
			return nil
		}
		if err := f.stalledFor(number, hash); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("shadow has not reached block %d %x", number, hash)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// stalledFor returns the stall of the direction responsible for replaying
// the given block, if any.
func (f *bintrieFollower) stalledFor(number uint64, hash common.Hash) error {
	header := rawdb.ReadHeader(f.db, hash, number)
	if header == nil {
		return nil
	}
	t := f.peek(!f.config.IsBinaryTrie(header.Number, header.Time))
	if t == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return t.stall
}

// cursorRoot returns the direction's cursor root, zero when it never
// resolved.
func (f *bintrieFollower) cursorRoot(pbt bool) common.Hash {
	t := f.peek(pbt)
	if t == nil {
		return common.Hash{}
	}
	_, _, root := t.cursor()
	return root
}

// journal persists and releases the follower's directions.
func (f *bintrieFollower) journal(head *types.Header) {
	for _, t := range f.live() {
		t.journal(head)
	}
}

// journal persists the tree's layers at the newest root the handle holds -
// the head's root while execution commits this flavour, the replay cursor's
// otherwise - and releases it. Owned handles only; a failed journal
// re-replays on the next start.
func (t *followerTree) journal(head *types.Header) {
	t.f.mu.Lock()
	handle, owned, root := t.handle, t.owned, t.cursorRoot
	t.f.mu.Unlock()
	if handle == nil || !owned {
		return
	}
	if head != nil && t.f.config.IsBinaryTrie(head.Number, head.Time) == t.pbt {
		root = head.Root
	}
	if err := handle.Journal(root); err != nil {
		log.Warn("Failed to journal shadow trie", "err", err)
	}
	if err := handle.Close(); err != nil {
		log.Error("Failed to close shadow trie", "err", err)
	}
}

func (t *followerTree) cursor() (uint64, common.Hash, common.Hash) {
	t.f.mu.Lock()
	defer t.f.mu.Unlock()
	return t.cursorNum, t.cursorHash, t.cursorRoot
}

func (t *followerTree) setCursor(num uint64, hash common.Hash, root common.Hash) {
	t.f.mu.Lock()
	t.cursorNum, t.cursorHash, t.cursorRoot = num, hash, root
	t.f.mu.Unlock()
}

func (t *followerTree) hasState(root common.Hash) bool {
	t.f.mu.Lock()
	handle := t.handle
	t.f.mu.Unlock()
	_, err := handle.NodeReader(root)
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

// MigrationProgress reports where a migrating chain stands.
type MigrationProgress struct {
	Phase  string             // inactive, running or done
	Binary *DirectionProgress // the binary shadow; nil before it runs
	Merkle *DirectionProgress // the merkle window; nil before it runs
}

// DirectionProgress reports one direction's replay position.
type DirectionProgress struct {
	Phase      string      // idle, following, synced, parked or stalled
	Cursor     uint64      // last replayed block number
	CursorHash common.Hash // last replayed block hash
	ShadowRoot common.Hash // the tree root of that block
	Error      string      // what stalled the direction, if anything
}

// SetBALRequester wires the fetcher that resolves missing access lists; the
// migration hands it the blocks a stalled replay needs.
func (bc *BlockChain) SetBALRequester(req func([]BALRequest)) {
	if bc.follower != nil {
		bc.follower.setRequester(req)
	}
}

// KickMigration re-arms a stalled migration sync, typically after new access
// lists landed.
func (bc *BlockChain) KickMigration() {
	if bc.follower != nil {
		bc.follower.kick()
	}
}

func (bc *BlockChain) MigrationProgress() MigrationProgress {
	if rawdb.ReadPBTMigrationDone(bc.db) {
		return MigrationProgress{Phase: "done"}
	}
	if bc.follower == nil {
		return MigrationProgress{Phase: "inactive"}
	}
	head := bc.CurrentBlock()
	if head == nil {
		return MigrationProgress{Phase: "running"}
	}
	return bc.follower.progress(head)
}

func (f *bintrieFollower) progress(head *types.Header) MigrationProgress {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := MigrationProgress{Phase: "running"}
	if t := f.dirs[dirIndex(true)]; t != nil {
		p.Binary = t.progressLocked(head)
	}
	if t := f.dirs[dirIndex(false)]; t != nil {
		p.Merkle = t.progressLocked(head)
	}
	return p
}

// progressLocked snapshots the direction under the follower's mutex. A
// direction whose flavour matches the head has nothing to replay: parked.
func (t *followerTree) progressLocked(head *types.Header) *DirectionProgress {
	p := &DirectionProgress{
		Phase:      "following",
		Cursor:     t.cursorNum,
		CursorHash: t.cursorHash,
		ShadowRoot: t.cursorRoot,
	}
	switch {
	case t.stall != nil:
		p.Phase, p.Error = "stalled", t.stall.Error()
	case t.handle == nil:
		p.Phase = "idle"
	case t.f.config.IsBinaryTrie(head.Number, head.Time) == t.pbt:
		p.Phase = "parked"
	case p.Cursor >= head.Number.Uint64():
		p.Phase = "synced"
	}
	return p
}
