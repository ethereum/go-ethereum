package contracts

import (
	"fmt"
	"strings"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	keys := getKeyStorage(statedb, address, parsed, methodName, input...)
	rets := []common.Address{}
	ret := common.Address{}
	for _, key := range keys {
		value := statedb.GetState(address, key)
		method := parsed.Methods[methodName]
		switch method.Outputs[0].Type.T {
		case abi.StringTy:
			//do nothing - output can't be string in this method
			//ret = string(value.Bytes())
		default:
			parsed.Unpack(&ret, methodName, value.Bytes())
			rets = append(rets, ret)
		}
	}
	return rets
}

func getKeyStorage(statedb *state.StateDB, address common.Address, parsed abi.ABI, methodName string, input ...common.Hash) ([]common.Hash) {
	method, ok := parsed.Methods[methodName]
	slot := slotBlockSignerMapping[methodName]
	keys := []common.Hash{}

	// do not support function call
	if ok && len(method.Inputs) <= 1 || len(method.Outputs) == 1 {
		if len(method.Inputs) == 0 {
			keys = append(keys, getKey(slot))
		} else {
			// support first input
			keyArrSlot := mapLocAtKey(input[0], slot)
			arrSlot := statedb.GetState(address, keyArrSlot)
			arrLength := arrSlot.Big().Uint64()
			for i := uint64(0); i < arrLength; i++ {
				valueHash := arrDynamicLocAtElement(keyArrSlot, i, 1)
				keys = append(keys, valueHash)
			}
		}
	}
	return keys
}

func getKey(slot uint64) common.Hash {
	updatedKey := common.BigToHash(new(big.Int).SetUint64(slot))
	return updatedKey
}

func mapLocAtKey(key common.Hash, slot uint64) common.Hash {
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	updatedKey := crypto.Keccak256Hash(key.Bytes(), slotHash.Bytes())
	return updatedKey
}

func arrDynamicLocAtElement(slotHash common.Hash, index uint64, elementSize uint64) common.Hash {
	slotKecBig := crypto.Keccak256Hash(slotHash.Bytes()).Big()
	//arrBig = slotKecBig + index * elementSize
	arrBig := slotKecBig.Add(slotKecBig, new(big.Int).SetUint64(index * elementSize))
	arrHash := common.BigToHash(arrBig)
	return arrHash
}
