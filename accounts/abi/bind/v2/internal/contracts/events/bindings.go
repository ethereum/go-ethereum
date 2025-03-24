// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package events

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

// CMetaData contains all meta data concerning the C contract.
var CMetaData = bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"data\",\"type\":\"uint256\"}],\"name\":\"basic1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"data\",\"type\":\"uint256\"}],\"name\":\"basic2\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"EmitMulti\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"EmitOne\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	ID:  "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin: "0x6080604052348015600e575f5ffd5b506101a08061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063cb49374914610038578063e8e49a7114610042575b5f5ffd5b61004061004c565b005b61004a6100fd565b005b60017f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd207600260405161007e9190610151565b60405180910390a260037f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd20760046040516100b89190610151565b60405180910390a25f15157f3b29b9f6d15ba80d866afb3d70b7548ab1ffda3ef6e65f35f1cb05b0e2b29f4e60016040516100f39190610151565b60405180910390a2565b60017f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd207600260405161012f9190610151565b60405180910390a2565b5f819050919050565b61014b81610139565b82525050565b5f6020820190506101645f830184610142565b9291505056fea26469706673582212207331c79de16a73a1639c4c4b3489ea78a3ed35fe62a178824f586df12672ac0564736f6c634300081c0033",
}

// C is an auto generated Go binding around an Ethereum contract.
type C struct {
	abi abi.ABI
}

// NewC creates a new instance of C.
func NewC() *C {
	parsed, err := CMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &C{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *C) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackEmitMulti is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xcb493749.
//
// Solidity: function EmitMulti() returns()
func (c *C) PackEmitMulti() []byte {
	enc, err := c.abi.Pack("EmitMulti")
	if err != nil {
		panic(err)
	}
	return enc
}

// PackEmitOne is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe8e49a71.
//
// Solidity: function EmitOne() returns()
func (c *C) PackEmitOne() []byte {
	enc, err := c.abi.Pack("EmitOne")
	if err != nil {
		panic(err)
	}
	return enc
}

// CBasic1 represents a basic1 event raised by the C contract.
type CBasic1 struct {
	Id   *big.Int
	Data *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const CBasic1EventName = "basic1"

// ContractEventName returns the user-defined event name.
func (CBasic1) ContractEventName() string {
	return CBasic1EventName
}

// UnpackBasic1Event is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event basic1(uint256 indexed id, uint256 data)
func (c *C) UnpackBasic1Event(log *types.Log) (*CBasic1, error) {
	event := "basic1"
	if log.Topics[0] != c.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CBasic1)
	if len(log.Data) > 0 {
		if err := c.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.abi.Events[event].Inputs {
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

// CBasic2 represents a basic2 event raised by the C contract.
type CBasic2 struct {
	Flag bool
	Data *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const CBasic2EventName = "basic2"

// ContractEventName returns the user-defined event name.
func (CBasic2) ContractEventName() string {
	return CBasic2EventName
}

// UnpackBasic2Event is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event basic2(bool indexed flag, uint256 data)
func (c *C) UnpackBasic2Event(log *types.Log) (*CBasic2, error) {
	event := "basic2"
	if log.Topics[0] != c.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CBasic2)
	if len(log.Data) > 0 {
		if err := c.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.abi.Events[event].Inputs {
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
