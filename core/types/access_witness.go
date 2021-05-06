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
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
)

type VerkleStem [31]byte

// Mode specifies how a tree location has been accessed
// for the byte value:
//	the first bit is set if the branch has been edited
//	the second bit is set if the branch has been read
type Mode byte

const (
	AccessWitnessReadFlag  = Mode(1)
	AccessWitnessWriteFlag = Mode(2)
)

// AccessWitness lists the locations of the state that are being accessed
// during the production of a block.
type AccessWitness struct {
	// Branches flags if a given branch has been loaded
	Branches map[VerkleStem]Mode

	// Chunks contains the initial value of each address
	Chunks map[common.Hash]Mode

	// InitialValue contains either `nil` if the location
	// didn't exist before it was accessed, or the value
	// that a location had before the execution of this
	// block.
	InitialValue map[string][]byte
}

func NewAccessWitness() *AccessWitness {
	return &AccessWitness{
		Branches:     make(map[VerkleStem]Mode),
		Chunks:       make(map[common.Hash]Mode),
		InitialValue: make(map[string][]byte),
	}
}

func (aw *AccessWitness) SetLeafValue(addr []byte, value []byte) {
	var stem [31]byte
	copy(stem[:], addr[:31])

	// Sanity check: ensure that the location has been declared
	if _, exist := aw.InitialValue[string(addr)]; !exist {
		if len(value) == 32 || len(value) == 0 {
			aw.InitialValue[string(addr)] = value
		} else {
			var aligned [32]byte
			copy(aligned[:len(value)], value)

			aw.InitialValue[string(addr)] = aligned[:]
		}
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
	if ((chunkValue & AccessWitnessWriteFlag) == 0) && ((chunkValue | AccessWitnessWriteFlag) != 0) {
		chunkWrite = true
		chunkValue |= AccessWitnessWriteFlag
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
		stem                                [31]byte
		stemRead, selectorRead              bool
		stemWrite, selectorWrite, chunkFill bool
	)
	copy(stem[:], addr[:31])

	// Check for the presence of the stem
	if _, hasStem := aw.Branches[stem]; !hasStem {
		stemRead = true
		aw.Branches[stem] = AccessWitnessReadFlag
	}

	// Check for the presence of the leaf selector
	if _, hasSelector := aw.Chunks[common.BytesToHash(addr)]; !hasSelector {
		selectorRead = true
		aw.Chunks[common.BytesToHash(addr)] = AccessWitnessReadFlag
	}

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

	for k, v := range other.InitialValue {
		if _, ok := aw.InitialValue[k]; !ok {
			aw.InitialValue[k] = v
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
	for k, v := range aw.InitialValue {
		result[k] = v
	}
	return result
}

func (aw *AccessWitness) Copy() *AccessWitness {
	naw := &AccessWitness{
		Branches:     make(map[VerkleStem]Mode),
		Chunks:       make(map[common.Hash]Mode),
		InitialValue: make(map[string][]byte),
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

// TouchAndChargeContractCreateInit charges access costs to initiate
// a contract creation
func (aw *AccessWitness) TouchAndChargeContractCreateInit(addr []byte, createSendsValue bool) uint64 {
	var gas uint64
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyVersion(addr[:]))
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyNonce(addr[:]))
	if createSendsValue {
		gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyBalance(addr[:]))
	}
	gas += aw.TouchAddressOnWriteAndComputeGas(utils.GetTreeKeyCodeKeccak(addr[:]))
	return gas
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

func (aw *AccessWitness) SetTxOriginTouchedLeaves(originAddr, originBalance, originNonce []byte, codeSize int) {
	var version [32]byte
	aw.SetLeafValue(utils.GetTreeKeyVersion(originAddr[:]), version[:])
	aw.SetLeafValue(utils.GetTreeKeyBalance(originAddr[:]), originBalance)
	aw.SetLeafValue(utils.GetTreeKeyNonce(originAddr[:]), originNonce)
	var cs [32]byte
	binary.LittleEndian.PutUint64(cs[:8], uint64(codeSize))
	aw.SetLeafValue(utils.GetTreeKeyCodeSize(originAddr[:]), cs[:])
}

func (aw *AccessWitness) SetTxExistingTouchedLeaves(targetAddr, targetBalance, targetNonce, targetCodeSize, targetCodeHash []byte) {
	var version [32]byte
	aw.SetLeafValue(utils.GetTreeKeyVersion(targetAddr[:]), version[:])
	aw.SetLeafValue(utils.GetTreeKeyBalance(targetAddr[:]), targetBalance)
	aw.SetLeafValue(utils.GetTreeKeyNonce(targetAddr[:]), targetNonce)
	aw.SetLeafValue(utils.GetTreeKeyCodeSize(targetAddr[:]), targetCodeSize)
	aw.SetLeafValue(utils.GetTreeKeyCodeKeccak(targetAddr[:]), targetCodeHash)
}

func (aw *AccessWitness) SetGetObjectTouchedLeaves(targetAddr, version, targetBalance, targetNonce, targetCodeHash []byte) {
	aw.SetLeafValue(utils.GetTreeKeyVersion(targetAddr[:]), version[:])
	aw.SetLeafValue(utils.GetTreeKeyBalance(targetAddr[:]), targetBalance)
	aw.SetLeafValue(utils.GetTreeKeyNonce(targetAddr[:]), targetNonce)
	aw.SetLeafValue(utils.GetTreeKeyCodeKeccak(targetAddr[:]), targetCodeHash)
}

func (aw *AccessWitness) SetObjectCodeTouchedLeaves(addr, cs, ch []byte) {
	aw.SetLeafValue(utils.GetTreeKeyCodeSize(addr[:]), cs)
	aw.SetLeafValue(utils.GetTreeKeyCodeKeccak(addr[:]), ch)
}
