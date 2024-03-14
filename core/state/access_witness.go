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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

// mode specifies how a tree location has been accessed
// for the byte value:
// * the first bit is set if the branch has been edited
// * the second bit is set if the branch has been read
type mode byte

const (
	AccessWitnessReadFlag  = mode(1)
	AccessWitnessWriteFlag = mode(2)
)

var zeroTreeIndex uint256.Int

// AccessWitness lists the locations of the state that are being accessed
// during the production of a block.
type AccessWitness struct {
	branches map[branchAccessKey]mode
	chunks   map[chunkAccessKey]mode

	pointCache *utils.PointCache
}

func NewAccessWitness() *AccessWitness {
	return &AccessWitness{
		branches:   make(map[branchAccessKey]mode),
		chunks:     make(map[chunkAccessKey]mode),
		pointCache: utils.NewPointCache(),
	}
}

// Merge is used to merge the witness that got generated during the execution
// of a tx, with the accumulation of witnesses that were generated during the
// execution of all the txs preceding this one in a given block.
func (aw *AccessWitness) Merge(o AccessList) {
	other := o.(*AccessWitness)
	for k := range other.branches {
		aw.branches[k] |= other.branches[k]
	}
	for k, chunk := range other.chunks {
		aw.chunks[k] |= chunk
	}
}

// Key returns, predictably, the list of keys that were touched during the
// buildup of the access witness.
func (aw *AccessWitness) Keys() [][]byte {
	// TODO: consider if parallelizing this is worth it, probably depending on len(aw.chunks).
	keys := make([][]byte, 0, len(aw.chunks))
	for chunk := range aw.chunks {
		basePoint := aw.pointCache.GetTreeKeyHeader(chunk.addr[:])
		key := utils.GetTreeKeyWithEvaluatedAddess(basePoint, &chunk.treeIndex, chunk.leafKey)
		keys = append(keys, key)
	}
	return keys
}

func (aw *AccessWitness) Copy() AccessList {
	naw := &AccessWitness{
		branches:   make(map[branchAccessKey]mode),
		chunks:     make(map[chunkAccessKey]mode),
		pointCache: aw.pointCache,
	}
	naw.Merge(aw)
	return naw
}

func (aw *AccessWitness) TouchAndChargeValueTransfer(callerAddr, targetAddr []byte) uint64 {
	var gas uint64
	gas += aw.touchAddressAndChargeGas(callerAddr, zeroTreeIndex, utils.BalanceLeafKey, AccessListWrite)
	gas += aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BalanceLeafKey, AccessListWrite)
	return gas
}

// TouchAndChargeContractCreateInit charges access costs to initiate
// a contract creation
func (aw *AccessWitness) TouchAndChargeContractCreateInit(addr []byte, createSendsValue bool) uint64 {
	var gas uint64
	gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.VersionLeafKey, AccessListWrite)
	gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.NonceLeafKey, AccessListWrite)
	// FIXME note that this is incompatible with the spec
	if createSendsValue {
		gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BalanceLeafKey, AccessListWrite)
	}
	return gas
}

func (aw *AccessWitness) TouchTxOriginAndComputeGas(originAddr []byte) uint64 {
	aw.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.VersionLeafKey, AccessListRead)
	aw.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.CodeSizeLeafKey, AccessListRead)
	aw.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.CodeKeccakLeafKey, AccessListRead)
	aw.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.NonceLeafKey, AccessListWrite)
	aw.touchAddressAndChargeGas(originAddr, zeroTreeIndex, utils.BalanceLeafKey, AccessListWrite)

	// Kaustinen note: we're currently experimenting with stop chargin gas for the origin address
	// so simple transfer still take 21000 gas. This is to potentially avoid breaking existing tooling.
	// This is the reason why we return 0 instead of `gas`.
	// Note that we still have to touch the addresses to make sure the witness is correct.
	return 0
}

func (aw *AccessWitness) TouchTxExistingAndComputeGas(targetAddr []byte, sendsValue bool) uint64 {
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.VersionLeafKey, AccessListRead)
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.CodeSizeLeafKey, AccessListRead)
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.CodeKeccakLeafKey, AccessListRead)
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.NonceLeafKey, AccessListRead)
	if sendsValue {
		aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BalanceLeafKey, AccessListWrite)
	} else {
		aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BalanceLeafKey, AccessListRead)
	}

	// Kaustinen note: we're currently experimenting with stop chargin gas for the origin address
	// so simple transfer still take 21000 gas. This is to potentially avoid breaking existing tooling.
	// This is the reason why we return 0 instead of `gas`.
	// Note that we still have to touch the addresses to make sure the witness is correct.
	return 0
}

func (aw *AccessWitness) TouchAddressOnReadAndComputeGas(addr []byte, treeIndex uint256.Int, subIndex byte) uint64 {
	return aw.touchAddressAndChargeGas(addr, treeIndex, subIndex, AccessListRead)
}

func (aw *AccessWitness) touchAddressAndChargeGas(addr []byte, treeIndex uint256.Int, subIndex byte, isWrite ALAccessMode) uint64 {
	stemRead, suffixRead, stemWrite, selectorWrite, selectorFill := aw.touchAddress(addr, treeIndex, subIndex, isWrite)

	var gas uint64
	if stemRead {
		gas += params.WitnessBranchReadCost
	}
	if suffixRead {
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

// touchAddress adds any missing access event to the witness.
func (aw *AccessWitness) touchAddress(addr []byte, treeIndex uint256.Int, subIndex byte, isWrite ALAccessMode) (bool, bool, bool, bool, bool) {
	branchKey := newBranchAccessKey(addr, treeIndex)
	chunkKey := newChunkAccessKey(branchKey, subIndex)

	// Read access.
	var branchRead, chunkRead bool
	if _, hasStem := aw.branches[branchKey]; !hasStem {
		branchRead = true
		aw.branches[branchKey] = AccessWitnessReadFlag
	}
	if _, hasSelector := aw.chunks[chunkKey]; !hasSelector {
		chunkRead = true
		aw.chunks[chunkKey] = AccessWitnessReadFlag
	}

	// Write access.
	var branchWrite, chunkWrite, chunkFill bool
	if isWrite {
		if (aw.branches[branchKey] & AccessWitnessWriteFlag) == 0 {
			branchWrite = true
			aw.branches[branchKey] |= AccessWitnessWriteFlag
		}

		chunkValue := aw.chunks[chunkKey]
		if (chunkValue & AccessWitnessWriteFlag) == 0 {
			chunkWrite = true
			aw.chunks[chunkKey] |= AccessWitnessWriteFlag
		}

		// TODO: charge chunk filling costs if the leaf was previously empty in the state
	}

	return branchRead, chunkRead, branchWrite, chunkWrite, chunkFill
}

type branchAccessKey struct {
	addr      common.Address
	treeIndex uint256.Int
}

func newBranchAccessKey(addr []byte, treeIndex uint256.Int) branchAccessKey {
	var sk branchAccessKey
	copy(sk.addr[20-len(addr):], addr)
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

func (aw *AccessWitness) AddAddress(addr common.Address, items ALAccountItem, isWrite ALAccessMode) uint64 {
	var gas uint64
	for index := utils.VersionLeafKey; index <= utils.CodeSizeLeafKey; index++ {
		if (1<<index)&items != 0 {
			gas += aw.touchAddressAndChargeGas(addr[:], zeroTreeIndex, byte(index), isWrite)
		}
	}
	return gas
}

func (aw *AccessWitness) AddSlot(address common.Address, slot common.Hash, isWrite ALAccessMode) uint64 {
	treeIndex, suffix := utils.GetTreeKeyStorageSlotTreeIndexes(slot.Bytes())
	return aw.touchAddressAndChargeGas(address[:], *treeIndex, suffix, isWrite)
}

func (aw *AccessWitness) DeleteSlot(address common.Address, slot common.Hash) {
	// Should remain in the witness
}

func (aw *AccessWitness) DeleteAddress(address common.Address) {
	// Should remain in the witness
}

func (aw *AccessWitness) ContainsAddress(address common.Address) bool {
	panic("not implemented") // TODO: Implement
}

func (aw *AccessWitness) Contains(address common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	panic("not implemented") // TODO: Implement
}
