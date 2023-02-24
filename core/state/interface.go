// Copyright 2014 The go-ethereum Authors
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

// Package state provides a caching layer atop the Ethereum state trie.

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// StateDBI is an EVM database for full state querying.
type StateDBI interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	GetCommittedState(common.Address, common.Hash) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	GetTransientState(addr common.Address, key common.Hash) common.Hash
	SetTransientState(addr common.Address, key, value common.Hash)

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
	// AddAddressToAccessList adds the given address to the access list. This operation is safe to perform
	// even if the feature/fork is not active yet
	AddAddressToAccessList(addr common.Address)
	// AddSlotToAccessList adds the given (address,slot) to the access list. This operation is safe to perform
	// even if the feature/fork is not active yet
	AddSlotToAccessList(addr common.Address, slot common.Hash)
	Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	Logs() []*types.Log
	GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash) []*types.Log
	TxIndex() int
	AddPreimage(common.Hash, []byte)
	Preimages() map[common.Hash][]byte

	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error

	GetOrNewStateObject(addr common.Address) *StateObject

	DumpToCollector(c DumpCollector, conf *DumpConfig) (nextKey []byte)
	Dump(opts *DumpConfig) []byte
	RawDump(opts *DumpConfig) Dump
	IteratorDump(opts *DumpConfig) IteratorDump
	Database() Database
	StorageTrie(addr common.Address) (Trie, error)
	Error() error
	GetStorageProof(a common.Address, key common.Hash) ([][]byte, error)
	GetProof(addr common.Address) ([][]byte, error)
	SetBalance(addr common.Address, amount *big.Int)
	SetStorage(addr common.Address, storage map[common.Hash]common.Hash)
	Finalise(deleteEmptyObjects bool)
	Commit(deleteEmptyObjects bool) (common.Hash, error)
	Copy() StateDBI
	SetTxContext(thash common.Hash, ti int)
	StopPrefetcher()
	StartPrefetcher(namespace string)
	IntermediateRoot(deleteEmptyObjects bool) common.Hash
}
