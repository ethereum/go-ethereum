// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package events

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

// CPoint is an auto generated low-level Go binding around an user-defined struct.
type CPoint struct {
	X *big.Int
	Y *big.Int
}

// CMetaData contains all meta data concerning the C contract.
var CMetaData = bind.MetaData{
	ABI:     "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"data\",\"type\":\"uint256\"}],\"name\":\"basic1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"data\",\"type\":\"uint256\"}],\"name\":\"basic2\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DoSomethingWithManyArgs\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y\",\"type\":\"uint256\"}],\"internalType\":\"structC.Point\",\"name\":\"p\",\"type\":\"tuple\"}],\"name\":\"DoSomethingWithPoint\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y\",\"type\":\"uint256\"}],\"internalType\":\"structC.Point\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"EmitMulti\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"EmitOne\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Pattern: "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin:     "0x6080604052348015600e575f80fd5b5061042c8061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061004a575f3560e01c80636fd8b9681461004e578063cb4937491461006f578063e8e49a7114610079578063edcdc89414610083575b5f80fd5b6100566100b3565b6040516100669493929190610244565b60405180910390f35b6100776100c9565b005b61008161017a565b005b61009d600480360381019061009891906102ad565b6101b6565b6040516100aa9190610364565b60405180910390f35b5f805f805f805f80935093509350935090919293565b60017f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd20760026040516100fb919061037d565b60405180910390a260037f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd2076004604051610135919061037d565b60405180910390a25f15157f3b29b9f6d15ba80d866afb3d70b7548ab1ffda3ef6e65f35f1cb05b0e2b29f4e6001604051610170919061037d565b60405180910390a2565b60017f8f17dc823e2f9fcdf730b8182c935574691e811e7d46399fe0ff0087795cd20760026040516101ac919061037d565b60405180910390a2565b366101bf6101fa565b6001835f01356101cf91906103c3565b815f018181525050600183602001356101e891906103c3565b81602001818152505082915050919050565b60405180604001604052805f81526020015f81525090565b5f819050919050565b61022481610212565b82525050565b5f8115159050919050565b61023e8161022a565b82525050565b5f6080820190506102575f83018761021b565b610264602083018661021b565b610271604083018561021b565b61027e6060830184610235565b95945050505050565b5f80fd5b5f80fd5b5f604082840312156102a4576102a361028b565b5b81905092915050565b5f604082840312156102c2576102c1610287565b5b5f6102cf8482850161028f565b91505092915050565b6102e181610212565b81146102eb575f80fd5b50565b5f813590506102fc816102d8565b92915050565b5f61031060208401846102ee565b905092915050565b61032181610212565b82525050565b604082016103375f830183610302565b6103435f850182610318565b506103516020830183610302565b61035e6020850182610318565b50505050565b5f6040820190506103775f830184610327565b92915050565b5f6020820190506103905f83018461021b565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6103cd82610212565b91506103d883610212565b92508282019050808211156103f0576103ef610396565b5b9291505056fea264697066735822122037c4a3caaa4ac1fad7bb712bf2dc85b5d19726dd357808a46ac3b90d2f03dff564736f6c634300081a0033",
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
	return bind.NewContractInstance(backend, addr, c.abi)
}

// DoSomethingWithManyArgs is a free data retrieval call binding the contract method 0x6fd8b968.
//
// Solidity: function DoSomethingWithManyArgs() pure returns(uint256, uint256, uint256, bool)
func (c *C) PackDoSomethingWithManyArgs() []byte {
	enc, err := c.abi.Pack("DoSomethingWithManyArgs")
	if err != nil {
		panic(err)
	}
	return enc
}

type DoSomethingWithManyArgsOutput struct {
	Arg0 *big.Int
	Arg1 *big.Int
	Arg2 *big.Int
	Arg3 bool
}

func (c *C) UnpackDoSomethingWithManyArgs(data []byte) (DoSomethingWithManyArgsOutput, error) {
	out, err := c.abi.Unpack("DoSomethingWithManyArgs", data)

	outstruct := new(DoSomethingWithManyArgsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg0 = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Arg1 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	outstruct.Arg2 = abi.ConvertType(out[2], new(big.Int)).(*big.Int)

	outstruct.Arg3 = *abi.ConvertType(out[3], new(bool)).(*bool)

	return *outstruct, err

}

// DoSomethingWithPoint is a free data retrieval call binding the contract method 0xedcdc894.
//
// Solidity: function DoSomethingWithPoint((uint256,uint256) p) pure returns((uint256,uint256))
func (c *C) PackDoSomethingWithPoint(P CPoint) []byte {
	enc, err := c.abi.Pack("DoSomethingWithPoint", P)
	if err != nil {
		panic(err)
	}
	return enc
}

func (c *C) UnpackDoSomethingWithPoint(data []byte) (CPoint, error) {
	out, err := c.abi.Unpack("DoSomethingWithPoint", data)

	if err != nil {
		return *new(CPoint), err
	}

	out0 := *abi.ConvertType(out[0], new(CPoint)).(*CPoint)

	return out0, err

}

// EmitMulti is a free data retrieval call binding the contract method 0xcb493749.
//
// Solidity: function EmitMulti() returns()
func (c *C) PackEmitMulti() []byte {
	enc, err := c.abi.Pack("EmitMulti")
	if err != nil {
		panic(err)
	}
	return enc
}

// EmitOne is a free data retrieval call binding the contract method 0xe8e49a71.
//
// Solidity: function EmitOne() returns()
func (c *C) PackEmitOne() []byte {
	enc, err := c.abi.Pack("EmitOne")
	if err != nil {
		panic(err)
	}
	return enc
}

// CBasic1 represents a Basic1 event raised by the C contract.
type CBasic1 struct {
	Id   *big.Int
	Data *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const CBasic1EventName = "basic1"

func (CBasic1) ContractEventName() string {
	return CBasic1EventName
}

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

// CBasic2 represents a Basic2 event raised by the C contract.
type CBasic2 struct {
	Flag bool
	Data *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const CBasic2EventName = "basic2"

func (CBasic2) ContractEventName() string {
	return CBasic2EventName
}

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
