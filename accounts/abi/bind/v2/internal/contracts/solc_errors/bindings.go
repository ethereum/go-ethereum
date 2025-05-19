// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package solc_errors

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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arg1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg3\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"arg4\",\"type\":\"bool\"}],\"name\":\"BadThing\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arg1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg3\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg4\",\"type\":\"uint256\"}],\"name\":\"BadThing2\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Bar\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Foo\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin: "0x6080604052348015600e575f5ffd5b506101c58061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063b0a378b014610038578063bfb4ebcf14610042575b5f5ffd5b61004061004c565b005b61004a610092565b005b5f6001600260036040517fd233a24f00000000000000000000000000000000000000000000000000000000815260040161008994939291906100ef565b60405180910390fd5b5f600160025f6040517fbb6a82f10000000000000000000000000000000000000000000000000000000081526004016100ce949392919061014c565b60405180910390fd5b5f819050919050565b6100e9816100d7565b82525050565b5f6080820190506101025f8301876100e0565b61010f60208301866100e0565b61011c60408301856100e0565b61012960608301846100e0565b95945050505050565b5f8115159050919050565b61014681610132565b82525050565b5f60808201905061015f5f8301876100e0565b61016c60208301866100e0565b61017960408301856100e0565b610186606083018461013d565b9594505050505056fea26469706673582212206a82b4c28576e4483a81102558271cfefc891cd63b95440dea521185c1ff6a2a64736f6c634300081c0033",
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

// PackBar is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xb0a378b0.
//
// Solidity: function Bar() pure returns()
func (c *C) PackBar() []byte {
	enc, err := c.abi.Pack("Bar")
	if err != nil {
		panic(err)
	}
	return enc
}

// PackFoo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xbfb4ebcf.
//
// Solidity: function Foo() pure returns()
func (c *C) PackFoo() []byte {
	enc, err := c.abi.Pack("Foo")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackError attempts to decode the provided error data using user-defined
// error definitions.
func (c *C) UnpackError(raw []byte) (any, error) {
	if bytes.Equal(raw[:4], c.abi.Errors["BadThing"].ID.Bytes()[:4]) {
		return c.UnpackBadThingError(raw[4:])
	}
	if bytes.Equal(raw[:4], c.abi.Errors["BadThing2"].ID.Bytes()[:4]) {
		return c.UnpackBadThing2Error(raw[4:])
	}
	return nil, errors.New("Unknown error")
}

// CBadThing represents a BadThing error raised by the C contract.
type CBadThing struct {
	Arg1 *big.Int
	Arg2 *big.Int
	Arg3 *big.Int
	Arg4 bool
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error BadThing(uint256 arg1, uint256 arg2, uint256 arg3, bool arg4)
func CBadThingErrorID() common.Hash {
	return common.HexToHash("0xbb6a82f123854747ef4381e30e497f934a3854753fec99a69c35c30d4b46714d")
}

// UnpackBadThingError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error BadThing(uint256 arg1, uint256 arg2, uint256 arg3, bool arg4)
func (c *C) UnpackBadThingError(raw []byte) (*CBadThing, error) {
	out := new(CBadThing)
	if err := c.abi.UnpackIntoInterface(out, "BadThing", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// CBadThing2 represents a BadThing2 error raised by the C contract.
type CBadThing2 struct {
	Arg1 *big.Int
	Arg2 *big.Int
	Arg3 *big.Int
	Arg4 *big.Int
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error BadThing2(uint256 arg1, uint256 arg2, uint256 arg3, uint256 arg4)
func CBadThing2ErrorID() common.Hash {
	return common.HexToHash("0xd233a24f02271fe7c9470e060d0fda6447a142bf12ab31fed7ab65affd546175")
}

// UnpackBadThing2Error is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error BadThing2(uint256 arg1, uint256 arg2, uint256 arg3, uint256 arg4)
func (c *C) UnpackBadThing2Error(raw []byte) (*CBadThing2, error) {
	out := new(CBadThing2)
	if err := c.abi.UnpackIntoInterface(out, "BadThing2", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// C2MetaData contains all meta data concerning the C2 contract.
var C2MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arg1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg3\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"arg4\",\"type\":\"bool\"}],\"name\":\"BadThing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Foo\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "78ef2840de5b706112ca2dbfa765501a89",
	Bin: "0x6080604052348015600e575f5ffd5b506101148061001c5f395ff3fe6080604052348015600e575f5ffd5b50600436106026575f3560e01c8063bfb4ebcf14602a575b5f5ffd5b60306032565b005b5f600160025f6040517fbb6a82f1000000000000000000000000000000000000000000000000000000008152600401606c949392919060a3565b60405180910390fd5b5f819050919050565b6085816075565b82525050565b5f8115159050919050565b609d81608b565b82525050565b5f60808201905060b45f830187607e565b60bf6020830186607e565b60ca6040830185607e565b60d560608301846096565b9594505050505056fea2646970667358221220e90bf647ffc897060e44b88d54995ed0c03c988fbccaf034375c2ff4e594690764736f6c634300081c0033",
}

// C2 is an auto generated Go binding around an Ethereum contract.
type C2 struct {
	abi abi.ABI
}

// NewC2 creates a new instance of C2.
func NewC2() *C2 {
	parsed, err := C2MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &C2{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *C2) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackFoo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xbfb4ebcf.
//
// Solidity: function Foo() pure returns()
func (c2 *C2) PackFoo() []byte {
	enc, err := c2.abi.Pack("Foo")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackError attempts to decode the provided error data using user-defined
// error definitions.
func (c2 *C2) UnpackError(raw []byte) (any, error) {
	if bytes.Equal(raw[:4], c2.abi.Errors["BadThing"].ID.Bytes()[:4]) {
		return c2.UnpackBadThingError(raw[4:])
	}
	return nil, errors.New("Unknown error")
}

// C2BadThing represents a BadThing error raised by the C2 contract.
type C2BadThing struct {
	Arg1 *big.Int
	Arg2 *big.Int
	Arg3 *big.Int
	Arg4 bool
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error BadThing(uint256 arg1, uint256 arg2, uint256 arg3, bool arg4)
func C2BadThingErrorID() common.Hash {
	return common.HexToHash("0xbb6a82f123854747ef4381e30e497f934a3854753fec99a69c35c30d4b46714d")
}

// UnpackBadThingError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error BadThing(uint256 arg1, uint256 arg2, uint256 arg3, bool arg4)
func (c2 *C2) UnpackBadThingError(raw []byte) (*C2BadThing, error) {
	out := new(C2BadThing)
	if err := c2.abi.UnpackIntoInterface(out, "BadThing", raw); err != nil {
		return nil, err
	}
	return out, nil
}
