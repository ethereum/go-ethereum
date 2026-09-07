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

package vm

import (
	"errors"
	stdmath "math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	errNoFrameContext = errors.New("no frame transaction context")
	errInvalidTxParam = errors.New("invalid tx parameter")
)

// FrameContext holds the transaction-scoped state of an executing EIP-8141
// frame transaction. It is shared by all frames of the transaction.
type FrameContext struct {
	Sender               common.Address
	Nonce                uint64
	MaxPriorityFeePerGas *uint256.Int
	MaxFeePerGas         *uint256.Int
	MaxFeePerBlobGas     *uint256.Int
	MaxCost              *uint256.Int
	SigHash              common.Hash
	Frames               []types.FrameTxFrame
	Signatures           []types.FrameTxSignature

	CurrentFrame int

	// Receipts holds the live receipts of the completed frames, growing as
	// frames complete. Entries are not final until the transaction ends: an
	// atomic batch unroll zeroes the state gas of the unrolled frames'
	// receipts. Logs are materialized by the frame loop once the
	// transaction has finished.
	Receipts []types.FrameReceipt

	SenderApproved bool
	Payer          *common.Address
}

// CurrentTarget returns the resolved target of the currently executing frame.
func (fc *FrameContext) CurrentTarget() common.Address {
	return fc.Frames[fc.CurrentFrame].ResolvedTarget(fc.Sender)
}

// FrameContextSnapshot captures the mutable, transaction-scoped fields of a
// frame context so they can be restored when the call that changed them
// fails. The context follows the same journaling scope as state: an approval
// granted or a receipt edited in a child call is discarded together with
// that call's state changes.
type FrameContextSnapshot struct {
	senderApproved bool
	payer          *common.Address
	receipts       []types.FrameReceipt
}

// Snapshot captures the current frame context.
func (fc *FrameContext) Snapshot() FrameContextSnapshot {
	if fc == nil {
		return FrameContextSnapshot{}
	}
	receipts := make([]types.FrameReceipt, len(fc.Receipts))
	copy(receipts, fc.Receipts)
	return FrameContextSnapshot{
		senderApproved: fc.SenderApproved,
		payer:          fc.Payer,
		receipts:       receipts,
	}
}

// RestoreSnapshot restores a previously captured frame context.
func (fc *FrameContext) RestoreSnapshot(s FrameContextSnapshot) {
	if fc == nil {
		return
	}
	fc.SenderApproved = s.senderApproved
	fc.Payer = s.payer
	fc.Receipts = s.receipts
}

// FrameApprove validates and performs an APPROVE of the given scope for the
// currently executing frame, per EIP-8141. On payment approval the sender's
// nonce is incremented and the transaction's maximum cost is collected from
// the frame's resolved target; a sender account created by the nonce
// increment is charged from the executing frame's state gas pool through
// budget. ErrExecutionReverted is returned when the request is not allowed,
// ErrOutOfGas when the pool cannot cover the sender-creation charge.
func FrameApprove(statedb StateDB, fc *FrameContext, budget *GasBudget, scope uint64) error {
	frame := &fc.Frames[fc.CurrentFrame]
	target := frame.ResolvedTarget(fc.Sender)
	allowed := frame.Flags & types.FrameTxApproveScopeMask
	if scope == 0 || scope&^allowed != 0 {
		return ErrExecutionReverted
	}
	if scope&types.FrameTxApproveExecution != 0 {
		if fc.SenderApproved {
			return ErrExecutionReverted
		}
		if target != fc.Sender {
			return ErrExecutionReverted
		}
	}
	if scope&types.FrameTxApprovePayment != 0 {
		if fc.Payer != nil {
			return ErrExecutionReverted
		}
		if scope&types.FrameTxApproveExecution == 0 && !fc.SenderApproved {
			return ErrExecutionReverted
		}
		if statedb.GetBalance(target).Cmp(fc.MaxCost) < 0 {
			return ErrExecutionReverted
		}
	}
	if scope&types.FrameTxApproveExecution != 0 {
		fc.SenderApproved = true
	}
	if scope&types.FrameTxApprovePayment != 0 {
		// Incrementing the nonce of a non-existent sender creates the
		// account: charge the creation from the frame's state gas pool
		// immediately before the increment. A pool that cannot cover the
		// charge halts the current call frame, discarding every approval
		// effect with the halt's rollback.
		if statedb.Empty(fc.Sender) {
			if _, ok := budget.Charge(GasCosts{StateGas: params.AccountCreationSize * params.CostPerStateByte}); !ok {
				return ErrOutOfGas
			}
		}
		statedb.SetNonce(fc.Sender, statedb.GetNonce(fc.Sender)+1, tracing.NonceChangeEoACall)
		statedb.SubBalance(target, fc.MaxCost, tracing.BalanceDecreaseGasBuy)
		payer := target
		fc.Payer = &payer
	}
	return nil
}

// opApprove implements the APPROVE instruction (EIP-8141). It exits the
// current call frame successfully, like RETURN, while updating the
// transaction-scoped approval context. Only the memory expansion is charged.
func opApprove(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	offset := scope.Stack.pop()
	length := scope.Stack.pop()
	scopeArg := scope.Stack.pop()

	if scope.Contract.Address() != fc.CurrentTarget() {
		return nil, ErrExecutionReverted
	}
	scopeVal, overflow := scopeArg.Uint64WithOverflow()
	if overflow {
		return nil, ErrExecutionReverted
	}
	if err := FrameApprove(evm.StateDB, fc, &scope.Contract.Gas, scopeVal); err != nil {
		return nil, err
	}
	ret := scope.Memory.GetCopy(offset.Uint64(), length.Uint64())
	return ret, errStopToken
}

func frameByIndex(fc *FrameContext, index *uint256.Int) (*types.FrameTxFrame, error) {
	i, overflow := index.Uint64WithOverflow()
	if overflow || i >= uint64(len(fc.Frames)) {
		return nil, errInvalidTxParam
	}
	return &fc.Frames[i], nil
}

func pushWord(scope *ScopeContext, word []byte) {
	scope.Stack.push(new(uint256.Int).SetBytes(word))
}

func pushUint(scope *ScopeContext, v uint64) {
	scope.Stack.push(new(uint256.Int).SetUint64(v))
}

func pushAddress(scope *ScopeContext, addr common.Address) {
	scope.Stack.push(new(uint256.Int).SetBytes(addr[:]))
}

// opTxParam implements the TXPARAM instruction (EIP-8141), giving access to
// transaction-scoped information.
func opTxParam(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	param := scope.Stack.pop()
	selector, overflow := param.Uint64WithOverflow()
	if overflow {
		return nil, errInvalidTxParam
	}
	switch selector {
	case 0x00:
		pushUint(scope, types.FrameTxType)
	case 0x01:
		pushUint(scope, fc.Nonce)
	case 0x02:
		pushAddress(scope, fc.Sender)
	case 0x03:
		scope.Stack.push(new(uint256.Int).Set(fc.MaxPriorityFeePerGas))
	case 0x04:
		scope.Stack.push(new(uint256.Int).Set(fc.MaxFeePerGas))
	case 0x05:
		scope.Stack.push(new(uint256.Int).Set(fc.MaxFeePerBlobGas))
	case 0x06:
		scope.Stack.push(new(uint256.Int).Set(fc.MaxCost))
	case 0x07:
		pushUint(scope, uint64(len(evm.TxContext.BlobHashes)))
	case 0x08:
		pushWord(scope, fc.SigHash[:])
	case 0x09:
		pushUint(scope, uint64(len(fc.Frames)))
	case 0x0a:
		pushUint(scope, uint64(fc.CurrentFrame))
	case 0x0b:
		pushUint(scope, uint64(len(fc.Signatures)))
	default:
		return nil, errInvalidTxParam
	}
	return nil, nil
}

// opFrameDataLoad implements the FRAMEDATALOAD instruction (EIP-8141),
// loading one 32-byte word of the chosen frame's data with CALLDATALOAD
// semantics.
func opFrameDataLoad(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	offset := scope.Stack.pop()
	frameIndex := scope.Stack.pop()

	frame, err := frameByIndex(fc, &frameIndex)
	if err != nil {
		return nil, err
	}
	off, overflow := offset.Uint64WithOverflow()
	if overflow {
		off = stdmath.MaxUint64
	}
	pushWord(scope, getData(frame.Data, off, 32))
	return nil, nil
}

// opFrameDataCopy implements the FRAMEDATACOPY instruction (EIP-8141),
// copying the chosen frame's data into memory with CALLDATACOPY semantics.
func opFrameDataCopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	memOffset := scope.Stack.pop()
	dataOffset := scope.Stack.pop()
	length := scope.Stack.pop()
	frameIndex := scope.Stack.pop()

	frame, err := frameByIndex(fc, &frameIndex)
	if err != nil {
		return nil, err
	}
	off, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		off = stdmath.MaxUint64
	}
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), getData(frame.Data, off, length.Uint64()))
	return nil, nil
}

// opFrameParam implements the FRAMEPARAM instruction (EIP-8141), giving
// access to frame-scoped information. A frame's status and gas usage are
// read from its live receipt, which exists only once the frame has
// completed: requesting them for the current or a future frame results in
// an exceptional halt.
func opFrameParam(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	frameIndex := scope.Stack.pop()
	param := scope.Stack.pop()

	frame, err := frameByIndex(fc, &frameIndex)
	if err != nil {
		return nil, err
	}
	// completedReceipt returns the receipt of a completed frame, or an
	// exceptional halt for the current or a future frame.
	completedReceipt := func() (*types.FrameReceipt, error) {
		index := frameIndex.Uint64()
		if index >= uint64(fc.CurrentFrame) || index >= uint64(len(fc.Receipts)) {
			return nil, errInvalidTxParam
		}
		return &fc.Receipts[index], nil
	}
	selector, overflow := param.Uint64WithOverflow()
	if overflow {
		return nil, errInvalidTxParam
	}
	switch selector {
	case 0x00:
		pushAddress(scope, frame.ResolvedTarget(fc.Sender))
	case 0x01:
		pushUint(scope, frame.GasLimits.Execution)
	case 0x02:
		pushUint(scope, frame.Mode)
	case 0x03:
		pushUint(scope, frame.Flags)
	case 0x04:
		pushUint(scope, uint64(len(frame.Data)))
	case 0x05:
		receipt, err := completedReceipt()
		if err != nil {
			return nil, err
		}
		pushUint(scope, receipt.Status)
	case 0x06:
		pushUint(scope, frame.Flags&types.FrameTxApproveScopeMask)
	case 0x07:
		pushUint(scope, (frame.Flags&types.FrameTxAtomicBatchFlag)>>2)
	case 0x08:
		value := new(uint256.Int)
		if frame.Value != nil {
			value.Set(frame.Value)
		}
		scope.Stack.push(value)
	default:
		return nil, errInvalidTxParam
	}
	return nil, nil
}

func sigByIndex(fc *FrameContext, index *uint256.Int) (*types.FrameTxSignature, error) {
	i, overflow := index.Uint64WithOverflow()
	if overflow || i >= uint64(len(fc.Signatures)) {
		return nil, errInvalidTxParam
	}
	return &fc.Signatures[i], nil
}

// opSigParam implements the SIGPARAM instruction (EIP-8141), giving access
// to signature-scoped metadata. Each scheme family withholds the metadata
// the protocol does not define for it: the resolved signer is available only
// for protocol-validated entries, and the signature byte length only for
// ARBITRARY entries — the raw signature bytes of protocol-validated schemes,
// including their length, are not introspectable.
func opSigParam(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	sigIndex := scope.Stack.pop()
	param := scope.Stack.pop()

	sig, err := sigByIndex(fc, &sigIndex)
	if err != nil {
		return nil, err
	}
	selector, overflow := param.Uint64WithOverflow()
	if overflow {
		return nil, errInvalidTxParam
	}
	switch selector {
	case 0x00:
		if sig.Scheme == types.FrameTxSchemeArbitrary {
			return nil, errInvalidTxParam
		}
		signer := sig.ResolvedSigner(fc.Sender)
		pushWord(scope, signer[:])
	case 0x01:
		pushUint(scope, sig.Scheme)
	case 0x02:
		pushWord(scope, sig.Msg)
	case 0x03:
		if sig.Scheme != types.FrameTxSchemeArbitrary {
			return nil, errInvalidTxParam
		}
		pushUint(scope, uint64(len(sig.Signature)))
	default:
		return nil, errInvalidTxParam
	}
	return nil, nil
}

// opSigDataCopy implements the SIGDATACOPY instruction (EIP-8141), copying
// the raw bytes of an ARBITRARY signature entry into memory with
// CALLDATACOPY semantics. The raw bytes of protocol-validated schemes are
// not accessible.
func opSigDataCopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil {
		return nil, errNoFrameContext
	}
	memOffset := scope.Stack.pop()
	dataOffset := scope.Stack.pop()
	length := scope.Stack.pop()
	sigIndex := scope.Stack.pop()

	sig, err := sigByIndex(fc, &sigIndex)
	if err != nil {
		return nil, err
	}
	if sig.Scheme != types.FrameTxSchemeArbitrary {
		return nil, errInvalidTxParam
	}
	off, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		off = stdmath.MaxUint64
	}
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), getData(sig.Signature, off, length.Uint64()))
	return nil, nil
}

// memoryApprove returns the memory size required by APPROVE.
func memoryApprove(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(1))
}

// memoryFrameDataCopy returns the memory size required by FRAMEDATACOPY and
// SIGDATACOPY.
func memoryFrameDataCopy(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(2))
}
