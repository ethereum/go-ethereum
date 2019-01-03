package contracts

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	blockSignerContract "github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	slotBlockSignerMapping = map[string]uint64{
		"blockSigners": 0,
		"blocks":       1,
	}
	ParsedBlockSignerABI, _ = abi.JSON(strings.NewReader(blockSignerContract.BlockSignerABI))
)

func GetSigners(statedb *state.StateDB, parsed abi.ABI, block *types.Block) []common.Address {
	methodName := "getSigners"
	fmt.Printf("---%s---\n", methodName)
	start := time.Now()
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
		fmt.Printf("%v\n", common.HexToAddress(ret.Hex()).Hex())
	}

	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return rets
}
