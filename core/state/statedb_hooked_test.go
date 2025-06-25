// Copyright 2024 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// This method tests that the 'burn' from sending-to-selfdestructed accounts
// is accounted for.
// (There is also a higher-level test in eth/tracers: TestSupplySelfDestruct )
func TestBurn(t *testing.T) {
	// Note: burn can happen even after EIP-6780, if within one single transaction,
	// the following occur:
	// 1. contract B creates contract A
	// 2. contract A is destructed
	// 3. contract B sends ether to A

	var burned = new(uint256.Int)
	s, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	hooked := NewHookedState(s, &tracing.Hooks{
		OnBalanceChange: func(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
			if reason == tracing.BalanceDecreaseSelfdestructBurn {
				burned.Add(burned, uint256.MustFromBig(prev))
			}
		},
	})
	createAndDestroy := func(addr common.Address) {
		hooked.AddBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
		hooked.CreateContract(addr)
		hooked.SelfDestruct(addr)
		// sanity-check that balance is now 0
		if have, want := hooked.GetBalance(addr), new(uint256.Int); !have.Eq(want) {
			t.Fatalf("post-destruct balance wrong: have %v want %v", have, want)
		}
	}
	addA := common.Address{0xaa}
	addB := common.Address{0xbb}
	addC := common.Address{0xcc}

	// Tx 1: create and destroy address A and B in one tx
	createAndDestroy(addA)
	createAndDestroy(addB)
	hooked.AddBalance(addA, uint256.NewInt(200), tracing.BalanceChangeUnspecified)
	hooked.AddBalance(addB, uint256.NewInt(200), tracing.BalanceChangeUnspecified)
	hooked.Finalise(true)

	// Tx 2: create and destroy address C, then commit
	createAndDestroy(addC)
	hooked.AddBalance(addC, uint256.NewInt(200), tracing.BalanceChangeUnspecified)
	hooked.Finalise(true)

	s.Commit(0, false, false)
	if have, want := burned, uint256.NewInt(600); !have.Eq(want) {
		t.Fatalf("burn-count wrong, have %v want %v", have, want)
	}
}
