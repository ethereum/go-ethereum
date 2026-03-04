// Copyright 2025 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// buildApproveCode returns EVM bytecode that replicates the EELS approve_bytecode
// with a trampoline self-CALL pattern for the APPROVE caller check.
// Thanks Claude for this fn :)
func buildApproveCode(scope uint8) []byte {

	code := []byte{}
	b := func(ops ...byte) { code = append(code, ops...) }

	// ---- Compute frame_target ----
	// stack: []
	// TXPARAMLOAD(selector=0x10, index=0, offset=0) → current_frame_index
	b(byte(vm.PUSH1), 0x00) // offset=0
	b(byte(vm.PUSH1), 0x00) // index=0
	b(byte(vm.PUSH1), 0x10) // selector=0x10 (current frame index)
	b(byte(vm.TXPARAMLOAD)) // → [frame_idx]

	// TXPARAMLOAD(selector=0x11, index=frame_idx, offset=0) → frame target raw
	b(byte(vm.PUSH1), 0x00) // offset=0
	b(byte(vm.DUP2))        // frame_idx
	b(byte(vm.PUSH1), 0x11) // selector=0x11 (frame target)
	b(byte(vm.TXPARAMLOAD)) // → [frame_target_raw, frame_idx]
	b(byte(vm.SWAP1))       // → [frame_idx, frame_target_raw]
	b(byte(vm.POP))         // → [frame_target_raw]

	// Resolve null target → sender:
	// if frame_target_raw == 0: frame_target = TXPARAMLOAD(0x02, 0, 0) (= sender)
	// frame_target = frame_target_raw | (iszero(frame_target_raw) * sender)
	b(byte(vm.DUP1))   // → [frame_target_raw, frame_target_raw]
	b(byte(vm.ISZERO)) // → [is_null, frame_target_raw]
	// TXPARAMLOAD(0x02, 0, 0) → sender
	b(byte(vm.PUSH1), 0x00) // offset
	b(byte(vm.PUSH1), 0x00) // index
	b(byte(vm.PUSH1), 0x02) // selector = sender
	b(byte(vm.TXPARAMLOAD)) // → [sender, is_null, frame_target_raw]
	b(byte(vm.MUL))         // → [sender*is_null, frame_target_raw]
	b(byte(vm.OR))          // → [frame_target]

	// ---- Check CALLER == frame_target OR CALLER == ADDRESS ----
	b(byte(vm.CALLER))  // → [caller, frame_target]
	b(byte(vm.DUP2))    // → [frame_target, caller, frame_target]
	b(byte(vm.EQ))      // → [caller==frame_target, frame_target]
	b(byte(vm.CALLER))  // → [caller, caller==ft, frame_target]
	b(byte(vm.ADDRESS)) // → [address, caller, caller==ft, frame_target]
	b(byte(vm.EQ))      // → [caller==addr, caller==ft, frame_target]
	b(byte(vm.OR))      // → [condition, frame_target]
	b(byte(vm.SWAP1))   // → [frame_target, condition]
	b(byte(vm.POP))     // → [condition]

	// ---- Branch: JUMPI to approve block ----
	// We'll patch the jump destination after we know the layout
	jumpiPos := len(code)
	b(byte(vm.PUSH1), 0x00) // placeholder for approve_pc
	b(byte(vm.JUMPI))

	// ---- Else: trampoline self-CALL ----
	// CALL(gas=100000, address=ADDRESS, value=0, argsOffset=0, argsSize=0, retOffset=0, retSize=0)
	b(byte(vm.PUSH1), 0x00)             // retSize=0
	b(byte(vm.PUSH1), 0x00)             // retOffset=0
	b(byte(vm.PUSH1), 0x00)             // argsSize=0
	b(byte(vm.PUSH1), 0x00)             // argsOffset=0
	b(byte(vm.PUSH1), 0x00)             // value=0
	b(byte(vm.ADDRESS))                 // to=self
	b(byte(vm.PUSH3), 0x01, 0x86, 0xA0) // gas=100000
	b(byte(vm.CALL))
	b(byte(vm.POP)) // discard success flag

	// RETURNDATACOPY(0, 0, RETURNDATASIZE)
	b(byte(vm.RETURNDATASIZE))
	b(byte(vm.PUSH1), 0x00)
	b(byte(vm.PUSH1), 0x00)
	b(byte(vm.RETURNDATACOPY))

	// RETURN(0, RETURNDATASIZE)
	b(byte(vm.RETURNDATASIZE))
	b(byte(vm.PUSH1), 0x00)
	b(byte(vm.RETURN))

	// ---- Approve block ----
	approvePC := len(code)
	b(byte(vm.JUMPDEST))
	// APPROVE(offset=0, length=0, scope=<scope>)
	b(byte(vm.PUSH1), scope) // scope
	b(byte(vm.PUSH1), 0x00)  // length=0
	b(byte(vm.PUSH1), 0x00)  // offset=0
	b(byte(vm.APPROVE))
	b(byte(vm.STOP))

	// Patch the JUMPI destination
	code[jumpiPos+1] = byte(approvePC)

	return code
}

// buildSstoreCode returns EVM bytecode: SSTORE(slot, value) + STOP
func buildSstoreCode(slot, value byte) []byte {
	return []byte{
		byte(vm.PUSH1), value,
		byte(vm.PUSH1), slot,
		byte(vm.SSTORE),
		byte(vm.STOP),
	}
}

// TestFrameTxHappyPathSelfPaid mirrors the Python test_happy_path_self_paid:
// Todo - add more tests , writing one tests only curretly to debug Errors arised
func TestFrameTxHappyPathSelfPaid(t *testing.T) {
	var (
		config          = params.AllDevChainProtocolChanges
		sender          = common.HexToAddress("0x1000000000000000000000000000000000001000")
		executionTarget = common.HexToAddress("0x2000000000000000000000000000000000002000")
		senderCode      = buildApproveCode(0x02)
		execCode        = buildSstoreCode(0x01, 0x01)
		baseFee         = big.NewInt(7)
	)

	// --- State ---
	db := state.NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
	statedb, _ := state.New(types.EmptyRootHash, db)

	statedb.SetCode(sender, senderCode, tracing.CodeChangeUnspecified)
	statedb.SetNonce(sender, 1, tracing.NonceChangeUnspecified)
	statedb.SetBalance(sender,
		uint256.NewInt(1_000_000_000_000_000_000),
		tracing.BalanceChangeUnspecified,
	)
	statedb.SetCode(executionTarget, execCode, tracing.CodeChangeUnspecified)
	statedb.SetNonce(executionTarget, 1, tracing.NonceChangeUnspecified)
	statedb.Commit(0, false, false)

	// --- Build tx ---
	frames := []types.Frame{
		{
			Mode:     types.FrameModeVerify,
			Target:   sender,
			GasLimit: 50_000,
			Data:     []byte{},
		},
		{
			Mode:     types.FrameModeSender,
			Target:   executionTarget,
			GasLimit: 50_000,
			Data:     []byte{},
		},
	}
	frameTx := &types.FrameTx{
		ChainID:   uint256.NewInt(1),
		Nonce:     1,
		Sender:    sender,
		Frames:    frames,
		GasTipCap: uint256.NewInt(0),
		GasFeeCap: uint256.NewInt(7),
	}
	tx := types.NewTx(frameTx)

	rnd := common.BigToHash(big.NewInt(42))
	header := &types.Header{
		Number:     big.NewInt(1),
		Time:       0,
		Difficulty: big.NewInt(0),
		GasLimit:   10_000_000,
		BaseFee:    baseFee,
		Extra:      []byte{},
		MixDigest:  rnd,
	}

	blockCtx := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(n uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.HexToAddress("0xC014BA5E"),
		GasLimit:    header.GasLimit,
		BlockNumber: header.Number,
		Time:        header.Time,
		Difficulty:  header.Difficulty,
		BaseFee:     baseFee,
		Random:      &rnd,
	}
	evm := vm.NewEVM(blockCtx, statedb, config, vm.Config{})

	gp := new(GasPool).AddGas(header.GasLimit)
	var usedGas uint64

	receipt, err := ApplyTransaction(evm, gp, statedb, header, tx, &usedGas)
	if err != nil {
		t.Fatalf("ApplyTransaction failed: %v", err)
	}
	// jsonBytes, _ := json.MarshalIndent(receipt, "", "  ")
	// fmt.Printf("\n\nreceipt: ")
	// fmt.Println(string(jsonBytes))
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("receipt status: got %d want 1", receipt.Status)
	}

	// Sender nonce should be 2
	if n := statedb.GetNonce(sender); n != 2 {
		t.Errorf("sender nonce: got %d want 2", n)
	}

	// Execution target storage slot 1 should be 1
	slot := statedb.GetState(executionTarget, common.BigToHash(big.NewInt(1)))
	if slot != common.BigToHash(big.NewInt(1)) {
		t.Errorf("slot 1: got %x want 1", slot)
	}

	// Payer should be sender
	if receipt.Payer == nil || *receipt.Payer != sender {
		t.Errorf("payer: got %v want %s", receipt.Payer, sender.Hex())
	}

	// Frame receipts should be [success, success]
	if len(receipt.FrameReceipts) != 2 {
		t.Fatalf("frame receipts: got %d want 2", len(receipt.FrameReceipts))
	}
	for i, fr := range receipt.FrameReceipts {
		if fr.Status != types.ReceiptStatusSuccessful {
			t.Errorf("frame %d status: got %d want 1", i, fr.Status)
		}
		t.Logf("FrameReceipt[%d]: status=%d gasUsed=%d logs=%d", i, fr.Status, fr.GasUsed, len(fr.Logs))
	}
}
