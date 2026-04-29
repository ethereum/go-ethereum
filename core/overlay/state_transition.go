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

package overlay

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// Storage slots used by the binary transition registry system contract at
// params.BinaryTransitionRegistryAddress.
var (
	transitionStartedKey               = common.Hash{}
	conversionProgressAddressKey       = common.BytesToHash([]byte{1})
	conversionProgressSlotKey          = common.BytesToHash([]byte{2})
	conversionProgressStorageProcessed = common.BytesToHash([]byte{3})
	transitionEndedKey                 = common.BytesToHash([]byte{4})
	baseRootKey                        = common.BytesToHash([]byte{5})
)

// StorageReader is a minimal interface for reading contract storage slots.
// It is satisfied by *state.flatReader, allowing the transition state to be
// loaded without a full state.StateDB.
type StorageReader interface {
	Storage(addr common.Address, slot common.Hash) (common.Hash, error)
}

// TransitionState holds the progress markers of the MPT-to-binary
// translation process. It is reconstructed on demand from the storage of the
// binary transition registry system contract.
type TransitionState struct {
	CurrentAccountAddress *common.Address // address of the last translated account
	CurrentSlotHash       common.Hash     // hash of the last translated storage slot
	Started, Ended        bool

	// StorageProcessed marks whether the storage of the current account has
	// been fully processed. Useful when the maximum number of leaves of the
	// conversion is reached before the storage is exhausted.
	StorageProcessed bool

	BaseRoot common.Hash // frozen MPT base root captured at the fork block
}

// InTransition returns true if the translation process is in progress.
func (ts *TransitionState) InTransition() bool {
	return ts != nil && ts.Started && !ts.Ended
}

// Transitioned returns true if the translation process has been completed.
func (ts *TransitionState) Transitioned() bool {
	return ts != nil && ts.Ended
}

// Copy returns a deep copy of the TransitionState object.
func (ts *TransitionState) Copy() *TransitionState {
	ret := &TransitionState{
		Started:          ts.Started,
		Ended:            ts.Ended,
		CurrentSlotHash:  ts.CurrentSlotHash,
		StorageProcessed: ts.StorageProcessed,
		BaseRoot:         ts.BaseRoot,
	}
	if ts.CurrentAccountAddress != nil {
		addr := *ts.CurrentAccountAddress
		ret.CurrentAccountAddress = &addr
	}
	return ret
}

// IsTransitionActive checks whether the binary transition registry has been
// initialised by reading slot 0 (started) from the system contract.
func IsTransitionActive(reader StorageReader) bool {
	val, err := reader.Storage(params.BinaryTransitionRegistryAddress, transitionStartedKey)
	if err != nil {
		return false
	}
	return val != (common.Hash{})
}

// LoadTransitionState reads the full transition state from the binary
// transition registry system contract storage. Returns nil when the
// registry has not been initialised (i.e. the chain has not yet reached the
// UBT fork block).
//
// The root parameter is unused; it is retained on the signature so callers
// can express the state version they intend to read.
func LoadTransitionState(reader StorageReader, root common.Hash) *TransitionState {
	started, err := reader.Storage(params.BinaryTransitionRegistryAddress, transitionStartedKey)
	if err != nil || started == (common.Hash{}) {
		return nil
	}

	ended, _ := reader.Storage(params.BinaryTransitionRegistryAddress, transitionEndedKey)
	baseRoot, _ := reader.Storage(params.BinaryTransitionRegistryAddress, baseRootKey)

	var currentAddr *common.Address
	addrVal, _ := reader.Storage(params.BinaryTransitionRegistryAddress, conversionProgressAddressKey)
	if addrVal != (common.Hash{}) {
		addr := common.BytesToAddress(addrVal.Bytes())
		currentAddr = &addr
	}

	slotHash, _ := reader.Storage(params.BinaryTransitionRegistryAddress, conversionProgressSlotKey)
	storageProcessed, _ := reader.Storage(params.BinaryTransitionRegistryAddress, conversionProgressStorageProcessed)

	return &TransitionState{
		Started:               true,
		Ended:                 ended != (common.Hash{}),
		BaseRoot:              baseRoot,
		CurrentAccountAddress: currentAddr,
		CurrentSlotHash:       slotHash,
		StorageProcessed:      storageProcessed != (common.Hash{}),
	}
}
