// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package solc_errors

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
	ABI:     "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arg1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg3\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"arg4\",\"type\":\"bool\"}],\"name\":\"BadThing\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arg1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg3\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg4\",\"type\":\"uint256\"}],\"name\":\"BadThing2\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Bar\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Foo\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Pattern: "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin:     "0x6080604052348015600e575f80fd5b506101c58061001c5f395ff3fe608060405234801561000f575f80fd5b5060043610610034575f3560e01c8063b0a378b014610038578063bfb4ebcf14610042575b5f80fd5b61004061004c565b005b61004a610092565b005b5f6001600260036040517fd233a24f00000000000000000000000000000000000000000000000000000000815260040161008994939291906100ef565b60405180910390fd5b5f600160025f6040517fbb6a82f10000000000000000000000000000000000000000000000000000000081526004016100ce949392919061014c565b60405180910390fd5b5f819050919050565b6100e9816100d7565b82525050565b5f6080820190506101025f8301876100e0565b61010f60208301866100e0565b61011c60408301856100e0565b61012960608301846100e0565b95945050505050565b5f8115159050919050565b61014681610132565b82525050565b5f60808201905061015f5f8301876100e0565b61016c60208301866100e0565b61017960408301856100e0565b610186606083018461013d565b9594505050505056fea26469706673582212203f89da086f6d7e52e75f82a20ebbf7337f166a6dbae309180c8bb95e1a157e6e64736f6c634300081a0033",
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

// Bar is a free data retrieval call binding the contract method 0xb0a378b0.
//
// Solidity: function Bar() pure returns()
func (_C *C) PackBar() ([]byte, error) {
	return _C.abi.Pack("Bar")
}

// Foo is a free data retrieval call binding the contract method 0xbfb4ebcf.
//
// Solidity: function Foo() pure returns()
func (_C *C) PackFoo() ([]byte, error) {
	return _C.abi.Pack("Foo")
}

func (_C *C) UnpackError(raw []byte) any {

	if val, err := _C.UnpackBadThingError(raw); err != nil {
		return val

	} else if val, err := _C.UnpackBadThing2Error(raw); err != nil {
		return val

	}
	return nil
}

// CBadThing represents a BadThing error raised by the C contract.
type CBadThing struct {
	Arg1 *big.Int
	Arg2 *big.Int
	Arg3 *big.Int
	Arg4 bool
}

func CBadThingErrorID() common.Hash {
	return common.HexToHash("0xbb6a82f123854747ef4381e30e497f934a3854753fec99a69c35c30d4b46714d")
}

func (_C *C) UnpackBadThingError(raw []byte) (*CBadThing, error) {
	errName := "BadThing"
	out := new(CBadThing)
	if err := _C.abi.UnpackIntoInterface(out, errName, raw); err != nil {
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

func CBadThing2ErrorID() common.Hash {
	return common.HexToHash("0xd233a24f02271fe7c9470e060d0fda6447a142bf12ab31fed7ab65affd546175")
}

func (_C *C) UnpackBadThing2Error(raw []byte) (*CBadThing2, error) {
	errName := "BadThing2"
	out := new(CBadThing2)
	if err := _C.abi.UnpackIntoInterface(out, errName, raw); err != nil {
		return nil, err
	}
	return out, nil
}
