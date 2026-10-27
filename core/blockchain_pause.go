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
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/syncx"
	"github.com/ethereum/go-ethereum/log"
)

// ClonePause is a pause of the chain writes for a copy of the database, taken
// by PauseForClone. While it's held, the database on disk is consistent at Head.
type ClonePause struct {
	Head        *types.Header // The head a copy of the database opens at
	Start       time.Time     // When the chain writes were held
	Persist     time.Duration // The time it took to make the database consistent
	JournalSize uint64        // The size of the state journal written, in bytes

	bc   *BlockChain
	once sync.Once
}

// PauseForClone holds the chain writes and makes the database consistent on
// disk at the current head, for a copy of the data directory to be taken in the
// meantime. The copy opens at the head of the pause, without the rewind that
// follows an unclean shutdown. The pause lasts until Resume is called, or the
// chain is stopped. Only the path scheme supports it.
//
// The pause holds the chain mutex: no block is imported and the head doesn't
// move, while reads go on. It journals the state layers up to the head without
// closing the trie database, and syncs the ancient and the key-value stores.
// Other writers keep running, among them the key-value store's compactions and
// the chain freezer, which moves old blocks to the ancient store. So the copy
// must capture every store at one instant, as a filesystem snapshot of the data
// directory does; it then holds their state as a crash would.
//
// The chain mutex is waited for until ctx is done. A pause is given up if ctx is
// done once it holds the mutex, or once it is ready.
func (bc *BlockChain) PauseForClone(ctx context.Context) (*ClonePause, error) {
	if bc.triedb.Scheme() != rawdb.PathScheme {
		return nil, errors.New("pause for a clone requires the path scheme")
	}
	if err := bc.chainmu.LockContext(ctx); err != nil {
		if errors.Is(err, syncx.ErrClosed) {
			return nil, errChainStopped
		}
		return nil, err
	}
	if bc.stopping.Load() {
		bc.chainmu.Unlock()
		return nil, errChainStopped
	}
	// A free mutex is taken even if ctx is done.
	if err := ctx.Err(); err != nil {
		bc.chainmu.Unlock()
		return nil, err
	}
	p := &ClonePause{Head: bc.CurrentBlock(), Start: time.Now(), bc: bc}
	err := p.persist()
	if err == nil {
		// Making the database consistent can outlast ctx, whose caller may be
		// gone by now.
		err = ctx.Err()
	}
	if err != nil {
		bc.chainmu.Unlock()
		return nil, err
	}
	p.Persist = time.Since(p.Start)

	// Register the pause for Stop to end it, and end it now if Stop is running:
	// it may have looked for a pause before this one was registered.
	bc.clonePause.Store(p)
	if bc.stopping.Load() {
		p.Resume()
		return nil, errChainStopped
	}
	log.Info("Paused chain writes for a clone", "number", p.Head.Number, "hash", p.Head.Hash(), "root", p.Head.Root, "journal", common.StorageSize(p.JournalSize), "elapsed", common.PrettyDuration(p.Persist))
	return p, nil
}

// persist makes the database consistent on disk at the head of the pause.
func (p *ClonePause) persist() error {
	size, err := p.bc.triedb.JournalLive(p.Head.Root)
	if err != nil {
		return err
	}
	p.JournalSize = size

	// Write out the preimages cached in memory, as a shutdown does.
	p.bc.triedb.WritePreimages()

	// Sync the ancient store before the key-value store: the chain freezer
	// appends to the former before it deletes the moved blocks from the latter.
	if _, err := p.bc.db.AncientDatadir(); err == nil {
		if err := p.bc.db.SyncAncient(); err != nil {
			return err
		}
	}
	return p.bc.db.SyncKeyValue()
}

// Resume releases the chain writes. Only the first call has an effect. The state
// journal written for the pause stays behind like a clean shutdown's: a restart
// loads it only if it still matches the state on disk.
func (p *ClonePause) Resume() {
	p.once.Do(func() {
		p.bc.clonePause.CompareAndSwap(p, nil)
		p.bc.chainmu.Unlock()
		log.Info("Resumed chain writes after a pause for a clone", "number", p.Head.Number, "held", common.PrettyDuration(time.Since(p.Start)))
	})
}
