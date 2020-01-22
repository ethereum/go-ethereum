package tests

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
)

func TestOvm(t *testing.T) {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := state.New(common.Hash{}, db)
	address := common.HexToAddress("0x0a")
	code := []byte{
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE8),
		byte(vm.PUSH1), 0x96,
		byte(vm.PUSH1), 1,
		byte(vm.MSTORE8),
		byte(vm.PUSH1), 0x62,
		byte(vm.PUSH1), 2,
		byte(vm.MSTORE8),
		byte(vm.PUSH1), 0x08,
		byte(vm.PUSH1), 3,
		byte(vm.MSTORE8),
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
	}
	state.SetCode(address, code)

	_, _, err := runtime.Call(address, nil, &runtime.Config{State: state, Debug: true})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}
}
