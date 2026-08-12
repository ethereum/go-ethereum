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

// Transaction-level tests for EIP-8141 (frame transactions). They apply whole
// frame transactions and inspect the per-frame receipts, the approval context
// and the resulting state.

package core

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// Contracts used as frame targets.
var (
	// codeStore writes 1 to slot 0 and returns.
	codeStore = common.FromHex("600160005500")
	// codeRevert reverts immediately with no return data.
	codeRevert = common.FromHex("5f5ffd")
	// codeSigParamCopy invokes the SIGPARAM copy form (param 0x04) carrying only
	// the two operands the metadata form needs, leaving the deeper operands
	// absent. Stack from the top: signatureIndex=0, param=4.
	codeSigParamCopy = common.FromHex("60046000b4")
)

var (
	frameStoreAddr     = common.Address{0xc0, 0x01}
	frameRevertAddr    = common.Address{0xc0, 0x02}
	frameSigParamAddr  = common.Address{0xc0, 0x03}
	frameSenderKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	frameSenderAddress = crypto.PubkeyToAddress(frameSenderKey.PublicKey)
)

// frameChainConfig activates the Frame fork on top of Bogota.
func frameChainConfig() *params.ChainConfig {
	cfg := *params.MergedTestChainConfig
	cfg.AmsterdamTime = new(uint64)
	cfg.BogotaTime = new(uint64)
	cfg.FrameTime = new(uint64)
	return &cfg
}

// frameTestState builds a state with the sender funded, the expiry verifier
// predeployed and the helper contracts installed.
func frameTestState() *state.StateDB {
	return mkState(types.GenesisAlloc{
		frameSenderAddress:                {Balance: newGwei(1_000_000_000)},
		frameStoreAddr:                    {Code: codeStore, Balance: common.Big0},
		frameRevertAddr:                   {Code: codeRevert, Balance: common.Big0},
		frameSigParamAddr:                 {Code: codeSigParamCopy, Balance: common.Big0},
		params.FrameExpiryVerifierAddress: {Nonce: 1, Code: params.FrameExpiryVerifierCode, Balance: common.Big0},
	})
}

// frameEVM builds a Frame-fork EVM over statedb with fees disabled.
func frameEVM(sdb *state.StateDB) *vm.EVM {
	ctx := vm.BlockContext{
		CanTransfer:      CanTransfer,
		Transfer:         Transfer,
		GetHash:          func(uint64) common.Hash { return common.Hash{} },
		BlockNumber:      big.NewInt(0),
		Time:             1000,
		Random:           &common.Hash{},
		Difficulty:       big.NewInt(0),
		BaseFee:          big.NewInt(0),
		BlobBaseFee:      big.NewInt(0),
		GasLimit:         60_000_000,
		CostPerStateByte: params.CostPerStateByte,
	}
	return vm.NewEVM(ctx, sdb, frameChainConfig(), vm.Config{NoBaseFee: true})
}

// selfVerifyFrame returns a VERIFY frame targeting the sender with both approval
// scopes set. Against an account with no code this takes the EIP-8141 default
// code path, which approves on the strength of signature 0.
func selfVerifyFrame(gas uint64) types.Frame {
	return types.Frame{
		Mode:     types.FrameModeVerify,
		Flags:    types.FrameFlagApproveExecutionPayment,
		GasLimit: gas,
		Value:    new(uint256.Int),
	}
}

// senderFrame returns a SENDER frame calling target.
func senderFrame(target common.Address, gas uint64, flags byte) types.Frame {
	t := target
	return types.Frame{
		Mode:     types.FrameModeSender,
		Flags:    flags,
		Target:   &t,
		GasLimit: gas,
		Value:    new(uint256.Int),
	}
}

// signFrameTx fills in signature 0 as the sender's secp256k1 signature over the
// canonical signature hash, which is what the default code requires.
func signFrameTx(t *testing.T, ftx *types.FrameTx) *types.Transaction {
	t.Helper()
	ftx.Signatures = []types.FrameSignature{{Scheme: types.SignatureSchemeSecp256k1}}
	sigHash := ftx.ComputeSigHash()
	sig, err := crypto.Sign(sigHash[:], frameSenderKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// EIP-8141 encodes secp256k1 signatures as v || r || s; crypto.Sign gives r || s || v.
	frameSig := make([]byte, 65)
	frameSig[0] = sig[64]
	copy(frameSig[1:33], sig[:32])
	copy(frameSig[33:65], sig[32:65])
	ftx.Signatures[0].Signature = frameSig
	return types.NewTx(ftx)
}

// newFrameTx assembles a frame transaction from the given frames, signed by the
// sender over the canonical signature hash.
func newFrameTx(t *testing.T, frames ...types.Frame) *types.Transaction {
	t.Helper()
	return signFrameTx(t, &types.FrameTx{
		ChainID:    frameChainConfig().ChainID,
		Nonce:      0,
		Sender:     frameSenderAddress,
		Frames:     frames,
		GasTipCap:  big.NewInt(0),
		GasFeeCap:  big.NewInt(0),
		BlobFeeCap: new(uint256.Int),
	})
}

// applyFrameTx applies a frame transaction against sdb and returns the result.
func applyFrameTx(t *testing.T, sdb *state.StateDB, tx *types.Transaction) (*ExecutionResult, error) {
	t.Helper()
	cfg := frameChainConfig()
	evm := frameEVM(sdb)
	msg, err := TransactionToMessage(tx, types.MakeSigner(cfg, evm.Context.BlockNumber, evm.Context.Time), evm.Context.BaseFee)
	if err != nil {
		t.Fatalf("to message: %v", err)
	}
	sdb.SetTxContext(tx.Hash(), 0, 0)
	evm.SetTxContext(NewEVMTxContext(msg))
	gp := NewGasPool(evm.Context.GasLimit)
	return newStateTransition(evm, msg, gp).execute()
}

// slotSet reports whether the store contract's slot 0 was written.
func slotSet(sdb *state.StateDB, addr common.Address) bool {
	return sdb.GetState(addr, common.Hash{}) != (common.Hash{})
}

// TestFrameTxDefaultCode checks the EIP-8141 default code path: an account with
// no code of its own validates a frame transaction and approves both scopes on
// the strength of the sender's signature, so the transaction can execute.
func TestFrameTxDefaultCode(t *testing.T) {
	sdb := frameTestState()
	tx := newFrameTx(t, selfVerifyFrame(100_000), senderFrame(frameStoreAddr, 300_000, 0))

	res, err := applyFrameTx(t, sdb, tx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Payer != frameSenderAddress {
		t.Errorf("payer = %v, want the sender %v", res.Payer, frameSenderAddress)
	}
	if len(res.FrameReceipts) != 2 {
		t.Fatalf("got %d frame receipts, want 2", len(res.FrameReceipts))
	}
	for i, fr := range res.FrameReceipts {
		if fr.Status != types.ReceiptStatusSuccessful {
			t.Errorf("frame %d status = %d, want success", i, fr.Status)
		}
	}
	if !slotSet(sdb, frameStoreAddr) {
		t.Error("SENDER frame did not run: slot 0 unset")
	}
	// The sender's nonce is consumed exactly once, when payment is approved.
	if got := sdb.GetNonce(frameSenderAddress); got != 1 {
		t.Errorf("sender nonce = %d, want 1", got)
	}
	// The VERIFY frame must be charged for the work it did rather than running free.
	if res.FrameReceipts[0].GasUsed == 0 {
		t.Error("VERIFY frame reported zero gas")
	}
}

// TestFrameTxContractSenderApprove checks the APPROVE opcode itself, as opposed
// to the default code path: a sender that carries its own code approves both
// scopes from EVM bytecode. It also covers the EIP-8141 rule that EIP-3607 does
// not apply, so a contract may be tx.sender.
func TestFrameTxContractSenderApprove(t *testing.T) {
	// APPROVE takes offset, length, scope from the top of the stack, so scope is
	// pushed first: PUSH1 0x03 (APPROVE_EXECUTION_AND_PAYMENT), PUSH0, PUSH0, APPROVE.
	codeApprove := common.FromHex("60035f5faa")
	sdb := mkState(types.GenesisAlloc{
		frameSenderAddress: {Balance: newGwei(1_000_000_000), Code: codeApprove},
		frameStoreAddr:     {Code: codeStore, Balance: common.Big0},
	})
	tx := newFrameTx(t, selfVerifyFrame(100_000), senderFrame(frameStoreAddr, 300_000, 0))

	res, err := applyFrameTx(t, sdb, tx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Payer != frameSenderAddress {
		t.Errorf("payer = %v, want %v", res.Payer, frameSenderAddress)
	}
	if got := res.FrameReceipts[0].Status; got != types.ReceiptStatusSuccessful {
		t.Errorf("VERIFY frame status = %d, want success", got)
	}
	if !slotSet(sdb, frameStoreAddr) {
		t.Error("SENDER frame did not run")
	}
}

// TestFrameTxApproveScopeExceedsFlags checks that APPROVE reverts when it asks
// for a scope the frame's flags do not permit.
func TestFrameTxApproveScopeExceedsFlags(t *testing.T) {
	// Asks for scope 0x3 while the frame only allows 0x1.
	codeApprove := common.FromHex("60035f5faa")
	sdb := mkState(types.GenesisAlloc{
		frameSenderAddress: {Balance: newGwei(1_000_000_000), Code: codeApprove},
	})
	frame := selfVerifyFrame(100_000)
	frame.Flags = types.FrameFlagApprovePayment // 0x1, so 0x3 is out of scope
	tx := newFrameTx(t, frame)

	// The VERIFY frame reverts, which invalidates the whole transaction.
	if _, err := applyFrameTx(t, sdb, tx); !errors.Is(err, ErrFrameInvalid) {
		t.Fatalf("err = %v, want ErrFrameInvalid", err)
	}
}

// codeYulAccount is a minimal EIP-8141 smart account compiled from standalone
// Yul with solc 0.8.30 (`solc --strict-assembly`), using verbatim builtins to
// emit the new opcodes. Source in the project notes; the runtime is:
//
//	PUSH0 SLOAD              // owner from slot 0
//	PUSH0 PUSH0 SIGPARAM     // resolved_signer of signature 0
//	DUP2 DUP2 SUB PUSH1 0x10 JUMPI
//	PUSH1 0x03 PUSH0 PUSH0 APPROVE   // scope, length, offset
//	JUMPDEST PUSH0 PUSH0 REVERT
//
// It approves execution and payment iff the protocol-validated signer of
// signature 0 matches the owner. The protocol has already checked that
// signature against the canonical signature hash, so the account only decides
// whether it trusts the signer.
var codeYulAccount = common.FromHex("5f545f5fb481810360105760035f5faa5b5f5ffd")

var frameOwnerKey, _ = crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")

// TestFrameTxYulSmartAccount runs a smart account compiled by an unmodified
// solc through the frame execution path, proving the opcodes are reachable from
// a real toolchain: standalone Yul + verbatim needs no compiler fork.
func TestFrameTxYulSmartAccount(t *testing.T) {
	owner := crypto.PubkeyToAddress(frameOwnerKey.PublicKey)

	buildTx := func(t *testing.T, signWith *ecdsa.PrivateKey, signer common.Address) *types.Transaction {
		t.Helper()
		ftx := &types.FrameTx{
			ChainID: frameChainConfig().ChainID,
			Sender:  frameSenderAddress,
			Frames: []types.Frame{
				selfVerifyFrame(100_000),
				senderFrame(frameStoreAddr, 300_000, 0),
			},
			GasTipCap:  big.NewInt(0),
			GasFeeCap:  big.NewInt(0),
			BlobFeeCap: new(uint256.Int),
			// An explicit signer, so the account can distinguish the key that
			// signed from the account address itself.
			Signatures: []types.FrameSignature{{
				Scheme: types.SignatureSchemeSecp256k1,
				Signer: signer.Bytes(),
			}},
		}
		sigHash := ftx.ComputeSigHash()
		sig, err := crypto.Sign(sigHash[:], signWith)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		frameSig := make([]byte, 65)
		frameSig[0] = sig[64]
		copy(frameSig[1:33], sig[:32])
		copy(frameSig[33:65], sig[32:65])
		ftx.Signatures[0].Signature = frameSig
		return types.NewTx(ftx)
	}

	newState := func() *state.StateDB {
		sdb := mkState(types.GenesisAlloc{
			frameSenderAddress: {
				Balance: newGwei(1_000_000_000),
				Code:    codeYulAccount,
				Storage: map[common.Hash]common.Hash{{}: common.BytesToHash(owner.Bytes())},
			},
			frameStoreAddr: {Code: codeStore, Balance: common.Big0},
		})
		return sdb
	}

	t.Run("owner signs", func(t *testing.T) {
		sdb := newState()
		res, err := applyFrameTx(t, sdb, buildTx(t, frameOwnerKey, owner))
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if res.Payer != frameSenderAddress {
			t.Errorf("payer = %v, want the account %v", res.Payer, frameSenderAddress)
		}
		if got := res.FrameReceipts[0].Status; got != types.ReceiptStatusSuccessful {
			t.Errorf("VERIFY frame status = %d, want success", got)
		}
		if !slotSet(sdb, frameStoreAddr) {
			t.Error("SENDER frame did not run")
		}
	})

	t.Run("stranger signs", func(t *testing.T) {
		// A validly signed transaction from a key the account does not trust:
		// the protocol accepts the signature, the account rejects the signer, so
		// the VERIFY frame reverts and the transaction is invalid.
		stranger := crypto.PubkeyToAddress(frameSenderKey.PublicKey)
		sdb := newState()
		if _, err := applyFrameTx(t, sdb, buildTx(t, frameSenderKey, stranger)); !errors.Is(err, ErrFrameInvalid) {
			t.Fatalf("err = %v, want ErrFrameInvalid", err)
		}
	})
}

// TestFrameTxNoPayerIsInvalid checks that a transaction whose frames never
// approve payment is rejected outright.
func TestFrameTxNoPayerIsInvalid(t *testing.T) {
	sdb := frameTestState()
	// A DEFAULT frame cannot approve anything, so no payer is ever set.
	target := frameStoreAddr
	tx := newFrameTx(t, types.Frame{Mode: types.FrameModeDefault, Target: &target, GasLimit: 100_000, Value: new(uint256.Int)})

	if _, err := applyFrameTx(t, sdb, tx); !errors.Is(err, ErrFrameInvalid) {
		t.Fatalf("err = %v, want ErrFrameInvalid", err)
	}
}

// TestFrameTxAtomicBatchTerminatorFailure covers EIP-8141 Example 2: the batch
// terminator is the frame *without* the flag, so its failure must still roll the
// whole batch back.
func TestFrameTxAtomicBatchTerminatorFailure(t *testing.T) {
	sdb := frameTestState()
	tx := newFrameTx(t,
		selfVerifyFrame(100_000),
		// Batch: the store succeeds, the terminating revert fails.
		senderFrame(frameStoreAddr, 300_000, types.FrameFlagAtomicBatch),
		senderFrame(frameRevertAddr, 100_000, 0),
	)
	res, err := applyFrameTx(t, sdb, tx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if slotSet(sdb, frameStoreAddr) {
		t.Error("batch was not rolled back: the store frame's write survived the terminator's failure")
	}
	// The frames that ran keep their own status; only their logs are dropped.
	if got := res.FrameReceipts[1].Status; got != types.ReceiptStatusSuccessful {
		t.Errorf("frame 1 status = %d, want success (it did execute)", got)
	}
	if got := res.FrameReceipts[2].Status; got != types.ReceiptStatusFailed {
		t.Errorf("frame 2 status = %d, want failure", got)
	}
}

// TestFrameTxAtomicBatchSkipsRemainingFrames checks that when a frame inside a
// batch fails, the remaining frames of that batch are not executed at all: they
// are reported as skipped and consume none of their gas allotment.
func TestFrameTxAtomicBatchSkipsRemainingFrames(t *testing.T) {
	sdb := frameTestState()
	tx := newFrameTx(t,
		selfVerifyFrame(100_000),
		// Batch: the revert fails first, so the store must never run.
		senderFrame(frameRevertAddr, 100_000, types.FrameFlagAtomicBatch),
		senderFrame(frameStoreAddr, 300_000, types.FrameFlagAtomicBatch),
		senderFrame(frameStoreAddr, 300_000, 0),
	)
	res, err := applyFrameTx(t, sdb, tx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if slotSet(sdb, frameStoreAddr) {
		t.Error("a frame after the batch failure executed and its write survived")
	}
	if got := res.FrameReceipts[1].Status; got != types.ReceiptStatusFailed {
		t.Errorf("frame 1 status = %d, want failure", got)
	}
	for _, i := range []int{2, 3} {
		fr := res.FrameReceipts[i]
		if fr.Status != types.ReceiptStatusSkipped {
			t.Errorf("frame %d status = %d, want skipped (%d)", i, fr.Status, types.ReceiptStatusSkipped)
		}
		if fr.GasUsed != 0 {
			t.Errorf("frame %d used %d gas, want 0: a skipped frame's allotment is refunded", i, fr.GasUsed)
		}
	}
}

// TestFrameTxAtomicBatchSuccess checks that a batch which fully succeeds keeps
// its state changes.
func TestFrameTxAtomicBatchSuccess(t *testing.T) {
	sdb := frameTestState()
	tx := newFrameTx(t,
		selfVerifyFrame(100_000),
		senderFrame(frameStoreAddr, 300_000, types.FrameFlagAtomicBatch),
		senderFrame(frameStoreAddr, 300_000, 0),
	)
	if _, err := applyFrameTx(t, sdb, tx); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !slotSet(sdb, frameStoreAddr) {
		t.Error("a successful batch was rolled back")
	}
}

// TestFrameTxSigParamCopyUnderflow checks that the SIGPARAM copy form with too
// few operands fails the frame instead of panicking the node. The copy form
// reads five stack items but the jump table can only require two.
func TestFrameTxSigParamCopyUnderflow(t *testing.T) {
	sdb := frameTestState()
	tx := newFrameTx(t, selfVerifyFrame(100_000), senderFrame(frameSigParamAddr, 100_000, 0))

	res, err := applyFrameTx(t, sdb, tx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := res.FrameReceipts[1].Status; got != types.ReceiptStatusFailed {
		t.Errorf("frame status = %d, want failure from the stack underflow", got)
	}
}

// TestFrameTxExpiryVerifier checks the expiry verifier predeploy: a deadline in
// the future passes and costs gas, and one in the past reverts, which invalidates
// the transaction because the frame runs in VERIFY mode.
func TestFrameTxExpiryVerifier(t *testing.T) {
	expiryFrame := func(deadline uint64) types.Frame {
		addr := params.FrameExpiryVerifierAddress
		data := make([]byte, params.FrameTxExpiryDataLength)
		for i := range data {
			data[i] = byte(deadline >> (8 * (len(data) - 1 - i)))
		}
		return types.Frame{
			Mode:     types.FrameModeVerify,
			Target:   &addr,
			GasLimit: 100_000,
			Value:    new(uint256.Int),
			Data:     data,
		}
	}
	// Block time is 1000, so 2000 is still in the future.
	t.Run("valid", func(t *testing.T) {
		sdb := frameTestState()
		tx := newFrameTx(t, expiryFrame(2000), selfVerifyFrame(100_000), senderFrame(frameStoreAddr, 300_000, 0))
		res, err := applyFrameTx(t, sdb, tx)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if got := res.FrameReceipts[0].Status; got != types.ReceiptStatusSuccessful {
			t.Errorf("expiry frame status = %d, want success", got)
		}
		// Executing the predeploy must be charged for, not evaluated for free.
		if res.FrameReceipts[0].GasUsed == 0 {
			t.Error("expiry frame reported zero gas")
		}
	})
	t.Run("expired", func(t *testing.T) {
		sdb := frameTestState()
		tx := newFrameTx(t, expiryFrame(999), selfVerifyFrame(100_000))
		if _, err := applyFrameTx(t, sdb, tx); !errors.Is(err, ErrFrameInvalid) {
			t.Fatalf("err = %v, want ErrFrameInvalid for an expired deadline", err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		sdb := frameTestState()
		frame := expiryFrame(2000)
		frame.Data = frame.Data[:4] // not EXPIRY_DATA_LENGTH
		tx := newFrameTx(t, frame, selfVerifyFrame(100_000))
		if _, err := applyFrameTx(t, sdb, tx); !errors.Is(err, ErrFrameInvalid) {
			t.Fatalf("err = %v, want ErrFrameInvalid for a malformed expiry frame", err)
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		sdb := frameTestState()
		tx := newFrameTx(t, expiryFrame(2000), expiryFrame(2000), selfVerifyFrame(100_000))
		if _, err := applyFrameTx(t, sdb, tx); !errors.Is(err, ErrFrameInvalid) {
			t.Fatalf("err = %v, want ErrFrameInvalid for two expiry frames", err)
		}
	})
}

// TestFrameTxMaxCostOverflow checks that a fee cap large enough to overflow
// max_cost is rejected. Without the check the payer refund underflows and the
// payer is credited a wrapped balance.
func TestFrameTxMaxCostOverflow(t *testing.T) {
	sdb := frameTestState()
	hugeFee := new(big.Int).Lsh(big.NewInt(1), 255)
	tx := signFrameTx(t, &types.FrameTx{
		ChainID:    frameChainConfig().ChainID,
		Nonce:      0,
		Sender:     frameSenderAddress,
		Frames:     []types.Frame{selfVerifyFrame(100_000)},
		GasTipCap:  big.NewInt(0),
		GasFeeCap:  hugeFee,
		BlobFeeCap: new(uint256.Int),
	})
	balanceBefore := sdb.GetBalance(frameSenderAddress).Clone()
	if _, err := applyFrameTx(t, sdb, tx); err == nil {
		t.Fatal("an overflowing max cost was accepted")
	}
	if got := sdb.GetBalance(frameSenderAddress); got.Cmp(balanceBefore) > 0 {
		t.Fatalf("payer balance grew from %v to %v", balanceBefore, got)
	}
}

// TestFrameTxSenderFrameBeforeApproval checks that a SENDER frame is rejected
// until some frame has granted APPROVE_EXECUTION.
func TestFrameTxSenderFrameBeforeApproval(t *testing.T) {
	sdb := frameTestState()
	tx := newFrameTx(t, senderFrame(frameStoreAddr, 300_000, 0), selfVerifyFrame(100_000))

	if _, err := applyFrameTx(t, sdb, tx); !errors.Is(err, ErrFrameInvalid) {
		t.Fatalf("err = %v, want ErrFrameInvalid", err)
	}
}

// TestFrameTxTransientStorageClearedBetweenFrames checks the EIP-8141 rule that
// transient storage does not carry from one frame to the next.
func TestFrameTxTransientStorageClearedBetweenFrames(t *testing.T) {
	var (
		// tstoreAddr writes 1 to transient slot 0.
		tstoreAddr = common.Address{0xd0, 0x01}
		// tloadAddr reverts if transient slot 0 is set, else returns.
		tloadAddr = common.Address{0xd0, 0x02}
		// PUSH1 01 PUSH0 TSTORE STOP
		codeTStore = common.FromHex("60015f5d00")
		// PUSH0 TLOAD PUSH1 09 JUMPI STOP PUSH0 PUSH0 REVERT
		codeTLoad = common.FromHex("5f5c600657005b5f5ffd")
	)
	sdb := mkState(types.GenesisAlloc{
		frameSenderAddress: {Balance: newGwei(1_000_000_000)},
		tstoreAddr:         {Code: codeTStore, Balance: common.Big0},
		tloadAddr:          {Code: codeTLoad, Balance: common.Big0},
	})
	tx := newFrameTx(t,
		selfVerifyFrame(100_000),
		senderFrame(tstoreAddr, 300_000, 0),
		senderFrame(tloadAddr, 300_000, 0),
	)
	res, err := applyFrameTx(t, sdb, tx)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := res.FrameReceipts[2].Status; got != types.ReceiptStatusSuccessful {
		t.Errorf("frame 2 status = %d, want success: transient storage leaked across frames", got)
	}
}
