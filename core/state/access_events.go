// Copyright 2021 The go-ethereum Authors
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

package state

import (
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

// mode specifies how a tree location has been accessed
// for the byte value:
// * the first bit is set if the branch has been read
// * the second bit is set if the branch has been edited
type mode byte

const (
	AccessWitnessReadFlag  = mode(1)
	AccessWitnessWriteFlag = mode(2)
)

var zeroTreeIndex uint256.Int

// AccessEvents lists the locations of the state that are being accessed
// during the production of a block.
type AccessEvents struct {
	branches map[branchAccessKey]mode
	chunks   map[chunkAccessKey]mode

	pointCache *utils.PointCache
}

func NewAccessEvents(pointCache *utils.PointCache) *AccessEvents {
	return &AccessEvents{
		branches:   make(map[branchAccessKey]mode),
		chunks:     make(map[chunkAccessKey]mode),
		pointCache: pointCache,
	}
}

// Merge is used to merge the access events that were generated during the
// execution of a tx, with the accumulation of all access events that were
// generated during the execution of all txs preceding this one in a block.
func (ae *AccessEvents) Merge(other *AccessEvents) {
	for k := range other.branches {
		ae.branches[k] |= other.branches[k]
	}
	for k, chunk := range other.chunks {
		ae.chunks[k] |= chunk
	}
}

// Keys returns, predictably, the list of keys that were touched during the
// buildup of the access witness.
func (ae *AccessEvents) Keys() [][]byte {
	// TODO: consider if parallelizing this is worth it, probably depending on len(ae.chunks).
	keys := make([][]byte, 0, len(ae.chunks))
	for chunk := range ae.chunks {
		basePoint := ae.pointCache.Get(chunk.addr[:])
		key := utils.GetTreeKeyWithEvaluatedAddress(basePoint, &chunk.treeIndex, chunk.leafKey)
		keys = append(keys, key)
	}
	return keys
}

func (ae *AccessEvents) Copy() *AccessEvents {
	cpy := &AccessEvents{
		branches:   maps.Clone(ae.branches),
		chunks:     maps.Clone(ae.chunks),
		pointCache: ae.pointCache,
	}
	return cpy
}

// AddAccount returns the gas to be charged for each of the currently cold
// member fields of an account.
func (ae *AccessEvents) AddAccount(addr common.Address, isWrite bool) uint64 {
	var gas uint64
	gas += ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, isWrite)
	gas += ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, isWrite)
	return gas
}

// MessageCallGas returns the gas to be charged for each of the currently
// cold member fields of an account, that need to be touched when making a message
// call to that account.
func (ae *AccessEvents) MessageCallGas(destination common.Address) uint64 {
	var gas uint64
	gas += ae.touchAddressAndChargeGas(destination, zeroTreeIndex, utils.BasicDataLeafKey, false)
	return gas
}

// ValueTransferGas returns the gas to be charged for each of the currently
// cold balance member fields of the caller and the callee accounts.
func (ae *AccessEvents) ValueTransferGas(callerAddr, targetAddr common.Address) uint64 {
	var gas uint64
	gas += ae.touchAddressAndChargeGas(callerAddr, zeroTreeIndex, utils.BasicDataLeafKey, true)
	gas += ae.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BasicDataLeafKey, true)
	return gas
}

// ContractCreatePreCheckGas charges access costs before
// a contract creation is initiated. It is just reads, because the
// address collision is done before the transfer, and so no write
// are guaranteed to happen at this point.
func (ae *AccessEvents) ContractCreatePreCheckGas(addr common.Address) uint64 {
	var gas uint64
	gas += ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, false)
	gas += ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, false)
	return gas
}

// ContractCreateInitGas returns the access gas costs for the initialization of
// a contract creation.
func (ae *AccessEvents) ContractCreateInitGas(addr common.Address) uint64 {
	var gas uint64
	gas += ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, true)
	gas += ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, true)
	return gas
}

// AddTxOrigin adds the member fields of the sender account to the access event list,
// so that cold accesses are not charged, since they are covered by the 21000 gas.
func (ae *AccessEvents) AddTxOrigin(originAddr common.Address) {
	ae.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.BasicDataLeafKey, true)
	ae.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.CodeHashLeafKey, false)
}

// AddTxDestination adds the member fields of the sender account to the access event list,
// so that cold accesses are not charged, since they are covered by the 21000 gas.
func (ae *AccessEvents) AddTxDestination(addr common.Address, sendsValue bool) {
	ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, sendsValue)
	ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, false)
}

// SlotGas returns the amount of gas to be charged for a cold storage access.
func (ae *AccessEvents) SlotGas(addr common.Address, slot common.Hash, isWrite bool) uint64 {
	treeIndex, subIndex := utils.StorageIndex(slot.Bytes())
	return ae.touchAddressAndChargeGas(addr, *treeIndex, subIndex, isWrite)
}

// touchAddressAndChargeGas adds any missing access event to the access event list, and returns the cold
// access cost to be charged, if need be.
func (ae *AccessEvents) touchAddressAndChargeGas(addr common.Address, treeIndex uint256.Int, subIndex byte, isWrite bool) uint64 {
	stemRead, selectorRead, stemWrite, selectorWrite, selectorFill := ae.touchAddress(addr, treeIndex, subIndex, isWrite)

	var gas uint64
	if stemRead {
		gas += params.WitnessBranchReadCost
	}
	if selectorRead {
		gas += params.WitnessChunkReadCost
	}
	if stemWrite {
		gas += params.WitnessBranchWriteCost
	}
	if selectorWrite {
		gas += params.WitnessChunkWriteCost
	}
	if selectorFill {
		gas += params.WitnessChunkFillCost
	}
	return gas
}

// touchAddress adds any missing access event to the access event list.
func (ae *AccessEvents) touchAddress(addr common.Address, treeIndex uint256.Int, subIndex byte, isWrite bool) (bool, bool, bool, bool, bool) {
	branchKey := newBranchAccessKey(addr, treeIndex)
	chunkKey := newChunkAccessKey(branchKey, subIndex)

	// Read access.
	var branchRead, chunkRead bool
	if _, hasStem := ae.branches[branchKey]; !hasStem {
		branchRead = true
		ae.branches[branchKey] = AccessWitnessReadFlag
	}
	if _, hasSelector := ae.chunks[chunkKey]; !hasSelector {
		chunkRead = true
		ae.chunks[chunkKey] = AccessWitnessReadFlag
	}

	// Write access.
	var branchWrite, chunkWrite, chunkFill bool
	if isWrite {
		if (ae.branches[branchKey] & AccessWitnessWriteFlag) == 0 {
			branchWrite = true
			ae.branches[branchKey] |= AccessWitnessWriteFlag
		}

		chunkValue := ae.chunks[chunkKey]
		if (chunkValue & AccessWitnessWriteFlag) == 0 {
			chunkWrite = true
			ae.chunks[chunkKey] |= AccessWitnessWriteFlag
		}
		// TODO: charge chunk filling costs if the leaf was previously empty in the state
	}
	return branchRead, chunkRead, branchWrite, chunkWrite, chunkFill
}

type branchAccessKey struct {
	addr      common.Address
	treeIndex uint256.Int
}

func newBranchAccessKey(addr common.Address, treeIndex uint256.Int) branchAccessKey {
	var sk branchAccessKey
	sk.addr = addr
	sk.treeIndex = treeIndex
	return sk
}

type chunkAccessKey struct {
	branchAccessKey
	leafKey byte
}

func newChunkAccessKey(branchKey branchAccessKey, leafKey byte) chunkAccessKey {
	var lk chunkAccessKey
	lk.branchAccessKey = branchKey
	lk.leafKey = leafKey
	return lk
}

// CodeChunksRangeGas is a helper function to touch every chunk in a code range and charge witness gas costs
func (ae *AccessEvents) CodeChunksRangeGas(contractAddr common.Address, startPC, size uint64, codeLen uint64, isWrite bool) uint64 {
	// note that in the case where the copied code is outside the range of the
	// contract code but touches the last leaf with contract code in it,
	// we don't include the last leaf of code in the AccessWitness.  The
	// reason that we do not need the last leaf is the account's code size
	// is already in the AccessWitness so a stateless verifier can see that
	// the code from the last leaf is not needed.
	if (codeLen == 0 && size == 0) || startPC > codeLen {
		return 0
	}

	endPC := startPC + size
	if endPC > codeLen {
		endPC = codeLen
	}
	if endPC > 0 {
		endPC -= 1 // endPC is the last bytecode that will be touched.
	}

	var statelessGasCharged uint64
	for chunkNumber := startPC / 31; chunkNumber <= endPC/31; chunkNumber++ {
		treeIndex := *uint256.NewInt((chunkNumber + 128) / 256)
		subIndex := byte((chunkNumber + 128) % 256)
		gas := ae.touchAddressAndChargeGas(contractAddr, treeIndex, subIndex, isWrite)
		var overflow bool
		statelessGasCharged, overflow = math.SafeAdd(statelessGasCharged, gas)
		if overflow {
			panic("overflow when adding gas")
		}
	}
	return statelessGasCharged
}

// BasicDataGas adds the account's basic data to the accessed data, and returns the
// amount of gas that it costs.
// Note that an access in write mode implies an access in read mode, whereas an
// access in read mode does not imply an access in write mode.
func (ae *AccessEvents) BasicDataGas(addr common.Address, isWrite bool) uint64 {
	return ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, isWrite)
}

// CodeHashGas adds the account's code hash to the accessed data, and returns the
// amount of gas that it costs.
// in write mode. If false, the charged gas corresponds to an access in read mode.
// Note that an access in write mode implies an access in read mode, whereas an access in
// read mode does not imply an access in write mode.
func (ae *AccessEvents) CodeHashGas(addr common.Address, isWrite bool) uint64 {
	return ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, isWrite)
}
