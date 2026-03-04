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

package runtime

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/arena"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TestExecuteWithBumpAllocator runs bytecode through the EVM with a
// BumpAllocator and verifies the result matches the HeapAllocator path.
func TestExecuteWithBumpAllocator(t *testing.T) {
	// Bytecode: PUSH1 0x2a, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	// Returns the 32-byte big-endian encoding of 42.
	code := []byte{
		byte(vm.PUSH1), 0x2a,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}

	// Run with heap allocator (default).
	heapRet, _, err := Execute(code, nil, nil)
	if err != nil {
		t.Fatalf("heap execute failed: %v", err)
	}

	// Run with bump allocator.
	slab := make([]byte, 8<<20) // 8 MiB
	bumpRet, _, err := Execute(code, nil, &Config{
		EVMConfig: vm.Config{
			Allocator: arena.NewBumpAllocator(slab),
		},
	})
	if err != nil {
		t.Fatalf("bump execute failed: %v", err)
	}

	if !bytes.Equal(heapRet, bumpRet) {
		t.Fatalf("results differ:\n  heap: %x\n  bump: %x", heapRet, bumpRet)
	}
}

// TestCallWithBumpAllocator deploys a simple contract, then calls it using
// both allocator types and verifies matching results.
func TestCallWithBumpAllocator(t *testing.T) {
	// Contract code: returns the caller's address (CALLER, PUSH1 0, MSTORE, PUSH1 20, PUSH1 12, RETURN)
	code := []byte{
		byte(vm.CALLER),
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 20,
		byte(vm.PUSH1), 12,
		byte(vm.RETURN),
	}
	address := common.HexToAddress("0xaa")

	run := func(alloc arena.Allocator) ([]byte, error) {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		statedb.CreateAccount(address)
		statedb.SetCode(address, code, tracing.CodeChangeUnspecified)
		statedb.Finalise(true)

		ret, _, err := Call(address, nil, &Config{
			State: statedb,
			EVMConfig: vm.Config{
				Allocator: alloc,
			},
		})
		return ret, err
	}

	heapRet, err := run(nil)
	if err != nil {
		t.Fatalf("heap call failed: %v", err)
	}

	slab := make([]byte, 8<<20)
	bumpRet, err := run(arena.NewBumpAllocator(slab))
	if err != nil {
		t.Fatalf("bump call failed: %v", err)
	}

	if !bytes.Equal(heapRet, bumpRet) {
		t.Fatalf("results differ:\n  heap: %x\n  bump: %x", heapRet, bumpRet)
	}
}

// TestNestedCallsWithBumpAllocator exercises nested EVM calls (CALL opcode)
// with a BumpAllocator to stress-test per-call-frame arena allocation of
// Contract, Memory, Stack, and ScopeContext.
func TestNestedCallsWithBumpAllocator(t *testing.T) {
	// Inner contract: returns 0x42.
	innerCode := []byte{
		byte(vm.PUSH1), 0x42,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}
	innerAddr := common.HexToAddress("0xbb")

	// Outer contract calls inner and returns its result.
	// PUSH1 32       - retSize
	// PUSH1 0        - retOffset
	// PUSH1 0        - argsSize
	// PUSH1 0        - argsOffset
	// PUSH1 0        - value
	// PUSH20 <inner> - address
	// PUSH3 0xffffff - gas
	// CALL
	// PUSH1 32       - size
	// PUSH1 0        - offset
	// RETURN
	outerCode := []byte{
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH20),
	}
	outerCode = append(outerCode, innerAddr.Bytes()...)
	outerCode = append(outerCode,
		byte(vm.PUSH3), 0xff, 0xff, 0xff,
		byte(vm.CALL),
		byte(vm.POP), // pop success flag
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	)
	outerAddr := common.HexToAddress("0xcc")

	run := func(alloc arena.Allocator) ([]byte, error) {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		statedb.CreateAccount(innerAddr)
		statedb.SetCode(innerAddr, innerCode, tracing.CodeChangeUnspecified)
		statedb.CreateAccount(outerAddr)
		statedb.SetCode(outerAddr, outerCode, tracing.CodeChangeUnspecified)
		statedb.Finalise(true)

		ret, _, err := Call(outerAddr, nil, &Config{
			State: statedb,
			EVMConfig: vm.Config{
				Allocator: alloc,
			},
		})
		return ret, err
	}

	heapRet, err := run(nil)
	if err != nil {
		t.Fatalf("heap nested call failed: %v", err)
	}

	slab := make([]byte, 8<<20)
	bumpRet, err := run(arena.NewBumpAllocator(slab))
	if err != nil {
		t.Fatalf("bump nested call failed: %v", err)
	}

	if !bytes.Equal(heapRet, bumpRet) {
		t.Fatalf("nested call results differ:\n  heap: %x\n  bump: %x", heapRet, bumpRet)
	}
}

// TestCreateWithBumpAllocator exercises contract creation (CREATE opcode)
// with a BumpAllocator.
func TestCreateWithBumpAllocator(t *testing.T) {
	// Simple init code that returns 0x42 as runtime code.
	initCode := []byte{
		byte(vm.PUSH1), 0x42, // runtime code is just "0x42" (1 byte)
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		// Return 1 byte from memory offset 31 (the 0x42 byte in big-endian word).
		byte(vm.PUSH1), 1,
		byte(vm.PUSH1), 31,
		byte(vm.RETURN),
	}

	run := func(alloc arena.Allocator) ([]byte, common.Address, error) {
		code, addr, _, err := Create(initCode, &Config{
			EVMConfig: vm.Config{
				Allocator: alloc,
			},
			Value: big.NewInt(0),
		})
		return code, addr, err
	}

	heapCode, heapAddr, err := run(nil)
	if err != nil {
		t.Fatalf("heap create failed: %v", err)
	}

	slab := make([]byte, 8<<20)
	bumpCode, bumpAddr, err := run(arena.NewBumpAllocator(slab))
	if err != nil {
		t.Fatalf("bump create failed: %v", err)
	}

	if !bytes.Equal(heapCode, bumpCode) {
		t.Fatalf("created code differs:\n  heap: %x\n  bump: %x", heapCode, bumpCode)
	}
	if heapAddr != bumpAddr {
		t.Fatalf("created address differs: heap=%s bump=%s", heapAddr, bumpAddr)
	}
}
