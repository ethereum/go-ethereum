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
// OutputCheckerMetaData contains all meta data concerning the OutputChecker contract.
var OutputCheckerMetaData = &bind.MetaData{
	ABI:     "[{\"type\":\"function\",\"name\":\"noOutput\",\"constant\":true,\"inputs\":[],\"outputs\":[]},{\"type\":\"function\",\"name\":\"namedOutput\",\"constant\":true,\"inputs\":[],\"outputs\":[{\"name\":\"str\",\"type\":\"string\"}]},{\"type\":\"function\",\"name\":\"anonOutput\",\"constant\":true,\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\"}]},{\"type\":\"function\",\"name\":\"namedOutputs\",\"constant\":true,\"inputs\":[],\"outputs\":[{\"name\":\"str1\",\"type\":\"string\"},{\"name\":\"str2\",\"type\":\"string\"}]},{\"type\":\"function\",\"name\":\"collidingOutputs\",\"constant\":true,\"inputs\":[],\"outputs\":[{\"name\":\"str\",\"type\":\"string\"},{\"name\":\"Str\",\"type\":\"string\"}]},{\"type\":\"function\",\"name\":\"anonOutputs\",\"constant\":true,\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"\",\"type\":\"string\"}]},{\"type\":\"function\",\"name\":\"mixedOutputs\",\"constant\":true,\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"str\",\"type\":\"string\"}]}]",
	Pattern: "cc1d4e235801a590b506d5130b0cca90a1",
}

// OutputChecker is an auto generated Go binding around an Ethereum contract.
type OutputChecker struct {
	abi abi.ABI
}

// NewOutputChecker creates a new instance of OutputChecker.
func NewOutputChecker() (*OutputChecker, error) {
	parsed, err := OutputCheckerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &OutputChecker{abi: *parsed}, nil
}

func (outputChecker *OutputChecker) PackConstructor() []byte {
	res, _ := outputChecker.abi.Pack("")
	return res
}

// AnonOutput is a free data retrieval call binding the contract method 0x008bda05.
//
// Solidity: function anonOutput() returns(string)
func (outputChecker *OutputChecker) PackAnonOutput() ([]byte, error) {
	return outputChecker.abi.Pack("anonOutput")
}

func (outputChecker *OutputChecker) UnpackAnonOutput(data []byte) (string, error) {
	out, err := outputChecker.abi.Unpack("anonOutput", data)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// AnonOutputs is a free data retrieval call binding the contract method 0x3c401115.
//
// Solidity: function anonOutputs() returns(string, string)
func (outputChecker *OutputChecker) PackAnonOutputs() ([]byte, error) {
	return outputChecker.abi.Pack("anonOutputs")
}

type AnonOutputsOutput struct {
	Arg0 string
	Arg1 string
}

func (outputChecker *OutputChecker) UnpackAnonOutputs(data []byte) (AnonOutputsOutput, error) {
	out, err := outputChecker.abi.Unpack("anonOutputs", data)

	outstruct := new(AnonOutputsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg0 = *abi.ConvertType(out[0], new(string)).(*string)

	outstruct.Arg1 = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// CollidingOutputs is a free data retrieval call binding the contract method 0xeccbc1ee.
//
// Solidity: function collidingOutputs() returns(string str, string Str)
func (outputChecker *OutputChecker) PackCollidingOutputs() ([]byte, error) {
	return outputChecker.abi.Pack("collidingOutputs")
}

type CollidingOutputsOutput struct {
	Str  string
	Str0 string
}

func (outputChecker *OutputChecker) UnpackCollidingOutputs(data []byte) (CollidingOutputsOutput, error) {
	out, err := outputChecker.abi.Unpack("collidingOutputs", data)

	outstruct := new(CollidingOutputsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Str = *abi.ConvertType(out[0], new(string)).(*string)

	outstruct.Str0 = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// MixedOutputs is a free data retrieval call binding the contract method 0x21b77b44.
//
// Solidity: function mixedOutputs() returns(string, string str)
func (outputChecker *OutputChecker) PackMixedOutputs() ([]byte, error) {
	return outputChecker.abi.Pack("mixedOutputs")
}

type MixedOutputsOutput struct {
	Arg0 string
	Str  string
}

func (outputChecker *OutputChecker) UnpackMixedOutputs(data []byte) (MixedOutputsOutput, error) {
	out, err := outputChecker.abi.Unpack("mixedOutputs", data)

	outstruct := new(MixedOutputsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg0 = *abi.ConvertType(out[0], new(string)).(*string)

	outstruct.Str = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// NamedOutput is a free data retrieval call binding the contract method 0x5e632bd5.
//
// Solidity: function namedOutput() returns(string str)
func (outputChecker *OutputChecker) PackNamedOutput() ([]byte, error) {
	return outputChecker.abi.Pack("namedOutput")
}

func (outputChecker *OutputChecker) UnpackNamedOutput(data []byte) (string, error) {
	out, err := outputChecker.abi.Unpack("namedOutput", data)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// NamedOutputs is a free data retrieval call binding the contract method 0x7970a189.
//
// Solidity: function namedOutputs() returns(string str1, string str2)
func (outputChecker *OutputChecker) PackNamedOutputs() ([]byte, error) {
	return outputChecker.abi.Pack("namedOutputs")
}

type NamedOutputsOutput struct {
	Str1 string
	Str2 string
}

func (outputChecker *OutputChecker) UnpackNamedOutputs(data []byte) (NamedOutputsOutput, error) {
	out, err := outputChecker.abi.Unpack("namedOutputs", data)

	outstruct := new(NamedOutputsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Str1 = *abi.ConvertType(out[0], new(string)).(*string)

	outstruct.Str2 = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// NoOutput is a free data retrieval call binding the contract method 0x625f0306.
//
// Solidity: function noOutput() returns()
func (outputChecker *OutputChecker) PackNoOutput() ([]byte, error) {
	return outputChecker.abi.Pack("noOutput")
}