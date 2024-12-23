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
// NumericMethodNameMetaData contains all meta data concerning the NumericMethodName contract.
var NumericMethodNameMetaData = &bind.MetaData{
	ABI:     "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_param\",\"type\":\"address\"}],\"name\":\"_1TestEvent\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"_1test\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"__1test\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"__2test\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Pattern: "a691b347afbc44b90dd9a1dfbc65661904",
	Bin:     "0x6080604052348015600f57600080fd5b5060958061001e6000396000f3fe6080604052348015600f57600080fd5b5060043610603c5760003560e01c80639d993132146041578063d02767c7146049578063ffa02795146051575b600080fd5b60476059565b005b604f605b565b005b6057605d565b005b565b565b56fea26469706673582212200382ca602dff96a7e2ba54657985e2b4ac423a56abe4a1f0667bc635c4d4371f64736f6c63430008110033",
}

// NumericMethodName is an auto generated Go binding around an Ethereum contract.
type NumericMethodName struct {
	abi abi.ABI
}

// NewNumericMethodName creates a new instance of NumericMethodName.
func NewNumericMethodName() (*NumericMethodName, error) {
	parsed, err := NumericMethodNameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &NumericMethodName{abi: *parsed}, nil
}

func (_NumericMethodName *NumericMethodName) PackConstructor() []byte {
	res, _ := _NumericMethodName.abi.Pack("")
	return res
}

// E1test is a free data retrieval call binding the contract method 0xffa02795.
//
// Solidity: function _1test() pure returns()
func (_NumericMethodName *NumericMethodName) PackE1test() ([]byte, error) {
	return _NumericMethodName.abi.Pack("_1test")
}

// E1test0 is a free data retrieval call binding the contract method 0xd02767c7.
//
// Solidity: function __1test() pure returns()
func (_NumericMethodName *NumericMethodName) PackE1test0() ([]byte, error) {
	return _NumericMethodName.abi.Pack("__1test")
}

// E2test is a free data retrieval call binding the contract method 0x9d993132.
//
// Solidity: function __2test() pure returns()
func (_NumericMethodName *NumericMethodName) PackE2test() ([]byte, error) {
	return _NumericMethodName.abi.Pack("__2test")
}

// NumericMethodNameE1TestEvent represents a E1TestEvent event raised by the NumericMethodName contract.
type NumericMethodNameE1TestEvent struct {
	Param common.Address
	Raw   *types.Log // Blockchain specific contextual infos
}

const NumericMethodNameE1TestEventEventName = "_1TestEvent"

func (_NumericMethodName *NumericMethodName) UnpackE1TestEventEvent(log *types.Log) (*NumericMethodNameE1TestEvent, error) {
	event := "_1TestEvent"
	if log.Topics[0] != _NumericMethodName.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(NumericMethodNameE1TestEvent)
	if len(log.Data) > 0 {
		if err := _NumericMethodName.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _NumericMethodName.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}
