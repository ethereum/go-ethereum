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
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// bintrieDiskStore is the bintrie equivalent of diskStore (the merkle
// reader used by the snapshot generator). The two differ in how
// NodeReader validates the requested state root: the merkle store
// hashes the on-disk account-trie root with keccak256, while the
// bintrie root must be deserialized as a binary node and rehashed with
// sha256 (the bintrie's native hash function). Sharing the merkle store
// would always fail validation for a bintrie root.
//
// Once validated, both stores read trie nodes by path via
// rawdb.ReadAccountTrieNode — the path-based key space is shared
// between the two schemes (the bintrie sits in the same namespace as
// the account trie because EIP-7864 unifies storage under accounts).
type bintrieDiskStore struct {
	db ethdb.KeyValueStore
}

// NodeReader validates that the bintrie root currently persisted at the
// account-trie nil path matches the requested state root. The returned
// reader is a plain path-based diskReader (the same one used by the
// merkle generator) — only the validation logic differs.
func (s *bintrieDiskStore) NodeReader(stateRoot common.Hash) (database.NodeReader, error) {
	// EmptyBinaryHash and the legacy EmptyRootHash are both treated as
	// "trie has no persisted root" — neither has a corresponding on-disk
	// node, and the bintrie itself short-circuits these cases inside
	// NewBinaryTrie. We accept them here without touching the disk.
	if stateRoot == (common.Hash{}) || stateRoot == types.EmptyBinaryHash || stateRoot == types.EmptyRootHash {
		return &diskReader{s.db}, nil
	}
	blob := rawdb.ReadAccountTrieNode(s.db, nil)
	if len(blob) == 0 {
		return nil, fmt.Errorf("bintrie state %x is not available (empty root node)", stateRoot)
	}
	// DeserializeNode rehashes via sha256 internally; the resulting node's
	// Hash() is the canonical bintrie root hash for the on-disk blob.
	root, err := bintrie.DeserializeNode(blob, 0)
	if err != nil {
		return nil, fmt.Errorf("bintrie state %x: deserialize root: %w", stateRoot, err)
	}
	if got := root.Hash(); got != stateRoot {
		return nil, fmt.Errorf("bintrie state %x is not available (have %x)", stateRoot, got)
	}
	return &diskReader{s.db}, nil
}

// bintrieGeneratorContext holds the state needed by a single bintrie
// snapshot generation cycle. Unlike generatorContext (which manages two
// holdable iterators over the on-disk merkle account/storage prefixes),
// the bintrie path iterates the trie itself and never re-reads the
// existing flat state. As a result the bintrie context is small: just
// a write batch, the target root, and a single 32-byte progress marker
// (the bintrie key (stem || offset) at which the previous run was
// interrupted).
//
// The context is recreated on every generator restart, mirroring the
// merkle generatorContext lifecycle.
type bintrieGeneratorContext struct {
	root   common.Hash         // State root of the generation target
	marker []byte              // Resume marker — a full 32-byte (stem || offset) key
	db     ethdb.KeyValueStore // Key-value store containing trie nodes and stem blobs
	batch  ethdb.Batch         // Database batch for atomic writes
	logged time.Time           // Timestamp of the last progress log message
}

// newBintrieGeneratorContext initializes a fresh context bound to the
// given target root, starting from the supplied resume marker. A nil or
// zero-length marker means "start from the beginning of the trie".
func newBintrieGeneratorContext(root common.Hash, marker []byte, db ethdb.KeyValueStore) *bintrieGeneratorContext {
	return &bintrieGeneratorContext{
		root:   root,
		marker: marker,
		db:     db,
		batch:  db.NewBatch(),
		logged: time.Now(),
	}
}

// close releases any resources held by the context. The bintrie path
// holds no long-lived iterators outside of generateBinTrieStems (which
// owns its iterator and releases it on return), so this is currently a
// no-op. It exists symmetrically with generatorContext.close so future
// resource additions have an obvious place to land.
func (ctx *bintrieGeneratorContext) close() {}

// generateBinTrieStems regenerates the bintrie flat-state by iterating
// the entire bintrie and emitting one stem blob per stem. The iterator
// yields leaves in stem-then-offset order, so we accumulate offsets in a
// per-stem builder and flush whenever the stem changes (and once more
// at the end of iteration).
//
// Resume support is structural: ctx.marker — a 32-byte (stem || offset)
// key — is fed straight to BinaryTrie.NodeIterator which positions on the
// first leaf with key >= marker via binaryNodeIterator.seek (added in
// Commit 1). Resuming inside a stem is permitted; we re-encode the stem
// from scratch on each visit, so paying the disk cost twice for the
// "interrupted" stem is preferable to introducing a "partial-stem"
// resume protocol.
//
// Range proofs are deliberately not used here. The bintrie's Prove path
// is not implemented yet, and an iteration-only generation cycle is
// acceptable because regeneration is a one-time cost paid at startup.
//
// Code chunks (offsets 128..255) are written to the same stem blobs as
// account header and storage offsets — it keeps the stem encoding
// symmetric with the trie and means a future re-iteration regenerates
// the entire stem layout in one pass.
func (g *generator) generateBinTrieStems(ctx *bintrieGeneratorContext) error {
	// Open the bintrie via the same disk-backed reader that the merkle
	// generator uses. The diskStore reads trie nodes via
	// rawdb.ReadAccountTrieNode/ReadStorageTrieNode against the
	// already-namespaced verkle table (db.diskdb wraps it under
	// VerklePrefix), so the same accessor works for both schemes.
	tr, err := bintrie.NewBinaryTrie(ctx.root, &bintrieDiskStore{db: ctx.db})
	if err != nil {
		log.Info("Bintrie missing, snapshotting paused", "state", ctx.root, "err", err)
		return errMissingTrie
	}
	it, err := tr.NodeIterator(ctx.marker)
	if err != nil {
		return err
	}

	var (
		// currentStem is a freshly-allocated copy of the most recently
		// observed leaf's stem. We never alias the iterator's slice
		// because it can be invalidated on Next.
		currentStem []byte
		builder     = newStemBuilder()
	)

	// flushStem encodes the accumulated builder into a stem blob and
	// writes it to the batch (or deletes the key if the result is
	// empty — which can happen if every observed offset was nil, but
	// that should be impossible for a well-formed trie).
	flushStem := func() {
		if currentStem == nil || builder.empty() {
			return
		}
		blob := builder.encode()
		if blob == nil {
			rawdb.DeleteBinTrieStem(ctx.batch, currentStem)
		} else {
			rawdb.WriteBinTrieStem(ctx.batch, currentStem, blob)
		}
		builder.reset()
		// Bookkeeping: count one stem per emitted blob.
		g.stats.accounts++
	}

	for it.Next(true) {
		if !it.Leaf() {
			continue
		}
		key := it.LeafKey()
		val := it.LeafBlob()

		// A well-formed bintrie leaf is always (32-byte key, 32-byte value).
		// Defensive check so a malformed trie surfaces as an error rather
		// than corrupting the flat state.
		if len(key) != bintrie.StemSize+1 {
			return fmt.Errorf("bintrie leaf key has len %d, want %d", len(key), bintrie.StemSize+1)
		}
		if len(val) != stemBlobValueSize {
			return fmt.Errorf("bintrie leaf value has len %d, want %d", len(val), stemBlobValueSize)
		}

		// Stem boundary detection: if we've moved to a new stem, persist
		// the previous one before starting a new builder.
		if currentStem != nil && !bytes.Equal(key[:bintrie.StemSize], currentStem) {
			flushStem()
			currentStem = nil
		}
		if currentStem == nil {
			currentStem = make([]byte, bintrie.StemSize)
			copy(currentStem, key[:bintrie.StemSize])
		}
		// builder.set takes an owning copy internally so it's safe to
		// hand it the iterator's transient value slice.
		builder.set(key[bintrie.StemSize], val)

		g.stats.slots++
		g.stats.storage += common.StorageSize(1 + bintrie.StemSize + len(val))

		// Use the FULL leaf key (stem || offset) as the progress marker
		// so an interrupted run can resume mid-stem. checkAndFlushBin
		// takes an owning copy because the iterator's key may be
		// invalidated on the next call.
		marker := make([]byte, len(key))
		copy(marker, key)
		if err := g.checkAndFlushBin(ctx, marker); err != nil {
			return err
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	// Flush the trailing stem (the loop only flushes on transitions).
	flushStem()
	return nil
}

// checkAndFlushBin is the bintrie analogue of checkAndFlush. It saves
// progress as a single 32-byte (stem || offset) key and writes the
// batch when it exceeds IdealBatchSize, or when an abort signal is
// received.
//
// Unlike the merkle variant, there are no snapshot iterators to reopen
// here — the bintrie path iterates the trie itself, and the trie
// iterator manages its own resource lifetime.
func (g *generator) checkAndFlushBin(ctx *bintrieGeneratorContext, current []byte) error {
	var abort chan struct{}
	select {
	case abort = <-g.abort:
	default:
	}
	if ctx.batch.ValueSize() > ethdb.IdealBatchSize || abort != nil {
		if bytes.Compare(current, g.progress) < 0 {
			log.Error("Bintrie generator went backwards",
				"current", fmt.Sprintf("%x", current),
				"genMarker", fmt.Sprintf("%x", g.progress))
		}
		// Persist progress regardless of whether the batch is empty —
		// it may be that all observed stems were already on disk and
		// nothing actually changed.
		g.journalProgress(ctx.batch, current, g.stats)

		if err := ctx.batch.Write(); err != nil {
			return err
		}
		ctx.batch.Reset()

		g.lock.Lock()
		g.progress = current
		g.lock.Unlock()

		if abort != nil {
			g.stats.log("Aborting bintrie snapshot generation", ctx.root, g.progress)
			return newAbortErr(abort)
		}
	}
	if time.Since(ctx.logged) > 8*time.Second {
		g.stats.log("Generating bintrie snapshot", ctx.root, g.progress)
		ctx.logged = time.Now()
	}
	return nil
}

// generateBintrie is the bintrie analogue of the merkle `generate`
// background loop. The shapes mirror each other so the lifecycle and
// shutdown protocol look identical to callers (`run` / `stop`):
//
//  1. Persist the initial progress marker if this is a fresh run
//     (so a crash after the first batch can find the genesis marker
//     during recovery).
//  2. Drive generateBinTrieStems to completion (or until an abort).
//  3. On clean completion, write the "done" sentinel marker, log a
//     summary, and close g.done.
//  4. On abort (internal error or external signal), close the abort
//     channel and return.
func (g *generator) generateBintrie(ctx *bintrieGeneratorContext) {
	g.stats.log("Resuming bintrie snapshot generation", ctx.root, g.progress)
	defer ctx.close()

	if len(g.progress) == 0 {
		batch := ctx.db.NewBatch()
		rawdb.WriteSnapshotRoot(batch, ctx.root)
		g.journalProgress(batch, g.progress, g.stats)
		if err := batch.Write(); err != nil {
			log.Crit("Failed to write initialized bintrie state marker", "err", err)
		}
	}

	var abort chan struct{}
	if err := g.generateBinTrieStems(ctx); err != nil {
		var aerr *abortErr
		if errors.As(err, &aerr) {
			abort = aerr.abort
		}
		// Internal error: wait for an external abort signal so the
		// caller's stop() invocation can synchronize.
		if abort == nil {
			abort = <-g.abort
		}
		close(abort)
		return
	}

	// Successful completion: write the nil "done" marker so subsequent
	// loads know the snapshot is complete.
	g.journalProgress(ctx.batch, nil, g.stats)
	if err := ctx.batch.Write(); err != nil {
		log.Error("Failed to flush bintrie batch", "err", err)
		abort = <-g.abort
		close(abort)
		return
	}
	ctx.batch.Reset()

	log.Info("Generated bintrie snapshot",
		"stems", g.stats.accounts,
		"leaves", g.stats.slots,
		"storage", g.stats.storage,
		"elapsed", common.PrettyDuration(time.Since(g.stats.start)))

	g.lock.Lock()
	g.progress = nil
	g.lock.Unlock()
	close(g.done)

	// Block until the eventual stop() so the caller can wait for us.
	abort = <-g.abort
	close(abort)
}
