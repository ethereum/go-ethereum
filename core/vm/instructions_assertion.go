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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

var errInvalidAssertion = errors.New("invalid transaction assertion parameter or context")

type assertionDiffCache struct {
	frame int
	diff  *state.TransactionDiff
}

func postTxContext(evm *EVM) (*FrameContext, error) {
	fc := evm.TxContext.FrameContext
	if fc == nil || fc.CurrentFrame < 0 || fc.CurrentFrame >= len(fc.Frames) || fc.Frames[fc.CurrentFrame].Mode != types.ModePostTx {
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

func opTxTrace(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := postTxContext(evm)
	if err != nil {
		return nil, err
	}
	param, index := scope.Stack.pop(), scope.Stack.pop()
	if !param.IsUint64() || param.Uint64() > 0x15 {
		return nil, errInvalidAssertion
	}
	diff := assertionDiff(evm, fc)
	var value uint256.Int
	p, i := param.Uint64(), index.Uint64()
	switch p {
	case 0, 1, 2, 0x0c, 0x14, 0x15:
		if !index.IsZero() {
			return nil, errInvalidAssertion
		}
		switch p {
		case 0:
			value.SetUint64(uint64(len(diff.Balances)))
		case 1:
			value.SetUint64(uint64(len(diff.Slots)))
		case 2:
			value.SetUint64(uint64(len(diff.Deployed)))
		case 0x0c:
			value.SetUint64(uint64(len(diff.Events)))
		case 0x14:
			if fc.MaxCost != nil {
				value.Set(fc.MaxCost)
			}
		case 0x15:
			if fc.Payer != nil {
				value.SetBytes(fc.Payer[:])
			}
		}
	case 3, 4, 5:
		if !assertionIndex(&index, len(diff.Balances)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Balances[i]
		switch p {
		case 3:
			value.SetBytes(entry.Address[:])
		case 4:
			value.Set(&entry.Before)
		case 5:
			value.Set(&entry.After)
		}
	case 6, 7, 8, 9:
		if !assertionIndex(&index, len(diff.Slots)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Slots[i]
		switch p {
		case 6:
			value.SetBytes(entry.Address[:])
		case 7:
			value.SetBytes(entry.Key[:])
		case 8:
			value.SetBytes(entry.Before[:])
		case 9:
			value.SetBytes(entry.After[:])
		}
	case 0x0a, 0x0b:
		if !assertionIndex(&index, len(diff.Deployed)) {
			return nil, errInvalidAssertion
		}
		entry := &diff.Deployed[i]
		if p == 0x0a {
			value.SetBytes(entry.Address[:])
		} else {
			value.SetBytes(entry.CodeHash[:])
		}
	default:
		if !assertionIndex(&index, len(diff.Events)) {
			return nil, errInvalidAssertion
		}
		event := diff.Events[i]
		switch p {
		case 0x0d:
			value.SetBytes(event.Address[:])
		case 0x0e:
			value.SetUint64(uint64(len(event.Topics)))
		case 0x13:
			value.SetUint64(uint64(len(event.Data)))
		default:
			position := p - 0x0f
			if position >= uint64(len(event.Topics)) {
				return nil, errInvalidAssertion
			}
			value.SetBytes(event.Topics[position][:])
		}
	}
	scope.Stack.push(&value)
	return nil, nil
}

func opTxDiff(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := postTxContext(evm)
	if err != nil {
		return nil, err
	}
	param, key, index := scope.Stack.pop(), scope.Stack.pop(), scope.Stack.pop()
	if !param.IsUint64() || param.Uint64() > 0x0c {
		return nil, errInvalidAssertion
	}
	p := param.Uint64()
	if p >= 2 && p != 7 && p != 9 && p != 0x0c && !index.IsZero() {
		return nil, errInvalidAssertion
	}
	address := common.Address(key.Bytes20())
	var value uint256.Int
	switch p {
	case 0, 1:
		slot := common.Hash(index.Bytes32())
		live := evm.StateDB.GetState(address, slot)
		if p == 0 {
			live = evm.StateDB.GetTransactionState(address, slot)
		}
		value.SetBytes(live[:])
	case 2, 3:
		value.Set(evm.StateDB.GetBalance(address))
		if p == 2 {
			before := evm.StateDB.GetTransactionAccount(address)
			value.Set(&before.Balance)
		}
	case 4, 5:
		hash := evm.StateDB.GetCodeHash(address)
		if hash == (common.Hash{}) {
			hash = types.EmptyCodeHash
		}
		if p == 4 {
			hash = evm.StateDB.GetTransactionAccount(address).CodeHash
		}
		value.SetBytes(hash[:])
	default:
		diff := assertionDiff(evm, fc)
		var indices []uint64
		switch p {
		case 6, 7:
			indices = diff.SlotIndices[address]
		case 8, 9:
			indices = diff.EventIndices[address]
		case 0x0a:
			value.SetUint64(diff.AccountFlags[address])
		case 0x0b, 0x0c:
			indices = diff.TopicIndices[common.Hash(key.Bytes32())]
		}
		switch p {
		case 6, 8, 0x0b:
			value.SetUint64(uint64(len(indices)))
		case 7, 9, 0x0c:
			if !assertionIndex(&index, len(indices)) {
				return nil, errInvalidAssertion
			}
			value.SetUint64(indices[index.Uint64()])
		}
	}
	scope.Stack.push(&value)
	return nil, nil
}

func opEventDataCopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	fc, err := postTxContext(evm)
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
