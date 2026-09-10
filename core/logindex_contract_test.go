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
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

// eip8304ChainContext is a minimal ChainContext for exercising the EIP-8304
// index contract in isolation.
type eip8304ChainContext struct{ cfg *params.ChainConfig }

func (eip8304ChainContext) Engine() consensus.Engine                    { return beacon.New(ethash.NewFaker()) }
func (c eip8304ChainContext) Config() *params.ChainConfig               { return c.cfg }
func (eip8304ChainContext) GetHeader(common.Hash, uint64) *types.Header { return nil }
func (eip8304ChainContext) CurrentHeader() *types.Header                { return nil }
func (eip8304ChainContext) GetHeaderByHash(common.Hash) *types.Header   { return nil }
func (eip8304ChainContext) GetHeaderByNumber(uint64) *types.Header      { return nil }

// newEIP8304ContractState creates a fresh state for exercising the index
// contract in isolation.
func newEIP8304ContractState(t *testing.T) *state.StateDB {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(diskdb, nil)
	statedb, err := state.New(common.Hash{}, state.NewMPTDatabase(tdb, state.NewCodeDB(diskdb)))
	if err != nil {
		t.Fatal(err)
	}
	return statedb
}

// newEIP8304ContractEVM creates an EVM over the given state with the given
// block number, which drives the contract's NUMBER-based freshness checks.
func newEIP8304ContractEVM(t *testing.T, statedb *state.StateDB, blockNumber uint64) *vm.EVM {
	t.Helper()
	header := &types.Header{
		Number:     new(big.Int).SetUint64(blockNumber),
		Time:       blockNumber,
		GasLimit:   30_000_000,
		Coinbase:   common.Address{},
		BaseFee:    big.NewInt(0),
		Difficulty: big.NewInt(0),
	}
	ctx := NewEVMBlockContext(header, eip8304ChainContext{params.MergedTestChainConfig}, &header.Coinbase)
	return vm.NewEVM(ctx, statedb, params.MergedTestChainConfig, vm.Config{})
}

// deployEIP8304IndexContract deploys the EIP-8304 index contract from its init
// code and returns the resulting address.
func deployEIP8304IndexContract(t *testing.T, evm *vm.EVM) common.Address {
	t.Helper()
	_, addr, _, err := evm.Create(crypto.PubkeyToAddress(eip8304TestKey.PublicKey), params.IndexContractInitCode, vm.NewGasBudget(5_000_000, 0), common.U2560)
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
	return addr
}

// setCalldata builds the 96-byte set payload: first_block, table_size,
// table_root (all big-endian 32-byte words).
func setCalldata(firstBlock, tableSize uint64, root common.Hash) []byte {
	calldata := make([]byte, 96)
	binary.BigEndian.PutUint64(calldata[24:32], firstBlock)
	binary.BigEndian.PutUint64(calldata[56:64], tableSize)
	copy(calldata[64:96], root[:])
	return calldata
}

// getCalldata builds the 64-byte get payload: first_block, table_size.
func getCalldata(firstBlock, tableSize uint64) []byte {
	calldata := make([]byte, 64)
	binary.BigEndian.PutUint64(calldata[24:32], firstBlock)
	binary.BigEndian.PutUint64(calldata[56:64], tableSize)
	return calldata
}

// indexSlot is the storage slot for a table root per the EIP-8304 spec:
// table_size * TABLES_PER_LEVEL + (first_block / table_size) % TABLES_PER_LEVEL.
func indexSlot(firstBlock, tableSize uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(tableSize*0x400 + (firstBlock/tableSize)%0x400))
}

var eip8304TestKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

// TestEIP8304IndexContract pins the behavior of the EIP-8304 index contract
// against the spec:
//
//   - the deployed code must be exactly params.IndexContractCode (guards the
//     init-code/runtime consistency, which has regressed before)
//   - set: only the system address may write; the root lands at
//     table_size*1024 + (first_block/table_size)%1024
//   - get: 64-byte calldata returns the stored root; reverts on wrong size or
//     a first_block that is not a multiple of table_size
func TestEIP8304IndexContract(t *testing.T) {
	t.Run("deployment", func(t *testing.T) {
		statedb := newEIP8304ContractState(t)
		evm := newEIP8304ContractEVM(t, statedb, 1)
		addr := deployEIP8304IndexContract(t, evm)
		if got := statedb.GetCode(addr); !bytes.Equal(got, params.IndexContractCode) {
			t.Fatalf("deployed code mismatch: got %d bytes, want %d", len(got), len(params.IndexContractCode))
		}
	})

	t.Run("set", func(t *testing.T) {
		root := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
		cases := []struct{ fb, ts uint64 }{
			{2, 1}, {5, 2}, {8, 4}, {1030, 4}, {1024, 256},
		}
		for _, c := range cases {
			statedb := newEIP8304ContractState(t)
			evm := newEIP8304ContractEVM(t, statedb, 1)
			addr := deployEIP8304IndexContract(t, evm)
			if _, _, err := evm.Call(params.SystemAddress, addr, setCalldata(c.fb, c.ts, root), vm.NewGasBudget(5_000_000, 0), common.U2560); err != nil {
				t.Fatalf("set(%d, %d) failed: %v", c.fb, c.ts, err)
			}
			if got := statedb.GetState(addr, indexSlot(c.fb, c.ts)); got != root {
				t.Fatalf("set(%d, %d): slot %s = %x, want %x", c.fb, c.ts, indexSlot(c.fb, c.ts), got, root)
			}
		}
	})

	t.Run("set-caller-guard", func(t *testing.T) {
		statedb := newEIP8304ContractState(t)
		evm := newEIP8304ContractEVM(t, statedb, 1)
		addr := deployEIP8304IndexContract(t, evm)
		root := common.HexToHash("0xdeadbeef")
		sender := crypto.PubkeyToAddress(eip8304TestKey.PublicKey)
		if _, _, err := evm.Call(sender, addr, setCalldata(2, 1, root), vm.NewGasBudget(5_000_000, 0), common.U2560); err == nil {
			t.Fatal("non-system caller with set calldata: expected revert")
		}
		if got := statedb.GetState(addr, indexSlot(2, 1)); got != (common.Hash{}) {
			t.Fatalf("non-system caller wrote slot %s = %x", indexSlot(2, 1), got)
		}
	})

	t.Run("get", func(t *testing.T) {
		root := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000cafebabe")
		sender := crypto.PubkeyToAddress(eip8304TestKey.PublicKey)
		// The get path's freshness check is one-sided: table (fb, ts) remains
		// readable while num < fb + 1025*ts + ts/4 (the ring-buffer overwrite
		// window plus the table_size/4 publication delay, with truncating
		// division). There is no lower age bound. Write at block 1, then read
		// at the boundaries.
		for _, c := range []struct {
			fb, ts, num uint64
			wantErr     bool
		}{
			{fb: 2, ts: 1, num: 100, wantErr: false},  // no lower age bound
			{fb: 2, ts: 1, num: 1026, wantErr: false}, // last readable block
			{fb: 2, ts: 1, num: 1027, wantErr: true},  // ring window expired
			{fb: 8, ts: 4, num: 4108, wantErr: false}, // ts=4 window
			{fb: 8, ts: 4, num: 4109, wantErr: true},
		} {
			statedb := newEIP8304ContractState(t)
			evm := newEIP8304ContractEVM(t, statedb, 1)
			addr := deployEIP8304IndexContract(t, evm)
			if _, _, err := evm.Call(params.SystemAddress, addr, setCalldata(c.fb, c.ts, root), vm.NewGasBudget(5_000_000, 0), common.U2560); err != nil {
				t.Fatalf("set(%d, %d) failed: %v", c.fb, c.ts, err)
			}
			evm = newEIP8304ContractEVM(t, statedb, c.num)
			ret, _, err := evm.Call(sender, addr, getCalldata(c.fb, c.ts), vm.NewGasBudget(5_000_000, 0), common.U2560)
			if c.wantErr {
				if err == nil {
					t.Fatalf("get(%d, %d) at block %d: expected revert", c.fb, c.ts, c.num)
				}
				continue
			}
			if err != nil {
				t.Fatalf("get(%d, %d) at block %d failed: %v", c.fb, c.ts, c.num, err)
			}
			if got := common.BytesToHash(ret); got != root {
				t.Fatalf("get(%d, %d) at block %d returned %x, want %x", c.fb, c.ts, c.num, got, root)
			}
		}
	})

	t.Run("get-guards", func(t *testing.T) {
		root := common.HexToHash("0xcafebabe")
		statedb := newEIP8304ContractState(t)
		evm := newEIP8304ContractEVM(t, statedb, 259)
		addr := deployEIP8304IndexContract(t, evm)
		if _, _, err := evm.Call(params.SystemAddress, addr, setCalldata(2, 1, root), vm.NewGasBudget(5_000_000, 0), common.U2560); err != nil {
			t.Fatalf("set failed: %v", err)
		}
		sender := crypto.PubkeyToAddress(eip8304TestKey.PublicKey)
		// Wrong calldata size (32 bytes instead of 64).
		if _, _, err := evm.Call(sender, addr, make([]byte, 32), vm.NewGasBudget(5_000_000, 0), common.U2560); err == nil {
			t.Fatal("get with 32-byte calldata: expected revert")
		}
		// first_block not a multiple of table_size.
		if _, _, err := evm.Call(sender, addr, getCalldata(3, 2), vm.NewGasBudget(5_000_000, 0), common.U2560); err == nil {
			t.Fatal("get(3, 2): expected revert (3 is not a multiple of 2)")
		}
	})
}
