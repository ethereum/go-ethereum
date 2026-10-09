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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestFramePostTxBodyRollback(t *testing.T) {
	target := common.HexToAddress("0xcafe")
	assertion := common.HexToAddress("0xdead")
	for _, code := range [][]byte{
		program.New().Push(0).Push(0).Op(vm.REVERT).Bytes(),
		program.New().Push(types.ApprovePayment).Push(0).Push(0).Op(vm.APPROVE).Bytes(),
	} {
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			target:    {Code: program.New().Sstore(42, 42).Push(0).Push(0).Op(vm.LOG0).Bytes()},
			assertion: {Code: code},
		}))
		evm := prefixTestEVM(sdb, 5, 0)
		tx := prefixTestTx(t, []types.Frame{
			prefixVerifyFrame(),
			{Mode: types.ModeSender, Target: &target, GasLimits: types.Limits{Execution: 100_000, State: 200_000}, Value: new(uint256.Int)},
			{Mode: types.ModePostTx, Target: &assertion, Flags: types.ApprovePayment, GasLimits: types.Limits{Execution: 10_000}, Value: new(uint256.Int)},
			{Mode: types.ModePostTx, GasLimits: types.Limits{Execution: 10_000}, Value: new(uint256.Int)},
		}, 10, 2)
		result, err := ApplyMessage(evm, prefixTestMessage(t, evm, tx), nil)
		if err != nil {
			t.Fatal(err)
		}
		for i, status := range []uint64{frameStatusSuccess, frameStatusSuccess, frameStatusFailed, frameStatusSkipped} {
			receipt := result.FrameReceipts[i]
			if receipt.Status != status || receipt.StateGasUsed != 0 || len(receipt.Logs) != 0 {
				t.Fatalf("frame %d: unexpected receipt %+v", i, receipt)
			}
		}
		if result.FrameReceipts[1].GasUsed == 0 || result.FrameReceipts[3].GasUsed != 0 {
			t.Fatal("body gas was refunded or skipped frame was charged")
		}
		if sdb.GetState(target, common.BigToHash(big.NewInt(42))) != (common.Hash{}) || sdb.GetNonce(senderAddr) != 1 {
			t.Fatal("body survived or validation prefix was reverted")
		}
	}
}

func TestFrameKeyedNoncePrefix(t *testing.T) {
	keys := []uint256.Int{*uint256.NewInt(1), *uint256.NewInt(2)}
	approve := program.New().Push(types.ApproveExecutionAndPayment).Push(0).Push(0).Op(vm.APPROVE).Bytes()
	sdb := mkState(senderAlloc(types.GenesisAlloc{
		senderAddr:                 {Nonce: 7, Balance: big.NewInt(1e18), Code: approve},
		params.NonceManagerAddress: {Nonce: 1, Code: params.NonceManagerCode},
	}))
	evm := prefixTestEVM(sdb, 5, 0)
	frame := prefixVerifyFrame()
	frame.GasLimits.State = 2 * params.StorageCreationSize * params.CostPerStateByte
	tx := types.NewTx(&types.FrameTx{
		ChainID: uint256.MustFromBig(cfg8037.ChainID), NonceKeys: keys, Sender: senderAddr,
		Frames: []types.Frame{frame},
		Fees:   types.Fees{MaxFeePerGas: uint256.NewInt(10), MaxPriorityFeePerGas: uint256.NewInt(2), MaxFeePerBlobGas: new(uint256.Int)},
	})
	if _, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 0); err != nil {
		t.Fatal(err)
	}
	if sdb.GetNonce(senderAddr) != 7 {
		t.Fatal("keyed nonce consumed account nonce")
	}
	for i := range keys {
		slot := types.FrameTxNonceSlot(senderAddr, &keys[i])
		if sdb.GetState(params.NonceManagerAddress, slot) != common.BigToHash(big.NewInt(1)) {
			t.Fatal("keyed nonce not consumed")
		}
		if address, slotWarm := sdb.SlotInAccessList(params.NonceManagerAddress, slot); address || slotWarm {
			t.Fatal("protocol nonce access warmed the access list")
		}
	}
}
