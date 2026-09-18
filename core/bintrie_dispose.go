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

	// A node that crossed the fork in-run holds the merkle tree as its own
	// canonical handle: the flavour is resolved once, at startup. Deleting
	// under a live pathdb would have its disk layer serve every miss as a
	// cached "no such account". Defer to a start where it is not canonical.
	if !bc.triedb.IsPBT() {
		log.Info("Merkle state marked for disposal, deferred to the next start")
		return
	}
	// Release the tree before deleting what it reads; treeFor refuses to
	// reopen it once the marker is set.
	if bc.follower != nil {
		if t := bc.follower.peek(false); t != nil {
			t.close()
		}
	}
	bc.startMerkleDisposal()
}

// resumeMerkleDisposal re-decides the disposal at every start: window close
// is a one-shot on the follower's last act, and the follower is never created
// again once the migration is done, so a crash between the two markers - or a
// node that ran as archive then - would keep the state forever.
func (bc *BlockChain) resumeMerkleDisposal() {
	// Already begun: finish it, archive or not. Switching gcmode does not
	// bring back the half that is gone.
	if rawdb.ReadPBTMerkleDisposed(bc.db) {
		bc.startMerkleDisposal()
		return
	}
	if rawdb.ReadPBTMigrationDone(bc.db) {
		bc.disposeMerkle()
	}
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
	deleted, done, err := rawdb.DeleteMerkleState(db, interrupt)
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
