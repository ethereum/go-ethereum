// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package convertedv1bindtests

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

// TODO: convert this type to value type after everything works.
// InputCheckerMetaData contains all meta data concerning the InputChecker contract.
var InputCheckerMetaData = &bind.MetaData{
	ABI:     "[{\"type\":\"function\",\"name\":\"noInput\",\"constant\":true,\"inputs\":[],\"outputs\":[]},{\"type\":\"function\",\"name\":\"namedInput\",\"constant\":true,\"inputs\":[{\"name\":\"str\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"anonInput\",\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"namedInputs\",\"constant\":true,\"inputs\":[{\"name\":\"str1\",\"type\":\"string\"},{\"name\":\"str2\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"anonInputs\",\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"\",\"type\":\"string\"}],\"outputs\":[]},{\"type\":\"function\",\"name\":\"mixedInputs\",\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"str\",\"type\":\"string\"}],\"outputs\":[]}]",
	Pattern: "e551ce092312e54f54f45ffdf06caa4cdc",
}

// InputChecker is an auto generated Go binding around an Ethereum contract.
type InputChecker struct {
	abi abi.ABI
}

// NewInputChecker creates a new instance of InputChecker.
func NewInputChecker() (*InputChecker, error) {
	parsed, err := InputCheckerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &InputChecker{abi: *parsed}, nil
}

func (_InputChecker *InputChecker) PackConstructor() []byte {
	res, _ := _InputChecker.abi.Pack("")
	return res
}

// AnonInput is a free data retrieval call binding the contract method 0x3e708e82.
//
// Solidity: function anonInput(string ) returns()
func (_InputChecker *InputChecker) PackAnonInput(arg0 string) ([]byte, error) {
	return _InputChecker.abi.Pack("anonInput", arg0)
}

// AnonInputs is a free data retrieval call binding the contract method 0x28160527.
//
// Solidity: function anonInputs(string , string ) returns()
func (_InputChecker *InputChecker) PackAnonInputs(arg0 string, arg1 string) ([]byte, error) {
	return _InputChecker.abi.Pack("anonInputs", arg0, arg1)
}

// MixedInputs is a free data retrieval call binding the contract method 0xc689ebdc.
//
// Solidity: function mixedInputs(string , string str) returns()
func (_InputChecker *InputChecker) PackMixedInputs(arg0 string, str string) ([]byte, error) {
	return _InputChecker.abi.Pack("mixedInputs", arg0, str)
}

// NamedInput is a free data retrieval call binding the contract method 0x0d402005.
//
// Solidity: function namedInput(string str) returns()
func (_InputChecker *InputChecker) PackNamedInput(str string) ([]byte, error) {
	return _InputChecker.abi.Pack("namedInput", str)
}

// NamedInputs is a free data retrieval call binding the contract method 0x63c796ed.
//
// Solidity: function namedInputs(string str1, string str2) returns()
func (_InputChecker *InputChecker) PackNamedInputs(str1 string, str2 string) ([]byte, error) {
	return _InputChecker.abi.Pack("namedInputs", str1, str2)
}

// NoInput is a free data retrieval call binding the contract method 0x53539029.
//
// Solidity: function noInput() returns()
func (_InputChecker *InputChecker) PackNoInput() ([]byte, error) {
	return _InputChecker.abi.Pack("noInput")
}
