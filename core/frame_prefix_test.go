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
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func prefixTestEVM(sdb *state.StateDB, baseFee int64, timestamp uint64) *vm.EVM {
	config := *cfg8037
	config.BogotaTime = new(uint64)
	ctx := amsterdamCoreEVM(sdb).Context
	ctx.BaseFee, ctx.Time = big.NewInt(baseFee), timestamp
	ctx.Coinbase = common.HexToAddress("0xc01b")
	return vm.NewEVM(ctx, sdb, &config, vm.Config{})
}

func prefixTestTx(t *testing.T, frames []types.Frame, feeCap, tip uint64) *types.Transaction {
	t.Helper()
	ftx := &types.FrameTx{
		ChainID: uint256.MustFromBig(cfg8037.ChainID), Sender: senderAddr,
		Frames:     frames,
		Signatures: types.SignatureList{{Scheme: types.FrameTxSchemeSecp256k1}},
		Fees:       types.Fees{MaxFeePerGas: uint256.NewInt(feeCap), MaxPriorityFeePerGas: uint256.NewInt(tip), MaxFeePerBlobGas: new(uint256.Int)},
	}
	signer := types.NewBogotaSigner(cfg8037.ChainID)
	hash := signer.Hash(types.NewTx(ftx))
	sig, err := crypto.Sign(hash[:], senderKey)
	if err != nil {
		t.Fatal(err)
	}
	ftx.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
	return types.NewTx(ftx)
}

func prefixTestMessage(t *testing.T, evm *vm.EVM, tx *types.Transaction) *Message {
	t.Helper()
	msg, err := TransactionToMessage(tx, types.NewBogotaSigner(cfg8037.ChainID), evm.Context.BaseFee)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func prefixVerifyFrame() types.Frame {
	return types.Frame{Mode: types.ModeVerify, Flags: types.ApproveExecutionAndPayment, GasLimits: types.Limits{Execution: 50_000}, Value: new(uint256.Int)}
}

func TestFramePrefixExecution(t *testing.T) {
	target := common.HexToAddress("0xcafe")
	alloc := senderAlloc(types.GenesisAlloc{target: {Code: program.New().Sstore(42, 42).Bytes(), Balance: big.NewInt(100)}})
	tx := prefixTestTx(t, []types.Frame{prefixVerifyFrame(), {Mode: types.ModeSender, Target: &target, GasLimits: types.Limits{Execution: 100_000, State: 200_000}, Value: uint256.NewInt(7)}}, 10, 2)
	for _, prefixOnly := range []bool{true, false} {
		sdb := mkState(alloc)
		evm := prefixTestEVM(sdb, 5, 0)
		msg := prefixTestMessage(t, evm, tx)
		if prefixOnly {
			result, err := ApplyFramePrefix(evm, msg, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			if result.Payer != senderAddr || result.PayerFrame != 0 {
				t.Fatalf("wrong prefix result: %+v", result)
			}
			if sdb.GetState(target, common.BigToHash(big.NewInt(42))) != (common.Hash{}) || sdb.GetBalance(target).Uint64() != 100 {
				t.Fatal("prefix executed user operation")
			}
			if !sdb.GetBalance(evm.Context.Coinbase).IsZero() {
				t.Fatal("prefix paid coinbase")
			}
			// No settlement: the full transaction's maximum cost remains debited.
			want := new(uint256.Int).Sub(uint256.NewInt(1e18), new(uint256.Int).Mul(uint256.NewInt(msg.GasLimit), msg.GasFeeCap))
			if sdb.GetBalance(senderAddr).Cmp(want) != 0 {
				t.Fatalf("payer balance %v, want full max-cost debit %v", sdb.GetBalance(senderAddr), want)
			}
		} else {
			result, err := ApplyMessage(evm, msg, nil)
			if err != nil {
				t.Fatal(err)
			}
			if result.FrameReceipts[1].Status != frameStatusSuccess || sdb.GetState(target, common.BigToHash(big.NewInt(42))) != common.BigToHash(big.NewInt(42)) || sdb.GetBalance(target).Uint64() != 107 {
				t.Fatal("full execution did not apply user operation")
			}
			if sdb.GetBalance(evm.Context.Coinbase).IsZero() {
				t.Fatal("full execution did not pay coinbase")
			}
		}
	}
}

func TestFramePrefixFeeChecks(t *testing.T) {
	for _, tc := range []struct {
		name               string
		cap, tip           uint64
		prefixErr, fullErr error
	}{
		{"below base fee", 4, 2, nil, ErrFeeCapTooLow},
		{"tip above cap", 4, 5, ErrTipAboveFeeCap, ErrTipAboveFeeCap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := prefixTestTx(t, []types.Frame{prefixVerifyFrame()}, tc.cap, tc.tip)
			evm := prefixTestEVM(mkState(senderAlloc(nil)), 5, 0)
			_, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 0)
			if !errors.Is(err, tc.prefixErr) {
				t.Fatalf("prefix: got %v, want %v", err, tc.prefixErr)
			}
			evm = prefixTestEVM(mkState(senderAlloc(nil)), 5, 0)
			_, err = ApplyMessage(evm, prefixTestMessage(t, evm, tx), nil)
			if !errors.Is(err, tc.fullErr) {
				t.Fatalf("full: got %v, want %v", err, tc.fullErr)
			}
		})
	}
}

func TestFramePrefixFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		code []byte
	}{
		{"verify reverts", program.New().Push(0).Push(0).Op(vm.REVERT).Bytes()},
		{"no payment approval", program.New().Op(vm.STOP).Bytes()},
		{"execution approval only", program.New().Push(types.ApproveExecution).Push(0).Push(0).Op(vm.APPROVE).Bytes()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evm := prefixTestEVM(mkState(senderAlloc(types.GenesisAlloc{senderAddr: {Balance: big.NewInt(1e18), Code: tc.code}})), 5, 0)
			tx := prefixTestTx(t, []types.Frame{prefixVerifyFrame()}, 10, 2)
			if _, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 0); !errors.Is(err, ErrFrameTxInvalidExecution) {
				t.Fatalf("got %v, want invalid execution", err)
			}
		})
	}
}

func TestFramePrefixFailedDeploy(t *testing.T) {
	factory := common.HexToAddress("0xfac7")
	for _, tc := range []struct {
		name string
		code []byte
	}{
		{"revert", program.New().Push(0).Push(0).Op(vm.REVERT).Bytes()},
		{"exceptional halt", program.New().Op(vm.INVALID).Bytes()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			alloc := senderAlloc(types.GenesisAlloc{factory: {Code: tc.code}})
			tx := prefixTestTx(t, []types.Frame{
				{Mode: types.ModeDefault, Target: &factory, GasLimits: types.Limits{Execution: 20_000}, Value: new(uint256.Int)},
				prefixVerifyFrame(),
			}, 10, 2)
			evm := prefixTestEVM(mkState(alloc), 5, 0)
			result, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 1)
			if result != nil || !errors.Is(err, ErrFrameTxInvalidExecution) {
				t.Fatalf("failed deploy prefix: result=%+v err=%v", result, err)
			}
			// Block execution retains its non-fatal DEFAULT-frame semantics.
			evm = prefixTestEVM(mkState(alloc), 5, 0)
			full, err := ApplyMessage(evm, prefixTestMessage(t, evm, tx), nil)
			if err != nil {
				t.Fatal(err)
			}
			if full.FrameReceipts[0].Status != frameStatusFailed || full.FrameReceipts[1].Status != frameStatusSuccess || full.FramePayer == nil || *full.FramePayer != senderAddr {
				t.Fatalf("unexpected full execution: %+v", full)
			}
		})
	}
}

func TestFramePrefixRevertedApproval(t *testing.T) {
	// The nested call approves both scopes and returns, but its enclosing
	// VERIFY frame reverts. Payment must not survive that frame's rollback.
	code := program.New().Op(vm.CALLDATASIZE).Push(byte(0)).Op(vm.JUMPI)
	code.StaticCall(uint256.NewInt(20_000), senderAddr.Bytes(), 0, 1, 0, 0).
		Op(vm.POP).Push(0).Push(0).Op(vm.REVERT)
	code.Bytes()[2] = byte(len(code.Bytes()))
	code.Op(vm.JUMPDEST).Push(3).Push(0).Push(0).Op(vm.APPROVE)
	sdb := mkState(senderAlloc(types.GenesisAlloc{senderAddr: {Balance: big.NewInt(1e18), Code: code.Bytes()}}))
	evm := prefixTestEVM(sdb, 5, 0)
	tx := prefixTestTx(t, []types.Frame{prefixVerifyFrame()}, 10, 2)
	result, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 0)
	if result != nil || !errors.Is(err, ErrFrameTxInvalidExecution) {
		t.Fatalf("reverted approval: result=%+v err=%v", result, err)
	}
	if sdb.GetNonce(senderAddr) != 0 || sdb.GetBalance(senderAddr).Uint64() != 1e18 {
		t.Fatal("reverted payment approval survived in state")
	}
}

func TestFramePrefixBound(t *testing.T) {
	payer := common.HexToAddress("0x9876")
	// VERIFY approves execution only, a legal subset of flags 0x3. The later
	// SENDER would write storage and approve payment if the bound were ignored.
	verifyCode := program.New().Push(types.ApproveExecution).Push(0).Push(0).Op(vm.APPROVE).Bytes()
	payerCode := program.New().Sstore(42, 42).Push(types.ApprovePayment).Push(0).Push(0).Op(vm.APPROVE).Bytes()
	alloc := senderAlloc(types.GenesisAlloc{
		senderAddr: {Balance: big.NewInt(1e18), Code: verifyCode},
		payer:      {Balance: big.NewInt(1e18), Code: payerCode},
	})
	tx := prefixTestTx(t, []types.Frame{prefixVerifyFrame(), {Mode: types.ModeSender, Flags: types.ApprovePayment, Target: &payer, GasLimits: types.Limits{Execution: 100_000, State: 200_000}, Value: new(uint256.Int)}}, 10, 2)
	sdb := mkState(alloc)
	evm := prefixTestEVM(sdb, 5, 0)
	if _, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 0); !errors.Is(err, ErrFrameTxInvalidExecution) {
		t.Fatalf("got %v, want invalid execution", err)
	}
	slot := common.BigToHash(big.NewInt(42))
	if sdb.GetState(payer, slot) != (common.Hash{}) || sdb.GetBalance(payer).Uint64() != 1e18 {
		t.Fatal("prefix executed later payer frame")
	}
	// Prove the skipped frame really would have chosen a payer.
	sdb = mkState(alloc)
	evm = prefixTestEVM(sdb, 5, 0)
	result, err := ApplyMessage(evm, prefixTestMessage(t, evm, tx), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FramePayer == nil || *result.FramePayer != payer || sdb.GetState(payer, slot) != slot {
		t.Fatal("full execution did not execute later payer frame")
	}
}

func TestFramePrefixExpiry(t *testing.T) {
	for _, deadline := range []uint64{99, 100, 101} {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, deadline)
		tx := prefixTestTx(t, []types.Frame{{Mode: types.ModeVerify, Target: &params.FrameTxExpiryVerifier, GasLimits: types.Limits{Execution: 10_000}, Value: new(uint256.Int), Data: data}, prefixVerifyFrame()}, 10, 2)
		evm := prefixTestEVM(mkState(senderAlloc(types.GenesisAlloc{params.FrameTxExpiryVerifier: {Code: params.FrameTxExpiryVerifierCode}})), 5, 100)
		result, err := ApplyFramePrefix(evm, prefixTestMessage(t, evm, tx), nil, 1)
		if deadline < 100 {
			if !errors.Is(err, ErrFrameTxInvalidExecution) {
				t.Fatalf("expired: got %v", err)
			}
		} else if err != nil || result.Payer != senderAddr || result.PayerFrame != 1 {
			t.Fatalf("deadline %d: result=%+v err=%v", deadline, result, err)
		}
	}
}

func TestFramePrefixIntrospection(t *testing.T) {
	later := types.Frame{Mode: types.ModeSender, GasLimits: types.Limits{Execution: 100_000}, Value: new(uint256.Int)}
	tx := prefixTestTx(t, []types.Frame{prefixVerifyFrame(), later}, 10, 2)
	sdb := mkState(senderAlloc(nil))
	evm := prefixTestEVM(sdb, 5, 0)
	msg := prefixTestMessage(t, evm, tx)
	checks := []struct {
		selector uint64
		want     *uint256.Int
	}{
		{3, msg.GasTipCap}, {4, msg.GasFeeCap},
		{6, new(uint256.Int).Mul(uint256.NewInt(msg.GasLimit), msg.GasFeeCap)},
		{8, new(uint256.Int).SetBytes(msg.FrameSigHash[:])},
		{9, uint256.NewInt(2)}, {11, uint256.NewInt(1)},
	}
	// Approve only if all transaction introspections observe the complete
	// original transaction, not a truncated or fee-rewritten simulation.
	code := program.New().Push(1)
	for _, check := range checks {
		code.Push(check.selector).Op(vm.TXPARAM).Push(check.want).Op(vm.EQ, vm.AND)
	}
	code.Push(3).Op(vm.MUL).Push(0).Push(0).Op(vm.APPROVE)
	sdb.SetCode(senderAddr, code.Bytes(), tracing.CodeChangeUnspecified)
	result, err := ApplyFramePrefix(evm, msg, nil, 0)
	if err != nil || result.Payer != senderAddr {
		t.Fatalf("introspection approval: result=%+v err=%v", result, err)
	}
}
