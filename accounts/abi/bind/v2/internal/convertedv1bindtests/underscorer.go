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

// UnderscorerMetaData contains all meta data concerning the Underscorer contract.
var UnderscorerMetaData = bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[],\"name\":\"LowerUpperCollision\",\"outputs\":[{\"name\":\"_res\",\"type\":\"int256\"},{\"name\":\"Res\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"_under_scored_func\",\"outputs\":[{\"name\":\"_int\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"UnderscoredOutput\",\"outputs\":[{\"name\":\"_int\",\"type\":\"int256\"},{\"name\":\"_string\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"PurelyUnderscoredOutput\",\"outputs\":[{\"name\":\"_\",\"type\":\"int256\"},{\"name\":\"res\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"UpperLowerCollision\",\"outputs\":[{\"name\":\"_Res\",\"type\":\"int256\"},{\"name\":\"res\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"AllPurelyUnderscoredOutput\",\"outputs\":[{\"name\":\"_\",\"type\":\"int256\"},{\"name\":\"__\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"UpperUpperCollision\",\"outputs\":[{\"name\":\"_Res\",\"type\":\"int256\"},{\"name\":\"Res\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"LowerLowerCollision\",\"outputs\":[{\"name\":\"_res\",\"type\":\"int256\"},{\"name\":\"res\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Pattern: "5873a90ab43c925dfced86ad53f871f01d",
	Bin:     "0x6060604052341561000f57600080fd5b6103858061001e6000396000f30060606040526004361061008e576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806303a592131461009357806346546dbe146100c357806367e6633d146100ec5780639df4848514610181578063af7486ab146101b1578063b564b34d146101e1578063e02ab24d14610211578063e409ca4514610241575b600080fd5b341561009e57600080fd5b6100a6610271565b604051808381526020018281526020019250505060405180910390f35b34156100ce57600080fd5b6100d6610286565b6040518082815260200191505060405180910390f35b34156100f757600080fd5b6100ff61028e565b6040518083815260200180602001828103825283818151815260200191508051906020019080838360005b8381101561014557808201518184015260208101905061012a565b50505050905090810190601f1680156101725780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b341561018c57600080fd5b6101946102dc565b604051808381526020018281526020019250505060405180910390f35b34156101bc57600080fd5b6101c46102f1565b604051808381526020018281526020019250505060405180910390f35b34156101ec57600080fd5b6101f4610306565b604051808381526020018281526020019250505060405180910390f35b341561021c57600080fd5b61022461031b565b604051808381526020018281526020019250505060405180910390f35b341561024c57600080fd5b610254610330565b604051808381526020018281526020019250505060405180910390f35b60008060016002819150809050915091509091565b600080905090565b6000610298610345565b61013a8090506040805190810160405280600281526020017f7069000000000000000000000000000000000000000000000000000000000000815250915091509091565b60008060016002819150809050915091509091565b60008060016002819150809050915091509091565b60008060016002819150809050915091509091565b60008060016002819150809050915091509091565b60008060016002819150809050915091509091565b6020604051908101604052806000815250905600a165627a7a72305820d1a53d9de9d1e3d55cb3dc591900b63c4f1ded79114f7b79b332684840e186a40029",
}

// Underscorer is an auto generated Go binding around an Ethereum contract.
type Underscorer struct {
	abi abi.ABI
}

// NewUnderscorer creates a new instance of Underscorer.
func NewUnderscorer() (*Underscorer, error) {
	parsed, err := UnderscorerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Underscorer{abi: *parsed}, nil
}

// AllPurelyUnderscoredOutput is a free data retrieval call binding the contract method 0xb564b34d.
//
// Solidity: function AllPurelyUnderscoredOutput() view returns(int256 _, int256 __)
func (underscorer *Underscorer) PackAllPurelyUnderscoredOutput() ([]byte, error) {
	return underscorer.abi.Pack("AllPurelyUnderscoredOutput")
}

type AllPurelyUnderscoredOutputOutput struct {
	Arg0 *big.Int
	Arg1 *big.Int
}

func (underscorer *Underscorer) UnpackAllPurelyUnderscoredOutput(data []byte) (AllPurelyUnderscoredOutputOutput, error) {
	out, err := underscorer.abi.Unpack("AllPurelyUnderscoredOutput", data)

	outstruct := new(AllPurelyUnderscoredOutputOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg0 = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Arg1 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// LowerLowerCollision is a free data retrieval call binding the contract method 0xe409ca45.
//
// Solidity: function LowerLowerCollision() view returns(int256 _res, int256 res)
func (underscorer *Underscorer) PackLowerLowerCollision() ([]byte, error) {
	return underscorer.abi.Pack("LowerLowerCollision")
}

type LowerLowerCollisionOutput struct {
	Res  *big.Int
	Res0 *big.Int
}

func (underscorer *Underscorer) UnpackLowerLowerCollision(data []byte) (LowerLowerCollisionOutput, error) {
	out, err := underscorer.abi.Unpack("LowerLowerCollision", data)

	outstruct := new(LowerLowerCollisionOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Res = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Res0 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// LowerUpperCollision is a free data retrieval call binding the contract method 0x03a59213.
//
// Solidity: function LowerUpperCollision() view returns(int256 _res, int256 Res)
func (underscorer *Underscorer) PackLowerUpperCollision() ([]byte, error) {
	return underscorer.abi.Pack("LowerUpperCollision")
}

type LowerUpperCollisionOutput struct {
	Res  *big.Int
	Res0 *big.Int
}

func (underscorer *Underscorer) UnpackLowerUpperCollision(data []byte) (LowerUpperCollisionOutput, error) {
	out, err := underscorer.abi.Unpack("LowerUpperCollision", data)

	outstruct := new(LowerUpperCollisionOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Res = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Res0 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// PurelyUnderscoredOutput is a free data retrieval call binding the contract method 0x9df48485.
//
// Solidity: function PurelyUnderscoredOutput() view returns(int256 _, int256 res)
func (underscorer *Underscorer) PackPurelyUnderscoredOutput() ([]byte, error) {
	return underscorer.abi.Pack("PurelyUnderscoredOutput")
}

type PurelyUnderscoredOutputOutput struct {
	Arg0 *big.Int
	Res  *big.Int
}

func (underscorer *Underscorer) UnpackPurelyUnderscoredOutput(data []byte) (PurelyUnderscoredOutputOutput, error) {
	out, err := underscorer.abi.Unpack("PurelyUnderscoredOutput", data)

	outstruct := new(PurelyUnderscoredOutputOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Arg0 = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Res = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// UnderscoredOutput is a free data retrieval call binding the contract method 0x67e6633d.
//
// Solidity: function UnderscoredOutput() view returns(int256 _int, string _string)
func (underscorer *Underscorer) PackUnderscoredOutput() ([]byte, error) {
	return underscorer.abi.Pack("UnderscoredOutput")
}

type UnderscoredOutputOutput struct {
	Int    *big.Int
	String string
}

func (underscorer *Underscorer) UnpackUnderscoredOutput(data []byte) (UnderscoredOutputOutput, error) {
	out, err := underscorer.abi.Unpack("UnderscoredOutput", data)

	outstruct := new(UnderscoredOutputOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Int = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.String = *abi.ConvertType(out[1], new(string)).(*string)

	return *outstruct, err

}

// UpperLowerCollision is a free data retrieval call binding the contract method 0xaf7486ab.
//
// Solidity: function UpperLowerCollision() view returns(int256 _Res, int256 res)
func (underscorer *Underscorer) PackUpperLowerCollision() ([]byte, error) {
	return underscorer.abi.Pack("UpperLowerCollision")
}

type UpperLowerCollisionOutput struct {
	Res  *big.Int
	Res0 *big.Int
}

func (underscorer *Underscorer) UnpackUpperLowerCollision(data []byte) (UpperLowerCollisionOutput, error) {
	out, err := underscorer.abi.Unpack("UpperLowerCollision", data)

	outstruct := new(UpperLowerCollisionOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Res = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Res0 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// UpperUpperCollision is a free data retrieval call binding the contract method 0xe02ab24d.
//
// Solidity: function UpperUpperCollision() view returns(int256 _Res, int256 Res)
func (underscorer *Underscorer) PackUpperUpperCollision() ([]byte, error) {
	return underscorer.abi.Pack("UpperUpperCollision")
}

type UpperUpperCollisionOutput struct {
	Res  *big.Int
	Res0 *big.Int
}

func (underscorer *Underscorer) UnpackUpperUpperCollision(data []byte) (UpperUpperCollisionOutput, error) {
	out, err := underscorer.abi.Unpack("UpperUpperCollision", data)

	outstruct := new(UpperUpperCollisionOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Res = abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	outstruct.Res0 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// UnderScoredFunc is a free data retrieval call binding the contract method 0x46546dbe.
//
// Solidity: function _under_scored_func() view returns(int256 _int)
func (underscorer *Underscorer) PackUnderScoredFunc() ([]byte, error) {
	return underscorer.abi.Pack("_under_scored_func")
}

func (underscorer *Underscorer) UnpackUnderScoredFunc(data []byte) (*big.Int, error) {
	out, err := underscorer.abi.Unpack("_under_scored_func", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}
