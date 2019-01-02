package contracts

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	slotBlockSignerMapping = map[string]uint64{
		"getSigners": 0,
	}
	ParsedBlockSignerABI, _ = abi.JSON(strings.NewReader(contract.BlockSignerABI))
)

///////////////////////////////////////
//////     BlockSigner SMC  ///////////
///////////////////////////////////////
func GetSigners(statedb *state.StateDB, parsed abi.ABI, block *types.Block) ([]common.Address) {
	methodName := "getSigners"
	fmt.Printf("---%s---\n", methodName)
	start := time.Now()
	signers := getSigners(parsed, statedb, common.HexToAddress(common.BlockSigners), methodName, block.Hash())
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	return signers
}

func getSigners(parsed abi.ABI, statedb *state.StateDB, address common.Address, methodName string, input ...common.Hash) ([]common.Address) {
	keys := getKeys(statedb, address, parsed, methodName, input...)
	rets := []common.Address{}
	ret := common.Address{}
	for _, key := range keys {
		value := statedb.GetState(address, key)
		method := parsed.Methods[methodName]
		switch method.Outputs[0].Type.T {
		case abi.AddressTy:
			ret = common.BytesToAddress(value.Bytes())
		default:
			err := parsed.Unpack(&ret, methodName, value.Bytes())
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}
			//ret = common.BytesToAddress(value.Bytes())
		}
		rets = append(rets, ret)
	}
	return rets
}

func getKeys(statedb *state.StateDB, address common.Address, parsed abi.ABI, methodName string, input ...common.Hash) ([]common.Hash) {
	method, ok := parsed.Methods[methodName]
	slot := slotBlockSignerMapping[methodName]
	keys := []common.Hash{}

	// do not support function call
	if ok && len(method.Inputs) <= 1 || len(method.Outputs) == 1 {
		if len(method.Inputs) == 0 {
			keys = append(keys, getLocSimpleVariable(slot))
		} else {
			// support first input
			keyArrSlot := getLocMappingAtKey(input[0], slot)
			arrSlot := statedb.GetState(address, keyArrSlot)
			arrLength := arrSlot.Big().Uint64()
			for i := uint64(0); i < arrLength; i++ {
				valueHash := getLocDynamicArrAtElement(keyArrSlot, i, 1)
				keys = append(keys, valueHash)
			}
		}
	}
	return keys
}
