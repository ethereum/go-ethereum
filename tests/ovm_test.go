package tests

import (
	"bytes"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
)

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
	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	state := newState()

	address := common.HexToAddress("9999999999999999999999999999999999999999")
	key := [32]byte{}
	value := [32]byte{}
	copy(key[:], []byte("hello"))
	copy(value[:], []byte("world"))

	storeCalldata, _ := stateManagerAbi.Pack("setStorage", address, key, value)
	getCalldata, _ := stateManagerAbi.Pack("getStorage", address, key)

	call(t, state, vm.StateManagerAddress, storeCalldata)
	getStorageReturnValue, _ := call(t, state, vm.StateManagerAddress, getCalldata)

	if !bytes.Equal(value[:], getStorageReturnValue) {
		t.Errorf("Expected %020x; got %020x", value[:], getStorageReturnValue)
	}
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
