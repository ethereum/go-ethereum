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
	ABI:     "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arg1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"arg3\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"arg4\",\"type\":\"bool\"}],\"name\":\"BadThing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Foo\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Pattern: "55ef3c19a0ab1c1845f9e347540c1e51f5",
	Bin:     "0x6080604052348015600e575f80fd5b506101148061001c5f395ff3fe6080604052348015600e575f80fd5b50600436106026575f3560e01c8063bfb4ebcf14602a575b5f80fd5b60306032565b005b5f600160025f6040517fbb6a82f1000000000000000000000000000000000000000000000000000000008152600401606c949392919060a3565b60405180910390fd5b5f819050919050565b6085816075565b82525050565b5f8115159050919050565b609d81608b565b82525050565b5f60808201905060b45f830187607e565b60bf6020830186607e565b60ca6040830185607e565b60d560608301846096565b9594505050505056fea26469706673582212205ce065ab1cfe16beba2b766e14009fc67ac66c214872149c889f0589720b870a64736f6c634300081a0033",
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

// Foo is a free data retrieval call binding the contract method 0xbfb4ebcf.
//
// Solidity: function Foo() pure returns()
func (_C *C) PackFoo() ([]byte, error) {
	return _C.abi.Pack("Foo")
}

func (_C *C) UnpackError(raw []byte) any {

	if val, err := _C.UnpackBadThingError(raw); err != nil {
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
