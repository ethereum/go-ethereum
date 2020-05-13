package vm

import (
  "os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
)

var (
  StateManagerAddress = common.HexToAddress(os.Getenv("STATE_MANAGER_ADDRESS"))
)

type ovmOperation func(*EVM, *Contract, []byte) ([]byte, error)
type methodId [4]byte

var funcs = map[string]ovmOperation{
	"getStorage(address,bytes32)": getStorage,
	"setStorage(address,bytes32,bytes32)":  setStorage,
}
var methodIds map[[4]byte]ovmOperation
var executionMangerBytecode []byte


func init() {
	methodIds = make(map[[4]byte]ovmOperation, len(funcs))
	for methodSignature, f := range funcs {
    methodIds[MethodSignatureToMethodId(methodSignature)] = f
	}
}


func MethodSignatureToMethodId(methodSignature string) [4]byte {
	var methodId [4]byte
	copy(methodId[:], crypto.Keccak256([]byte(methodSignature)))
	return methodId
}


func callStateManager(input []byte, evm *EVM, contract *Contract) (ret []byte, err error) {
	var methodId [4]byte
	copy(methodId[:], input[:4])
	ret, err = methodIds[methodId](evm, contract, input)
	return ret, err
}

func setStorage(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
  key := common.BytesToHash(input[4:36])
	val := common.BytesToHash(input[36:68])
	evm.StateDB.SetState(contract.Address(), key, val)
  return nil, nil
}
func getStorage(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
  key := common.BytesToHash(input[4:36])
	val := evm.StateDB.GetState(contract.Address(), key)
	return val.Bytes(), nil
  return []byte{}, nil
}
