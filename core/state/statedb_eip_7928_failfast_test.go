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

package state_test

import (
	"encoding/binary"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
)

var errBALPoison = errors.New("poisoned account read")

// balFaultReader implements state.Reader. It serves empty accounts, fails the
// poison address, and slows down every read issued after the failure so that
// sibling workers have ample time to observe it before the task queue drains.
type balFaultReader struct {
	poison   common.Address
	poisoned atomic.Bool
	reads    atomic.Int64
}

func (r *balFaultReader) Account(addr common.Address) (*types.StateAccount, error) {
	r.reads.Add(1)
	if addr == r.poison {
		r.poisoned.Store(true)
		return nil, errBALPoison
	}
	if r.poisoned.Load() {
		time.Sleep(50 * time.Microsecond)
	}
	return nil, nil
}

func (r *balFaultReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	return common.Hash{}, nil
}

func (r *balFaultReader) Has(addr common.Address, codeHash common.Hash) bool    { return false }
func (r *balFaultReader) Code(addr common.Address, codeHash common.Hash) []byte { return nil }
func (r *balFaultReader) CodeSize(addr common.Address, codeHash common.Hash) int {
	return 0
}

// balApplyList builds an access list with one nonce mutation for each of n
// accounts, whose addresses sort in index order.
func balApplyList(n int) (*bal.BlockAccessList, []common.Address) {
	construction := bal.NewConstructionBlockAccessList()
	addrs := make([]common.Address, n)
	for i := range addrs {
		addrs[i] = common.BytesToAddress(binary.BigEndian.AppendUint64(nil, uint64(i+1)))
		construction.NonceChange(addrs[i], 1, 1)
	}
	return construction.ToEncodingObj(), addrs
}

// TestParallelBALApplyFailFast checks that once one account fails to apply,
// the concurrent appliers stop claiming further accounts instead of grinding
// through the entire access list, and that the original failure (rather than
// the cancellation) is what surfaces to the caller.
func TestParallelBALApplyFailFast(t *testing.T) {
	if runtime.GOMAXPROCS(0) < 2 {
		t.Skip("fail-fast requires at least two concurrent workers")
	}
	const accounts = 2048

	// The poison address is the lowest one, so it is the first task claimed:
	// nearly the whole list still sits in the queue when the failure fires.
	// Without fail-fast every account is read exactly once regardless of the
	// failure, so the strict inequality below discriminates the behaviors.
	list, addrs := balApplyList(accounts)
	reader := &balFaultReader{poison: addrs[0]}
	sdb, err := state.NewWithReader(types.EmptyRootHash, state.NewDatabaseForTesting(), reader)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	if err := sdb.ApplyBlockAccessList(list); !errors.Is(err, errBALPoison) {
		t.Fatalf("apply error = %v, want %v", err, errBALPoison)
	}
	if n := reader.reads.Load(); n >= accounts {
		t.Fatalf("no fail-fast: all %d accounts were read after one failed", n)
	}
}

// TestParallelBALApplyHealthyComplete pins the happy path: with no failure,
// every account in the list is applied and no error is reported.
func TestParallelBALApplyHealthyComplete(t *testing.T) {
	const accounts = 64

	list, addrs := balApplyList(accounts)
	reader := &balFaultReader{} // zero poison address, never served
	sdb, err := state.NewWithReader(types.EmptyRootHash, state.NewDatabaseForTesting(), reader)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	if err := sdb.ApplyBlockAccessList(list); err != nil {
		t.Fatalf("apply error = %v, want nil", err)
	}
	if n := reader.reads.Load(); n != accounts {
		t.Fatalf("account reads = %d, want %d", n, accounts)
	}
	for _, addr := range addrs {
		if nonce := sdb.GetNonce(addr); nonce != 1 {
			t.Fatalf("account %x nonce = %d, want 1", addr, nonce)
		}
	}
}
