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
// SlicerMetaData contains all meta data concerning the Slicer contract.
var SlicerMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[{\"name\":\"input\",\"type\":\"address[]\"}],\"name\":\"echoAddresses\",\"outputs\":[{\"name\":\"output\",\"type\":\"address[]\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"input\",\"type\":\"uint24[23]\"}],\"name\":\"echoFancyInts\",\"outputs\":[{\"name\":\"output\",\"type\":\"uint24[23]\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"input\",\"type\":\"int256[]\"}],\"name\":\"echoInts\",\"outputs\":[{\"name\":\"output\",\"type\":\"int256[]\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"input\",\"type\":\"bool[]\"}],\"name\":\"echoBools\",\"outputs\":[{\"name\":\"output\",\"type\":\"bool[]\"}],\"type\":\"function\"}]",
	Pattern: "082c0740ab6537c7169cb573d097c52112",
	Bin:     "0x606060405261015c806100126000396000f3606060405260e060020a6000350463be1127a3811461003c578063d88becc014610092578063e15a3db71461003c578063f637e5891461003c575b005b604080516020600480358082013583810285810185019096528085526100ee959294602494909392850192829185019084908082843750949650505050505050604080516020810190915260009052805b919050565b604080516102e0818101909252610138916004916102e491839060179083908390808284375090955050505050506102e0604051908101604052806017905b60008152602001906001900390816100d15790505081905061008d565b60405180806020018281038252838181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019250505060405180910390f35b60405180826102e0808381846000600461015cf15090500191505060405180910390f3",
}

// Slicer is an auto generated Go binding around an Ethereum contract.
type Slicer struct {
	abi abi.ABI
}

// NewSlicer creates a new instance of Slicer.
func NewSlicer() (*Slicer, error) {
	parsed, err := SlicerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Slicer{abi: *parsed}, nil
}

func (_Slicer *Slicer) PackConstructor() []byte {
	res, _ := _Slicer.abi.Pack("")
	return res
}

// EchoAddresses is a free data retrieval call binding the contract method 0xbe1127a3.
//
// Solidity: function echoAddresses(address[] input) returns(address[] output)
func (_Slicer *Slicer) PackEchoAddresses(input []common.Address) ([]byte, error) {
	return _Slicer.abi.Pack("echoAddresses", input)
}

func (_Slicer *Slicer) UnpackEchoAddresses(data []byte) ([]common.Address, error) {
	out, err := _Slicer.abi.Unpack("echoAddresses", data)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// EchoBools is a free data retrieval call binding the contract method 0xf637e589.
//
// Solidity: function echoBools(bool[] input) returns(bool[] output)
func (_Slicer *Slicer) PackEchoBools(input []bool) ([]byte, error) {
	return _Slicer.abi.Pack("echoBools", input)
}

func (_Slicer *Slicer) UnpackEchoBools(data []byte) ([]bool, error) {
	out, err := _Slicer.abi.Unpack("echoBools", data)

	if err != nil {
		return *new([]bool), err
	}

	out0 := *abi.ConvertType(out[0], new([]bool)).(*[]bool)

	return out0, err

}

// EchoFancyInts is a free data retrieval call binding the contract method 0xd88becc0.
//
// Solidity: function echoFancyInts(uint24[23] input) returns(uint24[23] output)
func (_Slicer *Slicer) PackEchoFancyInts(input [23]*big.Int) ([]byte, error) {
	return _Slicer.abi.Pack("echoFancyInts", input)
}

func (_Slicer *Slicer) UnpackEchoFancyInts(data []byte) ([23]*big.Int, error) {
	out, err := _Slicer.abi.Unpack("echoFancyInts", data)

	if err != nil {
		return *new([23]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([23]*big.Int)).(*[23]*big.Int)

	return out0, err

}

// EchoInts is a free data retrieval call binding the contract method 0xe15a3db7.
//
// Solidity: function echoInts(int256[] input) returns(int256[] output)
func (_Slicer *Slicer) PackEchoInts(input []*big.Int) ([]byte, error) {
	return _Slicer.abi.Pack("echoInts", input)
}

func (_Slicer *Slicer) UnpackEchoInts(data []byte) ([]*big.Int, error) {
	out, err := _Slicer.abi.Unpack("echoInts", data)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}
