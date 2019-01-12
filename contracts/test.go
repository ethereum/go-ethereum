package contracts

//
//import (
//	"fmt"
//	"math/big"
//	"time"
//
//	"github.com/ethereum/go-ethereum/common"
//	"github.com/ethereum/go-ethereum/core"
//	"github.com/ethereum/go-ethereum/core/state"
//	"github.com/ethereum/go-ethereum/crypto"
//	"github.com/ethereum/go-ethereum/ethdb"
//	"github.com/ethereum/go-ethereum/core/types"
//)
//
//var (
//	slotValidatorMapping = map[string]uint64{
//		"withdrawsState":         0,
//		"validatorsState":        1,
//		"voters":                 2,
//		"candidates":             3,
//		"candidateCount":         4,
//		"minCandidateCap":        5,
//		"minVoterCap":            6,
//		"maxValidatorNumber":     7,
//		"candidateWithdrawDelay": 8,
//		"voterWithdrawDelay":     9,
//	}
//	slotBlockSignerMapping = map[string]uint64{
//		"blockSigners": 0,
//		"blocks": 1,
//	}
//	slotRandomizeMapping = map[string]uint64{
//		"randomSecret":  0,
//		"randomOpening": 1,
//	}
//	datadir   = "/mnt/sgp1_tuna_chaindata3/data/XDC/chaindata"
//	candidate = "0xd6fa3e7a89bf8c84f0ccd204a15c0d259daf2091"
//)
//
//func main() {
//	//Init
//	chaindb, err := ethdb.NewLDBDatabase(datadir, 0, 0)
//	if err != nil || chaindb == nil {
//		fmt.Printf("Can't get chaindb: %v", err)
//		return
//	}
//	headHash := core.GetHeadBlockHash(chaindb)
//	blockNumber := core.GetBlockNumber(chaindb, headHash)
//	block := core.GetBlock(chaindb, headHash, blockNumber)
//	if block == nil {
//		fmt.Println("Can't get head block")
//		return
//	}
//	database := state.NewDatabase(chaindb)
//	headerRootHash := block.Header().Root
//	headHeaderHash := core.GetHeadHeaderHash(chaindb)
//	statedb, _ := state.New(headHeaderHash, database)
//	if statedb == nil {
//		headHeaderHash = headerRootHash
//	}
//	candidateAddress := common.HexToAddress(candidate)
//	fmt.Printf("Block head :%d, header root:%v\n", blockNumber, headerRootHash.Hex())
//	statedb, _ = state.New(headHeaderHash, database)
//	if statedb == nil {
//		fmt.Println("Can't get state db")
//		return
//	}
//
//	//GetCandidates
//	_ = GetCandidates(statedb)
//
//	//GetCandidateOwner
//	_ = GetCandidateOwner(statedb, candidateAddress)
//
//	//GetCandidateCap
//	_ = GetCandidateCap(statedb, candidateAddress)
//
//	//GetVoters
//	voters := GetVoters(statedb, candidateAddress)
//
//	start := time.Now()
//	fmt.Printf("--------GetVoterCap---------\n")
//	for _, voter := range voters {
//		//GetVoterCap
//		_ = GetVoterCap(statedb, candidateAddress, voter)
//	}
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//
//	//GetSigners
//	blockInput := core.GetBlock(chaindb, common.HexToHash("0x632f2403ea19697082d794900275632eb3373f7a9943b1407461995bbbc2816a"), uint64(1800))
//	_ = GetSigners(statedb, blockInput)
//
//	//GetOpening
//	_ = GetOpening(statedb, candidateAddress)
//	//GetSecret
//	_ = GetSecret(statedb, candidateAddress)
//}
//
//func GetCandidates(statedb *state.StateDB) []common.Address {
//	start := time.Now()
//	fmt.Printf("--------GetCandidates---------\n")
//
//	slot := slotValidatorMapping["candidates"]
//	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
//	arrLength := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), slotHash)
//	fmt.Printf("Candidates length: %v\n", arrLength.Hex())
//	keys := []common.Hash{}
//	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
//		key := getLocDynamicArrAtElement(slotHash, i, 1)
//		keys = append(keys, key)
//	}
//	rets := []common.Address{}
//	for _, key := range keys {
//		ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), key)
//		rets = append(rets, common.HexToAddress(ret.Hex()))
//		fmt.Printf("%v\n", common.HexToAddress(ret.Hex()).Hex())
//	}
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return rets
//}
//
//func GetCandidateOwner(statedb *state.StateDB, candidate common.Address) common.Address {
//	start := time.Now()
//	fmt.Printf("--------GetCandidateOwner---------\n")
//
//	slot := slotValidatorMapping["validatorsState"]
//	// validatorsState[_candidate].owner;
//	locValidatorsState := getLocMappingAtKey(candidate.Hash(), slot)
//	locCandidateOwner := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(0)))
//	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locCandidateOwner))
//	fmt.Printf("ret: %v\n", common.HexToAddress(ret.Hex()).Hex())
//
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return common.HexToAddress(ret.Hex())
//}
//
//func GetCandidateCap(statedb *state.StateDB, candidate common.Address) string {
//	start := time.Now()
//	fmt.Printf("--------GetCandidateCap---------\n")
//
//	slot := slotValidatorMapping["validatorsState"]
//	// validatorsState[_candidate].cap;
//	locValidatorsState := getLocMappingAtKey(candidate.Hash(), slot)
//	locCandidateCap := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(1)))
//	ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locCandidateCap))
//	fmt.Printf("cap: %v\n", ret.Big().String())
//
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return ret.Hex()
//}
//
//func GetVoterCap(state *state.StateDB, candidate, voter common.Address) *big.Int {
//	//validatorsState[_candidate].voters[_voter]
//	slot := slotValidatorMapping["validatorsState"]
//	locValidatorsState := getLocMappingAtKey(candidate.Hash(), slot)
//	locCandidateVoters := locValidatorsState.Add(locValidatorsState, new(big.Int).SetUint64(uint64(2)))
//	retByte := crypto.Keccak256(voter.Hash().Bytes(), common.BigToHash(locCandidateVoters).Bytes())
//	ret := state.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BytesToHash(retByte))
//	fmt.Printf("voter: %v - cap: %v\n", voter.Hex(), ret.Big().String())
//	return ret.Big()
//}
//
//func GetVoters(statedb *state.StateDB, candidate common.Address) []common.Address {
//	start := time.Now()
//	fmt.Printf("--------GetVoters---------\n")
//
//	//mapping(address => address[]) voters;
//	slot := slotValidatorMapping["voters"]
//	locVoters := getLocMappingAtKey(candidate.Hash(), slot)
//	arrLength := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), common.BigToHash(locVoters))
//	fmt.Printf("Voters length: %v\n", arrLength.Hex())
//	keys := []common.Hash{}
//	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
//		key := getLocDynamicArrAtElement(common.BigToHash(locVoters), i, 1)
//		keys = append(keys, key)
//	}
//	rets := []common.Address{}
//	for _, key := range keys {
//		ret := statedb.GetState(common.HexToAddress(common.MasternodeVotingSMC), key)
//		rets = append(rets, common.HexToAddress(ret.Hex()))
//		fmt.Printf("%v\n", common.HexToAddress(ret.Hex()).Hex())
//	}
//
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return rets
//}
//
//func GetSigners(statedb *state.StateDB, block *types.Block) []common.Address {
//	methodName := "getSigners"
//	fmt.Printf("---%s---\n", methodName)
//	start := time.Now()
//	slot := slotBlockSignerMapping["blockSigners"]
//	keys := []common.Hash{}
//	keyArrSlot := getLocMappingAtKey(block.Hash(), slot)
//	arrSlot := statedb.GetState(common.HexToAddress(common.BlockSigners), common.BigToHash(keyArrSlot))
//	arrLength := arrSlot.Big().Uint64()
//	for i := uint64(0); i < arrLength; i++ {
//		key := getLocDynamicArrAtElement(common.BigToHash(keyArrSlot), i, 1)
//		keys = append(keys, key)
//	}
//	rets := []common.Address{}
//	for _, key := range keys {
//		ret := statedb.GetState(common.HexToAddress(common.BlockSigners), key)
//		rets = append(rets, common.HexToAddress(ret.Hex()))
//		fmt.Printf("%v\n", common.HexToAddress(ret.Hex()).Hex())
//	}
//
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return rets
//}
//
//func GetSecret(statedb *state.StateDB, address common.Address) [][32]byte {
//	start := time.Now()
//	fmt.Printf("--------GetSecret---------\n")
//
//	slot := slotRandomizeMapping["randomSecret"]
//	locSecret := getLocMappingAtKey(address.Hash(), slot)
//	arrLength := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locSecret))
//	fmt.Printf("Secret length: %v\n", arrLength.Hex())
//	keys := []common.Hash{}
//	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
//		key := getLocDynamicArrAtElement(common.BigToHash(locSecret), i, 1)
//		keys = append(keys, key)
//	}
//	rets := [][32]byte{}
//	for _, key := range keys {
//		ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), key)
//		rets = append(rets, ret)
//		fmt.Printf("ret hex: %v - ret byte: %v\n", ret.Hex(), ret.Bytes())
//	}
//	elapsed := time.Since(start)
//
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return rets
//}
//
//func GetOpening(statedb *state.StateDB, address common.Address) [32]byte {
//	start := time.Now()
//	fmt.Printf("--------GetOpening---------\n")
//
//	slot := slotRandomizeMapping["randomOpening"]
//	locOpening := getLocMappingAtKey(address.Hash(), slot)
//	ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locOpening))
//	fmt.Printf("ret hex: %v - ret byte: %v\n", ret.Hex(), ret.Bytes())
//	elapsed := time.Since(start)
//	fmt.Printf("Execution time: %s\n", elapsed)
//	return ret
//}
//
//
//////////////////////////////////////
///////////// Common lib /////////////
//////////////////////////////////////
//
//func getLocMappingAtKey(key common.Hash, slot uint64) *big.Int {
//	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
//	retByte := crypto.Keccak256(key.Bytes(), slotHash.Bytes())
//	ret := new(big.Int)
//	ret.SetBytes(retByte)
//	return ret
//}
//
//func getLocDynamicArrAtElement(slotHash common.Hash, index uint64, elementSize uint64) common.Hash {
//	slotKecBig := crypto.Keccak256Hash(slotHash.Bytes()).Big()
//	//arrBig = slotKecBig + index * elementSize
//	arrBig := slotKecBig.Add(slotKecBig, new(big.Int).SetUint64(index*elementSize))
//	return common.BigToHash(arrBig)
//}
//
//func getLocFixedArrAtElement(slot uint64, index uint64, elementSize uint64) common.Hash {
//	slotBig := new(big.Int).SetUint64(slot)
//	arrBig := slotBig.Add(slotBig, new(big.Int).SetUint64(index*elementSize))
//	return common.BigToHash(arrBig)
//}
