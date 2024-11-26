// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package v2_generated_testcase

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

// ExampleexampleStruct is an auto generated low-level Go binding around an user-defined struct.
type ExampleexampleStruct struct {
	Val1 *big.Int
	Val2 *big.Int
	Val3 string
}

// V2GeneratedTestcaseMetaData contains all meta data concerning the V2GeneratedTestcase contract.
var V2GeneratedTestcaseMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"firstArg\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"secondArg\",\"type\":\"string\"}],\"name\":\"Basic\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"firstArg\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"int256\",\"name\":\"val1\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"val2\",\"type\":\"int256\"},{\"internalType\":\"string\",\"name\":\"val3\",\"type\":\"string\"}],\"indexed\":false,\"internalType\":\"structExample.exampleStruct\",\"name\":\"secondArg\",\"type\":\"tuple\"}],\"name\":\"Struct\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"emitEvent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"emitEventsDiffTypes\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"emitTwoEvents\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"mutateStorageVal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"retrieveStorageVal\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b5061052b8061001c5f395ff3fe608060405234801561000f575f80fd5b5060043610610055575f3560e01c80636da1cd55146100595780637216c3331461007757806379da6d70146100815780637b0cb8391461008b578063bf54fad414610095575b5f80fd5b6100616100b1565b60405161006e9190610246565b60405180910390f35b61007f6100b9565b005b610089610129565b005b6100936101ec565b005b6100af60048036038101906100aa919061028d565b610225565b005b5f8054905090565b607b7f114f97f563bc13d79fae32f4746248fd563650659371fef0bc8a7011fdd7bc6a6040516100e890610312565b60405180910390a2607b7f114f97f563bc13d79fae32f4746248fd563650659371fef0bc8a7011fdd7bc6a60405161011f9061037a565b60405180910390a2565b607b7f114f97f563bc13d79fae32f4746248fd563650659371fef0bc8a7011fdd7bc6a60405161015890610312565b60405180910390a2607b7faa0aa64a4dab26dbe87da2dcff945b2197f80de1903db7334b8f496a94428d39604051806060016040528060018152602001600281526020016040518060400160405280600681526020017f737472696e6700000000000000000000000000000000000000000000000000008152508152506040516101e2919061046d565b60405180910390a2565b607b7f114f97f563bc13d79fae32f4746248fd563650659371fef0bc8a7011fdd7bc6a60405161021b906104d7565b60405180910390a2565b805f8190555050565b5f819050919050565b6102408161022e565b82525050565b5f6020820190506102595f830184610237565b92915050565b5f80fd5b61026c8161022e565b8114610276575f80fd5b50565b5f8135905061028781610263565b92915050565b5f602082840312156102a2576102a161025f565b5b5f6102af84828501610279565b91505092915050565b5f82825260208201905092915050565b7f6576656e743100000000000000000000000000000000000000000000000000005f82015250565b5f6102fc6006836102b8565b9150610307826102c8565b602082019050919050565b5f6020820190508181035f830152610329816102f0565b9050919050565b7f6576656e743200000000000000000000000000000000000000000000000000005f82015250565b5f6103646006836102b8565b915061036f82610330565b602082019050919050565b5f6020820190508181035f83015261039181610358565b9050919050565b5f819050919050565b6103aa81610398565b82525050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6103f2826103b0565b6103fc81856103ba565b935061040c8185602086016103ca565b610415816103d8565b840191505092915050565b5f606083015f8301516104355f8601826103a1565b50602083015161044860208601826103a1565b506040830151848203604086015261046082826103e8565b9150508091505092915050565b5f6020820190508181035f8301526104858184610420565b905092915050565b7f6576656e740000000000000000000000000000000000000000000000000000005f82015250565b5f6104c16005836102b8565b91506104cc8261048d565b602082019050919050565b5f6020820190508181035f8301526104ee816104b5565b905091905056fea2646970667358221220946fc97c32ae98514551443303280f456cca960cbdcc95d1cec614233dec100764736f6c634300081a0033",
}

// V2GeneratedTestcaseInstance represents a deployed instance of the V2GeneratedTestcase contract.
type V2GeneratedTestcaseInstance struct {
	V2GeneratedTestcase
	address common.Address // consider removing this, not clear what it's used for now (and why did we need custom deploy method on previous abi?)
	backend bind.ContractBackend
}

func NewV2GeneratedTestcaseInstance(c *V2GeneratedTestcase, address common.Address, backend bind.ContractBackend) *V2GeneratedTestcaseInstance {
	return &V2GeneratedTestcaseInstance{V2GeneratedTestcase: *c, address: address}
}

func (i *V2GeneratedTestcaseInstance) Address() common.Address {
	return i.address
}

func (i *V2GeneratedTestcaseInstance) Backend() bind.ContractBackend {
	return i.backend
}

// V2GeneratedTestcase is an auto generated Go binding around an Ethereum contract.
type V2GeneratedTestcase struct {
	abi        abi.ABI
	deployCode []byte
}

// NewV2GeneratedTestcase creates a new instance of V2GeneratedTestcase.
func NewV2GeneratedTestcase() (*V2GeneratedTestcase, error) {
	parsed, err := V2GeneratedTestcaseMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	code := common.Hex2Bytes(V2GeneratedTestcaseMetaData.Bin)
	return &V2GeneratedTestcase{abi: *parsed, deployCode: code}, nil
}

func (_V2GeneratedTestcase *V2GeneratedTestcase) DeployCode() []byte {
	return _V2GeneratedTestcase.deployCode
}

func (_V2GeneratedTestcase *V2GeneratedTestcase) PackConstructor() ([]byte, error) {
	return _V2GeneratedTestcase.abi.Pack("")
}

// EmitEvent is a free data retrieval call binding the contract method 0x7b0cb839.
//
// Solidity: function emitEvent() returns()
func (_V2GeneratedTestcase *V2GeneratedTestcase) PackEmitEvent() ([]byte, error) {
	return _V2GeneratedTestcase.abi.Pack("emitEvent")
}

// EmitEventsDiffTypes is a free data retrieval call binding the contract method 0x79da6d70.
//
// Solidity: function emitEventsDiffTypes() returns()
func (_V2GeneratedTestcase *V2GeneratedTestcase) PackEmitEventsDiffTypes() ([]byte, error) {
	return _V2GeneratedTestcase.abi.Pack("emitEventsDiffTypes")
}

// EmitTwoEvents is a free data retrieval call binding the contract method 0x7216c333.
//
// Solidity: function emitTwoEvents() returns()
func (_V2GeneratedTestcase *V2GeneratedTestcase) PackEmitTwoEvents() ([]byte, error) {
	return _V2GeneratedTestcase.abi.Pack("emitTwoEvents")
}

// MutateStorageVal is a free data retrieval call binding the contract method 0xbf54fad4.
//
// Solidity: function mutateStorageVal(uint256 num) returns()
func (_V2GeneratedTestcase *V2GeneratedTestcase) PackMutateStorageVal(num *big.Int) ([]byte, error) {
	return _V2GeneratedTestcase.abi.Pack("mutateStorageVal", num)
}

// RetrieveStorageVal is a free data retrieval call binding the contract method 0x6da1cd55.
//
// Solidity: function retrieveStorageVal() view returns(uint256)
func (_V2GeneratedTestcase *V2GeneratedTestcase) PackRetrieveStorageVal() ([]byte, error) {
	return _V2GeneratedTestcase.abi.Pack("retrieveStorageVal")
}

func (_V2GeneratedTestcase *V2GeneratedTestcase) UnpackRetrieveStorageVal(data []byte) (*big.Int, error) {
	out, err := _V2GeneratedTestcase.abi.Unpack("retrieveStorageVal", data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// V2GeneratedTestcaseBasic represents a Basic event raised by the V2GeneratedTestcase contract.
type V2GeneratedTestcaseBasic struct {
	FirstArg  *big.Int
	SecondArg string
	Raw       *types.Log // Blockchain specific contextual infos
}

func V2GeneratedTestcaseBasicEventID() common.Hash {
	return common.HexToHash("0x114f97f563bc13d79fae32f4746248fd563650659371fef0bc8a7011fdd7bc6a")
}

func (_V2GeneratedTestcase *V2GeneratedTestcase) UnpackBasicEvent(log *types.Log) (*V2GeneratedTestcaseBasic, error) {
	event := "Basic"
	if log.Topics[0] != _V2GeneratedTestcase.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(V2GeneratedTestcaseBasic)
	if len(log.Data) > 0 {
		if err := _V2GeneratedTestcase.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _V2GeneratedTestcase.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// V2GeneratedTestcaseStruct represents a Struct event raised by the V2GeneratedTestcase contract.
type V2GeneratedTestcaseStruct struct {
	FirstArg  *big.Int
	SecondArg ExampleexampleStruct
	Raw       *types.Log // Blockchain specific contextual infos
}

func V2GeneratedTestcaseStructEventID() common.Hash {
	return common.HexToHash("0xaa0aa64a4dab26dbe87da2dcff945b2197f80de1903db7334b8f496a94428d39")
}

func (_V2GeneratedTestcase *V2GeneratedTestcase) UnpackStructEvent(log *types.Log) (*V2GeneratedTestcaseStruct, error) {
	event := "Struct"
	if log.Topics[0] != _V2GeneratedTestcase.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(V2GeneratedTestcaseStruct)
	if len(log.Data) > 0 {
		if err := _V2GeneratedTestcase.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _V2GeneratedTestcase.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

