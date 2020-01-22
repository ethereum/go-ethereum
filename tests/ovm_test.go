package tests

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
)

var KEY = common.FromHex("0102030000000000000000000000000000000000000000000000000000000000")
var VALUE = common.FromHex("0405060000000000000000000000000000000000000000000000000000000000")

func mstoreBytes(bytes []byte, offset int) []byte {
	output := make([]byte, len(bytes)*5)
	for i, b := range bytes {
		output[i*5] = byte(vm.PUSH1)
		output[i*5+1] = b
		output[i*5+2] = byte(vm.PUSH1)
		output[i*5+3] = byte(offset + i)
		output[i*5+4] = byte(vm.MSTORE8)
	}
	return output
}

func call(addr []byte, value uint, inOffset uint, inSize uint, retOffset uint, retSize uint) []byte {
	output := []byte{
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), byte(retSize),
		byte(vm.PUSH1), byte(retOffset),
		byte(vm.PUSH1), byte(inSize),
		byte(vm.PUSH1), byte(inOffset),
		byte(vm.PUSH1), byte(value),
	}
	output = append(output, []byte{
		byte(vm.PUSH20)}...)
	output = append(output, addr...)
	output = append(output, []byte{
		byte(vm.GAS),
		byte(vm.CALL),
	}...)
	return output
}

func TestOvm(t *testing.T) {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := state.New(common.Hash{}, db)
	address := common.HexToAddress("0x0a")
	code := mstoreBytes(vm.OvmSSTOREMethodId, 0)
	code = append(code, mstoreBytes(KEY, 4)...)
	code = append(code, mstoreBytes(VALUE, 36)...)
	code = append(code,
		call(
			vm.OvmContractAddress,
			0,
			0,
			68,
			0,
			0)...)
	code = append(code, mstoreBytes(vm.OvmSLOADMethodId, 0)...)
	code = append(code, mstoreBytes(KEY, 4)...)
	code = append(code,
		call(
			vm.OvmContractAddress,
			0,
			0,
			36,
			0,
			0)...)
	code = append(code, []byte{
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}...)

	state.SetCode(address, code)

	returnValue, _, err := runtime.Call(address, nil, &runtime.Config{State: state, Debug: true})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	if !bytes.Equal(returnValue, VALUE) {
		t.Errorf("Expected %020x; got %020x", returnValue, VALUE)
	}

}
