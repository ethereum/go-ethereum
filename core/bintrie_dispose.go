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
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
)

// merkleDisposer deletes the retired merkle state in the background; Stop
// interrupts it and the marker resumes it.
type merkleDisposer struct {
	quit chan struct{}
	done chan struct{}
}

// stop interrupts the disposal and waits for it to let go of the database.
func (d *merkleDisposer) stop() {
	if d == nil {
		return
	}
	close(d.quit)
	<-d.done
}

// disposeMerkle retires and deletes the merkle state. Only the live node
// decides, and an archive node keeps it.
func (bc *BlockChain) disposeMerkle() {
	if !bc.live.Load() || bc.cfg.ArchiveMode {
		return
	}
	// The merkle state is the way back from a crash until a post-fork
	// binary state is on disk, which the write buffer alone can delay for
	// hours: put one there first. Flattening stops at the finalized block,
	// so the layers above it stay live for reorgs.
	keep := bc.CurrentBlock()
	if final := bc.CurrentFinalBlock(); final != nil && bc.chainConfig.IsBinaryTrie(final.Number, final.Time) {
		if child := bc.GetHeaderByNumber(final.Number.Uint64() + 1); child != nil && child.ParentHash == final.Hash() {
			keep = child
		}
	}
	if err := bc.persistBinary(keep); err != nil {
		log.Error("Failed to persist the binary tree, disposal deferred to the next start", "err", err)
		return
	}
	pbtPersistedGauge.Update(int64(keep.Number.Uint64() - 1))
	log.Info("Disposing of the merkle state", "persisted", keep.Number.Uint64()-1)
	rawdb.WritePBTMerkleDisposed(bc.db)
	bc.disposed.Store(true)

	// A live pathdb reads deleted records as empty accounts.
	if err := bc.retireMerkleTree(); err != nil {
		log.Error("Failed to retire the merkle trie, disposal deferred to the next start", "err", err)
		return
	}
	bc.startMerkleDisposal()
}

// persistBinary puts the binary state of the head's parent on disk, keeping
// the head and anything above it live. Nothing to do below the second
// post-fork block, or when the parent is at or below the disk layer.
func (bc *BlockChain) persistBinary(head *types.Header) error {
	parent := bc.GetHeader(head.ParentHash, head.Number.Uint64()-1)
	if parent == nil || !bc.chainConfig.IsBinaryTrie(parent.Number, parent.Time) {
		return nil
	}
	pbtdb := rawdb.NewTable(bc.db, string(rawdb.PBTPrefix))
	if id := rawdb.ReadStateID(pbtdb, parent.Root); id != nil && *id <= rawdb.ReadPersistentStateID(pbtdb) {
		return nil
	}
	tdb, err := bc.treeFor(true)
	if err != nil {
		return err
	}
	return tdb.Persist(head.Root)
}

// checkpointActivation runs once per process, on the first block whose parent
// is post-fork: it journals the frozen merkle trie at the boundary parent, so
// a kill during the window reopens the merkle trie there rather than at a
// disk layer hours older. The binary tree is not persisted here: that would
// flatten the pre-fork layers a straddle reorg still needs. The window close
// persists it, once the boundary is final.
func (bc *BlockChain) checkpointActivation(block *types.Block) {
	if bc.follower == nil || bc.activationCheckpointed.Load() {
		return
	}
	boundary := bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
	if boundary == nil || !bc.chainConfig.IsBinaryTrie(boundary.Number, boundary.Time) {
		return
	}
	merkle := bc.merkleHandle()
	if merkle == nil {
		return
	}
	for bc.chainConfig.IsBinaryTrie(boundary.Number, boundary.Time) {
		if boundary = bc.GetHeader(boundary.ParentHash, boundary.Number.Uint64()-1); boundary == nil {
			return
		}
	}
	if err := merkle.Checkpoint(boundary.Root); err != nil {
		log.Warn("Failed to journal the merkle trie at the boundary", "block", boundary.Number, "err", err)
	} else {
		log.Info("Journaled the merkle trie at the boundary", "block", boundary.Number)
	}
	bc.activationCheckpointed.Store(true)
}

// merkleHandle returns the live merkle trie: bc.trieDB() before the crossing
// and after an in-run one, the follower's when the node started post-fork.
func (bc *BlockChain) merkleHandle() *triedb.Database {
	if !bc.trieDB().IsPBT() {
		return bc.trieDB()
	}
	if bc.follower != nil {
		if t := bc.follower.peek(false); t != nil {
			bc.follower.mu.Lock()
			defer bc.follower.mu.Unlock()
			return t.handle
		}
	}
	return nil
}

// retireMerkleTree retires and closes the merkle handle. After an in-run
// crossing that is bc.trieDB(), and the chain adopts the follower's binary
// handle in its place: the tree it executes on, owned from here on like any
// other start's. Otherwise the follower's own merkle handle goes.
func (bc *BlockChain) retireMerkleTree() error {
	var handle *triedb.Database
	if bc.follower != nil {
		if t := bc.follower.peek(false); t != nil {
			handle = t.release()
		}
	}
	if merkle := bc.trieDB(); !merkle.IsPBT() {
		if bc.follower == nil {
			return errors.New("no binary tree to adopt")
		}
		binary := bc.follower.direction(true).adopt()
		if binary == nil {
			return errors.New("no binary tree to adopt")
		}
		bc.triedb.Store(binary)
		handle = merkle
	}
	if handle == nil {
		return nil
	}
	if err := handle.Retire(); err != nil {
		return err
	}
	return handle.Close()
}

// SettleMerkleDisposal marks this process as the live node, the only one that
// decides the disposal: gcmode is per run, and an offline command's default
// must not condemn an archive node's state. It also decides a window that
// closed undecided.
func (bc *BlockChain) SettleMerkleDisposal() error {
	bc.live.Store(true)
	if bc.disposed.Load() || !rawdb.ReadPBTMigrationDone(bc.db) {
		return nil
	}
	// Disposing would retire the handle this node executes on.
	if !bc.cfg.ArchiveMode && !bc.trieDB().IsPBT() {
		return errors.New("migration window closed at a pre-fork head: re-anchor or resync")
	}
	bc.disposeMerkle()
	return nil
}

// startMerkleDisposal launches the deletion once.
func (bc *BlockChain) startMerkleDisposal() {
	d := &merkleDisposer{quit: make(chan struct{}), done: make(chan struct{})}
	if !bc.disposer.CompareAndSwap(nil, d) {
		return
	}
	go func() {
		defer close(d.done)
		switch err := DisposeMerkleState(bc.db, bc.cfg.TrieJournalDirectory, d.quit); {
		case errors.Is(err, rawdb.ErrDeleteRangeInterrupted):
			log.Info("Merkle disposal interrupted")
		case err != nil:
			log.Error("Failed to dispose of the merkle state", "err", err)
		default:
			// Debug: every start after the first disposal reruns it.
			log.Debug("Disposed of the merkle state")
		}
	}()
}

// DisposeMerkleState deletes the merkle state, its history freezers and its
// journal file. Every step is idempotent, so an interrupted run restarts. The
// caller writes the disposal marker first and closes every merkle handle.
func DisposeMerkleState(db ethdb.Database, journalDir string, interrupt <-chan struct{}) error {
	if err := rawdb.DeleteMerkleState(db, interrupt); err != nil {
		return err
	}
	if ancient, err := db.AncientDatadir(); err == nil {
		for _, open := range []func(string, bool, bool) (ethdb.ResettableAncientStore, error){
			rawdb.NewStateFreezer,
			rawdb.NewTrienodeFreezer,
		} {
			store, err := open(ancient, false, false)
			if err != nil {
				return err
			}
			err = store.Reset()
			store.Close()
			if err != nil {
				return err
			}
		}
	}
	if journalDir != "" {
		if err := os.Remove(filepath.Join(journalDir, "merkle.journal")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// ImportsBinaryTree reports whether this chain runs an import or
// self-conversion: the flags were given and the tree is not in place yet.
func (bc *BlockChain) ImportsBinaryTree() bool {
	return bc.cfg.PBTImport
}
