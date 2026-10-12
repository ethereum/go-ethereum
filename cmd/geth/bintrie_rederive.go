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
// With a batch, the fold also writes the flat state in the keccak order it
// walks: storage rows as it goes, account rows spilled to a file under tmpDir
// and written after the last storage row, so no flushed table mixes the two
// key families. Everything is written by return, batch included, and digested
// into written.
func rederiveMerkleRoot(accounts, slots *bintrie.RecordStream, batch ethdb.Batch, written *writtenState, tmpDir string, start time.Time) (common.Hash, error) {
	var (
		slotHeld    = &heldStream{stream: slots}
		accountTrie = trie.NewStackTrie(nil)
		storageTrie = trie.NewStackTrie(nil)
		rederived   uint64
		acctRows    *spillFile
	)
	if batch != nil {
		var err error
		if acctRows, err = newSpillFile(tmpDir, "bintrie-acctrows-*"); err != nil {
			return common.Hash{}, err
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
	for {
		akey, avalue, err := accounts.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		if len(avalue) != merkleAccountRecordLen {
			return common.Hash{}, fmt.Errorf("account record for %x is %d bytes, want %d", akey, len(avalue), merkleAccountRecordLen)
		}
		var (
			accountHash = common.BytesToHash(akey)
			nonce       = binary.BigEndian.Uint64(avalue[:8])
			balance     = new(uint256.Int).SetBytes(avalue[8:40])
			codeHash    = common.CopyBytes(avalue[40:merkleAccountRecordLen])
		)
		storageTrie.Reset()
		storageRoot := types.EmptyRootHash
		hasStorage := false
		for {
			skey, svalue, err := slotHeld.current()
			if err != nil {
				return common.Hash{}, err
			}
			if skey == nil {
				break
			}
			switch bytes.Compare(skey[:common.HashLength], akey) {
			case -1:
				return common.Hash{}, fmt.Errorf("storage under account hash %x, which holds no account", skey[:common.HashLength])
			case 1:
			default:
				if err := storageTrie.Update(skey[common.HashLength:], svalue); err != nil {
					return common.Hash{}, err
				}
				if batch != nil {
					if err := written.storage.add(skey, svalue); err != nil {
						return common.Hash{}, err
					}
					rawdb.WriteStorageSnapshot(batch, accountHash, common.BytesToHash(skey[common.HashLength:]), svalue)
					if err := flush(); err != nil {
						return common.Hash{}, err
					}
				}
				hasStorage = true
				slotHeld.advance()
				continue
			}
			break
		}
		if hasStorage {
			storageRoot = storageTrie.Hash()
		}
		full, err := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    nonce,
			Balance:  balance,
			Root:     storageRoot,
			CodeHash: codeHash,
		})
		if err != nil {
			return common.Hash{}, err
		}
		if err := accountTrie.Update(akey, full); err != nil {
			return common.Hash{}, err
		}
		if acctRows != nil {
			slim := types.SlimAccountRLP(types.StateAccount{
				Nonce:    nonce,
				Balance:  balance,
				Root:     types.EmptyRootHash,
				CodeHash: codeHash,
			})
			if err := acctRows.add(akey, slim); err != nil {
				return common.Hash{}, err
			}
		}
		rederived++
		if rederived%100_000 == 0 {
			log.Info("Re-deriving merkle state", "accounts", rederived,
				"elapsed", common.PrettyDuration(time.Since(start)))
		}
	}
	if skey, _, err := slotHeld.current(); err != nil {
		return common.Hash{}, err
	} else if skey != nil {
		return common.Hash{}, fmt.Errorf("storage under account hash %x, which holds no account", skey[:common.HashLength])
	}
	if batch == nil {
		return accountTrie.Hash(), nil
	}
	err := acctRows.replay(bufio.NewReaderSize(nil, 256<<10), func(hash, row []byte) error {
		if err := written.accounts.add(hash, row); err != nil {
			return err
		}
		rawdb.WriteAccountSnapshot(batch, common.BytesToHash(hash), row)
		return flush()
	})
	if err != nil {
		return common.Hash{}, err
	}
	if err := batch.Write(); err != nil {
		return common.Hash{}, err
	}
	batch.Reset()
	return accountTrie.Hash(), nil
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
func verifyWritten(pbtdb ethdb.Database, written *writtenState) error {
	families := []struct {
		name   string
		prefix []byte
		want   *writeDigest
	}{
		{"tree node", rawdb.TrieNodeAccountPrefix, written.nodes},
		{"flat account", rawdb.SnapshotAccountPrefix, written.accounts},
		{"flat storage", rawdb.SnapshotStoragePrefix, written.storage},
	}
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
	if err := g.Wait(); err != nil {
		return err
	}
	log.Info("Verified the written state", "nodes", written.nodes.count,
		"accounts", written.accounts.count, "slots", written.storage.count)
	return nil
}
