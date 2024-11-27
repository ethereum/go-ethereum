// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package events

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
	ABI:     "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"data\",\"type\":\"uint256\"}],\"name\":\"basic1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"data\",\"type\":\"uint256\"}],\"name\":\"basic2\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"EmitMulti\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"EmitOne\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Pattern: "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin:     "0x6080604052348015600e575f80fd5b506101a08061001c5f395ff3fe608060405234801561000f575f80fd5b5060043610610034575f3560e01c8063cb49374914610038578063e8e49a7114610042575b5f80fd5b61004061004c565b005b61004a6100fd565b005b60017f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd207600260405161007e9190610151565b60405180910390a260037f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd20760046040516100b89190610151565b60405180910390a25f15157f3b29b9f6d15ba80d866afb3d70b7548ab1ffda3ef6e65f35f1cb05b0e2b29f4e60016040516100f39190610151565b60405180910390a2565b60017f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd207600260405161012f9190610151565b60405180910390a2565b5f819050919050565b61014b81610139565b82525050565b5f6020820190506101645f830184610142565b9291505056fea26469706673582212203624d263fed93ccf2a175b7c701e773f413c64394a51928aa2b1968299798fe664736f6c634300081a0033",
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

// TODO: create custom exported types where unpack would generate a struct return.

// TODO: test constructor with inputs
func (_C *C) PackConstructor() ([]byte, error) {
	return _C.abi.Pack("")
}

// EmitMulti is a free data retrieval call binding the contract method 0xcb493749.
//
// Solidity: function EmitMulti() returns()
func (_C *C) PackEmitMulti() ([]byte, error) {
	return _C.abi.Pack("EmitMulti")
}

// EmitOne is a free data retrieval call binding the contract method 0xe8e49a71.
//
// Solidity: function EmitOne() returns()
func (_C *C) PackEmitOne() ([]byte, error) {
	return _C.abi.Pack("EmitOne")
}

// CBasic1 represents a Basic1 event raised by the C contract.
type CBasic1 struct {
	Id   *big.Int
	Data *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

func CBasic1EventID() common.Hash {
	return common.HexToHash("0x8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd207")
}

func (_C *C) UnpackBasic1Event(log *types.Log) (*CBasic1, error) {
	event := "Basic1"
	if log.Topics[0] != _C.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CBasic1)
	if len(log.Data) > 0 {
		if err := _C.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _C.abi.Events[event].Inputs {
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

// CBasic2 represents a Basic2 event raised by the C contract.
type CBasic2 struct {
	Flag bool
	Data *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

func CBasic2EventID() common.Hash {
	return common.HexToHash("0x3b29b9f6d15ba80d866afb3d70b7548ab1ffda3ef6e65f35f1cb05b0e2b29f4e")
}

func (_C *C) UnpackBasic2Event(log *types.Log) (*CBasic2, error) {
	event := "Basic2"
	if log.Topics[0] != _C.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CBasic2)
	if len(log.Data) > 0 {
		if err := _C.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _C.abi.Events[event].Inputs {
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

