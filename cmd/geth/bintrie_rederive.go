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
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/holiman/uint256"
	"golang.org/x/sync/errgroup"
)

// The merkle re-derivation shared by both commands: the converter demands
// the state root it scanned, the importer the root its anchor commits.

const merkleAccountRecordLen = 8 + 32 + common.HashLength

// merkleAccountRecord encodes the account fields the fold needs. The storage
// root is absent: it is recomputed from the storage records.
func merkleAccountRecord(nonce uint64, balance *uint256.Int, codeHash common.Hash) []byte {
	record := make([]byte, 0, merkleAccountRecordLen)
	record = binary.BigEndian.AppendUint64(record, nonce)
	balance32 := balance.Bytes32()
	record = append(record, balance32[:]...)
	return append(record, codeHash.Bytes()...)
}

// heldStream wraps a sorted record stream with one record of lookahead.
type heldStream struct {
	stream     *bintrie.RecordStream
	key, value []byte
	done       bool
}

// current returns the held record, loading one if none is held.
func (h *heldStream) current() ([]byte, []byte, error) {
	if h.done || h.key != nil {
		return h.key, h.value, nil
	}
	key, value, err := h.stream.Next()
	if err == io.EOF {
		h.done = true
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	h.key, h.value = key, value
	return key, value, nil
}

// advance drops the held record.
func (h *heldStream) advance() { h.key, h.value = nil, nil }

// rederiveMerkleRoot folds account and storage records back into a
// merkle-patricia trie and returns its root. Both streams must ascend, which
// the sorters guarantee. Storage under no account is the loss this check
// exists to catch, so it errors rather than being skipped.
//
// The calling goroutine walks both streams in keccak order; workers fold the
// storage tries and one more goroutine the account trie, taking the storage
// roots in account order. With a batch, the walk also writes the flat state:
// storage rows as it goes, account rows spilled to a file under tmpDir and
// written after the last storage row, so no flushed table mixes the two key
// families. Everything is written by return, batch included, and digested
// into written.
func rederiveMerkleRoot(accounts, slots *bintrie.RecordStream, batch ethdb.Batch, written *writtenState, tmpDir string, workers int, start time.Time) (common.Hash, error) {
	workers = max(workers, 1)
	var (
		g, ctx    = errgroup.WithContext(context.Background())
		jobs      = make([]chan *foldBatch, workers)
		roots     = make([]chan []common.Hash, workers)
		free      = make(chan *foldBatch, 3*workers)
		accts     = make(chan []foldAccount, 4)
		acctsFree = make(chan []foldAccount, 6)
		root      common.Hash
	)
	for range cap(free) {
		free <- new(foldBatch)
	}
	for range cap(acctsFree) {
		acctsFree <- make([]foldAccount, 0, foldBatchSize)
	}
	for i := range workers {
		jobs[i], roots[i] = make(chan *foldBatch, 2), make(chan []common.Hash, 2)
		g.Go(func() error { return foldStorage(ctx, jobs[i], roots[i], free) })
	}
	g.Go(func() error {
		var err error
		root, err = foldAccounts(ctx, accts, acctsFree, roots)
		return err
	})
	g.Go(func() error {
		defer func() {
			for _, ch := range jobs {
				close(ch)
			}
			close(accts)
		}()
		return walkMerkleRecords(ctx, accounts, slots, batch, written, tmpDir, start, jobs, free, accts, acctsFree)
	})
	if err := g.Wait(); err != nil {
		return common.Hash{}, err
	}
	return root, nil
}

// foldBatchSize is how many slots, or accounts, a fold batch carries.
var foldBatchSize = 4096

// foldBatch carries storage slots to a fold worker, the value of keys[i]
// ending at vals[ends[i]]. Each of accounts closes one account's slots, at
// that index; slots past the last close continue into the worker's next
// batch. done ends the worker's job, after a close.
type foldBatch struct {
	keys     []common.Hash
	vals     []byte
	ends     []int
	accounts []int
	done     bool
}

// foldAccount is an account on its way to the account trie.
type foldAccount struct {
	hash     common.Hash
	nonce    uint64
	balance  uint256.Int
	codeHash common.Hash
	storage  bool
}

// walkMerkleRecords joins the account and slot streams in keccak order,
// writing the flat state if batch is set, and hands the slots to the fold
// workers and the accounts to the account trie: a job of whole accounts to
// one worker, the next job to the next.
func walkMerkleRecords(ctx context.Context, accounts, slots *bintrie.RecordStream, batch ethdb.Batch, written *writtenState, tmpDir string, start time.Time,
	jobs []chan *foldBatch, free chan *foldBatch, accts, acctsFree chan []foldAccount) error {
	var (
		slotHeld  = &heldStream{stream: slots}
		rederived uint64
		acctRows  *spillFile
		worker    int
		cur       *foldBatch
		pending   []foldAccount
	)
	if batch != nil {
		var err error
		if acctRows, err = newSpillFile(tmpDir, "bintrie-acctrows-*"); err != nil {
			return err
		}
		defer acctRows.close()
	}
	flush := func() error {
		if batch.ValueSize() < ethdb.IdealBatchSize {
			return nil
		}
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
		return nil
	}
	take := func() error {
		select {
		case cur = <-free:
		case <-ctx.Done():
			return ctx.Err()
		}
		cur.keys, cur.vals, cur.ends, cur.accounts, cur.done = cur.keys[:0], cur.vals[:0], cur.ends[:0], cur.accounts[:0], false
		return nil
	}
	send := func(done bool) error {
		cur.done = done
		select {
		case jobs[worker] <- cur:
		case <-ctx.Done():
			return ctx.Err()
		}
		if done {
			worker = (worker + 1) % len(jobs)
		}
		return take()
	}
	sendAccounts := func() error {
		select {
		case accts <- pending:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case pending = <-acctsFree:
		case <-ctx.Done():
			return ctx.Err()
		}
		pending = pending[:0]
		return nil
	}
	if err := take(); err != nil {
		return err
	}
	select {
	case pending = <-acctsFree:
	case <-ctx.Done():
		return ctx.Err()
	}
	for {
		akey, avalue, err := accounts.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(avalue) != merkleAccountRecordLen {
			return fmt.Errorf("account record for %x is %d bytes, want %d", akey, len(avalue), merkleAccountRecordLen)
		}
		acct := foldAccount{
			hash:     common.BytesToHash(akey),
			nonce:    binary.BigEndian.Uint64(avalue[:8]),
			codeHash: common.BytesToHash(avalue[40:merkleAccountRecordLen]),
		}
		acct.balance.SetBytes(avalue[8:40])
		for {
			skey, svalue, err := slotHeld.current()
			if err != nil {
				return err
			}
			if skey == nil {
				break
			}
			switch bytes.Compare(skey[:common.HashLength], acct.hash[:]) {
			case -1:
				return fmt.Errorf("storage under account hash %x, which holds no account", skey[:common.HashLength])
			case 1:
			default:
				cur.keys = append(cur.keys, common.Hash(skey[common.HashLength:]))
				cur.vals = append(cur.vals, svalue...)
				cur.ends = append(cur.ends, len(cur.vals))
				if batch != nil {
					if err := written.storage.add(skey, svalue); err != nil {
						return err
					}
					rawdb.WriteStorageSnapshot(batch, acct.hash, common.BytesToHash(skey[common.HashLength:]), svalue)
					if err := flush(); err != nil {
						return err
					}
				}
				acct.storage = true
				slotHeld.advance()
				// A storage trie past a batch goes on in the same worker's next.
				if len(cur.keys) == 2*foldBatchSize {
					if err := send(false); err != nil {
						return err
					}
				}
				continue
			}
			break
		}
		if acct.storage {
			cur.accounts = append(cur.accounts, len(cur.keys))
		}
		if acctRows != nil {
			slim := types.SlimAccountRLP(types.StateAccount{
				Nonce:    acct.nonce,
				Balance:  &acct.balance,
				Root:     types.EmptyRootHash,
				CodeHash: acct.codeHash[:],
			})
			if err := acctRows.add(akey, slim); err != nil {
				return err
			}
		}
		// Accounts reach the account trie only after their storage job is
		// out, so it never waits on roots no worker has.
		pending = append(pending, acct)
		if len(pending) == cap(pending) || len(cur.keys) >= foldBatchSize {
			if len(cur.accounts) > 0 {
				if err := send(true); err != nil {
					return err
				}
			}
			if err := sendAccounts(); err != nil {
				return err
			}
		}
		rederived++
		if rederived%100_000 == 0 {
			log.Info("Re-deriving merkle state", "accounts", rederived,
				"elapsed", common.PrettyDuration(time.Since(start)))
		}
	}
	if skey, _, err := slotHeld.current(); err != nil {
		return err
	} else if skey != nil {
		return fmt.Errorf("storage under account hash %x, which holds no account", skey[:common.HashLength])
	}
	if len(cur.accounts) > 0 {
		if err := send(true); err != nil {
			return err
		}
	}
	if len(pending) > 0 {
		if err := sendAccounts(); err != nil {
			return err
		}
	}
	if batch == nil {
		return nil
	}
	err := acctRows.replay(bufio.NewReaderSize(nil, 256<<10), func(hash, row []byte) error {
		if err := written.accounts.add(hash, row); err != nil {
			return err
		}
		rawdb.WriteAccountSnapshot(batch, common.BytesToHash(hash), row)
		return flush()
	})
	if err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		return err
	}
	batch.Reset()
	return nil
}

// foldStorage hashes the storage tries of the accounts it is sent and
// reports the roots of each job's accounts, in their order.
func foldStorage(ctx context.Context, in <-chan *foldBatch, out chan<- []common.Hash, free chan<- *foldBatch) error {
	var (
		st    = trie.NewStackTrie(nil)
		found []common.Hash
	)
	for b := range in {
		var next, val int
		update := func(to int) error {
			for ; next < to; next++ {
				if err := st.Update(b.keys[next][:], b.vals[val:b.ends[next]]); err != nil {
					return err
				}
				val = b.ends[next]
			}
			return nil
		}
		for _, end := range b.accounts {
			if err := update(end); err != nil {
				return err
			}
			found = append(found, st.Hash())
			st.Reset()
		}
		if err := update(len(b.keys)); err != nil {
			return err
		}
		done := b.done
		free <- b
		if done {
			select {
			case out <- found:
			case <-ctx.Done():
				return ctx.Err()
			}
			found = nil
		}
	}
	return nil
}

// foldAccounts builds the account trie from the accounts in order, taking
// each one's storage root from the workers' jobs, in the order they were
// handed out.
func foldAccounts(ctx context.Context, in <-chan []foldAccount, free chan<- []foldAccount, roots []chan []common.Hash) (common.Hash, error) {
	var (
		tr   = trie.NewStackTrie(nil)
		have []common.Hash
		job  int
	)
	for accounts := range in {
		for i := range accounts {
			acct := &accounts[i]
			root := types.EmptyRootHash
			if acct.storage {
				for len(have) == 0 {
					select {
					case have = <-roots[job%len(roots)]:
						job++
					case <-ctx.Done():
						return common.Hash{}, ctx.Err()
					}
				}
				root, have = have[0], have[1:]
			}
			full, err := rlp.EncodeToBytes(&types.StateAccount{
				Nonce:    acct.nonce,
				Balance:  &acct.balance,
				Root:     root,
				CodeHash: acct.codeHash[:],
			})
			if err != nil {
				return common.Hash{}, err
			}
			if err := tr.Update(acct.hash[:], full); err != nil {
				return common.Hash{}, err
			}
		}
		free <- accounts
	}
	return tr.Hash(), nil
}

// Both commands prove what they wrote into the binary tree namespace: each
// writer digests its records in key order, and a re-read of each key family
// must reproduce the digest.

// writeDigest hashes one key family's records in the order written. Keys are
// the family-relative suffixes the namespace stores them under.
type writeDigest struct {
	hasher  crypto.KeccakState
	count   uint64
	lastKey []byte
}

func newWriteDigest() *writeDigest {
	return &writeDigest{hasher: crypto.NewKeccakState()}
}

// add digests one record. Keys must ascend: the re-read walks key order.
func (d *writeDigest) add(key, value []byte) error {
	if d.count > 0 && bytes.Compare(key, d.lastKey) <= 0 {
		return fmt.Errorf("record %x written after %x, out of key order", key, d.lastKey)
	}
	d.lastKey = append(d.lastKey[:0], key...)
	var lengths [8]byte
	binary.BigEndian.PutUint32(lengths[:4], uint32(len(key)))
	binary.BigEndian.PutUint32(lengths[4:], uint32(len(value)))
	d.hasher.Write(lengths[:])
	d.hasher.Write(key)
	d.hasher.Write(value)
	d.count++
	return nil
}

func (d *writeDigest) sum() common.Hash {
	return common.BytesToHash(d.hasher.Sum(nil))
}

// writtenState digests the three key families a command writes.
type writtenState struct {
	nodes, accounts, storage *writeDigest
}

func newWrittenState() *writtenState {
	return &writtenState{nodes: newWriteDigest(), accounts: newWriteDigest(), storage: newWriteDigest()}
}

// writtenFamily is one key family a command writes, with the digest of what
// it wrote.
type writtenFamily struct {
	name   string
	prefix []byte
	want   *writeDigest
}

// families lists the key families written: tree nodes, flat accounts, flat
// storage.
func (w *writtenState) families() []writtenFamily {
	return []writtenFamily{
		{"tree node", rawdb.TrieNodeAccountPrefix, w.nodes},
		{"flat account", rawdb.SnapshotAccountPrefix, w.accounts},
		{"flat storage", rawdb.SnapshotStoragePrefix, w.storage},
	}
}

// writeStagedNodes writes the staged tree nodes into the namespace in key
// order, digesting each.
func writeStagedNodes(pbtdb ethdb.Database, nodes *nodeStager, digest *writeDigest) error {
	batch := pbtdb.NewBatch()
	err := nodes.replay(func(path, blob []byte) error {
		if err := digest.add(path, blob); err != nil {
			return err
		}
		rawdb.WriteAccountTrieNode(batch, path, blob)
		if batch.ValueSize() < ethdb.IdealBatchSize {
			return nil
		}
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
		return nil
	})
	if err != nil {
		return err
	}
	return batch.Write()
}

// verifyWritten re-reads each family in key order, concurrently, and
// requires it to match what was written. The namespace was empty when the
// command began, so a family holding anything more is a mismatch as well.
func verifyWritten(pbtdb ethdb.Database, families ...writtenFamily) error {
	var g errgroup.Group
	for _, f := range families {
		g.Go(func() error {
			got := newWriteDigest()
			it := pbtdb.NewIterator(f.prefix, nil)
			defer it.Release()
			for it.Next() {
				if err := got.add(it.Key()[len(f.prefix):], it.Value()); err != nil {
					return err
				}
			}
			if err := it.Error(); err != nil {
				return err
			}
			if got.count != f.want.count || got.sum() != f.want.sum() {
				return fmt.Errorf("%s records read back differ from those written: %d read, %d written", f.name, got.count, f.want.count)
			}
			return nil
		})
	}
	return g.Wait()
}

// logWritten reports a verified write-back.
func logWritten(written *writtenState) {
	log.Info("Verified the written state", "nodes", written.nodes.count,
		"accounts", written.accounts.count, "slots", written.storage.count)
}
