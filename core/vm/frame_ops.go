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
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// errNotFrameTransaction is returned when a frame-only instruction is executed
// outside of a frame transaction, causing an exceptional halt.
var errNotFrameTransaction = fmt.Errorf("frame transaction instruction outside a frame transaction")

// getFrame returns the active frame context or halts exceptionally when the
// instruction is executed outside a frame transaction.
func getFrame(evm *EVM) (*FrameContext, error) {
	if evm.frameCtx == nil {
		return nil, errNotFrameTransaction
	}
	return evm.frameCtx, nil
}

// opApprove implements the EIP-8141 APPROVE instruction. It exits the current
// EVM call frame successfully (like RETURN) and updates the transaction-scoped
// approval context based on the scope operand.
func opApprove(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	offset, length, scopeVal := scope.Stack.pop3()

	fc, err := getFrame(evm)
	if err != nil {
		return nil, err
	}
	// ADDRESS must equal the resolved target of the current frame.
	if scope.Contract.Address() != fc.ResolvedTarget {
		return nil, ErrExecutionReverted
	}
	// Ensure the requested scope is allowed by the current frame's flags.
	allowed := fc.Frame.Flags & byte(ApproveScopeMask)
	if scopeVal.IsZero() || scopeVal.Uint64()&^uint64(allowed) != 0 {
		return nil, ErrExecutionReverted
	}
	s := fc.State
	maxCost := new(uint256.Int)
	maxCost.SetFromBig(fc.State.MaxCost)
	switch scopeVal.Uint64() {
	case ApproveExecution:
		if s.SenderApproved || fc.ResolvedTarget != fc.Tx.Sender {
			return nil, ErrExecutionReverted
		}
		s.SenderApproved = true
	case ApprovePayment:
		if s.PayerSet || !s.SenderApproved {
			return nil, ErrExecutionReverted
		}
		if evm.StateDB.GetBalance(fc.ResolvedTarget).Cmp(maxCost) < 0 {
			return nil, ErrExecutionReverted
		}
		if err := fc.collectPayment(evm, maxCost); err != nil {
			return nil, err
		}
	case ApproveExecutionAndPayment:
		if s.SenderApproved || s.PayerSet || fc.ResolvedTarget != fc.Tx.Sender {
			return nil, ErrExecutionReverted
		}
		if evm.StateDB.GetBalance(fc.ResolvedTarget).Cmp(maxCost) < 0 {
			return nil, ErrExecutionReverted
		}
		s.SenderApproved = true
		if err := fc.collectPayment(evm, maxCost); err != nil {
			return nil, err
		}
	default:
		return nil, ErrExecutionReverted
	}
	ret := scope.Memory.GetCopy(offset.Uint64(), length.Uint64())
	return ret, errStopToken
}

// collectPayment increments the sender nonce, sets the payer, and debits the
// transaction's maximum cost from the payer.
func (fc *FrameContext) collectPayment(evm *EVM, maxCost *uint256.Int) error {
	nonce := evm.StateDB.GetNonce(fc.Tx.Sender)
	evm.StateDB.SetNonce(fc.Tx.Sender, nonce+1, tracing.NonceChangeEoACall)
	fc.State.Payer = fc.ResolvedTarget
	fc.State.PayerSet = true
	evm.StateDB.SubBalance(fc.ResolvedTarget, maxCost, tracing.BalanceDecreaseGasBuy)
	return nil
}

// opTxparam implements the EIP-8141 TXPARAM instruction.
func opTxparam(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := getFrame(evm)
	if err != nil {
		return nil, err
	}
	param := scope.Stack.pop1()
	switch param.Uint64() {
	case 0x00: // current transaction type
		param.SetUint64(uint64(types.FrameTxType))
	case 0x01: // nonce
		param.SetUint64(fc.Tx.Nonce)
	case 0x02: // sender
		param.SetBytes(fc.Tx.Sender.Bytes())
	case 0x03: // max_priority_fee_per_gas
		param.SetFromBig(fc.Tx.GasTipCap)
	case 0x04: // max_fee_per_gas
		param.SetFromBig(fc.Tx.GasFeeCap)
	case 0x05: // max_fee_per_blob_gas
		if fc.Tx.BlobFeeCap != nil {
			param.SetFromBig(fc.Tx.BlobFeeCap.ToBig())
		} else {
			param.Clear()
		}
	case 0x06: // max cost
		param.SetFromBig(fc.State.MaxCost)
	case 0x07: // len(blob_versioned_hashes)
		param.SetUint64(uint64(len(fc.Tx.BlobHashes)))
	case 0x08: // compute_sig_hash(tx)
		param.SetBytes(fc.SigHash.Bytes())
	case 0x09: // len(frames)
		param.SetUint64(uint64(len(fc.Tx.Frames)))
	case 0x0A: // currently executing frame index
		param.SetUint64(uint64(fc.FrameIndex))
	case 0x0B: // len(signatures)
		param.SetUint64(uint64(len(fc.Tx.Signatures)))
	default:
		return nil, fmt.Errorf("invalid TXPARAM parameter %d", param.Uint64())
	}
	scope.Stack.push(param)
	return nil, nil
}

// opFrameDataLoad implements the EIP-8141 FRAMEDATALOAD instruction.
func opFrameDataLoad(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := getFrame(evm)
	if err != nil {
		return nil, err
	}
	offset, frameIndex := scope.Stack.pop2()
	fidx := frameIndex.Uint64()
	if fidx >= uint64(len(fc.Tx.Frames)) {
		return nil, fmt.Errorf("invalid frame index %d", fidx)
	}
	x := scope.Stack.get()
	if o, overflow := offset.Uint64WithOverflow(); !overflow {
		x.SetBytes(getData(fc.Tx.Frames[fidx].Data, o, 32))
	} else {
		x.Clear()
	}
	return nil, nil
}

// opFrameDataCopy implements the EIP-8141 FRAMEDATACOPY instruction.
func opFrameDataCopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := getFrame(evm)
	if err != nil {
		return nil, err
	}
	memOffset, dataOffset, length, frameIndex := scope.Stack.pop4()
	fidx := frameIndex.Uint64()
	if fidx >= uint64(len(fc.Tx.Frames)) {
		return nil, fmt.Errorf("invalid frame index %d", fidx)
	}
	dataOffset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		dataOffset64 = math.MaxUint64
	}
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), getData(fc.Tx.Frames[fidx].Data, dataOffset64, length.Uint64()))
	return nil, nil
}

// opFrameParam implements the EIP-8141 FRAMEPARAM instruction.
func opFrameParam(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := getFrame(evm)
	if err != nil {
		return nil, err
	}
	frameIndex, param := scope.Stack.pop2()
	fidx := frameIndex.Uint64()
	if fidx >= uint64(len(fc.Tx.Frames)) {
		return nil, fmt.Errorf("invalid frame index %d", fidx)
	}
	frame := &fc.Tx.Frames[fidx]
	switch param.Uint64() {
	case 0x00: // resolved_target
		param.SetBytes(fc.resolvedTargetFor(int(fidx)).Bytes())
	case 0x01: // gas_limit
		param.SetUint64(frame.GasLimit)
	case 0x02: // mode
		param.SetUint64(uint64(frame.Mode))
	case 0x03: // flags
		param.SetUint64(uint64(frame.Flags))
	case 0x04: // len(data)
		param.SetUint64(uint64(len(frame.Data)))
	case 0x05: // status (current/future not accessible)
		if fidx >= uint64(fc.FrameIndex) {
			return nil, fmt.Errorf("frame status not yet available for index %d", fidx)
		}
		param.SetUint64(uint64(fc.frameStatus(int(fidx))))
	case 0x06: // allowed_scope
		param.SetUint64(uint64(frame.Flags & byte(ApproveScopeMask)))
	case 0x07: // atomic_batch
		param.SetUint64(uint64((frame.Flags >> 2) & 0x01))
	case 0x08: // value
		param.SetFromBig(frame.Value.ToBig())
	default:
		return nil, fmt.Errorf("invalid FRAMEPARAM parameter %d", param.Uint64())
	}
	scope.Stack.push(param)
	return nil, nil
}

// opSigparam implements the EIP-8141 SIGPARAM instruction. For param 0x04 it
// copies raw ARBITRARY signature bytes into memory; otherwise it returns a
// signature-scoped value.
func opSigparam(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := getFrame(evm)
	if err != nil {
		return nil, err
	}
	sidx := scope.Stack.peek().Uint64()
	if sidx >= uint64(len(fc.Tx.Signatures)) {
		return nil, fmt.Errorf("invalid signature index %d", sidx)
	}
	param := scope.Stack.back(1).Uint64()
	sig := &fc.Tx.Signatures[sidx]
	switch param {
	case 0x00: // resolved_signer
		if sig.Scheme == types.SignatureSchemeArbitrary {
			return nil, fmt.Errorf("resolved_signer unavailable for ARBITRARY signature")
		}
		resolved, ok := sig.ResolvedSigner(fc.Tx.Sender)
		if !ok {
			return nil, fmt.Errorf("invalid signer for signature %d", sidx)
		}
		sigIndex := scope.Stack.pop1()
		scope.Stack.pop()
		sigIndex.SetBytes(resolved.Bytes())
		scope.Stack.push(sigIndex)
	case 0x01: // scheme
		sigIndex := scope.Stack.pop1()
		scope.Stack.pop()
		sigIndex.SetUint64(uint64(sig.Scheme))
		scope.Stack.push(sigIndex)
	case 0x02: // msg
		sigIndex := scope.Stack.pop1()
		scope.Stack.pop()
		sigIndex.SetBytes(sig.Msg)
		scope.Stack.push(sigIndex)
	case 0x03: // len(signature)
		sigIndex := scope.Stack.pop1()
		scope.Stack.pop()
		sigIndex.SetUint64(uint64(len(sig.Signature)))
		scope.Stack.push(sigIndex)
	case 0x04: // copy ARBITRARY signature bytes to memory
		if sig.Scheme != types.SignatureSchemeArbitrary {
			return nil, fmt.Errorf("signature %d is not ARBITRARY", sidx)
		}
		// Stack: signatureIndex(top), param, memOffset, dataOffset, length
		scope.Stack.pop()
		scope.Stack.pop()
		memOffset, dataOffset, length := scope.Stack.pop3()
		dataOffset64, overflow := dataOffset.Uint64WithOverflow()
		if overflow {
			dataOffset64 = math.MaxUint64
		}
		scope.Memory.Set(memOffset.Uint64(), length.Uint64(), getData(sig.Signature, dataOffset64, length.Uint64()))
	default:
		return nil, fmt.Errorf("invalid SIGPARAM parameter %d", param)
	}
	return nil, nil
}
