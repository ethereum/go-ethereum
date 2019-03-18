package contracts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	slotRandomizeMapping = map[string]uint64{
		"randomSecret":  0,
		"randomOpening": 1,
	}
)

func GetSecret(statedb *state.StateDB, address common.Address) [][32]byte {
	slot := slotRandomizeMapping["randomSecret"]
	locSecret := getLocMappingAtKey(address.Hash(), slot)
	arrLength := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locSecret))
	keys := []common.Hash{}
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := getLocDynamicArrAtElement(common.BigToHash(locSecret), i, 1)
		keys = append(keys, key)
	}
	rets := [][32]byte{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), key)
		rets = append(rets, ret)
	}
	return rets
}

func GetOpening(statedb *state.StateDB, address common.Address) [32]byte {
	slot := slotRandomizeMapping["randomOpening"]
	locOpening := getLocMappingAtKey(address.Hash(), slot)
	ret := statedb.GetState(common.HexToAddress(common.RandomizeSMC), common.BigToHash(locOpening))
	return ret
}
