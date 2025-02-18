// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindtests

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = bytes.Equal
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// CallbackParamMetaData contains all meta data concerning the CallbackParam contract.
var CallbackParamMetaData = bind.MetaData{
	ABI: "[{\"constant\":false,\"inputs\":[{\"name\":\"callback\",\"type\":\"function\"}],\"name\":\"test\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	ID:  "949f96f86d3c2e1bcc15563ad898beaaca",
	Bin: "0x608060405234801561001057600080fd5b5061015e806100206000396000f3fe60806040526004361061003b576000357c010000000000000000000000000000000000000000000000000000000090048063d7a5aba214610040575b600080fd5b34801561004c57600080fd5b506100be6004803603602081101561006357600080fd5b810190808035806c0100000000000000000000000090049068010000000000000000900463ffffffff1677ffffffffffffffffffffffffffffffffffffffffffffffff169091602001919093929190939291905050506100c0565b005b818160016040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040180828152602001915050600060405180830381600087803b15801561011657600080fd5b505af115801561012a573d6000803e3d6000fd5b50505050505056fea165627a7a7230582062f87455ff84be90896dbb0c4e4ddb505c600d23089f8e80a512548440d7e2580029",
}

// CallbackParam is an auto generated Go binding around an Ethereum contract.
type CallbackParam struct {
	abi abi.ABI
}

// NewCallbackParam creates a new instance of CallbackParam.
func NewCallbackParam() *CallbackParam {
	parsed, err := CallbackParamMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &CallbackParam{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *CallbackParam) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackTest is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd7a5aba2.
//
// Solidity: function test(function callback) returns()
func (callbackParam *CallbackParam) PackTest(callback [24]byte) []byte {
	enc, err := callbackParam.abi.Pack("test", callback)
	if err != nil {
		panic(err)
	}
	return enc
}
