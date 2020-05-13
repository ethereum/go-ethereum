package tests

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
)

var KEY = common.FromHex("0102030000000000000000000000000000000000000000000000000000000000")
var VALUE = common.FromHex("0405060000000000000000000000000000000000000000000000000000000000")
var chainConfig params.ChainConfig

func init() {
	chainConfig = params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      new(big.Int),
		ByzantiumBlock:      new(big.Int),
		ConstantinopleBlock: new(big.Int),
		DAOForkBlock:        new(big.Int),
		DAOForkSupport:      false,
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
	}
}

func TestSloadAndStore(t *testing.T) {
  vm.StateManagerAddress = common.HexToAddress("42")
	state := newState()
  setStorageMethodId := vm.MethodSignatureToMethodId("setStorage(address,bytes32,bytes32)")
  storeCode := setStorageMethodId[:]
	storeCode = append(storeCode, KEY...)
	storeCode = append(storeCode, VALUE...)
	getStorageMethodId := vm.MethodSignatureToMethodId("getStorage(address,bytes32)")
  loadCode := getStorageMethodId[:]
	loadCode = append(loadCode, KEY...)

	call(t, state, vm.StateManagerAddress, storeCode)
	codeReturnValue, _ := call(t, state, vm.StateManagerAddress, loadCode)

	if !bytes.Equal(VALUE, codeReturnValue) {
		t.Errorf("Expected %020x; got %020x", VALUE, codeReturnValue)
	}
}

/*
  callCode generates EVM bytecode which makes a single CALL with call data as
  it's input.
*/
func callCode(addr common.Address) []byte {
	output := []byte{}
	output = append(output, []byte{
		byte(vm.CALLDATASIZE),
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.CALLDATACOPY),
	}...)
	output = append(output, pushN(0))
	output = append(output, int64ToBytes(0)...)
	output = append(output, pushN(0))
	output = append(output, int64ToBytes(0)...)
	output = append(output, pushN(0))
	output = append(output, int64ToBytes(0)...)
	output = append(output, pushN(0))
	output = append(output, int64ToBytes(0)...)
	output = append(output, byte(vm.CALLDATASIZE))
	output = append(output, pushN(0))
	output = append(output, int64ToBytes(0)...)
	output = append(output, pushN(0))
	output = append(output, int64ToBytes(0)...)
	output = append(output, []byte{
		byte(vm.PUSH20)}...)
	output = append(output, addr.Bytes()...)
	output = append(output, []byte{
		byte(vm.GAS),
		byte(vm.CALL),
		byte(vm.POP),
		byte(vm.RETURNDATASIZE),
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.RETURNDATACOPY),
		byte(vm.RETURNDATASIZE),
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}...)
	return output
}

func newState() *state.StateDB {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := state.New(common.Hash{}, db, nil)
	return state
}
func call(t *testing.T, state *state.StateDB, address common.Address, callData []byte) ([]byte, error) {
	returnValue, _, err := runtime.Call(address, callData, &runtime.Config{
		State:       state,
		ChainConfig: &chainConfig,
	})

	return returnValue, err
}

func int64ToBytes(n int64) []byte {
	if bytes.Equal(big.NewInt(n).Bytes(), []byte{}) {
		return []byte{0}
	} else {
		return big.NewInt(n).Bytes()
	}
}
func pushN(n int64) byte {
	return byte(int(vm.PUSH1) + byteLength(n) - 1)
}
func byteLength(n int64) int {
	if bytes.Equal(big.NewInt(n).Bytes(), []byte{}) {
		return 1
	} else {
		return len(big.NewInt(n).Bytes())
	}
}

func mockPurityChecker(pure bool) []byte {
	var pureByte byte

	if pure {
		pureByte = 1
	} else {
		pureByte = 0
	}

	return []byte{
		byte(vm.PUSH1),
		pureByte,
		byte(vm.PUSH1),
		0,
		byte(vm.MSTORE8),
		byte(vm.PUSH1),
		1,
		byte(vm.PUSH1),
		0,
		byte(vm.RETURN),
	}
}
