package tests

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
)

func mstoreBytes(bytes []byte) []byte {
	output := make([]byte, len(bytes)*5)
	for i, b := range bytes {
		output[i*5] = byte(vm.PUSH1)
		output[i*5+1] = b
		output[i*5+2] = byte(vm.PUSH1)
		output[i*5+3] = byte(i)
		output[i*5+4] = byte(vm.MSTORE8)
	}
	return output
}

func TestOvm(t *testing.T) {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := state.New(common.Hash{}, db)
	address := common.HexToAddress("0x0a")
	fmt.Printf("%x\n", mstoreBytes(vm.OvmSLOADMethodId))
	code := mstoreBytes(vm.OvmSLOADMethodId)
	code = append(code, []byte{
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 4,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		// 0x00000000000000000001
		byte(vm.PUSH20), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
		byte(vm.GAS),
		byte(vm.CALL),
	}...)

	fmt.Printf("%x", code)
	state.SetCode(address, code)

	_, _, err := runtime.Call(address, nil, &runtime.Config{State: state, Debug: true})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}
}
