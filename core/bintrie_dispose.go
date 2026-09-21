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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
)

// merkleDisposer deletes the retired merkle state in the background. The
// deletion is interrupted at shutdown rather than awaited: a mainnet state is
// hundreds of millions of keys, and the marker makes it resumable.
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

// disposeMerkle retires the merkle state now that the window has closed;
// archive nodes keep it. The marker lands before the first deletion and is
// never cleared, so it means "gone or going", which is what makes the job
// resumable and lets treeFor refuse a handle over half-deleted state.
func (bc *BlockChain) disposeMerkle() {
	if bc.cfg.ArchiveMode {
		log.Info("Keeping the merkle state: archive node")
		return
	}
	rawdb.WritePBTMerkleDisposed(bc.db)

	// Stop the live merkle tree before deleting what it reads: a pathdb that
	// keeps serving answers every deleted record as an absent account.
	// Retiring marks its layers stale, so reads fail loudly instead, and
	// closing it lets go of the history freezers the deletion resets.
	if err := bc.retireMerkleTree(); err != nil {
		log.Error("Failed to retire the merkle tree, its deletion waits for the next start", "err", err)
		return
	}
	bc.startMerkleDisposal()
}

// retireMerkleTree stops whichever handle holds the merkle namespace in this
// run. At most one can: the chain's own when the node crossed the fork
// in-run - the flavour is resolved once, at open - and a follower-owned one
// otherwise, since followerTree.open shares the chain's handle when the
// flavours match.
func (bc *BlockChain) retireMerkleTree() error {
	var handle *triedb.Database
	if bc.follower != nil {
		if t := bc.follower.peek(false); t != nil {
			handle = t.release()
		}
	}
	if !bc.triedb.IsPBT() {
		handle = bc.triedb
	}
	if handle == nil {
		return nil
	}
	if err := handle.Retire(); err != nil {
		return err
	}
	return handle.Close()
}

// merkleRetired reports whether bc.triedb is the merkle handle this run
// retired and closed. Shutdown must not journal or close it again: its
// layers are stale and the state a journal would describe is gone. A node
// cannot start in this state - NewBlockChain refuses a merkle head once the
// marker is set - so the marker plus a merkle handle means "retired here".
func (bc *BlockChain) merkleRetired() bool {
	return !bc.triedb.IsPBT() && rawdb.ReadPBTMerkleDisposed(bc.db)
}

// resumeMerkleDisposal finishes a deletion a previous run began. It runs in
// every NewBlockChain, offline commands included, and reads the marker
// alone: a deletion under way is safe to continue from anywhere, archive or
// not, since switching gcmode does not bring back the half that is gone.
func (bc *BlockChain) resumeMerkleDisposal() {
	if rawdb.ReadPBTMerkleDisposed(bc.db) {
		bc.startMerkleDisposal()
	}
}

// SettleMerkleDisposal decides the disposal for a migration that finished
// without one: window close is a one-shot on the follower's last act, so a
// crash between the two markers - or a node that ran as archive then - would
// keep the state forever. Only the live node calls it. Whether this is an
// archive node is a fact of its configuration, not of the datadir, and the
// default an offline command runs with would condemn an archive node's
// state from an unrelated geth invocation.
func (bc *BlockChain) SettleMerkleDisposal() error {
	if rawdb.ReadPBTMerkleDisposed(bc.db) || !rawdb.ReadPBTMigrationDone(bc.db) {
		return nil
	}
	// A full node whose head is back under the boundary executes on the
	// merkle trie, and gets no follower to cross the fork again once the
	// window has closed: condemning it here would retire the handle it
	// runs on. Refuse instead, the way a start on a condemned datadir is.
	if !bc.cfg.ArchiveMode && !bc.triedb.IsPBT() {
		return errors.New("migration window closed but the head commits the merkle trie; this datadir cannot follow a pre-fork chain")
	}
	bc.disposeMerkle()
	return nil
}

// startMerkleDisposal launches the deletion once. The disposer is published
// atomically: the window closes on the follower's goroutine, startup and
// shutdown read it from their own.
func (bc *BlockChain) startMerkleDisposal() {
	d := &merkleDisposer{quit: make(chan struct{}), done: make(chan struct{})}
	if !bc.disposer.CompareAndSwap(nil, d) {
		return
	}
	go func() {
		defer close(d.done)
		disposeMerkleState(bc.db, bc.cfg.TrieJournalDirectory, d.quit)
	}()
}

// disposeMerkleState deletes the merkle state, then the history and journal
// that describe it. Every step is a deletion, so a partial run just restarts.
func disposeMerkleState(db ethdb.Database, journalDir string, interrupt <-chan struct{}) {
	deleted, done, err := rawdb.DeleteMerkleState(db, rawdb.PathScheme, interrupt)
	if err != nil {
		log.Error("Failed to dispose of the merkle state", "records", deleted, "err", err)
		return
	}
	if !done {
		log.Info("Merkle disposal interrupted, resumes on the next start", "records", deleted)
		return
	}
	// History only goes once the state it indexes has, never the reverse.
	if ancient, err := db.AncientDatadir(); err == nil {
		for _, open := range []func(string, bool, bool) (ethdb.ResettableAncientStore, error){
			rawdb.NewStateFreezer,
			rawdb.NewTrienodeFreezer,
		} {
			store, err := open(ancient, false, false)
			if err != nil {
				log.Error("Failed to open a merkle history freezer", "err", err)
				return
			}
			err = store.Reset()
			store.Close()
			if err != nil {
				log.Error("Failed to reset a merkle history freezer", "err", err)
				return
			}
		}
	}
	if journalDir != "" {
		if err := os.Remove(filepath.Join(journalDir, "merkle.journal")); err != nil && !os.IsNotExist(err) {
			log.Error("Failed to remove the merkle journal", "err", err)
			return
		}
	}
	log.Info("Disposed of the merkle state", "records", deleted)
}
