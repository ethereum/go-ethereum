// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package uint256arrayreturn

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

// MyContractMetaData contains all meta data concerning the MyContract contract.
var MyContractMetaData = bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"GetNums\",\"outputs\":[{\"internalType\":\"uint256[5]\",\"name\":\"\",\"type\":\"uint256[5]\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "e48e83c9c45b19a47bd451eedc725a6bff",
	Bin: "0x6080604052348015600e575f5ffd5b506101a78061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c8063bd6d10071461002d575b5f5ffd5b61003561004b565b6040516100429190610158565b60405180910390f35b610053610088565b5f6040518060a001604052805f8152602001600181526020016002815260200160038152602001600481525090508091505090565b6040518060a00160405280600590602082028036833780820191505090505090565b5f60059050919050565b5f81905092915050565b5f819050919050565b5f819050919050565b6100d9816100c7565b82525050565b5f6100ea83836100d0565b60208301905092915050565b5f602082019050919050565b61010b816100aa565b61011581846100b4565b9250610120826100be565b805f5b8381101561015057815161013787826100df565b9650610142836100f6565b925050600181019050610123565b505050505050565b5f60a08201905061016b5f830184610102565b9291505056fea2646970667358221220ef76cc678ca215c3e9e5261e3f33ac1cb9901c3186c2af167bfcd8f03b3b864c64736f6c634300081c0033",
}

// MyContract is an auto generated Go binding around an Ethereum contract.
type MyContract struct {
	abi abi.ABI
}

// NewMyContract creates a new instance of MyContract.
func NewMyContract() *MyContract {
	parsed, err := MyContractMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &MyContract{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *MyContract) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackGetNums is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xbd6d1007.
//
// Solidity: function GetNums() pure returns(uint256[5])
func (myContract *MyContract) PackGetNums() []byte {
	enc, err := myContract.abi.Pack("GetNums")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetNums is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xbd6d1007.
//
// Solidity: function GetNums() pure returns(uint256[5])
func (myContract *MyContract) UnpackGetNums(data []byte) ([5]*big.Int, error) {
	out, err := myContract.abi.Unpack("GetNums", data)
	if err != nil {
		return *new([5]*big.Int), err
	}
	out0 := *abi.ConvertType(out[0], new([5]*big.Int)).(*[5]*big.Int)
	return out0, err
}
