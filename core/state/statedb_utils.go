package state

import (
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"

	"github.com/XinFinOrg/XDPoSChain/crypto"
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
	arrSlot := statedb.GetState(common.HexToAddress(common.BlockSigners), common.BigToHash(keyArrSlot))
	arrLength := arrSlot.Big().Uint64()
	for i := uint64(0); i < arrLength; i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(keyArrSlot), i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.BlockSigners), key)
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
	locSecret := GetLocMappingAtKey(address.Hash(), slot)
	arrLength := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locSecret))
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(locSecret), i, 1)
		keys = append(keys, key)
	}
	rets := [][32]byte{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), key)
		rets = append(rets, ret)
	}
	return rets
}

func GetOpening(statedb *StateDB, address common.Address) [32]byte {
	slot := slotRandomizeMapping["randomOpening"]
	locOpening := GetLocMappingAtKey(address.Hash(), slot)
	ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locOpening))
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
	arrLength := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), slotHash)
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := GetLocDynamicArrAtElement(slotHash, i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}
	return rets
}

func GetCandidateOwner(statedb *StateDB, candidate common.Address) common.Address {
	slot := slotValidatorMapping["validatorsState"]
	// validatorsState[_candidate].owner;
	locValidatorsState := GetLocMappingAtKey(candidate.Hash(), slot)
	locCandidateOwner := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(0)))
	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locCandidateOwner))
	return common.HexToAddress(ret.Hex())
}

func GetCandidateCap(statedb *StateDB, candidate common.Address) *big.Int {
	slot := slotValidatorMapping["validatorsState"]
	// validatorsState[_candidate].cap;
	locValidatorsState := GetLocMappingAtKey(candidate.Hash(), slot)
	locCandidateCap := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(1)))
	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locCandidateCap))
	return ret.Big()
}

func GetVoters(statedb *StateDB, candidate common.Address) []common.Address {
	//mapping(address => address[]) voters;
	slot := slotValidatorMapping["voters"]
	locVoters := GetLocMappingAtKey(candidate.Hash(), slot)
	arrLength := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locVoters))
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(locVoters), i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}

	return rets
}

func GetVoterCap(statedb *StateDB, candidate, voter common.Address) *big.Int {
	slot := slotValidatorMapping["validatorsState"]
	locValidatorsState := GetLocMappingAtKey(candidate.Hash(), slot)
	locCandidateVoters := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(2)))
	retByte := crypto.Keccak256(voter.Hash().Bytes(), common.BigToHash(locCandidateVoters).Bytes())
	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BytesToHash(retByte))
	return ret.Big()
}
