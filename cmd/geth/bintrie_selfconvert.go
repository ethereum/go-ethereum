// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// convertAt converts the node's own state in the background, as an EIP-8347
// self-converter, with the preimage file anchored at block a. The node keeps
// following the chain meanwhile, so the scan reads the state at the block P
// it last persisted, P >= a, with the keys of the file plus those blocks
// a+1..P wrote. Replaying a+1..P over the tree anchored at a yields the tree
// converted at P (EIP-8347), so the tree is attested at P, once P is
// finalized, and the follower takes it from there. The build steps aside for
// blocks through yield.
func (p *pbtImporter) convertAt(ctx context.Context, a *types.Header, yield *blockYield) error {
	workers := cmp.Or(p.cfg.Workers, utils.PBTImportWorkersFlag.Value)
	f, ranges, pieces, err := splitPreimages(p.cfg.Preimages, workers)
	if err != nil {
		return err
	}
	defer f.Close()

	at, sources, err := p.capture(ctx, a, ranges)
	if err != nil {
		return err
	}
	release := func() {
		for _, s := range sources {
			s.release()
		}
	}
	defer release()

	// ponytail: the sources pin the database's files through this pass too;
	// for an A far back, sort the lists before the capture instead.
	lists, err := accessListKeys(ctx, p.accessList(ctx, at.Number.Uint64()), a.Number.Uint64(), at.Number.Uint64(), ranges, p.tmpDir)
	if err != nil {
		return err
	}
	defer func() {
		for _, k := range lists {
			k.close()
		}
	}()
	plan := scanPlan{ranges: ranges, release: release}
	for i := range ranges {
		keys, err := newDriftKeys(pieces[i], lists[i])
		if err != nil {
			return err
		}
		plan.sources = append(plan.sources, sources[i])
		plan.keys = append(plan.keys, keys)
	}
	log.Info("Starting binary tree conversion in the background", "anchor", at.Number, "hash", at.Hash(),
		"stateRoot", at.Root, "preimages", p.cfg.Preimages, "preimageAnchor", a.Number)
	ref := anchorRef{hash: at.Hash(), byHash: true}
	root, err := convertPlan(ctx, p.db, at.Root, plan, yield, conversionOptions{
		sortBudget: onlineSortBudget,
		tmpDir:     p.tmpDir,
		workers:    workers,
		anchor:     at,
		// EIP-8347 anchors at finalized blocks; P is only persisted.
		beforeAttest: func() error {
			_, err := p.awaitAnchor(ctx, ref)
			return err
		},
	})
	if err != nil {
		return err
	}
	log.Info("Conversion complete", "anchor", at.Number, "hash", at.Hash(), "binaryRoot", root)
	return nil
}

// capture waits until the node has persisted a complete merkle flat state
// at a canonical block at or past a, which needs the node out of snap sync
// and the flat state generated, then captures it: the block, and a source
// per range.
func (p *pbtImporter) capture(ctx context.Context, a *types.Header, ranges []hashRange) (*types.Header, []*persistedSource, error) {
	var logged time.Time
	for {
		if p.chain.TrieDB().SnapshotCompleted() {
			root, sources, err := capturePersisted(p.db, ranges)
			if err != nil {
				return nil, nil, err
			}
			if at := p.persistedBlock(root, a); at != nil {
				return at, sources, nil
			}
			for _, s := range sources {
				s.release()
			}
		}
		if time.Since(logged) >= time.Minute {
			log.Info("Binary tree conversion waiting for a complete state at or past its anchor", "anchor", a.Number)
			logged = time.Now()
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(p.poll):
		}
	}
}

// persistedBlock returns the canonical block at or past a whose state root
// is root, walking back from the head, or nil if none is.
func (p *pbtImporter) persistedBlock(root common.Hash, a *types.Header) *types.Header {
	for h := p.chain.CurrentBlock(); h != nil && h.Number.Cmp(a.Number) >= 0; h = p.chain.GetHeader(h.ParentHash, h.Number.Uint64()-1) {
		if h.Root == root {
			return h
		}
	}
	return nil
}

// accessList returns a reader of canonical blocks' access lists, proven
// against their headers. A list the node lacks is asked of its peers, along
// with the rest of the batch up to block to, and waited for.
func (p *pbtImporter) accessList(ctx context.Context, to uint64) func(uint64) (*bal.BlockAccessList, error) {
	return func(n uint64) (*bal.BlockAccessList, error) {
		var logged time.Time
		for {
			hash := rawdb.ReadCanonicalHash(p.db, n)
			header := rawdb.ReadHeader(p.db, hash, n)
			if header == nil {
				return nil, fmt.Errorf("missing canonical header %d", n)
			}
			if header.BlockAccessListHash == nil {
				return nil, fmt.Errorf("block %d commits no access list, which a conversion past its preimage file's anchor needs", n)
			}
			if list := rawdb.ReadAccessList(p.db, hash, n); list != nil {
				if h := list.Hash(); h != *header.BlockAccessListHash {
					return nil, fmt.Errorf("access list mismatch for block %d %x: have %x, header commits %x", n, hash, h, *header.BlockAccessListHash)
				}
				return list, nil
			}
			p.chain.RequestAccessLists(n, to)
			if time.Since(logged) >= time.Minute {
				log.Info("Binary tree conversion waiting for an access list", "block", n)
				logged = time.Now()
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(p.poll):
			}
		}
	}
}

// An online self-conversion reads the state of a node that keeps following
// the chain. pathdb's own iterators cannot serve it: they go stale whenever
// the disk layer moves, every block on a synced node. What pathdb leaves in
// the database, though, is always one whole state: it persists a state as one
// atomic batch carrying the state's root and id (triedb/pathdb/buffer.go),
// and on a synced node with a complete flat state nothing else writes the
// merkle flat state. A database iterator reads at the sequence number it was
// opened at, for as long as it lives. So iterators opened together, with no
// flush between the first and the last, scan one persisted state however far
// the chain moves on meanwhile.

// capturePersisted opens, at each range start, a raw iterator over the
// merkle flat accounts and one over the flat storage, all reading the state
// pathdb last persisted, and returns that state's root and a source per
// range. Reading the persisted id and root before and after opening them
// shows that no flush landed in between. Check 2 still proves the scan
// against the root. Release the sources once the scan is done: until then
// they pin what the database held when they opened.
func capturePersisted(db ethdb.Database, ranges []hashRange) (common.Hash, []*persistedSource, error) {
	for range 3 {
		id, root := rawdb.ReadPersistentStateID(db), rawdb.ReadSnapshotRoot(db)
		sources := make([]*persistedSource, len(ranges))
		for i, r := range ranges {
			sources[i] = &persistedSource{
				accts: db.NewIterator(rawdb.SnapshotAccountPrefix, r.start[:]),
				slots: db.NewIterator(rawdb.SnapshotStoragePrefix, r.start[:]),
			}
		}
		if rawdb.ReadPersistentStateID(db) == id && rawdb.ReadSnapshotRoot(db) == root {
			return root, sources, nil
		}
		for _, s := range sources {
			s.release()
		}
	}
	return common.Hash{}, nil, errors.New("the persisted state moved on while every scan iterator opened, three times running")
}

// persistedSource reads one worker range of the persisted merkle flat state
// through the two iterators capturePersisted opened for it, forward only:
// the accounts lead, and each account's storage comes off one shared cursor,
// since an iterator opened later would read a later state.
type persistedSource struct {
	accts, slots ethdb.Iterator
	held         bool // slots holds an entry no account has taken yet
	once         sync.Once
}

func (s *persistedSource) accounts(common.Hash) (pathdb.AccountIterator, error) {
	return &rawAccounts{it: s.accts}, nil
}

func (s *persistedSource) storage(accountHash, _ common.Hash) (pathdb.StorageIterator, error) {
	return &rawStorage{src: s, account: accountHash}, nil
}

// release closes both iterators; later calls do nothing.
func (s *persistedSource) release() {
	s.once.Do(func() {
		s.accts.Release()
		s.slots.Release()
	})
}

// rawAccounts walks flat account entries: "a" ‖ hash -> slim account RLP.
type rawAccounts struct{ it ethdb.Iterator }

func (r *rawAccounts) Next() bool {
	for r.it.Next() {
		if len(r.it.Key()) == len(rawdb.SnapshotAccountPrefix)+common.HashLength {
			return true
		}
	}
	return false
}
func (r *rawAccounts) Error() error { return r.it.Error() }
func (r *rawAccounts) Hash() common.Hash {
	return common.BytesToHash(r.it.Key()[len(rawdb.SnapshotAccountPrefix):])
}
func (r *rawAccounts) Account() []byte { return r.it.Value() }
func (r *rawAccounts) Release()        {} // the source releases the iterator

// rawStorage walks one account's flat storage entries off the source's
// shared cursor: "o" ‖ account hash ‖ slot hash -> slot RLP. It stops at the
// first entry of a later account, which stays held for that account.
type rawStorage struct {
	src     *persistedSource
	account common.Hash
}

func (r *rawStorage) Next() bool {
	s := r.src
	for {
		if !s.held {
			if !s.slots.Next() {
				return false
			}
			s.held = true
		}
		key := s.slots.Key()
		if len(key) != len(rawdb.SnapshotStoragePrefix)+2*common.HashLength {
			s.held = false
			continue
		}
		switch bytes.Compare(key[len(rawdb.SnapshotStoragePrefix):len(rawdb.SnapshotStoragePrefix)+common.HashLength], r.account[:]) {
		case -1:
			// Storage of an account the scan never asked for, which a
			// consistent state does not hold: nothing commits it.
			s.held = false
		case 0:
			s.held = false
			return true
		default:
			return false
		}
	}
}
func (r *rawStorage) Error() error { return r.src.slots.Error() }
func (r *rawStorage) Hash() common.Hash {
	return common.BytesToHash(r.src.slots.Key()[len(rawdb.SnapshotStoragePrefix)+common.HashLength:])
}
func (r *rawStorage) Slot() []byte { return r.src.slots.Value() }
func (r *rawStorage) Release()     {} // the cursor outlives the account

// A self-converter scanning the state at block P with a preimage file made
// at an earlier block A meets keys created after A, which the file cannot
// name, and misses keys deleted after A, which it still names. Access lists
// carry raw addresses and slot keys (EIP-7928), and creating or deleting a
// key writes it: an account's balance, nonce or code changes (an EIP-161
// removal is recorded as a zero balance, core/state/statedb.go), a slot's
// value changes. So the writes of blocks A+1..P name every key the file
// lacks and account for every key it holds in excess; reads add nothing.

// balKeySorterBudget is what each of a self-converter's access-list key
// sorters holds before spilling: they sit beside the scan's own sorters.
const balKeySorterBudget = 32 << 20

// balKeys are one worker range's written keys: keccak(address) ‖ block ->
// address, and keccak(address) ‖ keccak(slot) ‖ block -> slot. The block
// keeps a key several blocks wrote from colliding in the sorter.
type balKeys struct {
	accounts, slots *bintrie.RecordSorter
}

func (k *balKeys) close() {
	k.accounts.Close()
	k.slots.Close()
}

// accessListKeys sorts the keys the access lists of blocks from+1..to write
// into per-range sorters. read returns a block's list, proven against its
// header.
func accessListKeys(ctx context.Context, read func(uint64) (*bal.BlockAccessList, error), from, to uint64, ranges []hashRange, tmpDir string) ([]*balKeys, error) {
	keys := make([]*balKeys, len(ranges))
	for i := range keys {
		keys[i] = &balKeys{
			accounts: bintrie.NewRecordSorter(tmpDir, balKeySorterBudget, nil),
			slots:    bintrie.NewRecordSorter(tmpDir, balKeySorterBudget, nil),
		}
	}
	fail := func(err error) ([]*balKeys, error) {
		for _, k := range keys {
			k.close()
		}
		return nil, err
	}
	var key []byte
	for n := from + 1; n <= to; n++ {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		list, err := read(n)
		if err != nil {
			return fail(err)
		}
		for _, acct := range *list {
			addrHash := crypto.Keccak256Hash(acct.Address[:])
			k := keys[rangeOf(ranges, addrHash)]
			if len(acct.BalanceChanges)+len(acct.NonceChanges)+len(acct.CodeChanges) > 0 {
				key = binary.BigEndian.AppendUint64(append(key[:0], addrHash[:]...), n)
				if err := k.accounts.Add(key, acct.Address[:]); err != nil {
					return fail(err)
				}
			}
			for _, change := range acct.StorageChanges {
				slot := change.Slot.Bytes32()
				slotHash := crypto.Keccak256Hash(slot[:])
				key = binary.BigEndian.AppendUint64(append(append(key[:0], addrHash[:]...), slotHash[:]...), n)
				if err := k.slots.Add(key, slot[:]); err != nil {
					return fail(err)
				}
			}
		}
	}
	return keys, nil
}

// rangeOf returns the index of the range holding hash; ranges ascend and the
// first starts at the bottom of the hash space.
func rangeOf(ranges []hashRange, hash common.Hash) int {
	return sort.Search(len(ranges), func(i int) bool {
		return bytes.Compare(ranges[i].start[:], hash[:]) > 0
	}) - 1
}

// balCursor walks one sorted stream of written keys forward.
type balCursor struct {
	stream   *bintrie.RecordStream
	n        int // hash bytes ahead of the block: 32 for accounts, 64 for slots
	started  bool
	hash     []byte // the entry under the cursor, nil past the end
	preimage []byte
}

func (c *balCursor) next() error {
	key, value, err := c.stream.Next()
	if err == io.EOF {
		c.hash = nil
		return nil
	}
	if err != nil {
		return err
	}
	c.hash, c.preimage = append(c.hash[:0], key[:c.n]...), append(c.preimage[:0], value...)
	return nil
}

// seek moves the cursor to hash, which must not lie behind it, and returns
// the key behind it if a list wrote it.
func (c *balCursor) seek(hash []byte) ([]byte, bool, error) {
	if !c.started {
		c.started = true
		if err := c.next(); err != nil {
			return nil, false, err
		}
	}
	for c.hash != nil && bytes.Compare(c.hash, hash) < 0 {
		if err := c.next(); err != nil {
			return nil, false, err
		}
	}
	if c.hash == nil || !bytes.Equal(c.hash, hash) {
		return nil, false, nil
	}
	return c.preimage, true, nil
}

// driftKeys resolves one worker's hashes for a scan at block P from a piece
// of a preimage file made at an earlier block A and the keys blocks A+1..P
// wrote. A key the file lacks must be one a list wrote, and so must a file
// key the state lacks. A file made at another state therefore still fails,
// as step 2's exact match would fail it. A record's slot keys are read a
// window at a time, as fileKeys reads them.
type driftKeys struct {
	f               fileKeys // the piece, and the slot keys of the record under the scan
	accounts, slots *balCursor

	// The file's next record header, read ahead of the scan.
	pending  bool
	pendAddr common.Address
	pendHash common.Hash
	pendLen  int // its slot keys, none read yet

	addrHash common.Hash // the account under the scan
	slotKey  []byte      // scratch for slot cursor keys
}

func newDriftKeys(file *preimageReader, keys *balKeys) (*driftKeys, error) {
	accounts, err := keys.accounts.Sort()
	if err != nil {
		return nil, err
	}
	slots, err := keys.slots.Sort()
	if err != nil {
		return nil, err
	}
	return &driftKeys{
		f:        fileKeys{r: file, hasher: newKeccakHasher()},
		accounts: &balCursor{stream: accounts, n: common.HashLength},
		slots:    &balCursor{stream: slots, n: 2 * common.HashLength},
	}, nil
}

// peek reads the file's next record header, unless one is pending. It
// reports false at the end of the piece.
func (d *driftKeys) peek() (bool, error) {
	if d.pending {
		return true, nil
	}
	addr, count, err := d.f.r.nextRecord()
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	hash, err := d.f.order.record(d.f.hasher, d.f.r.records-1, addr)
	if err != nil {
		return false, err
	}
	d.pending, d.pendAddr, d.pendHash, d.pendLen = true, addr, hash, count
	return true, nil
}

// take makes the pending record's slot keys the ones to match.
func (d *driftKeys) take() {
	d.pending = false
	d.f.left, d.f.slots, d.f.next = d.pendLen, d.f.slots[:0], 0
}

// gone accepts the pending file record as an account deleted since A, if a
// list wrote it. Its storage went with it: its slot keys are read, their
// order checked, and passed over.
func (d *driftKeys) gone() error {
	_, written, err := d.accounts.seek(d.pendHash[:])
	if err != nil {
		return err
	}
	if !written {
		return fmt.Errorf("preimage file names address %x, which the state does not hold and no access list since its anchor wrote", d.pendAddr)
	}
	d.take()
	for {
		_, _, ok, err := d.f.pending()
		if err != nil || !ok {
			return err
		}
		d.f.next = len(d.f.slots)
	}
}

func (d *driftKeys) account(hash common.Hash) (common.Address, error) {
	d.addrHash = hash
	for {
		ok, err := d.peek()
		if err != nil {
			return common.Address{}, err
		}
		if !ok || bytes.Compare(d.pendHash[:], hash[:]) > 0 {
			break
		}
		if d.pendHash == hash {
			d.take()
			return d.pendAddr, nil
		}
		if err := d.gone(); err != nil {
			return common.Address{}, err
		}
	}
	// Not in the file: created since A, so a list wrote it.
	d.f.left, d.f.slots, d.f.next = 0, d.f.slots[:0], 0
	preimage, ok, err := d.accounts.seek(hash[:])
	if err != nil {
		return common.Address{}, err
	}
	if !ok {
		return common.Address{}, fmt.Errorf("missing preimage for account hash %x: neither the preimage file nor the access lists since its anchor name it", hash)
	}
	return common.BytesToAddress(preimage), nil
}

// slotSeek looks the current account's slot hash up in the lists.
func (d *driftKeys) slotSeek(hash common.Hash) ([]byte, bool, error) {
	d.slotKey = append(append(d.slotKey[:0], d.addrHash[:]...), hash[:]...)
	return d.slots.seek(d.slotKey)
}

// cleared accepts the current account's next file slot, slot under hash, as
// one cleared since A, if a list wrote it.
func (d *driftKeys) cleared(addr common.Address, slot, hash common.Hash) error {
	_, written, err := d.slotSeek(hash)
	if err != nil {
		return err
	}
	if !written {
		return fmt.Errorf("preimage file names slot %x of %x, which the state does not hold and no access list since its anchor wrote", slot, addr)
	}
	d.f.next++
	return nil
}

func (d *driftKeys) slot(addr common.Address, hash common.Hash) (common.Hash, error) {
	for {
		slot, have, ok, err := d.f.pending()
		if err != nil {
			return common.Hash{}, err
		}
		if ok && have == hash {
			d.f.next++
			return slot, nil
		}
		if !ok || bytes.Compare(have[:], hash[:]) > 0 {
			break
		}
		if err := d.cleared(addr, slot, have); err != nil {
			return common.Hash{}, err
		}
	}
	// Not in the file: written since A.
	preimage, ok, err := d.slotSeek(hash)
	if err != nil {
		return common.Hash{}, err
	}
	if !ok {
		return common.Hash{}, fmt.Errorf("missing preimage for storage key %x (account %x): neither the preimage file nor the access lists since its anchor name it", hash, addr)
	}
	return common.BytesToHash(preimage), nil
}

func (d *driftKeys) done(addr common.Address) error {
	for {
		slot, have, ok, err := d.f.pending()
		if err != nil || !ok {
			return err
		}
		if err := d.cleared(addr, slot, have); err != nil {
			return err
		}
	}
}

func (d *driftKeys) close() error {
	for {
		ok, err := d.peek()
		if err != nil || !ok {
			return err
		}
		if err := d.gone(); err != nil {
			return err
		}
	}
}
