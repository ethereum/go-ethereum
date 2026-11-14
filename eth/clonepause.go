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

package eth

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultCloneTimeout = 20 * time.Second // How long a pause for a clone is held at most, by default
	maxCloneTimeout     = 2 * time.Minute  // The longest timeout a pause for a clone may ask for
)

var (
	errClonePauseHeld = errors.New("a pause for a clone is already held")
	errCloneUnknown   = errors.New("unknown pause for a clone")
	errCloneShutdown  = errors.New("shutting down")
	errCloneNoDatadir = errors.New("no data directory to copy: the database is in memory")
)

// ClonePauseResult describes a pause of the chain writes for a clone.
type ClonePauseResult struct {
	ID          string            `json:"id"`          // The pause, to resume it
	Number      hexutil.Uint64    `json:"number"`      // The head a copy of the database opens at
	Hash        common.Hash       `json:"hash"`        // The hash of the head
	Root        common.Hash       `json:"root"`        // The state root of the head
	JournalSize uint64            `json:"journalSize"` // The size of the state journal written, in bytes
	Wait        int64             `json:"wait"`        // Milliseconds waited for the chain writes to stop
	Persist     int64             `json:"persist"`     // Milliseconds taken to make the database consistent
	Expires     int64             `json:"expires"`     // When the pause ends by itself, in Unix milliseconds
	Paths       map[string]string `json:"paths"`       // The stores of the database and the state journal
}

// CloneResumeResult tells how a pause for a clone ended.
type CloneResumeResult struct {
	Intact bool  `json:"intact"` // Whether the pause lasted until resumed: a copy taken during it is good only if so
	Held   int64 `json:"held"`   // Milliseconds the pause was held
}

// clonePauser hands out the pauses of the chain writes for a clone, one at a
// time. A pause is a lease: it ends by itself when its timeout passes.
type clonePauser struct {
	chain *core.BlockChain
	paths map[string]string // The stores of the database, reported with a pause

	lock       sync.Mutex
	current    *cloneLease        // The lease held or being taken
	last       *cloneLease        // The lease held before, once ended
	closed     bool               // Whether the node is shutting down
	quit       context.Context    // Done at shutdown, ending the wait of a lease being taken
	cancelQuit context.CancelFunc // Ends quit
}

// cloneLease is a pause for a clone held by a client.
type cloneLease struct {
	id     string
	pause  *core.ClonePause // The pause, nil while being taken
	timer  *time.Timer      // Ends the pause when the lease expires
	held   time.Duration    // How long the pause was held, once it ended
	intact bool             // Whether the pause lasted until resumed, once it ended
}

// newClonePauser creates the pauser of the chain writes, reporting the given
// paths of the stores of the database.
func newClonePauser(chain *core.BlockChain, paths map[string]string) *clonePauser {
	quit, cancelQuit := context.WithCancel(context.Background())
	return &clonePauser{chain: chain, paths: paths, quit: quit, cancelQuit: cancelQuit}
}

// clonePaths returns the paths of the stores of the database: the key-value
// store, the ancient store with the state histories, the directory of the state
// journal, empty if the journal is in the key-value store, and the era store if
// one is configured: by default it's inside the ancient store.
func clonePaths(chaindata string, db ethdb.Database, journal string, era string) map[string]string {
	paths := map[string]string{
		"chaindata": chaindata,
		"journal":   journal,
	}
	if ancient, err := db.AncientDatadir(); err == nil {
		paths["ancient"] = ancient
		if era != "" {
			paths["era"] = rawdb.EraDir(ancient, era)
		}
	}
	return paths
}

// pause holds the chain writes, after a new head if asked, and returns once
// the database is consistent on disk. A pause ready only after the ready timeout,
// or after ctx is done, is given up; writing the journal and the syncs aren't
// interrupted, so they can overrun it. A pause given out ends by itself after
// the hold timeout.
func (cp *clonePauser) pause(ctx context.Context, ready time.Duration, hold time.Duration, afterHead bool) (*ClonePauseResult, error) {
	if cp.paths["chaindata"] == "" {
		return nil, errCloneNoDatadir
	}
	cp.lock.Lock()
	if cp.closed {
		cp.lock.Unlock()
		return nil, errCloneShutdown
	}
	if cp.current != nil {
		cp.lock.Unlock()
		return nil, errClonePauseHeld
	}
	lease := &cloneLease{id: rand.Text()}
	cp.current = lease
	cp.lock.Unlock()

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, ready)
	defer cancel()
	// Shutdown ends the wait too.
	defer context.AfterFunc(cp.quit, cancel)()
	pause, err := cp.take(ctx, afterHead)

	cp.lock.Lock()
	defer cp.lock.Unlock()

	if err == nil && cp.closed {
		pause.Resume()
		err = errCloneShutdown
	}
	if err != nil {
		cp.current = nil
		return nil, err
	}
	lease.pause = pause
	lease.timer = time.AfterFunc(hold, func() { cp.expire(lease) })

	return &ClonePauseResult{
		ID:          lease.id,
		Number:      hexutil.Uint64(pause.Head.Number.Uint64()),
		Hash:        pause.Head.Hash(),
		Root:        pause.Head.Root,
		JournalSize: pause.JournalSize,
		Wait:        pause.Start.Sub(start).Milliseconds(),
		Persist:     pause.Persist.Milliseconds(),
		Expires:     time.Now().Add(hold).UnixMilli(),
		Paths:       cp.paths,
	}, nil
}

// take pauses the chain writes, after a new head if asked.
func (cp *clonePauser) take(ctx context.Context, afterHead bool) (*core.ClonePause, error) {
	if afterHead {
		// The head event is sent with the chain mutex held: take one, and stop
		// listening before waiting for the mutex.
		heads := make(chan core.ChainHeadEvent, 1)
		sub := cp.chain.SubscribeChainHeadEvent(heads)
		select {
		case <-heads:
		case <-sub.Err():
			return nil, errCloneShutdown
		case <-ctx.Done():
			sub.Unsubscribe()
			return nil, ctx.Err()
		}
		sub.Unsubscribe()
	}
	return cp.chain.PauseForClone(ctx)
}

// resume ends the pause with the given id, and tells whether it was held until
// then. A pause that ended before, by expiry or at shutdown, is reported too.
func (cp *clonePauser) resume(id string) (*CloneResumeResult, error) {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	lease := cp.current
	if lease != nil && lease.id == id && lease.pause != nil {
		cp.end(lease, true)
	} else if lease = cp.last; lease == nil || lease.id != id {
		return nil, errCloneUnknown
	}
	return &CloneResumeResult{Intact: lease.intact, Held: lease.held.Milliseconds()}, nil
}

// expire ends the pause of the lease when its timeout passes.
func (cp *clonePauser) expire(lease *cloneLease) {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	if cp.current != lease {
		return // resumed in the meantime
	}
	log.Warn("Pause for a clone expired", "id", lease.id)
	cp.end(lease, false)
}

// close ends the pause held, if any, and refuses new ones.
func (cp *clonePauser) close() {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	if cp.closed {
		return
	}
	cp.closed = true
	cp.cancelQuit()

	if lease := cp.current; lease != nil && lease.pause != nil {
		log.Warn("Ending the pause for a clone at shutdown", "id", lease.id)
		cp.end(lease, false)
	}
}

// end ends the pause of the current lease. The lock must be held.
func (cp *clonePauser) end(lease *cloneLease, intact bool) {
	lease.timer.Stop()
	lease.pause.Resume()
	lease.held = time.Since(lease.pause.Start)
	lease.intact = intact
	cp.current, cp.last = nil, lease
}
