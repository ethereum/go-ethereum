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
	"fmt"
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

// TestHooks is a basic sanity-check of all hooks
func TestHooks(t *testing.T) {
	inner, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	inner.SetTxContext(common.Hash{0x11}, 100) // For the log
	var result []string
	var wants = []string{
		"0xaa00000000000000000000000000000000000000.balance: 0->100 (Unspecified)",
		"0xaa00000000000000000000000000000000000000.balance: 100->50 (Transfer)",
		"0xaa00000000000000000000000000000000000000.nonce: 0->1337",
		"0xaa00000000000000000000000000000000000000.code:  (0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470) ->0x1325 (0xa12ae05590de0c93a00bc7ac773c2fdb621e44f814985e72194f921c0050f728)",
		"0xaa00000000000000000000000000000000000000.storage slot 0x0000000000000000000000000000000000000000000000000000000000000001: 0x0000000000000000000000000000000000000000000000000000000000000000 ->0x0000000000000000000000000000000000000000000000000000000000000011",
		"0xaa00000000000000000000000000000000000000.storage slot 0x0000000000000000000000000000000000000000000000000000000000000001: 0x0000000000000000000000000000000000000000000000000000000000000011 ->0x0000000000000000000000000000000000000000000000000000000000000022",
		"log 100",
	}
	emitF := func(format string, a ...any) {
		result = append(result, fmt.Sprintf(format, a...))
	}
	sdb := NewHookedState(inner, &tracing.Hooks{
		OnBalanceChange: func(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
			emitF("%v.balance: %v->%v (%v)", addr, prev, new, reason)
		},
		OnNonceChange: func(addr common.Address, prev, new uint64) {
			emitF("%v.nonce: %v->%v", addr, prev, new)
		},
		OnCodeChange: func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
			emitF("%v.code: %#x (%v) ->%#x (%v)", addr, prevCode, prevCodeHash, code, codeHash)
		},
		OnStorageChange: func(addr common.Address, slot common.Hash, prev, new common.Hash) {
			emitF("%v.storage slot %v: %v ->%v", addr, slot, prev, new)
		},
		OnLog: func(log *types.Log) {
			emitF("log %v", log.TxIndex)
		},
	})
	sdb.AddBalance(common.Address{0xaa}, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	sdb.SubBalance(common.Address{0xaa}, uint256.NewInt(50), tracing.BalanceChangeTransfer)
	sdb.SetNonce(common.Address{0xaa}, 1337, tracing.NonceChangeGenesis)
	sdb.SetCode(common.Address{0xaa}, []byte{0x13, 37})
	sdb.SetState(common.Address{0xaa}, common.HexToHash("0x01"), common.HexToHash("0x11"))
	sdb.SetState(common.Address{0xaa}, common.HexToHash("0x01"), common.HexToHash("0x22"))
	sdb.SetTransientState(common.Address{0xaa}, common.HexToHash("0x02"), common.HexToHash("0x01"))
	sdb.SetTransientState(common.Address{0xaa}, common.HexToHash("0x02"), common.HexToHash("0x02"))
	sdb.AddLog(&types.Log{
		Address: common.Address{0xbb},
	})
	for i, want := range wants {
		if have := result[i]; have != want {
			t.Fatalf("error event %d, have\n%v\nwant%v\n", i, have, want)
		}
	}
}
