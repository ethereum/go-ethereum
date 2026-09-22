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

// merkleDisposer deletes the retired merkle state in the background,
// interrupted at shutdown rather than awaited. The marker makes it resumable.
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

// disposeMerkle condemns the merkle state once the window has closed; archive
// nodes keep it. The marker means "gone or going": it lands before the first
// key and is never cleared.
func (bc *BlockChain) disposeMerkle() {
	if bc.cfg.ArchiveMode {
		log.Info("Keeping the merkle state: archive node")
		return
	}
	rawdb.WritePBTMerkleDisposed(bc.db)

	// A pathdb left serving answers deleted records as absent accounts, and
	// closing it does not stop reads reaching disk.
	if err := bc.retireMerkleTree(); err != nil {
		log.Error("Failed to retire the merkle tree, its deletion waits for the next start", "err", err)
		return
	}
	bc.startMerkleDisposal()
}

// retireMerkleTree stops whichever handle holds the merkle namespace: the
// chain's own if the node crossed the fork in-run, a follower-owned one
// otherwise.
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

// merkleRetired reports whether bc.triedb is the handle this run retired, so
// shutdown neither journals nor closes it again.
func (bc *BlockChain) merkleRetired() bool {
	return !bc.triedb.IsPBT() && rawdb.ReadPBTMerkleDisposed(bc.db)
}

// resumeMerkleDisposal finishes a deletion a previous run began. Marker only:
// switching gcmode does not bring back the half that is gone.
func (bc *BlockChain) resumeMerkleDisposal() {
	if rawdb.ReadPBTMerkleDisposed(bc.db) {
		bc.startMerkleDisposal()
	}
}

// SettleMerkleDisposal decides the disposal for a window that closed without
// one. Only the live node calls it: archive mode is this node's
// configuration, not a fact of the datadir, and an offline command's default
// would condemn an archive node's state.
func (bc *BlockChain) SettleMerkleDisposal() error {
	if rawdb.ReadPBTMerkleDisposed(bc.db) || !rawdb.ReadPBTMigrationDone(bc.db) {
		return nil
	}
	// Condemning here would retire the handle this node executes on, and it
	// gets no follower to cross the fork again.
	if !bc.cfg.ArchiveMode && !bc.triedb.IsPBT() {
		return errors.New("migration window closed but the head commits the merkle trie; this datadir cannot follow a pre-fork chain")
	}
	bc.disposeMerkle()
	return nil
}

// startMerkleDisposal launches the deletion once. Window close, startup and
// shutdown all reach the disposer from their own goroutine.
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
	switch err := rawdb.DeleteMerkleState(db, rawdb.PathScheme, interrupt); {
	case errors.Is(err, rawdb.ErrDeleteRangeInterrupted):
		log.Info("Merkle disposal interrupted, resumes on the next start")
		return
	case err != nil:
		log.Error("Failed to dispose of the merkle state", "err", err)
		return
	}
	// History goes only once the state it indexes has.
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
	log.Info("Disposed of the merkle state")
}
