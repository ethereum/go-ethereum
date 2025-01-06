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
// RangeKeywordMetaData contains all meta data concerning the RangeKeyword contract.
var RangeKeywordMetaData = &bind.MetaData{
	ABI:     "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"range\",\"type\":\"uint256\"}],\"name\":\"functionWithKeywordParameter\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Pattern: "cec8c872ba06feb1b8f0a00e7b237eb226",
	Bin:     "0x608060405234801561001057600080fd5b5060dc8061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063527a119f14602d575b600080fd5b60436004803603810190603f9190605b565b6045565b005b50565b6000813590506055816092565b92915050565b600060208284031215606e57606d608d565b5b6000607a848285016048565b91505092915050565b6000819050919050565b600080fd5b6099816083565b811460a357600080fd5b5056fea2646970667358221220d4f4525e2615516394055d369fb17df41c359e5e962734f27fd683ea81fd9db164736f6c63430008070033",
}

// RangeKeyword is an auto generated Go binding around an Ethereum contract.
type RangeKeyword struct {
	abi abi.ABI
}

// NewRangeKeyword creates a new instance of RangeKeyword.
func NewRangeKeyword() (*RangeKeyword, error) {
	parsed, err := RangeKeywordMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &RangeKeyword{abi: *parsed}, nil
}

func (_RangeKeyword *RangeKeyword) PackConstructor() []byte {
	res, _ := _RangeKeyword.abi.Pack("")
	return res
}

// FunctionWithKeywordParameter is a free data retrieval call binding the contract method 0x527a119f.
//
// Solidity: function functionWithKeywordParameter(uint256 range) pure returns()
func (_RangeKeyword *RangeKeyword) PackFunctionWithKeywordParameter(Arg0 *big.Int) ([]byte, error) {
	return _RangeKeyword.abi.Pack("functionWithKeywordParameter", Arg0)
}
