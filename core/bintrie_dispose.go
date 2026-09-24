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
	log.Info("Disposing of the merkle state")
	rawdb.WritePBTMerkleDisposed(bc.db)

	// A live pathdb reads deleted records as empty accounts.
	if err := bc.retireMerkleTree(); err != nil {
		log.Error("Failed to retire the merkle trie, disposal deferred to the next start", "err", err)
		return
	}
	bc.startMerkleDisposal()
}

// retireMerkleTree retires and closes the merkle handle: bc.triedb after an
// in-run crossing, the follower's otherwise.
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
	if handle == bc.triedb {
		bc.merkleRetired.Store(true)
	}
	return handle.Close()
}

// SettleMerkleDisposal marks this process as the live node, the only one that
// decides the disposal: gcmode is per run, and an offline command's default
// must not condemn an archive node's state. It also decides a window that
// closed undecided.
func (bc *BlockChain) SettleMerkleDisposal() error {
	bc.live.Store(true)
	if rawdb.ReadPBTMerkleDisposed(bc.db) || !rawdb.ReadPBTMigrationDone(bc.db) {
		return nil
	}
	// Disposing would retire the handle this node executes on.
	if !bc.cfg.ArchiveMode && !bc.triedb.IsPBT() {
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
