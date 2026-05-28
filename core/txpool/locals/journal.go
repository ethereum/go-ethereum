// Copyright 2017 The go-ethereum Authors
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

package locals

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// errNoActiveJournal is returned if a transaction is attempted to be inserted
// into the journal, but no such file is currently open.
var errNoActiveJournal = errors.New("no active journal")

// devNull is a WriteCloser that just discards anything written into it. Its
// goal is to allow the transaction journal to write into a fake journal when
// loading transactions on startup without printing warnings due to no file
// being read for write.
type devNull struct{}

func (*devNull) Write(p []byte) (n int, err error) { return len(p), nil }
func (*devNull) Close() error                      { return nil }

// journal is a rotating log of transactions with the aim of storing locally
// created transactions to allow non-executed ones to survive node restarts.
//
// writer is shared between the tracker loop goroutine (load / setupWriter /
// rotate / close) and any goroutine that calls TxTracker.Track / TrackAll
// (which reaches journal.insert). TxTracker.mu does not cover loop-side
// writer mutations — see #34983 for the race report from `go test -race`.
// A small dedicated mutex guards every read / write of the writer pointer.
// We deliberately drop the lock before doing the actual rlp.Encode in
// insert so that a concurrent close / rotate is not blocked by an in-flight
// write — Write on a closed file simply returns an error, which insert
// propagates back to the caller (already ignored at the call site in
// TrackAll).
type journal struct {
	path string // Filesystem path to store the transactions at

	mu     sync.Mutex
	writer io.WriteCloser // Output stream to write new transactions into
}

// getWriter returns the current journal writer under the writer mutex.
func (journal *journal) getWriter() io.WriteCloser {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	return journal.writer
}

// setWriter atomically swaps the writer pointer and returns the previous
// value so the caller can close it after releasing the lock.
func (journal *journal) setWriter(w io.WriteCloser) io.WriteCloser {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	prev := journal.writer
	journal.writer = w
	return prev
}

// newTxJournal creates a new transaction journal to
func newTxJournal(path string) *journal {
	return &journal{
		path: path,
	}
}

// load parses a transaction journal dump from disk, loading its contents into
// the specified pool.
func (journal *journal) load(add func([]*types.Transaction) []error) error {
	// Open the journal for loading any past transactions
	input, err := os.Open(journal.path)
	if errors.Is(err, fs.ErrNotExist) {
		// Skip the parsing if the journal file doesn't exist at all
		return nil
	}
	if err != nil {
		return err
	}
	defer input.Close()

	// Temporarily discard any journal additions (don't double add on load).
	// The add callback below dispatches to TxTracker.TrackAll → insert, which
	// reads journal.writer under journal.mu; setWriter publishes the devNull
	// atomically so those reads observe a consistent value.
	journal.setWriter(new(devNull))
	defer journal.setWriter(nil)

	// Inject all transactions from the journal into the pool
	stream := rlp.NewStream(input, 0)
	total, dropped := 0, 0

	// Create a method to load a limited batch of transactions and bump the
	// appropriate progress counters. Then use this method to load all the
	// journaled transactions in small-ish batches.
	loadBatch := func(txs types.Transactions) {
		for _, err := range add(txs) {
			if err != nil {
				log.Debug("Failed to add journaled transaction", "err", err)
				dropped++
			}
		}
	}
	var (
		failure error
		batch   types.Transactions
	)
	for {
		// Parse the next transaction and terminate on error
		tx := new(types.Transaction)
		if err = stream.Decode(tx); err != nil {
			if err != io.EOF {
				failure = err
			}
			if batch.Len() > 0 {
				loadBatch(batch)
			}
			break
		}
		// New transaction parsed, queue up for later, import if threshold is reached
		total++

		if batch = append(batch, tx); batch.Len() > 1024 {
			loadBatch(batch)
			batch = batch[:0]
		}
	}
	log.Info("Loaded local transaction journal", "transactions", total, "dropped", dropped)

	return failure
}

func (journal *journal) setupWriter() error {
	// Close any previously-installed writer; clear the slot first so concurrent
	// inserts can see the transition without racing on the close.
	if prev := journal.setWriter(nil); prev != nil {
		if err := prev.Close(); err != nil {
			return err
		}
	}

	// Re-open the journal file for appending.
	// Use O_APPEND to ensure we always write to the end of the file.
	sink, err := os.OpenFile(journal.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	journal.setWriter(sink)

	return nil
}

// insert adds the specified transaction to the local disk journal.
func (journal *journal) insert(tx *types.Transaction) error {
	// Snapshot the writer under the mutex, then release before doing the
	// actual rlp.Encode so a concurrent close / rotate is not blocked by an
	// in-flight write. Write on a closed file returns an error which we
	// propagate; the existing call site in TrackAll already ignores it.
	w := journal.getWriter()
	if w == nil {
		return errNoActiveJournal
	}
	return rlp.Encode(w, tx)
}

// rotate regenerates the transaction journal based on the current contents of
// the transaction pool.
func (journal *journal) rotate(all map[common.Address]types.Transactions) error {
	// Close the current journal (if any is open). Clear the slot first so
	// concurrent inserts observe the transition before the file is closed.
	if prev := journal.setWriter(nil); prev != nil {
		if err := prev.Close(); err != nil {
			return err
		}
	}
	// Generate a new journal with the contents of the current pool
	replacement, err := os.OpenFile(journal.path+".new", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	journaled := 0
	for _, txs := range all {
		for _, tx := range txs {
			if err = rlp.Encode(replacement, tx); err != nil {
				replacement.Close()
				return err
			}
		}
		journaled += len(txs)
	}
	replacement.Close()

	// Replace the live journal with the newly generated one
	if err = os.Rename(journal.path+".new", journal.path); err != nil {
		return err
	}
	sink, err := os.OpenFile(journal.path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	journal.setWriter(sink)

	logger := log.Info
	if len(all) == 0 {
		logger = log.Debug
	}
	logger("Regenerated local transaction journal", "transactions", journaled, "accounts", len(all))

	return nil
}

// close flushes the transaction journal contents to disk and closes the file.
func (journal *journal) close() error {
	if prev := journal.setWriter(nil); prev != nil {
		return prev.Close()
	}
	return nil
}
