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

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

var errInvalidAssertion = errors.New("invalid transaction assertion parameter or context")

// TXTRACE parameters (EIP-7906). Every state-change table lists its address,
// before and after params consecutively.
const (
	txTraceBalancesChanged = 0x00
	txTraceSlotsChanged    = 0x01
	txTraceCodesChanged    = 0x02
	txTraceNoncesChanged   = 0x03
	txTraceBalanceAddress  = 0x04 // then BALANCE_BEFORE, BALANCE_AFTER
	txTraceSlotAddress     = 0x07 // then SLOT_BEFORE, SLOT_AFTER, SLOT_KEY
	txTraceCodeAddress     = 0x0b // then CODE_BEFORE, CODE_AFTER
	txTraceNonceAddress    = 0x0e // then NONCE_BEFORE, NONCE_AFTER
	txTraceEventsCount     = 0x11
	txTraceEventAddress    = 0x12
	txTraceEventTopicCount = 0x13
	txTraceEventTopic0     = 0x14 // through EVENT_TOPIC3
	txTraceEventDataLen    = 0x18
	txTraceGasPreCharge    = 0x19
	txTraceGasPayer        = 0x1a
)

type assertionDiffCache struct {
	frame int
	diff  *state.TransactionDiff
}

// assertContext returns the executing frame transaction's context, or an
// error unless the executing frame is an ASSERT frame. The check keys on the
// frame, not the call depth, so the assertion instructions are valid anywhere
// in an ASSERT frame's call subtree and nowhere else.
func assertContext(evm *EVM) (*FrameContext, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil || fc.CurrentFrame < 0 || fc.CurrentFrame >= len(fc.Frames) || fc.Frames[fc.CurrentFrame].Mode != types.ModeAssert {
		return nil, errInvalidAssertion
	}
	return fc, nil
}

func assertionDiff(evm *EVM, fc *FrameContext) *state.TransactionDiff {
	if fc.assertionDiff == nil || fc.assertionDiff.frame != fc.CurrentFrame {
		fc.assertionDiff = &assertionDiffCache{fc.CurrentFrame, evm.StateDB.GetTransactionDiff()}
	}
	return fc.assertionDiff.diff
}

func assertionIndex(index *uint256.Int, length int) bool {
	return index.IsUint64() && index.Uint64() < uint64(length)
}

// opTxTrace implements the TXTRACE instruction (EIP-7906), pushing one value
// of the executing transaction's state diff selected by a parameter and, for
// the per-entry parameters, an index into the table the parameter enumerates.
func opTxTrace(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := assertContext(evm)
	if err != nil {
		return nil, err
	}
	param, index := scope.Stack.pop(), scope.Stack.pop()
	if !param.IsUint64() || param.Uint64() > txTraceGasPayer {
		return nil, errInvalidAssertion
	}
	diff := assertionDiff(evm, fc)
	var value uint256.Int
	p, i := param.Uint64(), index.Uint64()
	switch {
	case p <= txTraceNoncesChanged || p == txTraceEventsCount || p >= txTraceGasPreCharge:
		if !index.IsZero() {
			return nil, errInvalidAssertion
		}
		switch p {
		case txTraceBalancesChanged:
			value.SetUint64(uint64(len(diff.Balances)))
		case txTraceSlotsChanged:
			value.SetUint64(uint64(len(diff.Slots)))
		case txTraceCodesChanged:
			value.SetUint64(uint64(len(diff.Codes)))
		case txTraceNoncesChanged:
			value.SetUint64(uint64(len(diff.Nonces)))
		case txTraceEventsCount:
			value.SetUint64(uint64(len(diff.Events)))
		case txTraceGasPreCharge:
			if fc.MaxCost != nil {
				value.Set(fc.MaxCost)
			}
		case txTraceGasPayer:
			if fc.Payer != nil {
				value.SetBytes(fc.Payer[:])
			}
		}
	case p < txTraceSlotAddress:
		if !assertionIndex(&index, len(diff.Balances)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Balances[i]
		switch p - txTraceBalanceAddress {
		case 0:
			value.SetBytes(entry.Address[:])
		case 1:
			value.Set(&entry.Before)
		case 2:
			value.Set(&entry.After)
		}
	case p < txTraceCodeAddress:
		if !assertionIndex(&index, len(diff.Slots)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Slots[i]
		switch p - txTraceSlotAddress {
		case 0:
			value.SetBytes(entry.Address[:])
		case 1:
			value.SetBytes(entry.Before[:])
		case 2:
			value.SetBytes(entry.After[:])
		case 3:
			value.SetBytes(entry.Key[:])
		}
	case p < txTraceNonceAddress:
		if !assertionIndex(&index, len(diff.Codes)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Codes[i]
		switch p - txTraceCodeAddress {
		case 0:
			value.SetBytes(entry.Address[:])
		case 1:
			value.SetBytes(entry.Before[:])
		case 2:
			value.SetBytes(entry.After[:])
		}
	case p < txTraceEventsCount:
		if !assertionIndex(&index, len(diff.Nonces)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Nonces[i]
		switch p - txTraceNonceAddress {
		case 0:
			value.SetBytes(entry.Address[:])
		case 1:
			value.SetUint64(entry.Before)
		case 2:
			value.SetUint64(entry.After)
		}
	default:
		if !assertionIndex(&index, len(diff.Events)) {
			return nil, errInvalidAssertion
		}
		event := diff.Events[i]
		switch p {
		case txTraceEventAddress:
			value.SetBytes(event.Address[:])
		case txTraceEventTopicCount:
			value.SetUint64(uint64(len(event.Topics)))
		case txTraceEventDataLen:
			value.SetUint64(uint64(len(event.Data)))
		default:
			position := p - txTraceEventTopic0
			if position >= uint64(len(event.Topics)) {
				return nil, errInvalidAssertion
			}
			value.SetBytes(event.Topics[position][:])
		}
	}
	scope.Stack.push(&value)
	return nil, nil
}

// opEventDataCopy implements the EVENTDATACOPY instruction (EIP-7906),
// copying a portion of one event's non-indexed data into memory.
func opEventDataCopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := assertContext(evm)
	if err != nil {
		return nil, err
	}
	index, memoryOffset, dataOffset, length := scope.Stack.pop(), scope.Stack.pop(), scope.Stack.pop(), scope.Stack.pop()
	diff := assertionDiff(evm, fc)
	if !assertionIndex(&index, len(diff.Events)) {
		return nil, errInvalidAssertion
	}
	data := diff.Events[index.Uint64()].Data
	// Compare separately, avoiding both 256-bit and native integer wraparound.
	if !dataOffset.IsUint64() || !length.IsUint64() || dataOffset.Uint64() > uint64(len(data)) || length.Uint64() > uint64(len(data))-dataOffset.Uint64() {
		return nil, errInvalidAssertion
	}
	if !length.IsZero() {
		start, size := dataOffset.Uint64(), length.Uint64()
		scope.Memory.Set(memoryOffset.Uint64(), size, data[start:start+size])
	}
	return nil, nil
}
