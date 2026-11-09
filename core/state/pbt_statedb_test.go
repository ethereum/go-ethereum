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

package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

var (
	reviewRules = params.Rules{IsEIP158: true, IsByzantium: true, IsLondon: true, IsCancun: true}
	rvX         = common.HexToAddress("0x1234")
	rvOther     = common.HexToAddress("0xaaaa")
	oldLo       = common.Hash{31: 1}              // header slot
	oldHi       = common.BigToHash(common.Big256) // bucket slot
	newLo       = common.Hash{31: 5}              // header slot
	newHi       = common.BigToHash(big300)        // bucket slot
	big300      = uint256.NewInt(300).ToBig()
)

func newPBTState() *PBTDatabase {
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), &triedb.Config{PathDB: pathdb.Defaults, IsPBT: true})
	return NewPBTDatabase(tdb, nil)
}

// preState holds rvOther and an empty rvX with two slots, committed without
// EIP-161 as a genesis allocation can.
func preState(t *testing.T) (*PBTDatabase, common.Hash) {
	sdb := newPBTState()
	st, _ := New(types.EmptyBinaryHash, sdb)
	st.SetBalance(rvOther, uint256.NewInt(7), tracing.BalanceChangeUnspecified)
	st.SetState(rvX, oldLo, common.Hash{31: 0xaa})
	st.SetState(rvX, oldHi, common.Hash{31: 0xbb})
	root, err := st.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return sdb, root
}

func expect(t *testing.T, slots map[common.Hash]common.Hash) common.Hash {
	exp, _ := New(types.EmptyBinaryHash, newPBTState())
	exp.SetBalance(rvOther, uint256.NewInt(7), tracing.BalanceChangeUnspecified)
	exp.SetBalance(rvX, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	for k, v := range slots {
		exp.SetState(rvX, k, v)
	}
	root, _ := exp.Commit(reviewRules, 0)
	return root
}

func check(t *testing.T, sdb *PBTDatabase, got, want common.Hash, slots map[common.Hash]common.Hash) {
	t.Helper()
	if got != want {
		t.Fatalf("root mismatch: got %x want %x", got, want)
	}
	tr, _ := sdb.OpenTrie(got)
	st, _ := New(got, sdb)
	for _, k := range []common.Hash{oldLo, oldHi, newLo, newHi} {
		v, _ := tr.GetStorage(rvX, k.Bytes())
		want := slots[k]
		if common.BytesToHash(v) != want || st.GetState(rvX, k) != want {
			t.Fatalf("slot %x: tree %x flat %x want %x", k, v, st.GetState(rvX, k), want)
		}
	}
}

// tx1 clears X, tx2 re-funds it.
func TestReviewPBTRecreate(t *testing.T) {
	sdb, root := preState(t)
	st, _ := New(root, sdb)
	st.AddBalance(rvX, new(uint256.Int), tracing.BalanceChangeUnspecified)
	st.Finalise(reviewRules)
	st.AddBalance(rvX, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	st.Finalise(reviewRules)
	got, err := st.Commit(reviewRules, 1)
	if err != nil {
		t.Fatal(err)
	}
	check(t, sdb, got, expect(t, nil), nil)
}

// Recreated X writes storage, the root is taken, and a later tx writes more:
// the wipe must not run again and drop the first write.
func TestReviewPBTRecreateThenIntermediateRoot(t *testing.T) {
	sdb, root := preState(t)
	st, _ := New(root, sdb)
	st.AddBalance(rvX, new(uint256.Int), tracing.BalanceChangeUnspecified)
	st.Finalise(reviewRules)
	st.AddBalance(rvX, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	st.SetState(rvX, newLo, common.Hash{31: 0xcc})
	st.Finalise(reviewRules)
	st.IntermediateRoot(reviewRules)
	st.SetState(rvX, newHi, common.Hash{31: 0xdd})
	st.Finalise(reviewRules)
	got, err := st.Commit(reviewRules, 1)
	if err != nil {
		t.Fatal(err)
	}
	slots := map[common.Hash]common.Hash{newLo: {31: 0xcc}, newHi: {31: 0xdd}}
	check(t, sdb, got, expect(t, slots), slots)
}

// The deletion reaches the tree through IntermediateRoot before X is
// recreated, which was already correct.
func TestReviewPBTDeleteAppliedThenRecreate(t *testing.T) {
	sdb, root := preState(t)
	st, _ := New(root, sdb)
	st.AddBalance(rvX, new(uint256.Int), tracing.BalanceChangeUnspecified)
	st.Finalise(reviewRules)
	st.IntermediateRoot(reviewRules)
	st.AddBalance(rvX, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	st.SetState(rvX, newLo, common.Hash{31: 0xcc})
	st.Finalise(reviewRules)
	got, err := st.Commit(reviewRules, 1)
	if err != nil {
		t.Fatal(err)
	}
	slots := map[common.Hash]common.Hash{newLo: {31: 0xcc}}
	check(t, sdb, got, expect(t, slots), slots)
}
