package contracts

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	randomizeContract "github.com/ethereum/go-ethereum/contracts/randomize/contract"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	slotRandomizeMapping = map[string]uint64{
		"randomSecret":  0,
		"randomOpening": 1,
	}
	ParsedRandomizeABI, _ = abi.JSON(strings.NewReader(randomizeContract.TomoRandomizeABI))
)

func GetSecret(statedb *state.StateDB, address common.Address) [][32]byte {
	start := time.Now()
	fmt.Printf("--------GetSecret---------\n")

	slot := slotRandomizeMapping["randomSecret"]
	locSecret := getLocMappingAtKey(address.Hash(), slot)
	arrLength := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locSecret))
	fmt.Printf("Secret length: %v\n", arrLength.Hex())
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := getLocDynamicArrAtElement(common.BigToHash(locSecret), i, 1)
		keys = append(keys, key)
	}
	rets := [][32]byte{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), key)
		rets = append(rets, ret)
		fmt.Printf("ret hex: %v - ret byte: %v\n", ret.Hex(), ret.Bytes())
	}
	elapsed := time.Since(start)

	fmt.Printf("Execution time: %s\n", elapsed)
	return rets
}

func GetOpening(statedb *state.StateDB, address common.Address) [32]byte {
	start := time.Now()
	fmt.Printf("--------GetOpening---------\n")

	slot := slotRandomizeMapping["randomOpening"]
	locOpening := getLocMappingAtKey(address.Hash(), slot)
	ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locOpening))
	fmt.Printf("ret hex: %v - ret byte: %v\n", ret.Hex(), ret.Bytes())
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return ret
}
