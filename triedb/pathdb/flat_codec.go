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

package pathdb

import (
	"bytes"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

// flatStateCodec abstracts the trie-specific aspects of flat-state storage:
// key derivation from (address, slot), persistence of account/storage entries
// to disk, clean-cache key disambiguation, and iterator construction.
//
// It mirrors the existing nodeHasher pattern (a hot, small interface plugged
// into the Database struct), and complements the Hasher interface from
// state-hasher-iface-2 which abstracts trie-side hashing/commit.
//
// Two implementations are provided:
//   - merkleFlatCodec: keccak-keyed flat state, the historical MPT scheme.
//   - bintrieFlatCodec: per-stem flat state for the unified binary trie.
//     Wired into pathdb.Database.New when isVerkle is true.
//
// All methods MUST be safe for concurrent use; the codec is shared across
// goroutines (the disk layer's read path, the buffer flush path, and the
// background generator may all call into it simultaneously).
type flatStateCodec interface {
	// AccountKey derives the flat-state lookup key for an account.
	//
	// For Merkle: returns keccak256(addr).
	// For Bintrie: returns the full 32-byte tree key (stem || offset) for
	// the BasicData leaf. Since BasicDataLeafKey is 0, the last byte is
	// zero, but the result is a full key — callers use stemFromKey /
	// offsetFromKey to decompose it.
	AccountKey(addr common.Address) common.Hash

	// StorageKey derives the flat-state lookup keys for a storage slot.
	//
	// The first return value carries the account-side hash (e.g.
	// keccak256(addr) for Merkle, or zero for bintrie which has no per-account
	// grouping). The second return value carries the slot-side hash
	// (keccak256(slot) for Merkle, or the full bintrie key for bintrie).
	//
	// Read/Write methods receive the same pair, so the codec implementation
	// is the only place that has to interpret them.
	StorageKey(addr common.Address, slot common.Hash) (accountKey common.Hash, storageKey common.Hash)

	// ReadAccount loads an account flat-state entry from persistent storage.
	// Returns nil if the entry is not present.
	ReadAccount(db ethdb.KeyValueReader, key common.Hash) []byte

	// ReadStorage loads a storage flat-state entry from persistent storage.
	// Returns nil if the entry is not present.
	ReadStorage(db ethdb.KeyValueReader, accountKey common.Hash, storageKey common.Hash) []byte

	// WriteAccount persists an account flat-state entry into the supplied batch.
	WriteAccount(batch ethdb.Batch, key common.Hash, blob []byte)

	// DeleteAccount removes an account flat-state entry via the supplied batch.
	DeleteAccount(batch ethdb.Batch, key common.Hash)

	// WriteStorage persists a storage flat-state entry into the supplied batch.
	WriteStorage(batch ethdb.Batch, accountKey common.Hash, storageKey common.Hash, blob []byte)

	// DeleteStorage removes a storage flat-state entry via the supplied batch.
	DeleteStorage(batch ethdb.Batch, accountKey common.Hash, storageKey common.Hash)

	// AccountCacheKey returns the byte key used in the disk-layer clean state
	// cache (fastcache) for an account entry. The cache is shared between
	// account and storage lookups, so codecs must ensure their key spaces are
	// disjoint to avoid collisions.
	AccountCacheKey(key common.Hash) []byte

	// StorageCacheKey returns the byte key used in the disk-layer clean state
	// cache (fastcache) for a storage entry. See AccountCacheKey for the
	// disjointness requirement.
	StorageCacheKey(accountKey common.Hash, storageKey common.Hash) []byte

	// AccountPrefix returns the rawdb key prefix used by account entries on
	// disk. Used by the generator to set up its account-range iterator.
	AccountPrefix() []byte

	// StoragePrefix returns the rawdb key prefix used by storage entries on
	// disk. Used by the generator to set up its storage-range iterator.
	StoragePrefix() []byte

	// AccountKeyLength returns the expected total length (prefix + payload)
	// of an on-disk account key. The generator uses this to filter spurious
	// matches when iterating with a length-bounded iterator.
	AccountKeyLength() int

	// StorageKeyLength returns the expected total length (prefix + payload)
	// of an on-disk storage key. See AccountKeyLength.
	StorageKeyLength() int

	// AccountPrefixSize returns the per-entry on-disk overhead used by the
	// stateSet to estimate flush sizes. This is just the prefix length for
	// merkle codecs; bintrie codecs may use a different convention.
	AccountPrefixSize() int

	// StoragePrefixSize returns the per-entry on-disk overhead for storage
	// entries.
	StoragePrefixSize() int

	// SplitMarker decomposes a generation progress marker into the account
	// portion and the full marker. For Merkle the account part is the first
	// 32 bytes; for bintrie both halves are the same single 32-byte stem.
	SplitMarker(marker []byte) (accountMarker []byte, fullMarker []byte)

	// MarkerCompare compares a flat-state key against a generation progress
	// marker. Returns the same semantics as bytes.Compare. Used by the
	// disklayer.account/storage gating logic and by writeStates.
	MarkerCompare(key []byte, marker []byte) int

	// StorageMarkerKey returns the byte representation used to compare a
	// (accountHash, storageHash) pair against the generator progress
	// marker in disklayer.storage's generation-progress gate. Merkle
	// uses the 64-byte concatenation (two-tier keying); bintrie uses
	// the 32-byte storageHash directly (single-tier, stem||offset key
	// space matching the bintrie generator's 32-byte marker).
	StorageMarkerKey(accountHash, storageHash common.Hash) []byte

	// Flush drains all pending mutations from the in-memory accountData and
	// storageData maps into the supplied batch and updates the clean cache
	// in lockstep. The codec controls iteration order, key derivation, and
	// any aggregation that may be required (e.g. the bintrie codec must
	// merge per-offset writes into per-stem read-modify-writes to avoid
	// quadratic disk reads).
	//
	// Entries strictly past genMarker (per the codec's MarkerCompare
	// semantics) are skipped because they will be regenerated by the
	// background snapshot generator.
	//
	// Returns (account-entry count, storage-entry count) for metric
	// reporting; the merkle codec reports one per map entry, while the
	// bintrie codec reports one per logical offset write (so the metrics
	// remain comparable across schemes).
	Flush(batch ethdb.Batch, genMarker []byte, accountData map[common.Hash][]byte, storageData map[common.Hash]map[common.Hash][]byte, clean *fastcache.Cache) (int, int, error)
}

// merkleFlatCodec implements flatStateCodec for the keccak-keyed MPT flat
// state scheme. All methods are thin wrappers over rawdb accessors and
// existing helpers; this codec preserves the historical behavior bit-for-bit.
type merkleFlatCodec struct{}

// Compile-time interface check.
var _ flatStateCodec = (*merkleFlatCodec)(nil)

func (c *merkleFlatCodec) AccountKey(addr common.Address) common.Hash {
	return crypto.Keccak256Hash(addr.Bytes())
}

func (c *merkleFlatCodec) StorageKey(addr common.Address, slot common.Hash) (common.Hash, common.Hash) {
	return crypto.Keccak256Hash(addr.Bytes()), crypto.Keccak256Hash(slot.Bytes())
}

func (c *merkleFlatCodec) ReadAccount(db ethdb.KeyValueReader, key common.Hash) []byte {
	return rawdb.ReadAccountSnapshot(db, key)
}

func (c *merkleFlatCodec) ReadStorage(db ethdb.KeyValueReader, accountKey, storageKey common.Hash) []byte {
	return rawdb.ReadStorageSnapshot(db, accountKey, storageKey)
}

func (c *merkleFlatCodec) WriteAccount(batch ethdb.Batch, key common.Hash, blob []byte) {
	rawdb.WriteAccountSnapshot(batch, key, blob)
}

func (c *merkleFlatCodec) DeleteAccount(batch ethdb.Batch, key common.Hash) {
	rawdb.DeleteAccountSnapshot(batch, key)
}

func (c *merkleFlatCodec) WriteStorage(batch ethdb.Batch, accountKey, storageKey common.Hash, blob []byte) {
	rawdb.WriteStorageSnapshot(batch, accountKey, storageKey, blob)
}

func (c *merkleFlatCodec) DeleteStorage(batch ethdb.Batch, accountKey, storageKey common.Hash) {
	rawdb.DeleteStorageSnapshot(batch, accountKey, storageKey)
}

func (c *merkleFlatCodec) AccountCacheKey(key common.Hash) []byte {
	// The historical merkle clean cache uses the bare 32-byte account hash.
	// This is a slice into the caller's hash; callers must not retain it.
	return key[:]
}

func (c *merkleFlatCodec) StorageCacheKey(accountKey, storageKey common.Hash) []byte {
	return storageKeySlice(accountKey, storageKey)
}

func (c *merkleFlatCodec) AccountPrefix() []byte {
	return rawdb.SnapshotAccountPrefix
}

func (c *merkleFlatCodec) StoragePrefix() []byte {
	return rawdb.SnapshotStoragePrefix
}

func (c *merkleFlatCodec) AccountKeyLength() int {
	return len(rawdb.SnapshotAccountPrefix) + common.HashLength
}

func (c *merkleFlatCodec) StorageKeyLength() int {
	return len(rawdb.SnapshotStoragePrefix) + 2*common.HashLength
}

func (c *merkleFlatCodec) AccountPrefixSize() int {
	return len(rawdb.SnapshotAccountPrefix)
}

func (c *merkleFlatCodec) StoragePrefixSize() int {
	return len(rawdb.SnapshotStoragePrefix)
}

func (c *merkleFlatCodec) SplitMarker(marker []byte) ([]byte, []byte) {
	var accMarker []byte
	if len(marker) > 0 {
		accMarker = marker[:common.HashLength]
	}
	return accMarker, marker
}

func (c *merkleFlatCodec) MarkerCompare(key []byte, marker []byte) int {
	return bytes.Compare(key, marker)
}

func (c *merkleFlatCodec) StorageMarkerKey(accountHash, storageHash common.Hash) []byte {
	return storageKeySlice(accountHash, storageHash)
}

// Flush drains the supplied account/storage maps into the batch using the
// historical merkle per-entry layout: one rawdb write per accountData entry
// and one per storage slot. Entries past the genMarker are skipped (the
// generator will fill them in). The clean cache is kept in sync with each
// write so subsequent reads do not stale.
//
// This is the implementation that previously lived directly in writeStates.
// It has been moved into the codec so the bintrie codec can supply its own
// per-stem aggregating implementation alongside this one.
func (c *merkleFlatCodec) Flush(batch ethdb.Batch, genMarker []byte, accountData map[common.Hash][]byte, storageData map[common.Hash]map[common.Hash][]byte, clean *fastcache.Cache) (int, int, error) {
	var (
		accounts int
		slots    int
	)
	for addrHash, blob := range accountData {
		// Skip any account not yet covered by the snapshot. The account
		// at the generation marker position (addrHash == genMarker[:common.HashLength])
		// should still be updated, as it would be skipped in the next
		// generation cycle.
		if genMarker != nil && bytes.Compare(addrHash[:], genMarker) > 0 {
			continue
		}
		accounts++
		cacheKey := c.AccountCacheKey(addrHash)
		if len(blob) == 0 {
			c.DeleteAccount(batch, addrHash)
			if clean != nil {
				clean.Set(cacheKey, []byte{})
			}
		} else {
			c.WriteAccount(batch, addrHash, blob)
			if clean != nil {
				clean.Set(cacheKey, blob)
			}
		}
	}
	for addrHash, storages := range storageData {
		// Skip any account not covered yet by the snapshot
		if genMarker != nil && bytes.Compare(addrHash[:], genMarker) > 0 {
			continue
		}
		midAccount := genMarker != nil && bytes.Equal(addrHash[:], genMarker[:common.HashLength])

		for storageHash, blob := range storages {
			// Skip any storage slot not yet covered by the snapshot. The storage slot
			// at the generation marker position (addrHash == genMarker[:common.HashLength]
			// and storageHash == genMarker[common.HashLength:]) should still be updated,
			// as it would be skipped in the next generation cycle.
			if midAccount && bytes.Compare(storageHash[:], genMarker[common.HashLength:]) > 0 {
				continue
			}
			slots++
			cacheKey := c.StorageCacheKey(addrHash, storageHash)
			if len(blob) == 0 {
				c.DeleteStorage(batch, addrHash, storageHash)
				if clean != nil {
					clean.Set(cacheKey, []byte{})
				}
			} else {
				c.WriteStorage(batch, addrHash, storageHash, blob)
				if clean != nil {
					clean.Set(cacheKey, blob)
				}
			}
		}
	}
	return accounts, slots, nil
}
