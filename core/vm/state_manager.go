package vm

import (
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	StateManagerAddress = common.HexToAddress(os.Getenv("STATE_MANAGER_ADDRESS"))
)

type stateManagerFunction func(*EVM, *Contract, []byte) ([]byte, error)
type methodId [4]byte

var funcs = map[string]stateManagerFunction{
	"getStorage(address,bytes32)":         getStorage,
	"setStorage(address,bytes32,bytes32)": setStorage,
}
var methodIds map[[4]byte]stateManagerFunction
var executionMangerBytecode []byte

func init() {
	methodIds = make(map[[4]byte]stateManagerFunction, len(funcs))
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
	address := common.BytesToAddress(input[4:36])
	key := common.BytesToHash(input[36:68])
	val := common.BytesToHash(input[68:100])
	evm.StateDB.SetState(address, key, val)
	return nil, nil
}

func getStorage(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	key := common.BytesToHash(input[36:68])
	val := evm.StateDB.GetState(address, key)
	return val.Bytes(), nil
}
