// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package return_structs

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

var CLibraryDeps = []*bind.MetaData{}

// TODO: convert this type to value type after everything works.
// CMetaData contains all meta data concerning the C contract.
var CMetaData = &bind.MetaData{
	ABI:     "[{\"inputs\":[],\"name\":\"DoSomethingWithManyArgs\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Pattern: "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin:     "0x6080604052348015600e575f80fd5b5060fc8061001b5f395ff3fe6080604052348015600e575f80fd5b50600436106026575f3560e01c80636fd8b96814602a575b5f80fd5b60306047565b604051603e9493929190608b565b60405180910390f35b5f805f805f805f80935093509350935090919293565b5f819050919050565b606d81605d565b82525050565b5f8115159050919050565b6085816073565b82525050565b5f608082019050609c5f8301876066565b60a760208301866066565b60b260408301856066565b60bd6060830184607e565b9594505050505056fea2646970667358221220ca49ad4f133bcee385e1656fc277c9fd6546763a73df384f7d5d0fdd3c4808a564736f6c634300081a0033",
}

// C is an auto generated Go binding around an Ethereum contract.
type C struct {
	abi abi.ABI
}

// NewC creates a new instance of C.
func NewC() (*C, error) {
	parsed, err := CMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &C{abi: *parsed}, nil
}

func (_C *C) PackConstructor() ([]byte, error) {
	return _C.abi.Pack("")
}

// DoSomethingWithManyArgs is a free data retrieval call binding the contract method 0x6fd8b968.
//
// Solidity: function DoSomethingWithManyArgs() pure returns(uint256, uint256, uint256, bool)
func (_C *C) PackDoSomethingWithManyArgs() ([]byte, error) {
	return _C.abi.Pack("DoSomethingWithManyArgs")
}

type DoSomethingWithManyArgsOutput struct {
	Arg  *big.Int
	Arg0 *big.Int
	Arg1 *big.Int
	Arg2 bool
}

func (_C *C) UnpackDoSomethingWithManyArgs(data []byte) (DoSomethingWithManyArgsOutput, error) {
	out, err := _C.abi.Unpack("DoSomethingWithManyArgs", data)

	outstruct := new(DoSomethingWithManyArgsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Arg0 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Arg1 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Arg2 = *abi.ConvertType(out[3], new(bool)).(*bool)

	return *outstruct, err

}

