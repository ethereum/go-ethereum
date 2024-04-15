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
	"github.com/ethereum/go-ethereum/common/math"
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

func NewAccessWitness(pointCache *utils.PointCache) *AccessWitness {
	return &AccessWitness{
		branches:   make(map[branchAccessKey]mode),
		chunks:     make(map[chunkAccessKey]mode),
		pointCache: pointCache,
	}
}

// Merge is used to merge the witness that got generated during the execution
// of a tx, with the accumulation of witnesses that were generated during the
// execution of all the txs preceding this one in a given block.
func (aw *AccessWitness) Merge(other *AccessWitness) {
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

func (aw *AccessWitness) Copy() *AccessWitness {
	naw := &AccessWitness{
		branches:   make(map[branchAccessKey]mode),
		chunks:     make(map[chunkAccessKey]mode),
		pointCache: aw.pointCache,
	}
	naw.Merge(aw)
	return naw
}

func (aw *AccessWitness) TouchFullAccount(addr []byte, isWrite bool) uint64 {
	var gas uint64
	for i := utils.VersionLeafKey; i <= utils.CodeSizeLeafKey; i++ {
		gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, byte(i), isWrite)
	}
	return gas
}

func (aw *AccessWitness) TouchAndChargeMessageCall(addr []byte) uint64 {
	var gas uint64
	gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.VersionLeafKey, false)
	gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeSizeLeafKey, false)
	return gas
}

func (aw *AccessWitness) TouchAndChargeValueTransfer(callerAddr, targetAddr []byte) uint64 {
	var gas uint64
	gas += aw.touchAddressAndChargeGas(callerAddr, zeroTreeIndex, utils.BalanceLeafKey, true)
	gas += aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BalanceLeafKey, true)
	return gas
}

// TouchAndChargeContractCreateInit charges access costs to initiate
// a contract creation
func (aw *AccessWitness) TouchAndChargeContractCreateInit(addr []byte, createSendsValue bool) uint64 {
	var gas uint64
	gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.VersionLeafKey, true)
	gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.NonceLeafKey, true)
	if createSendsValue {
		gas += aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BalanceLeafKey, true)
	}
	return gas
}

func (aw *AccessWitness) TouchTxOriginAndComputeGas(originAddr []byte) uint64 {
	for i := utils.VersionLeafKey; i <= utils.CodeSizeLeafKey; i++ {
		aw.touchAddressAndChargeGas(originAddr, zeroTreeIndex, byte(i), i == utils.BalanceLeafKey || i == utils.NonceLeafKey)
	}

	// Kaustinen note: we're currently experimenting with stop chargin gas for the origin address
	// so simple transfer still take 21000 gas. This is to potentially avoid breaking existing tooling.
	// This is the reason why we return 0 instead of `gas`.
	// Note that we still have to touch the addresses to make sure the witness is correct.
	return 0
}

func (aw *AccessWitness) TouchTxExistingAndComputeGas(targetAddr []byte, sendsValue bool) uint64 {
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.VersionLeafKey, false)
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.CodeSizeLeafKey, false)
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.CodeHashLeafKey, false)
	aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.NonceLeafKey, false)
	if sendsValue {
		aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BalanceLeafKey, true)
	} else {
		aw.touchAddressAndChargeGas(targetAddr, zeroTreeIndex, utils.BalanceLeafKey, false)
	}

	// Kaustinen note: we're currently experimenting with stop chargin gas for the origin address
	// so simple transfer still take 21000 gas. This is to potentially avoid breaking existing tooling.
	// This is the reason why we return 0 instead of `gas`.
	// Note that we still have to touch the addresses to make sure the witness is correct.
	return 0
}

func (aw *AccessWitness) TouchSlotAndChargeGas(addr []byte, slot common.Hash, isWrite bool) uint64 {
	treeIndex, subIndex := utils.GetTreeKeyStorageSlotTreeIndexes(slot.Bytes())
	return aw.touchAddressAndChargeGas(addr, *treeIndex, subIndex, isWrite)
}

func (aw *AccessWitness) touchAddressAndChargeGas(addr []byte, treeIndex uint256.Int, subIndex byte, isWrite bool) uint64 {
	stemRead, selectorRead, stemWrite, selectorWrite, selectorFill := aw.touchAddress(addr, treeIndex, subIndex, isWrite)

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

// touchAddress adds any missing access event to the witness.
func (aw *AccessWitness) touchAddress(addr []byte, treeIndex uint256.Int, subIndex byte, isWrite bool) (bool, bool, bool, bool, bool) {
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

// touchCodeChunksRangeOnReadAndChargeGas is a helper function to touch every chunk in a code range and charge witness gas costs
func (aw *AccessWitness) TouchCodeChunksRangeAndChargeGas(contractAddr []byte, startPC, size uint64, codeLen uint64, isWrite bool) uint64 {
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
		gas := aw.touchAddressAndChargeGas(contractAddr, treeIndex, subIndex, isWrite)
		var overflow bool
		statelessGasCharged, overflow = math.SafeAdd(statelessGasCharged, gas)
		if overflow {
			panic("overflow when adding gas")
		}
	}

	return statelessGasCharged
}

func (aw *AccessWitness) TouchVersion(addr []byte, isWrite bool) uint64 {
	return aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.VersionLeafKey, isWrite)
}

func (aw *AccessWitness) TouchBalance(addr []byte, isWrite bool) uint64 {
	return aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.BalanceLeafKey, isWrite)
}

func (aw *AccessWitness) TouchNonce(addr []byte, isWrite bool) uint64 {
	return aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.NonceLeafKey, isWrite)
}

func (aw *AccessWitness) TouchCodeSize(addr []byte, isWrite bool) uint64 {
	return aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeSizeLeafKey, isWrite)
}

func (aw *AccessWitness) TouchCodeHash(addr []byte, isWrite bool) uint64 {
	return aw.touchAddressAndChargeGas(addr, zeroTreeIndex, utils.CodeHashLeafKey, isWrite)
}
