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
	gomath "math"

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
func (ae *AccessEvents) AddAccount(addr common.Address, isWrite bool, availableGas uint64) uint64 {
	var gas uint64 // accumulate the consumed gas
	consumed, expected := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, isWrite, availableGas)
	if consumed < expected {
		return expected
	}
	gas += consumed
	consumed, expected = ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, isWrite, availableGas-consumed)
	if consumed < expected {
		return expected + gas
	}
	gas += expected
	return gas
}

// MessageCallGas returns the gas to be charged for each of the currently
// cold member fields of an account, that need to be touched when making a message
// call to that account.
func (ae *AccessEvents) MessageCallGas(destination common.Address, availableGas uint64) uint64 {
	_, expected := ae.touchAddressAndChargeGas(destination, zeroTreeIndex, utils.BasicDataLeafKey, false, availableGas)
	if expected == 0 {
		expected = params.WarmStorageReadCostEIP2929
	}
	return expected
}

// ValueTransferGas returns the gas to be charged for each of the currently
// cold balance member fields of the caller and the callee accounts.
func (ae *AccessEvents) ValueTransferGas(callerAddr, targetAddr common.Address, availableGas uint64) uint64 {
	_, expected1 := ae.touchAddressAndChargeGas(callerAddr, zeroTreeIndex, utils.BasicDataLeafKey, true, availableGas)
	if expected1 > availableGas {
		return expected1
	}
	_, expected2 := ae.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BasicDataLeafKey, true, availableGas-expected1)
	if expected1+expected2 == 0 {
		return params.WarmStorageReadCostEIP2929
	}
	return expected1 + expected2
}

// ContractCreatePreCheckGas charges access costs before
// a contract creation is initiated. It is just reads, because the
// address collision is done before the transfer, and so no write
// are guaranteed to happen at this point.
func (ae *AccessEvents) ContractCreatePreCheckGas(addr common.Address, availableGas uint64) uint64 {
	consumed, expected1 := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, false, availableGas)
	_, expected2 := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, false, availableGas-consumed)
	return expected1 + expected2
}

// ContractCreateInitGas returns the access gas costs for the initialization of
// a contract creation.
func (ae *AccessEvents) ContractCreateInitGas(addr common.Address, availableGas uint64) (uint64, uint64) {
	var gas uint64
	consumed, expected1 := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, true, availableGas)
	gas += consumed
	consumed, expected2 := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, true, availableGas-consumed)
	gas += consumed
	return gas, expected1 + expected2
}

// AddTxOrigin adds the member fields of the sender account to the access event list,
// so that cold accesses are not charged, since they are covered by the 21000 gas.
func (ae *AccessEvents) AddTxOrigin(originAddr common.Address) {
	ae.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.BasicDataLeafKey, true, gomath.MaxUint64)
	ae.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.CodeHashLeafKey, false, gomath.MaxUint64)
}

// AddTxDestination adds the member fields of the sender account to the access event list,
// so that cold accesses are not charged, since they are covered by the 21000 gas.
func (ae *AccessEvents) AddTxDestination(addr common.Address, sendsValue, doesntExist bool) {
	ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, sendsValue, gomath.MaxUint64)
	ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, doesntExist, gomath.MaxUint64)
}

// SlotGas returns the amount of gas to be charged for a cold storage access.
func (ae *AccessEvents) SlotGas(addr common.Address, slot common.Hash, isWrite bool, availableGas uint64, chargeWarmCosts bool) uint64 {
	treeIndex, subIndex := utils.StorageIndex(slot.Bytes())
	_, expected := ae.touchAddressAndChargeGas(addr, *treeIndex, subIndex, isWrite, availableGas)
	if expected == 0 && chargeWarmCosts {
		expected = params.WarmStorageReadCostEIP2929
	}
	return expected
}

// touchAddressAndChargeGas adds any missing access event to the access event list, and returns the
// consumed and required gas.
func (ae *AccessEvents) touchAddressAndChargeGas(addr common.Address, treeIndex uint256.Int, subIndex byte, isWrite bool, availableGas uint64) (uint64, uint64) {
	branchKey := newBranchAccessKey(addr, treeIndex)
	chunkKey := newChunkAccessKey(branchKey, subIndex)

	// Read access.
	var branchRead, chunkRead bool
	if _, hasStem := ae.branches[branchKey]; !hasStem {
		branchRead = true
	}
	if _, hasSelector := ae.chunks[chunkKey]; !hasSelector {
		chunkRead = true
	}

	// Write access.
	var branchWrite, chunkWrite, chunkFill bool
	if isWrite {
		if (ae.branches[branchKey] & AccessWitnessWriteFlag) == 0 {
			branchWrite = true
		}

		chunkValue := ae.chunks[chunkKey]
		if (chunkValue & AccessWitnessWriteFlag) == 0 {
			chunkWrite = true
		}
	}

	var gas uint64
	if branchRead {
		gas += params.WitnessBranchReadCost
	}
	if chunkRead {
		gas += params.WitnessChunkReadCost
	}
	if branchWrite {
		gas += params.WitnessBranchWriteCost
	}
	if chunkWrite {
		gas += params.WitnessChunkWriteCost
	}
	if chunkFill {
		gas += params.WitnessChunkFillCost
	}

	if availableGas < gas {
		// consumed != expected
		return availableGas, gas
	}

	if branchRead {
		ae.branches[branchKey] = AccessWitnessReadFlag
	}
	if branchWrite {
		ae.branches[branchKey] |= AccessWitnessWriteFlag
	}
	if chunkRead {
		ae.chunks[chunkKey] = AccessWitnessReadFlag
	}
	if chunkWrite {
		ae.chunks[chunkKey] |= AccessWitnessWriteFlag
	}

	// consumed == expected
	return gas, gas
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
func (ae *AccessEvents) CodeChunksRangeGas(contractAddr common.Address, startPC, size uint64, codeLen uint64, isWrite bool, availableGas uint64) (uint64, uint64) {
	// note that in the case where the copied code is outside the range of the
	// contract code but touches the last leaf with contract code in it,
	// we don't include the last leaf of code in the AccessWitness.  The
	// reason that we do not need the last leaf is the account's code size
	// is already in the AccessWitness so a stateless verifier can see that
	// the code from the last leaf is not needed.
	if (codeLen == 0 && size == 0) || startPC > codeLen {
		return 0, 0
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
		consumed, expected := ae.touchAddressAndChargeGas(contractAddr, treeIndex, subIndex, isWrite, availableGas)
		// did we OOG ?
		if expected > consumed {
			return statelessGasCharged + consumed, statelessGasCharged + expected
		}
		var overflow bool
		statelessGasCharged, overflow = math.SafeAdd(statelessGasCharged, consumed)
		if overflow {
			panic("overflow when adding gas")
		}
		availableGas -= consumed
	}
	return statelessGasCharged, statelessGasCharged
}

// BasicDataGas adds the account's basic data to the accessed data, and returns the
// amount of gas that it costs.
// Note that an access in write mode implies an access in read mode, whereas an
// access in read mode does not imply an access in write mode.
func (ae *AccessEvents) BasicDataGas(addr common.Address, isWrite bool, availableGas uint64, chargeWarmCosts bool) uint64 {
	_, expected := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BasicDataLeafKey, isWrite, availableGas)
	if expected == 0 && chargeWarmCosts {
		if availableGas < params.WarmStorageReadCostEIP2929 {
			return availableGas
		}
		expected = params.WarmStorageReadCostEIP2929
	}
	return expected
}

// CodeHashGas adds the account's code hash to the accessed data, and returns the
// amount of gas that it costs.
// in write mode. If false, the charged gas corresponds to an access in read mode.
// Note that an access in write mode implies an access in read mode, whereas an access in
// read mode does not imply an access in write mode.
func (ae *AccessEvents) CodeHashGas(addr common.Address, isWrite bool, availableGas uint64, chargeWarmCosts bool) uint64 {
	_, expected := ae.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, isWrite, availableGas)
	if expected == 0 && chargeWarmCosts {
		if availableGas < params.WarmStorageReadCostEIP2929 {
			return availableGas
		}
		expected = params.WarmStorageReadCostEIP2929
	}
	return expected
}
