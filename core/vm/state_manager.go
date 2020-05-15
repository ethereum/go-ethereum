package vm

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ExecutionManagerAddress = common.HexToAddress("A193E42526F1FEA8C99AF609dcEabf30C1c29fAA")
	StateManagerAddress = common.HexToAddress("b0229ed527b40a36bc00eab1a29390a9bc1417a6")
  WORD_SIZE = 32
)

type stateManagerFunction func(*EVM, *Contract, []byte) ([]byte, error)
type methodId [4]byte

var funcs = map[string]stateManagerFunction{
	"getStorage(address,bytes32)":                getStorage,
	"setStorage(address,bytes32,bytes32)":        setStorage,
	"deployContract(address,bytes,bool,address)": deployContract,
	"getOvmContractNonce(address)":               getOvmContractNonce,
	"getCodeContractBytecode(address)":           getCodeContractBytecode,
	"getCodeContractHash(address)":               getCodeContractHash,
	"getCodeContractAddress(address)":            getCodeContractAddress,
	"associateCodeContract(address,address)":     associateCodeContract,
	"incrementOvmContractNonce(address)":         incrementOvmContractNonce,
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
	if len(input) == 0 {
		return nil, nil
	}
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

func getCodeContractBytecode(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	code := evm.StateDB.GetCode(address)
	return simpleAbiEncode(code), nil
}


func getCodeContractHash(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	codeHash := evm.StateDB.GetCodeHash(address)
	return codeHash.Bytes(), nil
}

func associateCodeContract(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	return []byte{}, nil
}

func getCodeContractAddress(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := input[4:36]
	return address, nil
}

func getOvmContractNonce(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, evm.StateDB.GetNonce(address))
	val := append(make([]byte, 24), b[:]...)
	return val, nil
}

func incrementOvmContractNonce(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	oldNonce := evm.StateDB.GetNonce(address)
	evm.StateDB.SetNonce(address, oldNonce+1)
	return nil, nil
}

func deployContract(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	callerAddress := common.BytesToAddress(input[100:132])
	initCodeLength := binary.BigEndian.Uint32(input[160:164])
	initCode := input[164 : 164+initCodeLength]
	callerContractRef := &Contract{self: AccountRef(callerAddress)}
	_, _, _, err = evm.OvmCreate(callerContractRef, address, initCode, contract.Gas, bigZero)

	return common.BytesToHash(address.Bytes()).Bytes(), nil
}

func simpleAbiEncode(bytes []byte) ([]byte) {
	encodedCode := make([]byte, WORD_SIZE)
	binary.BigEndian.PutUint64(encodedCode[WORD_SIZE-8:], uint64(len(bytes)))
	padding := make([]byte, len(bytes) % WORD_SIZE)
	codeWithLength := append(append(encodedCode, bytes...), padding...)
	offset := make([]byte, WORD_SIZE)
	binary.BigEndian.PutUint64(offset[WORD_SIZE-8:], uint64(2))
	return append([]byte{0, 0}, append(offset, codeWithLength...)...)
}
