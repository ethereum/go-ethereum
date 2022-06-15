// Copyright 2020 The go-ethereum Authors
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

package rawdb

import (
	"encoding/binary"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/sha3"
)

// ReadPreimage retrieves a single preimage of the provided hash.
func ReadPreimage(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(preimageKey(hash))
	return data
}

// WritePreimages writes the provided set of preimages to the database.
func WritePreimages(db ethdb.KeyValueWriter, preimages map[common.Hash][]byte) {
	for hash, preimage := range preimages {
		if err := db.Put(preimageKey(hash), preimage); err != nil {
			log.Crit("Failed to store trie preimage", "err", err)
		}
	}
	preimageCounter.Inc(int64(len(preimages)))
	preimageHitCounter.Inc(int64(len(preimages)))
}

// ReadCode retrieves the contract code of the provided code hash.
func ReadCode(db ethdb.KeyValueReader, hash common.Hash) []byte {
	// Try with the prefixed code scheme first, if not then try with legacy
	// scheme.
	data := ReadCodeWithPrefix(db, hash)
	if len(data) != 0 {
		return data
	}
	data, _ = db.Get(hash.Bytes())
	return data
}

// ReadCodeWithPrefix retrieves the contract code of the provided code hash.
// The main difference between this function and ReadCode is this function
// will only check the existence with latest scheme(with prefix).
func ReadCodeWithPrefix(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(codeKey(hash))
	return data
}

// HasCode checks if the contract code corresponding to the
// provided code hash is present in the db.
func HasCode(db ethdb.KeyValueReader, hash common.Hash) bool {
	// Try with the prefixed code scheme first, if not then try with legacy
	// scheme.
	if ok := HasCodeWithPrefix(db, hash); ok {
		return true
	}
	ok, _ := db.Has(hash.Bytes())
	return ok
}

// HasCodeWithPrefix checks if the contract code corresponding to the
// provided code hash is present in the db. This function will only check
// presence using the prefix-scheme.
func HasCodeWithPrefix(db ethdb.KeyValueReader, hash common.Hash) bool {
	ok, _ := db.Has(codeKey(hash))
	return ok
}

// WriteCode writes the provided contract code database.
func WriteCode(db ethdb.KeyValueWriter, hash common.Hash, code []byte) {
	if err := db.Put(codeKey(hash), code); err != nil {
		log.Crit("Failed to store contract code", "err", err)
	}
}

// DeleteCode deletes the specified contract code from the database.
func DeleteCode(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(codeKey(hash)); err != nil {
		log.Crit("Failed to delete contract code", "err", err)
	}
}

// hasher used to derive the hash of trie node.
type hasher struct{ sha crypto.KeccakState }

var hasherPool = sync.Pool{
	New: func() interface{} { return &hasher{sha: sha3.NewLegacyKeccak256().(crypto.KeccakState)} },
}

func newHasher() *hasher           { return hasherPool.Get().(*hasher) }
func returnHasherToPool(h *hasher) { hasherPool.Put(h) }

func (h *hasher) hashData(data []byte) (n common.Hash) {
	h.sha.Reset()
	h.sha.Write(data)
	h.sha.Read(n[:])
	return n
}

// ReadTrieNode retrieves the trie node and the associated node hash of
// the provided node key.
func ReadTrieNode(db ethdb.KeyValueReader, key []byte) ([]byte, common.Hash) {
	data, err := db.Get(trieNodeKey(key))
	if err != nil {
		return nil, common.Hash{}
	}
	hasher := newHasher()
	defer returnHasherToPool(hasher)
	return data, hasher.hashData(data)
}

// HasTrieNode checks the trie node presence with the provided node key.
func HasTrieNode(db ethdb.KeyValueReader, key []byte) bool {
	ok, _ := db.Has(trieNodeKey(key))
	return ok
}

// WriteTrieNode writes the provided trie node database.
func WriteTrieNode(db ethdb.KeyValueWriter, key []byte, node []byte) {
	if err := db.Put(trieNodeKey(key), node); err != nil {
		log.Crit("Failed to store trie node", "err", err)
	}
}

// DeleteTrieNode deletes the specified trie node from the database.
func DeleteTrieNode(db ethdb.KeyValueWriter, key []byte) {
	if err := db.Delete(trieNodeKey(key)); err != nil {
		log.Crit("Failed to delete trie node", "err", err)
	}
}

// ReadLegacyTrieNode retrieves the legacy trie node with the given
// associated node hash.
func ReadLegacyTrieNode(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, err := db.Get(hash.Bytes())
	if err != nil {
		return nil
	}
	return data
}

// WriteLegacyTrieNode writes the provided legacy trie node to database.
func WriteLegacyTrieNode(db ethdb.KeyValueWriter, hash []byte, node []byte) {
	if err := db.Put(hash, node); err != nil {
		log.Crit("Failed to store legacy trie node", "err", err)
	}
}

// ReadTrieNodeSnapshot retrieves the trie node snapshot and the associated
// node hash of the provided node key.
func ReadTrieNodeSnapshot(db ethdb.KeyValueReader, prefix []byte, key []byte) ([]byte, common.Hash) {
	data, err := db.Get(trieNodeSnapshotKey(prefix, key))
	if err != nil {
		return nil, common.Hash{}
	}
	hasher := newHasher()
	defer returnHasherToPool(hasher)
	return data, hasher.hashData(data)
}

// WriteTrieNodeSnapshot writes the provided trie node snapshot database.
func WriteTrieNodeSnapshot(db ethdb.KeyValueWriter, prefix []byte, key []byte, node []byte) {
	if err := db.Put(trieNodeSnapshotKey(prefix, key), node); err != nil {
		log.Crit("Failed to store trie node snapshot", "err", err)
	}
}

// DeleteTrieNodeSnapshot deletes the specified trie node snapshot from the
// database.
func DeleteTrieNodeSnapshot(db ethdb.KeyValueWriter, prefix, key []byte) {
	if err := db.Delete(trieNodeSnapshotKey(prefix, key)); err != nil {
		log.Crit("Failed to delete trie node snapshot", "err", err)
	}
}

// DeleteTrieNodeSnapshots deletes all the trie node snapshots under the given
// namespace.
func DeleteTrieNodeSnapshots(db ethdb.KeyValueStore, prefix []byte) {
	iter := db.NewIterator(append(TrieNodeSnapshotPrefix, prefix...), nil)
	defer iter.Release()

	batch := db.NewBatch()
	for iter.Next() {
		batch.Delete(iter.Key())
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				log.Crit("Failed to delete trie node snapshots", "err", err)
			}
			batch.Reset()
		}
	}
	batch.Write()
}

// ReadReverseDiff retrieves the state reverse diff with the given associated
// block hash and number. Because reverse diff is encoded from 1 in Geth, while
// encoded from 0 in freezer, so do the conversion here implicitly.
func ReadReverseDiff(db ethdb.AncientReaderOp, id uint64) []byte {
	blob, err := db.Ancient(freezerReverseDiffTable, id-1)
	if err != nil {
		return nil
	}
	return blob
}

// ReadReverseDiffHash retrieves the state root corresponding to the specified
// reverse diff. Because reverse diff is encoded from 1 in Geth, while encoded
// from 0 in freezer, so do the conversion here implicitly.
func ReadReverseDiffHash(db ethdb.AncientReaderOp, id uint64) common.Hash {
	blob, err := db.Ancient(freezerReverseDiffHashTable, id-1)
	if err != nil {
		return common.Hash{}
	}
	return common.BytesToHash(blob)
}

// WriteReverseDiff writes the provided reverse diff to database. Because reverse
// diff is encoded from 1 in Geth, while encoded from 0 in freezer, so do the
// conversion here implicitly.
func WriteReverseDiff(db ethdb.AncientWriter, id uint64, blob []byte, state common.Hash) {
	db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		op.AppendRaw(freezerReverseDiffTable, id-1, blob)
		op.AppendRaw(freezerReverseDiffHashTable, id-1, state.Bytes())
		return nil
	})
}

// ReadReverseDiffLookup retrieves the reverse diff id with the given associated
// state root. Return nil if it's not existent.
func ReadReverseDiffLookup(db ethdb.KeyValueReader, root common.Hash) *uint64 {
	data, err := db.Get(reverseDiffLookupKey(root))
	if err != nil || len(data) == 0 {
		return nil
	}
	id := binary.BigEndian.Uint64(data)
	return &id
}

// WriteReverseDiffLookup writes the provided reverse diff lookup to database.
func WriteReverseDiffLookup(db ethdb.KeyValueWriter, root common.Hash, id uint64) {
	var buff [8]byte
	binary.BigEndian.PutUint64(buff[:], id)
	if err := db.Put(reverseDiffLookupKey(root), buff[:]); err != nil {
		log.Crit("Failed to store reverse diff lookup", "err", err)
	}
}

// DeleteReverseDiffLookup deletes the specified reverse diff lookup from the database.
func DeleteReverseDiffLookup(db ethdb.KeyValueWriter, root common.Hash) {
	if err := db.Delete(reverseDiffLookupKey(root)); err != nil {
		log.Crit("Failed to delete reverse diff lookup", "err", err)
	}
}

// ReadReverseDiffHead retrieves the number of the latest reverse diff from
// the database.
func ReadReverseDiffHead(db ethdb.KeyValueReader) uint64 {
	data, _ := db.Get(ReverseDiffHeadKey)
	if len(data) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(data)
}

// WriteReverseDiffHead stores the number of the latest reverse diff id
// into database.
func WriteReverseDiffHead(db ethdb.KeyValueWriter, number uint64) {
	if err := db.Put(ReverseDiffHeadKey, encodeBlockNumber(number)); err != nil {
		log.Crit("Failed to store the head reverse diff id", "err", err)
	}
}

// ReadTrieJournal retrieves the serialized in-memory trie node diff layers saved at
// the last shutdown. The blob is expected to be max a few 10s of megabytes.
func ReadTrieJournal(db ethdb.KeyValueReader) []byte {
	data, _ := db.Get(triesJournalKey)
	return data
}

// WriteTrieJournal stores the serialized in-memory trie node diff layers to save at
// shutdown. The blob is expected to be max a few 10s of megabytes.
func WriteTrieJournal(db ethdb.KeyValueWriter, journal []byte) {
	if err := db.Put(triesJournalKey, journal); err != nil {
		log.Crit("Failed to store tries journal", "err", err)
	}
}

// DeleteTrieJournal deletes the serialized in-memory trie node diff layers saved at
// the last shutdown
func DeleteTrieJournal(db ethdb.KeyValueWriter) {
	if err := db.Delete(triesJournalKey); err != nil {
		log.Crit("Failed to remove tries journal", "err", err)
	}
}

// ReadLegacyStateRoot is a helper function used to load the state root
// stored in legacy scheme. It's mainly served as the fallback for geth
// node which just upgrades from legacy storage scheme.
func ReadLegacyStateRoot(db ethdb.Database) common.Hash {
	blockHash := ReadHeadBlockHash(db)
	for {
		if blockHash == (common.Hash{}) {
			break
		}
		number := ReadHeaderNumber(db, blockHash)
		if number == nil {
			break
		}
		block := ReadBlock(db, blockHash, *number)
		if block == nil {
			break
		}
		blob := ReadLegacyTrieNode(db, block.Root())
		if len(blob) > 0 {
			log.Info("Found fallback base layer root", "number", block.Number(), "hash", block.Hash(), "root", block.Root())
			return block.Root()
		}
		blockHash = block.ParentHash()
	}
	return common.Hash{}
}
