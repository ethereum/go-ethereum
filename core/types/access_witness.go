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

package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
)

type VerkleStem [31]byte

type Mode byte

type ChunkValue struct {
	mode  Mode
	value []byte
}

// AccessWitness lists the locations of the state that are being accessed
// during the production of a block.
// TODO(@gballet) this doesn't fully support deletions
type AccessWitness struct {
	// Branches flags if a given branch has been loaded
	// for the byte value:
	//	the first bit is set if the branch has been edited
	//	the second bit is set if the branch has been read
	Branches map[VerkleStem]Mode

	// Chunks contains the initial value of each address
	Chunks map[common.Hash]ChunkValue
}

func NewAccessWitness() *AccessWitness {
	return &AccessWitness{
		Branches: make(map[VerkleStem]Mode),
		Chunks:   make(map[common.Hash]ChunkValue),
	}
}

const (
	AccessWitnessReadFlag  = Mode(1)
	AccessWitnessWriteFlag = Mode(2)
)

// because of the way Geth's EVM is implemented, the gas cost of an operation
// may be needed before the value of the leaf-key can be retrieved. Hence, we
// break witness access (for the purpose of gas accounting), and filling witness
// values into two methods
func (aw *AccessWitness) SetLeafValue(addr []byte, value []byte) {
	var stem [31]byte
	copy(stem[:], addr[:31])

	if chunk, exists := aw.Chunks[common.BytesToHash(addr)]; exists {
		chunk.value = value
		aw.Chunks[common.BytesToHash(addr)] = chunk
	} else {
		panic(fmt.Sprintf("address not in access witness: %x", addr))
	}
}

func (aw *AccessWitness) touchAddressOnWrite(addr []byte) (bool, bool, bool) {
	var stem VerkleStem
	var stemWrite, chunkWrite, chunkFill bool
	copy(stem[:], addr[:31])

	// NOTE: stem, selector access flags already exist in their
	// respective maps because this function is called at the end of
	// processing a read access event

	if (aw.Branches[stem] & AccessWitnessWriteFlag) == 0 {
		stemWrite = true
		aw.Branches[stem] |= AccessWitnessWriteFlag
	}

	chunkValue := aw.Chunks[common.BytesToHash(addr)]
	// if chunkValue.mode XOR AccessWitnessWriteFlag
	if ((chunkValue.mode & AccessWitnessWriteFlag) == 0) && ((chunkValue.mode | AccessWitnessWriteFlag) != 0) {
		chunkWrite = true
		chunkValue.mode |= AccessWitnessWriteFlag
		aw.Chunks[common.BytesToHash(addr)] = chunkValue
	}

	// TODO charge chunk filling costs if the leaf was previously empty in the state
	/*
		if chunkWrite {
			if _, err := verkleDb.TryGet(addr); err != nil {
				chunkFill = true
			}
		}
	*/

	return stemWrite, chunkWrite, chunkFill
}

// TouchAddress adds any missing addr to the witness and returns respectively
// true if the stem or the stub weren't arleady present.
func (aw *AccessWitness) touchAddress(addr []byte, isWrite bool) (bool, bool, bool, bool, bool) {
	var (
		stem         [31]byte
		stemRead     bool
		selectorRead bool
	)
	copy(stem[:], addr[:31])

	// Check for the presence of the stem
	if _, hasStem := aw.Branches[stem]; !hasStem {
		stemRead = true
		aw.Branches[stem] = AccessWitnessReadFlag
	}

	selectorRead = true

	// Check for the presence of the leaf selector
	if _, hasSelector := aw.Chunks[common.BytesToHash(addr)]; !hasSelector {
		aw.Chunks[common.BytesToHash(addr)] = ChunkValue{
			AccessWitnessReadFlag,
			nil,
		}
	}

	var stemWrite, selectorWrite, chunkFill bool

	if isWrite {
		stemWrite, selectorWrite, chunkFill = aw.touchAddressOnWrite(addr)
	}

	return stemRead, selectorRead, stemWrite, selectorWrite, chunkFill
}

func (aw *AccessWitness) touchAddressAndChargeGas(addr []byte, isWrite bool) uint64 {
	var gas uint64

	stemRead, selectorRead, stemWrite, selectorWrite, selectorFill := aw.touchAddress(addr, isWrite)

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

func (aw *AccessWitness) TouchAddressOnWriteAndComputeGas(addr []byte) uint64 {
	return aw.touchAddressAndChargeGas(addr, true)
}

func (aw *AccessWitness) TouchAddressOnReadAndComputeGas(addr []byte) uint64 {
	return aw.touchAddressAndChargeGas(addr, false)
}

// Merge is used to merge the witness that got generated during the execution
// of a tx, with the accumulation of witnesses that were generated during the
// execution of all the txs preceding this one in a given block.
func (aw *AccessWitness) Merge(other *AccessWitness) {
	for k := range other.Branches {
		if _, ok := aw.Branches[k]; !ok {
			aw.Branches[k] = other.Branches[k]
		}
	}

	for k, chunk := range other.Chunks {
		if _, ok := aw.Chunks[k]; !ok {
			aw.Chunks[k] = chunk
		}
	}
}

// Key returns, predictably, the list of keys that were touched during the
// buildup of the access witness.
func (aw *AccessWitness) Keys() [][]byte {
	keys := make([][]byte, 0, len(aw.Chunks))
	for key := range aw.Chunks {
		var k [32]byte
		copy(k[:], key[:])
		keys = append(keys, k[:])
	}
	return keys
}

func (aw *AccessWitness) KeyVals() map[string][]byte {
	result := make(map[string][]byte)
	for k, v := range aw.Chunks {
		result[string(k[:])] = v.value
	}
	return result
}

func (aw *AccessWitness) Copy() *AccessWitness {
	naw := &AccessWitness{
		Branches: make(map[VerkleStem]Mode),
		Chunks:   make(map[common.Hash]ChunkValue),
	}

	naw.Merge(aw)

	return naw
}

// TODO TouchAndCharge + SetLeafValue* does redundant calls to GetTreeKey*

func (aw *AccessWitness) TouchAndChargeProofOfAbsence(addr []byte) uint64 {
	var gas uint64
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyVersion(addr[:]))
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyBalance(addr[:]))
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeSize(addr[:]))
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeKeccak(addr[:]))
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyNonce(addr[:]))
	return gas
}

func (aw *AccessWitness) TouchAndChargeMessageCall(addr []byte) uint64 {
	var gas uint64
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyVersion(addr[:]))
	gas += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeSize(addr[:]))
	return gas
}

func (aw *AccessWitness) SetLeafValuesMessageCall(addr, codeSize []byte) {
	var data [32]byte
	aw.SetLeafValue(utils.GetTreeKeyVersion(addr[:]), data[:])
	aw.SetLeafValue(utils.GetTreeKeyCodeSize(addr[:]), codeSize[:])
}

func (aw *AccessWitness) TouchAndChargeValueTransfer(callerAddr, targetAddr []byte) uint64 {
	var gas uint64
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(callerAddr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(targetAddr[:]))
	return gas
}

func (aw *AccessWitness) SetLeafValuesValueTransfer(callerAddr, targetAddr, callerBalance, targetBalance []byte) {
	aw.SetLeafValue(utils.GetTreeKeyBalance(callerAddr[:]), callerBalance)
	aw.SetLeafValue(utils.GetTreeKeyBalance(targetAddr[:]), targetBalance)
}

// TouchAndChargeContractCreateInit charges access costs to initiate
// a contract creation
func (aw *AccessWitness) TouchAndChargeContractCreateInit(addr []byte, createSendsValue bool) uint64 {
	var gas uint64
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyVersion(addr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyNonce(addr[:]))
	if createSendsValue {
		gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(addr[:]))
	}
	return gas
}

func (aw *AccessWitness) SetLeafValuesContractCreateInit(addr, nonce, value []byte) {
	var version [32]byte
	aw.SetLeafValue(utils.GetTreeKeyVersion(addr[:]), version[:])
	aw.SetLeafValue(utils.GetTreeKeyNonce(addr[:]), nonce)
	if value != nil {
		aw.SetLeafValue(utils.GetTreeKeyBalance(addr[:]), value)
	}
}

// TouchAndChargeContractCreateCompleted charges access access costs after
// the completion of a contract creation to populate the created account in
// the tree
func (aw *AccessWitness) TouchAndChargeContractCreateCompleted(addr []byte, withValue bool) uint64 {
	var gas uint64
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyVersion(addr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyNonce(addr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(addr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyCodeSize(addr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyCodeKeccak(addr[:]))
	return gas
}

func (aw *AccessWitness) SetLeafValuesContractCreateCompleted(addr, codeSize, codeKeccak []byte) {
	aw.SetLeafValue(utils.GetTreeKeyCodeSize(addr[:]), codeSize)
	aw.SetLeafValue(utils.GetTreeKeyCodeKeccak(addr[:]), codeKeccak)
}

func (aw *AccessWitness) TouchTxOriginAndComputeGas(originAddr []byte, sendsValue bool) uint64 {
	var gasUsed uint64
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyVersion(originAddr[:]))
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeKeccak(originAddr[:]))
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeSize(originAddr[:]))
	gasUsed += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyNonce(originAddr[:]))
	gasUsed += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(originAddr[:]))

	if sendsValue {
		gasUsed += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(originAddr[:]))
	}
	return gasUsed
}

func (aw *AccessWitness) TouchTxExistingAndComputeGas(targetAddr []byte, sendsValue bool) uint64 {
	var gasUsed uint64
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyVersion(targetAddr[:]))
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyBalance(targetAddr[:]))
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyNonce(targetAddr[:]))
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeSize(targetAddr[:]))
	gasUsed += aw.TouchAddressOnReadAndComputeGas(utils.GetTreeKeyCodeKeccak(targetAddr[:]))

	if sendsValue {
		gasUsed += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(targetAddr[:]))
	}
	return gasUsed
}

func (aw *AccessWitness) SetTxOriginTouchedLeaves(originAddr, originBalance, originNonce []byte) {
	var version [32]byte
	aw.SetLeafValue(utils.GetTreeKeyVersion(originAddr[:]), version[:])
	aw.SetLeafValue(utils.GetTreeKeyBalance(originAddr[:]), originBalance)
	aw.SetLeafValue(utils.GetTreeKeyNonce(originAddr[:]), originNonce)
}

func (aw *AccessWitness) SetTxExistingTouchedLeaves(targetAddr, targetBalance, targetNonce, targetCodeSize, targetCodeHash []byte) {
	var version [32]byte
	aw.SetLeafValue(utils.GetTreeKeyVersion(targetAddr[:]), version[:])
	aw.SetLeafValue(utils.GetTreeKeyBalance(targetAddr[:]), targetBalance)
	aw.SetLeafValue(utils.GetTreeKeyNonce(targetAddr[:]), targetNonce)
	aw.SetLeafValue(utils.GetTreeKeyCodeSize(targetAddr[:]), targetCodeSize)
	aw.SetLeafValue(utils.GetTreeKeyCodeKeccak(targetAddr[:]), targetCodeHash)
}
