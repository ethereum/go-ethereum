// Copyright 2021 The go-ethereum Authors
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

package downloader

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// hookedBackfiller is a tester backfiller with all interface methods mocked and
// hooked so tests can implement only the things they need.
type hookedBackfiller struct {
	// suspendHook is an optional hook to be called when the filler is requested
	// to be suspended.
	suspendHook func()

	// resumeHook is an optional hook to be called when the filler is requested
	// to be resumed.
	resumeHook func()
}

// suspend requests the backfiller to abort any running full or snap sync
// based on the skeleton chain as it might be invalid. The backfiller should
// gracefully handle multiple consecutive suspends without a resume, even
// on initial sartup.
func (hf *hookedBackfiller) suspend() {
	if hf.suspendHook != nil {
		hf.suspendHook()
	}
}

// resume requests the backfiller to start running fill or snap sync based on
// the skeleton chain as it has successfully been linked. Appending new heads
// to the end of the chain will not result in suspend/resume cycles.
func (hf *hookedBackfiller) resume() {
	if hf.resumeHook != nil {
		hf.resumeHook()
	}
}

// newNoopBackfiller creates a hooked backfiller with all callbacks disabled,
// essentially acting as a noop.
func newNoopBackfiller() backfiller {
	return new(hookedBackfiller)
}

// Tests various sync initialzations based on previous leftovers in the database
// and announced heads.
func TestSkeletonSyncInit(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	// Create a few key headers
	var (
		genesis  = &types.Header{Number: big.NewInt(0)}
		block49  = &types.Header{Number: big.NewInt(49)}
		block49B = &types.Header{Number: big.NewInt(49), Extra: []byte("B")}
		block50  = &types.Header{Number: big.NewInt(50), ParentHash: block49.Hash()}
	)
	tests := []struct {
		headers  []*types.Header // Database content (beside the genesis)
		oldstate []*subchain     // Old sync state with various interrupted subchains
		head     *types.Header   // New head header to announce to reorg to
		newstate []*subchain     // Expected sync state after the reorg
	}{
		// Completely empty database with only the genesis set. The sync is expected
		// to create a single subchain with the requested head.
		{
			head:     block50,
			newstate: []*subchain{{Head: 50, Tail: 50}},
		},
		// Empty database with only the genesis set with a leftover empty sync
		// progess. This is a synthetic case, just for the sake of covering things.
		{
			oldstate: []*subchain{},
			head:     block50,
			newstate: []*subchain{{Head: 50, Tail: 50}},
		},
		// A single leftover subchain is present, older than the new head. The
		// old subchain should be left as is and a new one appended to the sync
		// status.
		{
			oldstate: []*subchain{{Head: 10, Tail: 5}},
			head:     block50,
			newstate: []*subchain{
				{Head: 50, Tail: 50},
				{Head: 10, Tail: 5},
			},
		},
		// Multiple leftover subchains are present, older than the new head. The
		// old subchains should be left as is and a new one appended to the sync
		// status.
		{
			oldstate: []*subchain{
				{Head: 20, Tail: 15},
				{Head: 10, Tail: 5},
			},
			head: block50,
			newstate: []*subchain{
				{Head: 50, Tail: 50},
				{Head: 20, Tail: 15},
				{Head: 10, Tail: 5},
			},
		},
		// A single leftover subchain is present, newer than the new head. The
		// newer subchain should be deleted and a fresh one created for the head.
		{
			oldstate: []*subchain{{Head: 65, Tail: 60}},
			head:     block50,
			newstate: []*subchain{{Head: 50, Tail: 50}},
		},
		// Multiple leftover subchain is present, newer than the new head. The
		// newer subchains should be deleted and a fresh one created for the head.
		{
			oldstate: []*subchain{
				{Head: 75, Tail: 70},
				{Head: 65, Tail: 60},
			},
			head:     block50,
			newstate: []*subchain{{Head: 50, Tail: 50}},
		},

		// Two leftover subchains are present, one fully older and one fully
		// newer than the announced head. The head should delete the newer one,
		// keeping the older one.
		{
			oldstate: []*subchain{
				{Head: 65, Tail: 60},
				{Head: 10, Tail: 5},
			},
			head: block50,
			newstate: []*subchain{
				{Head: 50, Tail: 50},
				{Head: 10, Tail: 5},
			},
		},
		// Multiple leftover subchains are present, some fully older and some
		// fully newer than the announced head. The head should delete the newer
		// ones, keeping the older ones.
		{
			oldstate: []*subchain{
				{Head: 75, Tail: 70},
				{Head: 65, Tail: 60},
				{Head: 20, Tail: 15},
				{Head: 10, Tail: 5},
			},
			head: block50,
			newstate: []*subchain{
				{Head: 50, Tail: 50},
				{Head: 20, Tail: 15},
				{Head: 10, Tail: 5},
			},
		},
		// A single leftover subchain is present and the new head is extending
		// it with one more header. We expect the subchain head to be pushed
		// forward.
		{
			headers:  []*types.Header{block49},
			oldstate: []*subchain{{Head: 49, Tail: 5}},
			head:     block50,
			newstate: []*subchain{{Head: 50, Tail: 5}},
		},
		// A single leftover subchain is present and although the new head does
		// extend it number wise, the hash chain does not link up. We expect a
		// new subchain to be created for the dangling head.
		{
			headers:  []*types.Header{block49B},
			oldstate: []*subchain{{Head: 49, Tail: 5}},
			head:     block50,
			newstate: []*subchain{
				{Head: 50, Tail: 50},
				{Head: 49, Tail: 5},
			},
		},
		// A single leftover subchain is present. A new head is announced that
		// links into the middle of it, correctly anchoring into an existing
		// header. We expect the old subchain to be truncated and extended with
		// the new head.
		{
			headers:  []*types.Header{block49},
			oldstate: []*subchain{{Head: 100, Tail: 5}},
			head:     block50,
			newstate: []*subchain{{Head: 50, Tail: 5}},
		},
		// A single leftover subchain is present. A new head is announced that
		// links into the middle of it, but does not anchor into an existing
		// header. We expect the old subchain to be truncated and a new chain
		// be created for the dangling head.
		{
			headers:  []*types.Header{block49B},
			oldstate: []*subchain{{Head: 100, Tail: 5}},
			head:     block50,
			newstate: []*subchain{
				{Head: 50, Tail: 50},
				{Head: 49, Tail: 5},
			},
		},
	}
	for i, tt := range tests {
		// Create a fresh database and initialize it with the starting state
		db := rawdb.NewMemoryDatabase()

		rawdb.WriteHeader(db, genesis)
		for _, header := range tt.headers {
			rawdb.WriteSkeletonHeader(db, header)
		}
		if tt.oldstate != nil {
			blob, _ := json.Marshal(&skeletonProgress{Subchains: tt.oldstate})
			rawdb.WriteSkeletonSyncStatus(db, blob)
		}
		// Create a skeleton sync and run a cycle
		wait := make(chan struct{})

		skeleton := newSkeleton(db, newPeerSet(), func(string) {}, newNoopBackfiller())
		skeleton.syncStarting = func() { close(wait) }
		skeleton.Sync(tt.head)

		<-wait
		skeleton.Terminate()

		// Ensure the correct resulting sync status
		var progress skeletonProgress
		json.Unmarshal(rawdb.ReadSkeletonSyncStatus(db), &progress)

		if len(progress.Subchains) != len(tt.newstate) {
			t.Errorf("test %d: subchain count mismatch: have %d, want %d", i, len(progress.Subchains), len(tt.newstate))
			continue
		}
		for j := 0; j < len(progress.Subchains); j++ {
			if progress.Subchains[j].Head != tt.newstate[j].Head {
				t.Errorf("test %d: subchain %d head mismatch: have %d, want %d", i, j, progress.Subchains[j].Head, tt.newstate[j].Head)
			}
			if progress.Subchains[j].Tail != tt.newstate[j].Tail {
				t.Errorf("test %d: subchain %d tail mismatch: have %d, want %d", i, j, progress.Subchains[j].Tail, tt.newstate[j].Tail)
			}
		}
	}
}
