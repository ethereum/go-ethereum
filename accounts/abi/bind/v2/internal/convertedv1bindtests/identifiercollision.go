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
// IdentifierCollisionMetaData contains all meta data concerning the IdentifierCollision contract.
var IdentifierCollisionMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[],\"name\":\"MyVar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"_myVar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Pattern: "1863c5622f8ac2c09c42f063ca883fe438",
	Bin:     "0x60806040523480156100115760006000fd5b50610017565b60c3806100256000396000f3fe608060405234801560105760006000fd5b506004361060365760003560e01c806301ad4d8714603c5780634ef1f0ad146058576036565b60006000fd5b60426074565b6040518082815260200191505060405180910390f35b605e607d565b6040518082815260200191505060405180910390f35b60006000505481565b60006000600050549050608b565b9056fea265627a7a7231582067c8d84688b01c4754ba40a2a871cede94ea1f28b5981593ab2a45b46ac43af664736f6c634300050c0032",
}

// IdentifierCollision is an auto generated Go binding around an Ethereum contract.
type IdentifierCollision struct {
	abi abi.ABI
}

// NewIdentifierCollision creates a new instance of IdentifierCollision.
func NewIdentifierCollision() (*IdentifierCollision, error) {
	parsed, err := IdentifierCollisionMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &IdentifierCollision{abi: *parsed}, nil
}

func (identifierCollision *IdentifierCollision) PackConstructor() []byte {
	res, _ := identifierCollision.abi.Pack("")
	return res
}

// MyVar is a free data retrieval call binding the contract method 0x4ef1f0ad.
//
// Solidity: function MyVar() view returns(uint256)
func (identifierCollision *IdentifierCollision) PackMyVar() ([]byte, error) {
	return identifierCollision.abi.Pack("MyVar")
}

func (identifierCollision *IdentifierCollision) UnpackMyVar(data []byte) (*big.Int, error) {
	out, err := identifierCollision.abi.Unpack("MyVar", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// PubVar is a free data retrieval call binding the contract method 0x01ad4d87.
//
// Solidity: function _myVar() view returns(uint256)
func (identifierCollision *IdentifierCollision) PackPubVar() ([]byte, error) {
	return identifierCollision.abi.Pack("_myVar")
}

func (identifierCollision *IdentifierCollision) UnpackPubVar(data []byte) (*big.Int, error) {
	out, err := identifierCollision.abi.Unpack("_myVar", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}