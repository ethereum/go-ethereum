// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package convertedv1bindtests

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// InputCheckerMetaData contains all meta data concerning the InputChecker contract.
var InputCheckerMetaData = bind.MetaData{
	ABI:     "[{\"type\":\"function\",\"name\":\"noInput\",\"constant\":true,\"inputs\":[],\"outputs\":[]},{\"type\":\"function\",\"name\":\"namedInput\",\"constant\":true,\"inputs\":[{\"name\":\"str\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"anonInput\",\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"namedInputs\",\"constant\":true,\"inputs\":[{\"name\":\"str1\",\"type\":\"string\"},{\"name\":\"str2\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"anonInputs\",\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"mixedInputs\",\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"str\",\"type\":\"string\"}],\"outputs\":[]}]",
	Pattern: "e551ce092312e54f54f45ffdf06caa4cdc",
}

// InputChecker is an auto generated Go binding around an Ethereum contract.
type InputChecker struct {
	abi abi.ABI
}

// NewInputChecker creates a new instance of InputChecker.
func NewInputChecker() *InputChecker {
	parsed, err := InputCheckerMetaData.GetAbi()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &InputChecker{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *InputChecker) Instance(backend bind.ContractBackend, addr common.Address) bind.ContractInstance {
	return bind.NewContractInstance(backend, addr, c.abi)
}

// AnonInput is a free data retrieval call binding the contract method 0x3e708e82.
//
// Solidity: function anonInput(string ) returns()
func (inputChecker *InputChecker) PackAnonInput(Arg0 string) ([]byte, error) {
	return inputChecker.abi.Pack("anonInput", Arg0)
}

// AnonInputs is a free data retrieval call binding the contract method 0x28160527.
//
// Solidity: function anonInputs(string , string ) returns()
func (inputChecker *InputChecker) PackAnonInputs(Arg0 string, Arg1 string) ([]byte, error) {
	return inputChecker.abi.Pack("anonInputs", Arg0, Arg1)
}

// MixedInputs is a free data retrieval call binding the contract method 0xc689ebdc.
//
// Solidity: function mixedInputs(string , string str) returns()
func (inputChecker *InputChecker) PackMixedInputs(Arg0 string, Str string) ([]byte, error) {
	return inputChecker.abi.Pack("mixedInputs", Arg0, Str)
}

// NamedInput is a free data retrieval call binding the contract method 0x0d402005.
//
// Solidity: function namedInput(string str) returns()
func (inputChecker *InputChecker) PackNamedInput(Str string) ([]byte, error) {
	return inputChecker.abi.Pack("namedInput", Str)
}

// NamedInputs is a free data retrieval call binding the contract method 0x63c796ed.
//
// Solidity: function namedInputs(string str1, string str2) returns()
func (inputChecker *InputChecker) PackNamedInputs(Str1 string, Str2 string) ([]byte, error) {
	return inputChecker.abi.Pack("namedInputs", Str1, Str2)
}

// NoInput is a free data retrieval call binding the contract method 0x53539029.
//
// Solidity: function noInput() returns()
func (inputChecker *InputChecker) PackNoInput() ([]byte, error) {
	return inputChecker.abi.Pack("noInput")
}
