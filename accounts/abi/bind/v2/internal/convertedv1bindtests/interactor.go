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
// InteractorMetaData contains all meta data concerning the Interactor contract.
var InteractorMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[],\"name\":\"transactString\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"deployString\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"str\",\"type\":\"string\"}],\"name\":\"transact\",\"outputs\":[],\"type\":\"function\"},{\"inputs\":[{\"name\":\"str\",\"type\":\"string\"}],\"type\":\"constructor\"}]",
	Pattern: "f63980878028f3242c9033fdc30fd21a81",
	Bin:     "0x6060604052604051610328380380610328833981016040528051018060006000509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10608d57805160ff19168380011785555b50607c9291505b8082111560ba57838155600101606b565b50505061026a806100be6000396000f35b828001600101855582156064579182015b828111156064578251826000505591602001919060010190609e565b509056606060405260e060020a60003504630d86a0e181146100315780636874e8091461008d578063d736c513146100ea575b005b610190600180546020600282841615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156102295780601f106101fe57610100808354040283529160200191610229565b61019060008054602060026001831615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156102295780601f106101fe57610100808354040283529160200191610229565b60206004803580820135601f81018490049093026080908101604052606084815261002f946024939192918401918190838280828437509496505050505050508060016000509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061023157805160ff19168380011785555b506102619291505b808211156102665760008155830161017d565b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156101f05780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b820191906000526020600020905b81548152906001019060200180831161020c57829003601f168201915b505050505081565b82800160010185558215610175579182015b82811115610175578251826000505591602001919060010190610243565b505050565b509056",
}

// Interactor is an auto generated Go binding around an Ethereum contract.
type Interactor struct {
	abi abi.ABI
}

// NewInteractor creates a new instance of Interactor.
func NewInteractor() (*Interactor, error) {
	parsed, err := InteractorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Interactor{abi: *parsed}, nil
}

func (interactor *Interactor) PackConstructor(str string) []byte {
	res, _ := interactor.abi.Pack("", str)
	return res
}

// DeployString is a free data retrieval call binding the contract method 0x6874e809.
//
// Solidity: function deployString() returns(string)
func (interactor *Interactor) PackDeployString() ([]byte, error) {
	return interactor.abi.Pack("deployString")
}

func (interactor *Interactor) UnpackDeployString(data []byte) (string, error) {
	out, err := interactor.abi.Unpack("deployString", data)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Transact is a free data retrieval call binding the contract method 0xd736c513.
//
// Solidity: function transact(string str) returns()
func (interactor *Interactor) PackTransact(Str string) ([]byte, error) {
	return interactor.abi.Pack("transact", Str)
}

// TransactString is a free data retrieval call binding the contract method 0x0d86a0e1.
//
// Solidity: function transactString() returns(string)
func (interactor *Interactor) PackTransactString() ([]byte, error) {
	return interactor.abi.Pack("transactString")
}

func (interactor *Interactor) UnpackTransactString(data []byte) (string, error) {
	out, err := interactor.abi.Unpack("transactString", data)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}