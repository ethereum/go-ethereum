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

// EmptyMetaData contains all meta data concerning the Empty contract.
var EmptyMetaData = bind.MetaData{
	ABI:     "[]",
	Pattern: "c4ce3210982aa6fc94dabe46dc1dbf454d",
	Bin:     "0x606060405260068060106000396000f3606060405200",
}

// Empty is an auto generated Go binding around an Ethereum contract.
type Empty struct {
	abi abi.ABI
}

// NewEmpty creates a new instance of Empty.
func NewEmpty() (*Empty, error) {
	parsed, err := EmptyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Empty{abi: *parsed}, nil
}

func (empty *Empty) PackConstructor() []byte {
	res, _ := empty.abi.Pack("")
	return res
}
