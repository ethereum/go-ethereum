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
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// The merkle trie's retirement. EIP-8347 calls it disposable once the
// migration window has closed, and a full node agrees: execution commits the
// binary tree, and the frozen merkle state served nothing but pre-fork state
// queries. An archive node is the exception - pre-fork history is the service
// it exists to provide - so it keeps both the state and the queries.

// merkleDisposer deletes the retired merkle state in the background. The
// deletion is interruptible rather than awaited to completion: a mainnet
// state is hundreds of millions of keys, and the disposal marker makes an
// unfinished run resumable, so shutdown stops it between batches instead of
// waiting for it.
type merkleDisposer struct {
	quit chan struct{}
	done chan struct{}
}

// stop interrupts the disposal and waits for it to let go of the database,
// which the node is about to close underneath it.
func (d *merkleDisposer) stop() {
	if d == nil {
		return
	}
	close(d.quit)
	<-d.done
}

// disposeMerkle retires the merkle state now that the migration window has
// closed. Archive nodes keep it.
//
// The marker lands before the first deletion and is never cleared, so it
// means "gone or going", never "intact". That is what makes the job
// resumable, and what lets treeFor refuse a handle over state that is only
// half there - a reader with such a handle would take absence for answers.
func (bc *BlockChain) disposeMerkle() {
	if bc.cfg.ArchiveMode {
		log.Info("Keeping the merkle state: archive node")
		return
	}
	rawdb.WritePBTMerkleDisposed(bc.db)

	// Release the tree before deleting what it reads. Past activation the
	// merkle handle is the follower's own - the node's canonical one is the
	// binary tree - and pathdb owns freezers that cannot be reset underneath
	// it. Nothing reopens it: treeFor refuses once the marker is set.
	if bc.follower != nil {
		if t := bc.follower.peek(false); t != nil {
			t.close()
		}
	}
	bc.startMerkleDisposal()
}

// resumeMerkleDisposal finishes a disposal that a crash or a shutdown
// interrupted. A finished one costs one empty scan per key family, which is
// cheaper than a second marker to tell the two states apart.
func (bc *BlockChain) resumeMerkleDisposal() {
	if bc.cfg.ArchiveMode || !rawdb.ReadPBTMerkleDisposed(bc.db) {
		return
	}
	bc.startMerkleDisposal()
}

func (bc *BlockChain) startMerkleDisposal() {
	if bc.disposer != nil {
		return
	}
	bc.disposer = &merkleDisposer{quit: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(bc.disposer.done)
		disposeMerkleState(bc.db, bc.cfg.TrieJournalDirectory, bc.disposer.quit)
	}()
}

// disposeMerkleState deletes the merkle state, then the history freezers and
// journal file that describe it. Every step is a deletion, which is what lets
// an interrupted run simply start again.
func disposeMerkleState(db ethdb.Database, journalDir string, interrupt <-chan struct{}) {
	deleted, done, err := rawdb.DeleteMerkleState(db, interrupt)
	if err != nil {
		log.Error("Failed to dispose of the merkle state", "records", deleted, "err", err)
		return
	}
	if !done {
		log.Info("Merkle disposal interrupted, resumes on the next start", "records", deleted)
		return
	}
	// The history is only reset once the state it indexes is gone, so an
	// interrupted run leaves history for state that still exists rather than
	// the other way round.
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
