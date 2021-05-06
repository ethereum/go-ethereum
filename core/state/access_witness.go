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
)

type VerkleStem [31]byte

// Mode specifies how a tree location has been accessed
// for the byte value:
// * the first bit is set if the branch has been edited
// * the second bit is set if the branch has been read
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

	// Caches which code chunks have been accessed, in order
	// to reduce the number of times that GetTreeKeyCodeChunk
	// is called.
	CodeLocations map[string]map[uint64]struct{}

	statedb *StateDB
}

func NewAccessWitness(statedb *StateDB) *AccessWitness {
	return &AccessWitness{
		Branches:      make(map[VerkleStem]Mode),
		Chunks:        make(map[common.Hash]Mode),
		InitialValue:  make(map[string][]byte),
		CodeLocations: make(map[string]map[uint64]struct{}),
		statedb:       statedb,
	}
}

func (aw *AccessWitness) HasCodeChunk(addr []byte, chunknr uint64) bool {
	if locs, ok := aw.CodeLocations[string(addr)]; ok {
		if _, ok = locs[chunknr]; ok {
			return true
		}
	}

	return false
}

// SetCodeLeafValue does the same thing as SetLeafValue, but for code chunks. It
// maintains a cache of which (address, chunk) were calculated, in order to avoid
// calling GetTreeKey more than once per chunk.
func (aw *AccessWitness) SetCachedCodeChunk(addr []byte, chunknr uint64) {
	if locs, ok := aw.CodeLocations[string(addr)]; ok {
		if _, ok = locs[chunknr]; ok {
			return
		}
	} else {
		aw.CodeLocations[string(addr)] = map[uint64]struct{}{}
	}

	aw.CodeLocations[string(addr)][chunknr] = struct{}{}
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

	// TODO see if merging improves performance
	//for k, v := range other.addrToPoint {
	//if _, ok := aw.addrToPoint[k]; !ok {
	//aw.addrToPoint[k] = v
	//}
	//}
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

func (aw *AccessWitness) GetTreeKeyVersionCached(addr []byte) []byte {
	return aw.statedb.db.(*cachingDB).addrToPoint.GetTreeKeyVersionCached(addr)
}

func (aw *AccessWitness) TouchAndChargeProofOfAbsence(addr []byte) uint64 {
	var (
		balancekey, cskey, ckkey, noncekey [32]byte
		gas                                uint64
	)

	// Only evaluate the polynomial once
	versionkey := aw.GetTreeKeyVersionCached(addr[:])
	copy(balancekey[:], versionkey)
	balancekey[31] = utils.BalanceLeafKey
	copy(noncekey[:], versionkey)
	noncekey[31] = utils.NonceLeafKey
	copy(cskey[:], versionkey)
	cskey[31] = utils.CodeSizeLeafKey
	copy(ckkey[:], versionkey)
	ckkey[31] = utils.CodeKeccakLeafKey

	gas += aw.TouchAddressOnReadAndComputeGas(versionkey)
	gas += aw.TouchAddressOnReadAndComputeGas(balancekey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(cskey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(ckkey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(noncekey[:])
	return gas
}

func (aw *AccessWitness) TouchAndChargeMessageCall(addr []byte) uint64 {
	var (
		gas   uint64
		cskey [32]byte
	)
	// Only evaluate the polynomial once
	versionkey := aw.GetTreeKeyVersionCached(addr[:])
	copy(cskey[:], versionkey)
	cskey[31] = utils.CodeSizeLeafKey
	gas += aw.TouchAddressOnReadAndComputeGas(versionkey)
	gas += aw.TouchAddressOnReadAndComputeGas(cskey[:])
	return gas
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
	var (
		balancekey, ckkey, noncekey [32]byte
		gas                         uint64
	)

	// Only evaluate the polynomial once
	versionkey := aw.GetTreeKeyVersionCached(addr[:])
	copy(balancekey[:], versionkey)
	balancekey[31] = utils.BalanceLeafKey
	copy(noncekey[:], versionkey)
	noncekey[31] = utils.NonceLeafKey
	copy(ckkey[:], versionkey)
	ckkey[31] = utils.CodeKeccakLeafKey

	gas += aw.TouchAddressOnWriteAndComputeGas(versionkey)
	gas += aw.TouchAddressOnWriteAndComputeGas(noncekey[:])
	if createSendsValue {
		gas += aw.TouchAddressOnWriteAndComputeGas(balancekey[:])
	}
	gas += aw.TouchAddressOnWriteAndComputeGas(ckkey[:])
	return gas
}

// TouchAndChargeContractCreateCompleted charges access access costs after
// the completion of a contract creation to populate the created account in
// the tree
func (aw *AccessWitness) TouchAndChargeContractCreateCompleted(addr []byte, withValue bool) uint64 {
	var (
		balancekey, cskey, ckkey, noncekey [32]byte
		gas                                uint64
	)

	// Only evaluate the polynomial once
	versionkey := aw.GetTreeKeyVersionCached(addr[:])
	copy(balancekey[:], versionkey)
	balancekey[31] = utils.BalanceLeafKey
	copy(noncekey[:], versionkey)
	noncekey[31] = utils.NonceLeafKey
	copy(cskey[:], versionkey)
	cskey[31] = utils.CodeSizeLeafKey
	copy(ckkey[:], versionkey)
	ckkey[31] = utils.CodeKeccakLeafKey

	gas += aw.TouchAddressOnWriteAndComputeGas(versionkey)
	gas += aw.TouchAddressOnWriteAndComputeGas(balancekey[:])
	gas += aw.TouchAddressOnWriteAndComputeGas(cskey[:])
	gas += aw.TouchAddressOnWriteAndComputeGas(ckkey[:])
	gas += aw.TouchAddressOnWriteAndComputeGas(noncekey[:])
	return gas
}

func (aw *AccessWitness) TouchTxOriginAndComputeGas(originAddr []byte) uint64 {
	var (
		balancekey, cskey, ckkey, noncekey [32]byte
		gas                                uint64
	)

	// Only evaluate the polynomial once
	versionkey := aw.GetTreeKeyVersionCached(originAddr[:])
	copy(balancekey[:], versionkey)
	balancekey[31] = utils.BalanceLeafKey
	copy(noncekey[:], versionkey)
	noncekey[31] = utils.NonceLeafKey
	copy(cskey[:], versionkey)
	cskey[31] = utils.CodeSizeLeafKey
	copy(ckkey[:], versionkey)
	ckkey[31] = utils.CodeKeccakLeafKey

	gas += aw.TouchAddressOnReadAndComputeGas(versionkey)
	gas += aw.TouchAddressOnReadAndComputeGas(cskey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(ckkey[:])
	gas += aw.TouchAddressOnWriteAndComputeGas(noncekey[:])
	gas += aw.TouchAddressOnWriteAndComputeGas(balancekey[:])

	return gas
}

func (aw *AccessWitness) TouchTxExistingAndComputeGas(targetAddr []byte, sendsValue bool) uint64 {
	var (
		balancekey, cskey, ckkey, noncekey [32]byte
		gas                                uint64
	)

	// Only evaluate the polynomial once
	versionkey := aw.GetTreeKeyVersionCached(targetAddr[:])
	copy(balancekey[:], versionkey)
	balancekey[31] = utils.BalanceLeafKey
	copy(noncekey[:], versionkey)
	noncekey[31] = utils.NonceLeafKey
	copy(cskey[:], versionkey)
	cskey[31] = utils.CodeSizeLeafKey
	copy(ckkey[:], versionkey)
	ckkey[31] = utils.CodeKeccakLeafKey

	gas += aw.TouchAddressOnReadAndComputeGas(versionkey)
	gas += aw.TouchAddressOnReadAndComputeGas(cskey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(ckkey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(noncekey[:])
	gas += aw.TouchAddressOnReadAndComputeGas(balancekey[:])

	if sendsValue {
		gas += aw.TouchAddressOnWriteAndComputeGas(balancekey[:])
	}
	return gas
}
