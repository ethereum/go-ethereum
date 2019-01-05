package contracts

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	validatorContract "github.com/ethereum/go-ethereum/contracts/validator/contract"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	ParsedValidatorABI, _ = abi.JSON(strings.NewReader(validatorContract.TomoValidatorABI))
	slotValidatorMapping  = map[string]uint64{
		"withdrawsState":         0,
		"validatorsState":        1,
		"voters":                 2,
		"candidates":             3,
		"candidateCount":         4,
		"minCandidateCap":        5,
		"minVoterCap":            6,
		"maxValidatorNumber":     7,
		"candidateWithdrawDelay": 8,
		"voterWithdrawDelay":     9,
	}
)

func GetCandidates(statedb *state.StateDB, parsed abi.ABI) []common.Address {
	start := time.Now()
	slot := slotValidatorMapping["candidates"]
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	arrLength := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), slotHash)
	fmt.Printf("Candidates length: %v\n", arrLength.Hex())
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := getLocDynamicArrAtElement(slotHash, i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
		fmt.Printf("%v\n", common.HexToAddress(ret.Hex()).Hex())
	}
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return rets
}

func GetCandidateOwner(statedb *state.StateDB, candidate common.Address) common.Address {
	start := time.Now()
	fmt.Printf("--------GetCandidateOwner---------\n")

	slot := slotValidatorMapping["validatorsState"]
	// validatorsState[_candidate].owner;
	locValidatorsState := getLocMappingAtKey(candidate.Hash(), slot)
	locCandidateOwner := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(0)))
	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locCandidateOwner))
	fmt.Printf("ret: %v\n", common.HexToAddress(ret.Hex()).Hex())

	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return common.HexToAddress(ret.Hex())
}

func GetCandidateCap(statedb *state.StateDB, parsed abi.ABI, candidate common.Address) string {
	start := time.Now()

	slot := slotValidatorMapping["validatorsState"]
	// validatorsState[_candidate].cap;
	locValidatorsState := getLocMappingAtKey(candidate.Hash(), slot)
	locCandidateCap := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(2)))
	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locCandidateCap))
	fmt.Printf("ret hex: %v\n", ret.Hex())

	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return ret.Hex()
}

func GetVoters(statedb *state.StateDB, candidate common.Address) []common.Address {
	start := time.Now()
	fmt.Printf("--------GetVoters---------\n")

	//mapping(address => address[]) voters;
	slot := slotValidatorMapping["voters"]
	locVoters := getLocMappingAtKey(candidate.Hash(), slot)
	arrLength := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locVoters))
	fmt.Printf("Voters length: %v\n", arrLength.Hex())
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := getLocDynamicArrAtElement(common.BigToHash(locVoters), i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
		fmt.Printf("%v\n", common.HexToAddress(ret.Hex()).Hex())
	}

	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return rets
}

func GetVoterCap(state *state.StateDB, candidate, voter common.Address) *big.Int {
	//validatorsState[_candidate].voters[_voter]
	start := time.Now()
	fmt.Printf("--------GetVoterCap---------\n")
	slot := slotValidatorMapping["validatorsState"]
	locValidatorsState := getLocMappingAtKey(candidate.Hash(), slot)
	locCandidateVoters := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(3)))
	locVoters := getLocMappingAtKey(voter.Hash(), locCandidateVoters.Uint64())
	ret := state.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locVoters))
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return ret.Big()
}
