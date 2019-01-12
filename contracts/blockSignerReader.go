package contracts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	slotBlockSignerMapping = map[string]uint64{
		"blockSigners": 0,
		"blocks":       1,
	}
)

func GetSigners(statedb *state.StateDB, block *types.Block) []common.Address {
	slot := slotBlockSignerMapping["blockSigners"]
	keys := []common.Hash{}
	keyArrSlot := getLocMappingAtKey(block.Hash(), slot)
	arrSlot := statedb.GetState(common.HexToAddress(common.BlockSigners), common.BigToHash(keyArrSlot))
	arrLength := arrSlot.Big().Uint64()
	for i := uint64(0); i < arrLength; i++ {
		key := getLocDynamicArrAtElement(common.BigToHash(keyArrSlot), i, 1)
		keys = append(keys, key)
	}
	rets := []common.Address{}
	for _, key := range keys {
		ret := statedb.GetState(common.HexToAddress(common.BlockSigners), key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}

	return rets
}
