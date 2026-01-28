// Copyright (c) 2018 XDPoSChain
// This file provides utilities for accessing XDPoS state data.

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	slotBlockSignerMapping = map[string]uint64{
		"blockSigners": 0,
		"blocks":       1,
	}
)

func GetSigners(statedb *StateDB, block *types.Block) []common.Address {
	slot := slotBlockSignerMapping["blockSigners"]
	keys := []common.Hash{}
	keyArrSlot := GetLocMappingAtKey(block.Hash(), slot)
	arrSlot := statedb.GetState(common.BlockSignersBinary, common.BigToHash(keyArrSlot))
	arrLength := arrSlot.Big().Uint64()
	for i := uint64(0); i < arrLength; i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(keyArrSlot), i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.BlockSignersBinary, key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}

	return rets
}

var (
	slotRandomizeMapping = map[string]uint64{
		"randomSecret":  0,
		"randomOpening": 1,
	}
)

func GetSecret(statedb *StateDB, address common.Address) [][32]byte {
	slot := slotRandomizeMapping["randomSecret"]
	locSecret := GetLocMappingAtKey(common.BytesToHash(address.Bytes()), slot)
	arrLength := statedb.GetState(common.RandomizeSMCBinary, common.BigToHash(locSecret))
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(locSecret), i, 1)
		keys = append(keys, key)
	}
	rets := [][32]byte{}
	for _, key := range keys {
		ret := statedb.GetState(common.RandomizeSMCBinary, key)
		rets = append(rets, ret)
	}
	return rets
}

func GetOpening(statedb *StateDB, address common.Address) [32]byte {
	slot := slotRandomizeMapping["randomOpening"]
	locOpening := GetLocMappingAtKey(common.BytesToHash(address.Bytes()), slot)
	ret := statedb.GetState(common.RandomizeSMCBinary, common.BigToHash(locOpening))
	return ret
}

// The smart contract and the compiled byte code (in corresponding *.go file) is at commit "KYC Layer added." 7f856ffe672162dfa9c4006c89afb45a24fb7f9f
// Notice that if smart contract and the compiled byte code (in corresponding *.go file) changes, below also changes
var (
	slotValidatorMapping = map[string]uint64{
		"withdrawsState":         0,
		"validatorsState":        1,
		"voters":                 2,
		"KYCString":              3,
		"invalidKYCCount":        4,
		"hasVotedInvalid":        5,
		"ownerToCandidate":       6,
		"owners":                 7,
		"candidates":             8,
		"candidateCount":         9,
		"ownerCount":             10,
		"minCandidateCap":        11,
		"minVoterCap":            12,
		"maxValidatorNumber":     13,
		"candidateWithdrawDelay": 14,
		"voterWithdrawDelay":     15,
	}
)

func GetCandidates(statedb *StateDB) []common.Address {
	slot := slotValidatorMapping["candidates"]
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	arrLength := statedb.GetState(common.MasternodeVotingSMCBinary, slotHash)
	count := arrLength.Big().Uint64()
	rets := make([]common.Address, 0, count)

	emptyHash := common.Hash{}
	for i := uint64(0); i < count; i++ {
		key := GetLocDynamicArrAtElement(slotHash, i, 1)
		ret := statedb.GetState(common.MasternodeVotingSMCBinary, key)
		if ret != emptyHash {
			rets = append(rets, common.HexToAddress(ret.Hex()))
		}
	}

	return rets
}

func GetCandidateOwner(statedb *StateDB, candidate common.Address) common.Address {
	slot := slotValidatorMapping["validatorsState"]
	// validatorsState[_candidate].owner;
	locValidatorsState := GetLocMappingAtKey(common.BytesToHash(candidate.Bytes()), slot)
	locCandidateOwner := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(0)))
	ret := statedb.GetState(common.MasternodeVotingSMCBinary, common.BigToHash(locCandidateOwner))
	return common.HexToAddress(ret.Hex())
}

func GetCandidateCap(statedb *StateDB, candidate common.Address) *big.Int {
	slot := slotValidatorMapping["validatorsState"]
	// validatorsState[_candidate].cap;
	locValidatorsState := GetLocMappingAtKey(common.BytesToHash(candidate.Bytes()), slot)
	locCandidateCap := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(1)))
	ret := statedb.GetState(common.MasternodeVotingSMCBinary, common.BigToHash(locCandidateCap))
	return ret.Big()
}

func GetVoters(statedb *StateDB, candidate common.Address) []common.Address {
	//mapping(address => address[]) voters;
	slot := slotValidatorMapping["voters"]
	locVoters := GetLocMappingAtKey(common.BytesToHash(candidate.Bytes()), slot)
	arrLength := statedb.GetState(common.MasternodeVotingSMCBinary, common.BigToHash(locVoters))
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(locVoters), i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.MasternodeVotingSMCBinary, key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}

	return rets
}

func GetVoterCap(statedb *StateDB, candidate, voter common.Address) *big.Int {
	slot := slotValidatorMapping["validatorsState"]
	locValidatorsState := GetLocMappingAtKey(common.BytesToHash(candidate.Bytes()), slot)
	locCandidateVoters := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(2)))
	retByte := crypto.Keccak256(common.BytesToHash(voter.Bytes()).Bytes(), common.BigToHash(locCandidateVoters).Bytes())
	ret := statedb.GetState(common.MasternodeVotingSMCBinary, common.BytesToHash(retByte))
	return ret.Big()
}

var (
	slotMintedRecordTotalMinted  uint64 = 0
	slotMintedRecordLastEpochNum uint64 = 1
)

func GetTotalMinted(statedb *StateDB) common.Hash {
	hash := GetLocSimpleVariable(slotMintedRecordTotalMinted)
	totalMinted := statedb.GetState(common.MintedRecordAddressBinary, hash)
	return totalMinted
}

func PutTotalMinted(statedb *StateDB, value common.Hash) {
	hash := GetLocSimpleVariable(slotMintedRecordTotalMinted)
	statedb.SetState(common.MintedRecordAddressBinary, hash, value)
}

func GetLastEpochNum(statedb *StateDB) common.Hash {
	hash := GetLocSimpleVariable(slotMintedRecordLastEpochNum)
	totalMinted := statedb.GetState(common.MintedRecordAddressBinary, hash)
	return totalMinted
}

func PutLastEpochNum(statedb *StateDB, value common.Hash) {
	hash := GetLocSimpleVariable(slotMintedRecordLastEpochNum)
	statedb.SetState(common.MintedRecordAddressBinary, hash, value)
}
