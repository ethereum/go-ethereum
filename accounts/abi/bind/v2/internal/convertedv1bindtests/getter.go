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
// GetterMetaData contains all meta data concerning the Getter contract.
var GetterMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[],\"name\":\"getter\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"},{\"name\":\"\",\"type\":\"int256\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"type\":\"function\"}]",
	Pattern: "e23a74c8979fe93c9fff15e4f51535ad54",
	Bin:     "0x606060405260dc8060106000396000f3606060405260e060020a6000350463993a04b78114601a575b005b600060605260c0604052600260809081527f486900000000000000000000000000000000000000000000000000000000000060a05260017fc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a47060e0829052610100819052606060c0908152600261012081905281906101409060a09080838184600060046012f1505081517fffff000000000000000000000000000000000000000000000000000000000000169091525050604051610160819003945092505050f3",
}

// Getter is an auto generated Go binding around an Ethereum contract.
type Getter struct {
	abi abi.ABI
}

// NewGetter creates a new instance of Getter.
func NewGetter() (*Getter, error) {
	parsed, err := GetterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Getter{abi: *parsed}, nil
}

func (getter *Getter) PackConstructor() []byte {
	res, _ := getter.abi.Pack("")
	return res
}

// Getter is a free data retrieval call binding the contract method 0x993a04b7.
//
// Solidity: function getter() returns(string, int256, bytes32)
func (getter *Getter) PackGetter() ([]byte, error) {
	return getter.abi.Pack("getter")
}

type GetterOutput struct {
	Arg0 string
	Arg1 *big.Int
	Arg2 [32]byte
}

func (getter *Getter) UnpackGetter(data []byte) (GetterOutput, error) {
	out, err := getter.abi.Unpack("getter", data)

	outstruct := new(GetterOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg0 = *abi.ConvertType(out[0], new(string)).(*string)

	outstruct.Arg1 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	outstruct.Arg2 = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)

	return *outstruct, err

}