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
	"fmt"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// bintrieFlatCodec implements flatStateCodec for the binary trie using the
// stem-blob on-disk layout defined in stem_blob.go. Keys are the 32-byte
// stems of the EIP-7864 binary state tree (the first 31 bytes of the full
// bintrie key, zero-padded into a common.Hash) and values are packed stem
// blobs containing the subset of 256 offsets that have been written at
// that stem.
//
// Unlike merkleFlatCodec (which is a stateless singleton), this codec
// holds a reference to the underlying key-value store so its Write/Delete
// methods can perform a read-modify-write on the existing stem blob
// before merging in the new (offset, value) pair. ethdb.Batch is
// write-only, so the batch passed to Write* cannot be used to fetch the
// current state of a stem.
//
// Pre-aggregation requirement: within a single flush pass, the caller
// must NOT issue two Write* calls targeting the same stem. The codec
// reads the stem from the store (not from the in-flight batch), so a
// second write at the same stem would re-read the pre-flush state and
// clobber the first write. The codec's public surface area is designed
// around this assumption; Commit 8 of the bintrie flat-state plan
// restructures writeStates to pre-aggregate per-stem writes so callers
// do not have to handle this manually.
//
// This codec is NOT wired into pathdb.Database.New yet — that happens in a
// later commit once the leaf-production hook in binaryHasher and the
// stateUpdate wiring are in place. Until then, all call sites still
// dispatch through merkleFlatCodec and bintrie mode continues to use the
// (soon to be replaced) keccak-shaped flat-state layout.
type bintrieFlatCodec struct {
	// db is the underlying key-value store used by applyWrites to read
	// the current stem blob before merging in new (offset, value) pairs.
	// It is always the pathdb Database's already-wrapped diskdb (the
	// VerklePrefix-namespaced table) so reads and writes share the same
	// on-disk key space.
	db ethdb.KeyValueReader
}

// newBintrieFlatCodec constructs a bintrieFlatCodec bound to the given
// key-value reader. The reader is used for read-modify-write on stem
// blobs; writes still flow through the ethdb.Batch passed to each
// Write*/Delete* call.
func newBintrieFlatCodec(db ethdb.KeyValueReader) *bintrieFlatCodec {
	return &bintrieFlatCodec{db: db}
}

// Compile-time interface assertion.
var _ flatStateCodec = (*bintrieFlatCodec)(nil)

// bintrieCacheKeyPrefix is a one-byte prefix applied to all bintrie cache
// keys to keep them disjoint from merkle account keys (which are raw
// 32-byte hashes) and merkle storage keys (which are 64-byte
// accountHash||storageHash) in the shared clean-state fastcache. Without a
// prefix, a 32-byte merkle account hash and a 32-byte bintrie stem could
// collide on the same cache slot and return wrong data on read.
const bintrieCacheKeyPrefix byte = 0x01

// stemFromKey extracts the 31-byte stem from a 32-byte flat-state key.
// Bintrie keys follow the "stem || offset" layout (EIP-7864), so the stem
// is always bytes [0..30] and the byte at index 31 is the offset within
// the stem. Callers that use AccountKey()/StorageKey() followed by
// Read/Write never need to look at the offset themselves — the codec
// handles offset extraction internally.
func stemFromKey(key common.Hash) []byte {
	return key[:bintrie.StemSize]
}

// offsetFromKey returns the offset byte of a 32-byte flat-state key.
func offsetFromKey(key common.Hash) byte {
	return key[bintrie.StemSize]
}

// ---------------------------------------------------------------------
// Key derivation
// ---------------------------------------------------------------------

// AccountKey returns the bintrie BasicData key for the given address.
// The result has the account's 31-byte stem in bytes [0..30] and offset 0
// (BasicDataLeafKey) in byte 31. The CodeHash leaf lives at the same stem
// with offset 1, so a single ReadAccount is enough to materialize both
// offsets via the returned stem blob.
func (c *bintrieFlatCodec) AccountKey(addr common.Address) common.Hash {
	return common.BytesToHash(bintrie.GetBinaryTreeKeyBasicData(addr))
}

// StorageKey returns the bintrie key for a storage slot. The first return
// value (the "account key" in the merkle naming convention) is the zero
// hash because bintrie has no per-account grouping at the flat-state
// level; the second return value is the full 32-byte slot key (stem ||
// offset). Callers must pass both values back through the Read/Write
// storage methods so the codec can recover the stem and offset.
func (c *bintrieFlatCodec) StorageKey(addr common.Address, slot common.Hash) (common.Hash, common.Hash) {
	full := bintrie.GetBinaryTreeKeyStorageSlot(addr, slot[:])
	return common.Hash{}, common.BytesToHash(full)
}

// ---------------------------------------------------------------------
// Disk reads
// ---------------------------------------------------------------------

// ReadAccount returns the raw stem blob for the account's stem — NOT a
// decoded account. The caller (e.g. bintrieFlatReader in a later commit)
// is responsible for extracting BasicData (offset 0) and CodeHash
// (offset 1) from the blob.
//
// This signature asymmetry with merkleFlatCodec.ReadAccount (which
// returns slim-RLP-encoded account bytes) is intentional: a bintrie stem
// blob can contain data for many logical fields, and the caller decides
// which offsets to extract. A higher-level "return an assembled Account"
// helper would have to re-encode into a format no consumer wants.
func (c *bintrieFlatCodec) ReadAccount(db ethdb.KeyValueReader, key common.Hash) []byte {
	return rawdb.ReadBinTrieStem(db, stemFromKey(key))
}

// ReadStorage returns the 32-byte value stored at the storage slot's
// offset within its stem, or nil if the offset is not populated.
//
// Unlike ReadAccount, this method DOES perform offset extraction from
// the stem blob: storage-slot reads are always a single-offset query, so
// returning the whole blob would just force every caller to re-run the
// extraction. A malformed stem blob is treated as absent and logged
// (returning nil) to match the behavior of rawdb.ReadStorageSnapshot on
// the merkle path.
//
// The first parameter (accountKey) is ignored: see StorageKey for the
// reasoning behind the bintrie's zero-hash convention.
func (c *bintrieFlatCodec) ReadStorage(db ethdb.KeyValueReader, _ common.Hash, storageKey common.Hash) []byte {
	blob := rawdb.ReadBinTrieStem(db, stemFromKey(storageKey))
	if len(blob) == 0 {
		return nil
	}
	val, err := extractStemOffset(blob, offsetFromKey(storageKey))
	if err != nil {
		// A well-formed blob never errors on a point read. If we get
		// here the on-disk layout is corrupted — return nil rather than
		// propagating the error, since the interface has no error path
		// (the caller expects a value-or-nil just like
		// rawdb.ReadStorageSnapshot).
		return nil
	}
	return val
}

// ---------------------------------------------------------------------
// Disk writes
// ---------------------------------------------------------------------

// WriteAccount writes an account entry. The blob is expected to be a
// two-slot payload containing BasicData (bytes 0..31) followed by the
// code hash (bytes 32..63) — the caller (binaryHasher, in a later
// commit) packs these together because they live at the same stem and
// benefit from a single read-modify-write pass.
//
// Writing nil or an empty blob is equivalent to clearing offsets 0 and 1
// at this stem (a partial account deletion); the codec merges the
// resulting bitmap into the existing stem blob and deletes the key
// entirely if no offsets remain set.
//
// An error from mergeStemBlob (e.g. malformed existing blob) is logged
// via log.Crit because flat-state corruption is unrecoverable at this
// layer — same policy as rawdb.WriteAccountSnapshot.
func (c *bintrieFlatCodec) WriteAccount(batch ethdb.Batch, key common.Hash, blob []byte) {
	writes, err := splitAccountBlob(blob)
	if err != nil {
		crit("bintrie WriteAccount: %v", err)
		return
	}
	c.applyWrites(batch, stemFromKey(key), writes)
}

// DeleteAccount clears offsets 0 (BasicData) and 1 (CodeHash) at the
// account's stem. Other offsets at the same stem (e.g. header storage
// slots) are NOT touched — callers that want a full account wipe must
// walk storage separately, which is consistent with the bintrie's
// DeleteAccount semantics (see trie/bintrie/trie.go).
func (c *bintrieFlatCodec) DeleteAccount(batch ethdb.Batch, key common.Hash) {
	writes := []stemOffsetValue{
		{Offset: bintrie.BasicDataLeafKey, Value: nil},
		{Offset: bintrie.CodeHashLeafKey, Value: nil},
	}
	c.applyWrites(batch, stemFromKey(key), writes)
}

// WriteStorage writes a single storage-slot value. The blob must be 32
// bytes (the canonical storage value width); a shorter/longer blob is a
// caller bug and is logged via log.Crit.
//
// The first parameter (accountKey) is ignored — see StorageKey.
func (c *bintrieFlatCodec) WriteStorage(batch ethdb.Batch, _ common.Hash, storageKey common.Hash, blob []byte) {
	if len(blob) != stemBlobValueSize {
		crit("bintrie WriteStorage: value has len %d, want %d", len(blob), stemBlobValueSize)
		return
	}
	writes := []stemOffsetValue{{Offset: offsetFromKey(storageKey), Value: blob}}
	c.applyWrites(batch, stemFromKey(storageKey), writes)
}

// DeleteStorage clears a single offset at a stem. If the stem has no
// other populated offsets afterwards, the key is removed entirely.
func (c *bintrieFlatCodec) DeleteStorage(batch ethdb.Batch, _ common.Hash, storageKey common.Hash) {
	writes := []stemOffsetValue{{Offset: offsetFromKey(storageKey), Value: nil}}
	c.applyWrites(batch, stemFromKey(storageKey), writes)
}

// applyWrites performs a read-modify-write on the given stem: reads the
// existing blob via the codec's bound reader, merges in the supplied
// (offset, value) pairs, and writes the result back via the batch — or
// deletes the key if the merged result is empty. Shared by all four
// Write/Delete methods to ensure the policy (nil value clears, empty
// blob deletes) is consistent.
//
// Returns the merged blob (or nil if the stem was deleted) so callers
// such as Flush can repopulate the clean cache without an extra disk
// read. The returned slice is freshly allocated and owned by the caller.
//
// Important: the read comes from c.db, NOT from the batch. A second
// call for the same stem within a flush would re-read the pre-flush
// state; see the pre-aggregation requirement documented on
// bintrieFlatCodec.
func (c *bintrieFlatCodec) applyWrites(batch ethdb.Batch, stem []byte, writes []stemOffsetValue) []byte {
	existing := rawdb.ReadBinTrieStem(c.db, stem)
	merged, err := mergeStemBlob(existing, writes)
	if err != nil {
		crit("bintrie applyWrites: %v", err)
		return nil
	}
	if merged == nil {
		rawdb.DeleteBinTrieStem(batch, stem)
		return nil
	}
	rawdb.WriteBinTrieStem(batch, stem, merged)
	return merged
}

// splitAccountBlob validates and splits the two-slot account payload
// passed to WriteAccount. A nil or empty blob is interpreted as
// "clear both offsets".
func splitAccountBlob(blob []byte) ([]stemOffsetValue, error) {
	if len(blob) == 0 {
		return []stemOffsetValue{
			{Offset: bintrie.BasicDataLeafKey, Value: nil},
			{Offset: bintrie.CodeHashLeafKey, Value: nil},
		}, nil
	}
	if len(blob) != 2*stemBlobValueSize {
		return nil, fmt.Errorf("account blob len %d, want %d (BasicData || CodeHash)", len(blob), 2*stemBlobValueSize)
	}
	return []stemOffsetValue{
		{Offset: bintrie.BasicDataLeafKey, Value: blob[:stemBlobValueSize]},
		{Offset: bintrie.CodeHashLeafKey, Value: blob[stemBlobValueSize:]},
	}, nil
}

// ---------------------------------------------------------------------
// Clean-cache keys
// ---------------------------------------------------------------------

// AccountCacheKey returns a disambiguated byte key for the shared
// fastcache-backed clean state cache. The prefix byte
// bintrieCacheKeyPrefix keeps bintrie stem lookups disjoint from merkle
// account lookups (both of which use 32-byte keys), and from merkle
// storage lookups (which use 64-byte keys). The stem (31 bytes) is
// embedded after the prefix; the offset byte is not included because
// the cache entry caches the whole stem blob, not a single offset.
func (c *bintrieFlatCodec) AccountCacheKey(key common.Hash) []byte {
	out := make([]byte, 1+bintrie.StemSize)
	out[0] = bintrieCacheKeyPrefix
	copy(out[1:], stemFromKey(key))
	return out
}

// StorageCacheKey returns the cache key for a storage entry. For bintrie
// this is the same stem as the account cache key — storage slots and
// account header live at different stems in the general case, but
// multiple storage slots of the same stem share a single cache entry.
// The accountKey parameter is ignored (see StorageKey).
func (c *bintrieFlatCodec) StorageCacheKey(_ common.Hash, storageKey common.Hash) []byte {
	out := make([]byte, 1+bintrie.StemSize)
	out[0] = bintrieCacheKeyPrefix
	copy(out[1:], stemFromKey(storageKey))
	return out
}

// ---------------------------------------------------------------------
// Generator iterator configuration
// ---------------------------------------------------------------------

// AccountPrefix returns the rawdb key prefix used for bintrie flat-state
// entries. The generator iterator uses this prefix to walk all stem
// blobs for the initial population of the flat state from an existing
// bintrie.
func (c *bintrieFlatCodec) AccountPrefix() []byte {
	return rawdb.BinTrieStemPrefix
}

// StoragePrefix returns the same prefix as AccountPrefix because bintrie
// flat-state entries are stored in a single namespace (stems contain
// both account and storage data). The generator in a later commit uses
// a single iterator over this prefix rather than the two-tier
// account-then-storage walk used by the merkle generator.
func (c *bintrieFlatCodec) StoragePrefix() []byte {
	return rawdb.BinTrieStemPrefix
}

// AccountKeyLength returns the expected on-disk key length for a stem
// entry: 1 byte of prefix + 31 bytes of stem = 32 bytes total.
func (c *bintrieFlatCodec) AccountKeyLength() int {
	return len(rawdb.BinTrieStemPrefix) + bintrie.StemSize
}

// StorageKeyLength returns the same length as AccountKeyLength because
// bintrie stems are a single unified namespace.
func (c *bintrieFlatCodec) StorageKeyLength() int {
	return len(rawdb.BinTrieStemPrefix) + bintrie.StemSize
}

// AccountPrefixSize returns the per-entry on-disk overhead used by the
// stateSet to estimate flush sizes. For bintrie this is just the single
// byte of BinTrieStemPrefix.
func (c *bintrieFlatCodec) AccountPrefixSize() int {
	return len(rawdb.BinTrieStemPrefix)
}

// StoragePrefixSize returns the same as AccountPrefixSize.
func (c *bintrieFlatCodec) StoragePrefixSize() int {
	return len(rawdb.BinTrieStemPrefix)
}

// ---------------------------------------------------------------------
// Generation progress marker
// ---------------------------------------------------------------------

// SplitMarker splits a generation progress marker into the account and
// full components. For bintrie the marker is a single 31-byte stem (or
// the full 32-byte key with offset 0), not the merkle two-tier
// account-then-storage format, so both returned slices point at the
// same data. The second half of the merkle marker (storage offset) has
// no equivalent for bintrie: the generator iterates stems directly,
// not (account, storage) pairs.
func (c *bintrieFlatCodec) SplitMarker(marker []byte) ([]byte, []byte) {
	if len(marker) == 0 {
		return nil, marker
	}
	return marker, marker
}

// MarkerCompare compares a flat-state key against a progress marker with
// bytes.Compare semantics, mirroring the merkle codec. The bintrie keys
// being compared are stem bytes (31 bytes) or full keys (32 bytes); both
// are lexicographically ordered so bytes.Compare is the correct
// ordering.
func (c *bintrieFlatCodec) MarkerCompare(key []byte, marker []byte) int {
	return bytes.Compare(key, marker)
}

// Flush drains the in-memory accountData and storageData maps into the
// batch using the bintrie per-stem layout. The maps are expected to hold
// per-offset entries — each key is a 32-byte (stem || offset) tuple
// produced by AccountKey/StorageKey, and each value is a 32-byte leaf
// (or nil to clear that offset).
//
// All entries are first grouped by stem, then a single
// read-modify-write is issued per stem so the codec touches each stem
// at most once during a flush. This is what allows the per-call
// pre-aggregation requirement documented on bintrieFlatCodec to be
// satisfied even when many writes target the same stem.
//
// storageData is also walked because higher-level callers may emit
// storage entries that the codec routes through the storage map for
// historical reasons; for the bintrie path, entries should normally
// arrive on accountData but we accept either layout.
//
// Returns (offset count from accountData, offset count from storageData)
// so the metric reporting in writeStates remains comparable to the
// merkle path. The clean cache is updated with the merged stem blob
// (one cache entry per stem, not per offset) — readers extract the
// requested offset on hit.
func (c *bintrieFlatCodec) Flush(batch ethdb.Batch, genMarker []byte, accountData map[common.Hash][]byte, storageData map[common.Hash]map[common.Hash][]byte, clean *fastcache.Cache) (int, int) {
	// Aggregate per-offset writes into per-stem batches. We use [31]byte
	// as the map key because bytes slices aren't hashable in Go and the
	// stem itself is fixed size; the alternative (using common.Hash with
	// a zero pad) would waste a byte per entry.
	type aggregator struct {
		writes []stemOffsetValue
	}
	aggregated := make(map[[bintrie.StemSize]byte]*aggregator)

	addWrite := func(fullKey common.Hash, value []byte) {
		var stem [bintrie.StemSize]byte
		copy(stem[:], fullKey[:bintrie.StemSize])
		offset := fullKey[bintrie.StemSize]
		ag, exists := aggregated[stem]
		if !exists {
			ag = &aggregator{}
			aggregated[stem] = ag
		}
		ag.writes = append(ag.writes, stemOffsetValue{Offset: offset, Value: value})
	}

	var (
		accountWrites int
		storageWrites int
	)
	for fullKey, value := range accountData {
		// genMarker filtering: skip stems that the generator hasn't
		// reached yet. We compare against the FULL key (stem || offset)
		// because the bintrie marker is itself a 32-byte key.
		if genMarker != nil && bytes.Compare(fullKey[:], genMarker) > 0 {
			continue
		}
		accountWrites++
		addWrite(fullKey, value)
	}
	for _, slots := range storageData {
		for fullKey, value := range slots {
			if genMarker != nil && bytes.Compare(fullKey[:], genMarker) > 0 {
				continue
			}
			storageWrites++
			addWrite(fullKey, value)
		}
	}
	// Issue one RMW per stem and update the clean cache with the merged
	// blob (or invalidate it if the stem was deleted).
	for stem, ag := range aggregated {
		merged := c.applyWrites(batch, stem[:], ag.writes)
		if clean != nil {
			// Reuse AccountCacheKey to derive the cache key — for
			// bintrie this only depends on the stem so the trailing
			// offset byte in the synthetic full key is irrelevant.
			var fullKey common.Hash
			copy(fullKey[:bintrie.StemSize], stem[:])
			cacheKey := c.AccountCacheKey(fullKey)
			if merged == nil {
				clean.Set(cacheKey, nil)
			} else {
				clean.Set(cacheKey, merged)
			}
		}
	}
	return accountWrites, storageWrites
}

// crit is a shim around log.Crit that allows tests to replace the fatal
// behavior with a panic if needed. Defined at the package level to match
// the single-call-per-error style used by the merkle codec.
func crit(format string, args ...any) {
	// Import cycle avoidance: we delegate to log.Crit via the existing
	// import in this package (see flat_codec.go for the merkle codec,
	// which uses log.Crit through rawdb's own accessors).
	// Here we keep the dependency light by just panicking; production
	// flat-state corruption is unrecoverable and panicking surfaces the
	// issue immediately rather than letting a silently-corrupted state
	// root propagate.
	panic(fmt.Sprintf(format, args...))
}
