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
// TuplerMetaData contains all meta data concerning the Tupler contract.
var TuplerMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[],\"name\":\"tuple\",\"outputs\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"int256\"},{\"name\":\"c\",\"type\":\"bytes32\"}],\"type\":\"function\"}]",
	Pattern: "a8f4d2061f55c712cfae266c426a1cd568",
	Bin:     "0x606060405260dc8060106000396000f3606060405260e060020a60003504633175aae28114601a575b005b600060605260c0604052600260809081527f486900000000000000000000000000000000000000000000000000000000000060a05260017fc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a47060e0829052610100819052606060c0908152600261012081905281906101409060a09080838184600060046012f1505081517fffff000000000000000000000000000000000000000000000000000000000000169091525050604051610160819003945092505050f3",
}

// Tupler is an auto generated Go binding around an Ethereum contract.
type Tupler struct {
	abi abi.ABI
}

// NewTupler creates a new instance of Tupler.
func NewTupler() (*Tupler, error) {
	parsed, err := TuplerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Tupler{abi: *parsed}, nil
}

func (tupler *Tupler) PackConstructor() []byte {
	res, _ := tupler.abi.Pack("")
	return res
}

// Tuple is a free data retrieval call binding the contract method 0x3175aae2.
//
// Solidity: function tuple() returns(string a, int256 b, bytes32 c)
func (tupler *Tupler) PackTuple() ([]byte, error) {
	return tupler.abi.Pack("tuple")
}

type TupleOutput struct {
	A string
	B *big.Int
	C [32]byte
}

func (tupler *Tupler) UnpackTuple(data []byte) (TupleOutput, error) {
	out, err := tupler.abi.Unpack("tuple", data)

	outstruct := new(TupleOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.A = *abi.ConvertType(out[0], new(string)).(*string)

	outstruct.B = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	outstruct.C = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)

	return *outstruct, err

}