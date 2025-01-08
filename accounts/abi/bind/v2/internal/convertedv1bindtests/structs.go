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

// Struct0 is an auto generated low-level Go binding around an user-defined struct.
type Struct0 struct {
	B [32]byte
}

// StructsMetaData contains all meta data concerning the Structs contract.
var StructsMetaData = bind.MetaData{
	ABI:     "[{\"inputs\":[],\"name\":\"F\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"B\",\"type\":\"bytes32\"}],\"internalType\":\"structStructs.A[]\",\"name\":\"a\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256[]\",\"name\":\"c\",\"type\":\"uint256[]\"},{\"internalType\":\"bool[]\",\"name\":\"d\",\"type\":\"bool[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"G\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"B\",\"type\":\"bytes32\"}],\"internalType\":\"structStructs.A[]\",\"name\":\"a\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Pattern: "920a35318e7581766aec7a17218628a91d",
	Bin:     "0x608060405234801561001057600080fd5b50610278806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806328811f591461003b5780636fecb6231461005b575b600080fd5b610043610070565b604051610052939291906101a0565b60405180910390f35b6100636100d6565b6040516100529190610186565b604080516002808252606082810190935282918291829190816020015b610095610131565b81526020019060019003908161008d575050805190915061026960611b9082906000906100be57fe5b60209081029190910101515293606093508392509050565b6040805160028082526060828101909352829190816020015b6100f7610131565b8152602001906001900390816100ef575050805190915061026960611b90829060009061012057fe5b602090810291909101015152905090565b60408051602081019091526000815290565b815260200190565b6000815180845260208085019450808401835b8381101561017b578151518752958201959082019060010161015e565b509495945050505050565b600060208252610199602083018461014b565b9392505050565b6000606082526101b3606083018661014b565b6020838203818501528186516101c98185610239565b91508288019350845b818110156101f3576101e5838651610143565b9484019492506001016101d2565b505084810360408601528551808252908201925081860190845b8181101561022b57825115158552938301939183019160010161020d565b509298975050505050505050565b9081526020019056fea2646970667358221220eb85327e285def14230424c52893aebecec1e387a50bb6b75fc4fdbed647f45f64736f6c63430006050033",
}

// Structs is an auto generated Go binding around an Ethereum contract.
type Structs struct {
	abi abi.ABI
}

// NewStructs creates a new instance of Structs.
func NewStructs() (*Structs, error) {
	parsed, err := StructsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Structs{abi: *parsed}, nil
}

// F is a free data retrieval call binding the contract method 0x28811f59.
//
// Solidity: function F() view returns((bytes32)[] a, uint256[] c, bool[] d)
func (structs *Structs) PackF() ([]byte, error) {
	return structs.abi.Pack("F")
}

type FOutput struct {
	A []Struct0
	C []*big.Int
	D []bool
}

func (structs *Structs) UnpackF(data []byte) (FOutput, error) {
	out, err := structs.abi.Unpack("F", data)

	outstruct := new(FOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.A = *abi.ConvertType(out[0], new([]Struct0)).(*[]Struct0)

	outstruct.C = *abi.ConvertType(out[1], new([]*big.Int)).(*[]*big.Int)

	outstruct.D = *abi.ConvertType(out[2], new([]bool)).(*[]bool)

	return *outstruct, err

}

// G is a free data retrieval call binding the contract method 0x6fecb623.
//
// Solidity: function G() view returns((bytes32)[] a)
func (structs *Structs) PackG() ([]byte, error) {
	return structs.abi.Pack("G")
}

func (structs *Structs) UnpackG(data []byte) ([]Struct0, error) {
	out, err := structs.abi.Unpack("G", data)

	if err != nil {
		return *new([]Struct0), err
	}

	out0 := *abi.ConvertType(out[0], new([]Struct0)).(*[]Struct0)

	return out0, err

}
