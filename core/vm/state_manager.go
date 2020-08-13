package vm

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type stateManagerFunction func(*EVM, *Contract, []byte) ([]byte, error)

var funcs = map[string]stateManagerFunction{
	"getStorage(address,bytes32)":                   getStorage,
	"setStorage(address,bytes32,bytes32)":           setStorage,
	"getOvmContractNonce(address)":                  getOvmContractNonce,
	"incrementOvmContractNonce(address)":            incrementOvmContractNonce,
	"getCodeContractBytecode(address)":              getCodeContractBytecode,
	"getCodeContractHash(address)":                  getCodeContractHash,
	"getCodeContractAddressFromOvmAddress(address)": getCodeContractAddress,
	"associateCodeContract(address,address)":        associateCodeContract,
	"registerCreatedContract(address)":              registerCreatedContract,
}
var methodIds map[[4]byte]stateManagerFunction

func init() {
	methodIds = make(map[[4]byte]stateManagerFunction, len(funcs))
	for methodSignature, f := range funcs {
		methodIds[methodSignatureToMethodID(methodSignature)] = f
	}
}

func methodSignatureToMethodID(methodSignature string) [4]byte {
	var methodID [4]byte
	copy(methodID[:], crypto.Keccak256([]byte(methodSignature)))
	return methodID
}

func callStateManager(input []byte, evm *EVM, contract *Contract) (ret []byte, err error) {
	var methodID [4]byte
	if len(input) == 0 {
		return nil, nil
	}
	copy(methodID[:], input[:4])
	ret, err = methodIds[methodID](evm, contract, input)
	return ret, err
}

func setStorage(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	key := common.BytesToHash(input[36:68])
	val := common.BytesToHash(input[68:100])
	log.Debug("[State Mgr] Setting storage.", "Contract address:", hex.EncodeToString(address.Bytes()), "key:", hex.EncodeToString(key.Bytes()), "val:", hex.EncodeToString(val.Bytes()))
	evm.StateDB.SetState(address, key, val)
	return nil, nil
}

func getStorage(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	key := common.BytesToHash(input[36:68])
	val := evm.StateDB.GetState(address, key)
	log.Debug("[State Mgr] Getting storage.", "Contract address:", hex.EncodeToString(address.Bytes()), "key:", hex.EncodeToString(key.Bytes()), "val:", hex.EncodeToString(val.Bytes()))
	return val.Bytes(), nil
}

func getCodeContractBytecode(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	code := evm.StateDB.GetCode(address)
	log.Debug("[State Mgr] Getting Bytecode.", " Contract address:", hex.EncodeToString(address.Bytes()), "Code:", hex.EncodeToString(code))
	return simpleAbiEncode(code), nil
}

func getCodeContractHash(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	codeHash := evm.StateDB.GetCodeHash(address)
	log.Debug("[State Mgr] Getting Code Hash.", " Contract address:", hex.EncodeToString(address.Bytes()), "Code hash:", hex.EncodeToString(codeHash.Bytes()))
	return codeHash.Bytes(), nil
}

func associateCodeContract(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	log.Debug("[State Mgr] Associating code contract")
	return []byte{}, nil
}

func registerCreatedContract(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	log.Debug("[State Mgr] Registering created contract")
	return []byte{}, nil
}

func getCodeContractAddress(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := input[4:36]
	// Ensure 0x0000...deadXXXX is not called as they are banned addresses (the address space used for the OVM contracts)
	bannedAddresses := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 222, 173}
	if bytes.Equal(input[16:34], bannedAddresses) {
		log.Error("[State Mgr] forbidden 0x...DEAD address access!", "Address", hex.EncodeToString(address))
		return nil, errors.New("forbidden 0x...DEAD address access")
	}
	log.Debug("[State Mgr] Getting code contract.", "address:", hex.EncodeToString(address))
	return address, nil
}

func getOvmContractNonce(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, evm.StateDB.GetNonce(address))
	val := append(make([]byte, 24), b[:]...)
	log.Debug("[State Mgr] Getting nonce.", "Contract address:", hex.EncodeToString(address.Bytes()), "Nonce:", evm.StateDB.GetNonce(address))
	return val, nil
}

func incrementOvmContractNonce(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	address := common.BytesToAddress(input[4:36])
	oldNonce := evm.StateDB.GetNonce(address)
	evm.StateDB.SetNonce(address, oldNonce+1)
	log.Debug("[State Mgr] Incrementing nonce.", " Contract address:", hex.EncodeToString(address.Bytes()), "Nonce:", oldNonce+1)
	return nil, nil
}

func simpleAbiEncode(bytes []byte) []byte {
	encodedCode := make([]byte, WORD_SIZE)
	binary.BigEndian.PutUint64(encodedCode[WORD_SIZE-8:], uint64(len(bytes)))
	padding := make([]byte, len(bytes)%WORD_SIZE)
	codeWithLength := append(append(encodedCode, bytes...), padding...)
	offset := make([]byte, WORD_SIZE)
	// Hardcode a 2 because we will only return dynamic bytes with a single element
	binary.BigEndian.PutUint64(offset[WORD_SIZE-8:], uint64(2))
	return append([]byte{0, 0}, append(offset, codeWithLength...)...)
}