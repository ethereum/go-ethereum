// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package db

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = bytes.Equal
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// DBStats is an auto generated low-level Go binding around an user-defined struct.
type DBStats struct {
	Gets    *big.Int
	Inserts *big.Int
	Mods    *big.Int
}

// DBMetaData contains all meta data concerning the DB contract.
var DBMetaData = bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"key\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"Insert\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"key\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"KeyedInsert\",\"type\":\"event\"},{\"stateMutability\":\"nonpayable\",\"type\":\"fallback\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"k\",\"type\":\"uint256\"}],\"name\":\"get\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNamedStatParams\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gets\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"inserts\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"mods\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getStatParams\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getStatsStruct\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"gets\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"inserts\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"mods\",\"type\":\"uint256\"}],\"internalType\":\"structDB.Stats\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"k\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"v\",\"type\":\"uint256\"}],\"name\":\"insert\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	ID:  "253cc2574e2f8b5e909644530e4934f6ac",
	Bin: "0x60806040525f5f553480156011575f5ffd5b5060405180606001604052805f81526020015f81526020015f81525060035f820151815f015560208201518160010155604082015181600201559050506105f78061005b5f395ff3fe60806040526004361061004d575f3560e01c80631d834a1b146100cb5780636fcb9c70146101075780639507d39a14610133578063e369ba3b1461016f578063ee8161e01461019b5761006a565b3661006a57345f5f82825461006291906103eb565b925050819055005b348015610075575f5ffd5b505f36606082828080601f0160208091040260200160405190810160405280939291908181526020018383808284375f81840152601f19601f820116905080830192505050505050509050915050805190602001f35b3480156100d6575f5ffd5b506100f160048036038101906100ec919061044c565b6101c5565b6040516100fe9190610499565b60405180910390f35b348015610112575f5ffd5b5061011b6102ef565b60405161012a939291906104b2565b60405180910390f35b34801561013e575f5ffd5b50610159600480360381019061015491906104e7565b61030e565b6040516101669190610499565b60405180910390f35b34801561017a575f5ffd5b50610183610341565b604051610192939291906104b2565b60405180910390f35b3480156101a6575f5ffd5b506101af610360565b6040516101bc9190610561565b60405180910390f35b5f5f82036101da5760028054905090506102e9565b5f60015f8581526020019081526020015f20540361023757600283908060018154018082558091505060019003905f5260205f20015f909190919091505560036001015f81548092919061022d9061057a565b9190505550610252565b60036002015f81548092919061024c9061057a565b91905055505b8160015f8581526020019081526020015f20819055507f8b39ff47dca36ab5b8b80845238af53aa579625ac7fb173dc09376adada4176983836002805490506040516102a0939291906104b2565b60405180910390a1827f40bed843c6c5f72002f9b469cf4c1ee9f7fb1eb48f091c1267970f98522ac02d836040516102d89190610499565b60405180910390a260028054905090505b92915050565b5f5f5f60035f0154600360010154600360020154925092509250909192565b5f60035f015f8154809291906103239061057a565b919050555060015f8381526020019081526020015f20549050919050565b5f5f5f60035f0154600360010154600360020154925092509250909192565b610368610397565b60036040518060600160405290815f820154815260200160018201548152602001600282015481525050905090565b60405180606001604052805f81526020015f81526020015f81525090565b5f819050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6103f5826103b5565b9150610400836103b5565b9250828201905080821115610418576104176103be565b5b92915050565b5f5ffd5b61042b816103b5565b8114610435575f5ffd5b50565b5f8135905061044681610422565b92915050565b5f5f604083850312156104625761046161041e565b5b5f61046f85828601610438565b925050602061048085828601610438565b9150509250929050565b610493816103b5565b82525050565b5f6020820190506104ac5f83018461048a565b92915050565b5f6060820190506104c55f83018661048a565b6104d2602083018561048a565b6104df604083018461048a565b949350505050565b5f602082840312156104fc576104fb61041e565b5b5f61050984828501610438565b91505092915050565b61051b816103b5565b82525050565b606082015f8201516105355f850182610512565b5060208201516105486020850182610512565b50604082015161055b6040850182610512565b50505050565b5f6060820190506105745f830184610521565b92915050565b5f610584826103b5565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036105b6576105b56103be565b5b60018201905091905056fea264697066735822122063e58431f2afdc667f8e687d3e6a99085a93c1fd3ce40b218463b8ddd3cc093664736f6c634300081c0033",
}

// DB is an auto generated Go binding around an Ethereum contract.
type DB struct {
	abi abi.ABI
}

// NewDB creates a new instance of DB.
func NewDB() *DB {
	parsed, err := DBMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &DB{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *DB) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackGet is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x9507d39a.
//
// Solidity: function get(uint256 k) returns(uint256)
func (dB *DB) PackGet(k *big.Int) []byte {
	enc, err := dB.abi.Pack("get", k)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGet is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x9507d39a.
//
// Solidity: function get(uint256 k) returns(uint256)
func (dB *DB) UnpackGet(data []byte) (*big.Int, error) {
	out, err := dB.abi.Unpack("get", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackGetNamedStatParams is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe369ba3b.
//
// Solidity: function getNamedStatParams() view returns(uint256 gets, uint256 inserts, uint256 mods)
func (dB *DB) PackGetNamedStatParams() []byte {
	enc, err := dB.abi.Pack("getNamedStatParams")
	if err != nil {
		panic(err)
	}
	return enc
}

// GetNamedStatParamsOutput serves as a container for the return parameters of contract
// method GetNamedStatParams.
type GetNamedStatParamsOutput struct {
	Gets    *big.Int
	Inserts *big.Int
	Mods    *big.Int
}

// UnpackGetNamedStatParams is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xe369ba3b.
//
// Solidity: function getNamedStatParams() view returns(uint256 gets, uint256 inserts, uint256 mods)
func (dB *DB) UnpackGetNamedStatParams(data []byte) (GetNamedStatParamsOutput, error) {
	out, err := dB.abi.Unpack("getNamedStatParams", data)
	outstruct := new(GetNamedStatParamsOutput)
	if err != nil {
		return *outstruct, err
	}
	outstruct.Gets = abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	outstruct.Inserts = abi.ConvertType(out[1], new(big.Int)).(*big.Int)
	outstruct.Mods = abi.ConvertType(out[2], new(big.Int)).(*big.Int)
	return *outstruct, err

}

// PackGetStatParams is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x6fcb9c70.
//
// Solidity: function getStatParams() view returns(uint256, uint256, uint256)
func (dB *DB) PackGetStatParams() []byte {
	enc, err := dB.abi.Pack("getStatParams")
	if err != nil {
		panic(err)
	}
	return enc
}

// GetStatParamsOutput serves as a container for the return parameters of contract
// method GetStatParams.
type GetStatParamsOutput struct {
	Arg0 *big.Int
	Arg1 *big.Int
	Arg2 *big.Int
}

// UnpackGetStatParams is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x6fcb9c70.
//
// Solidity: function getStatParams() view returns(uint256, uint256, uint256)
func (dB *DB) UnpackGetStatParams(data []byte) (GetStatParamsOutput, error) {
	out, err := dB.abi.Unpack("getStatParams", data)
	outstruct := new(GetStatParamsOutput)
	if err != nil {
		return *outstruct, err
	}
	outstruct.Arg0 = abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	outstruct.Arg1 = abi.ConvertType(out[1], new(big.Int)).(*big.Int)
	outstruct.Arg2 = abi.ConvertType(out[2], new(big.Int)).(*big.Int)
	return *outstruct, err

}

// PackGetStatsStruct is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xee8161e0.
//
// Solidity: function getStatsStruct() view returns((uint256,uint256,uint256))
func (dB *DB) PackGetStatsStruct() []byte {
	enc, err := dB.abi.Pack("getStatsStruct")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetStatsStruct is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xee8161e0.
//
// Solidity: function getStatsStruct() view returns((uint256,uint256,uint256))
func (dB *DB) UnpackGetStatsStruct(data []byte) (DBStats, error) {
	out, err := dB.abi.Unpack("getStatsStruct", data)
	if err != nil {
		return *new(DBStats), err
	}
	out0 := *abi.ConvertType(out[0], new(DBStats)).(*DBStats)
	return out0, err
}

// PackInsert is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x1d834a1b.
//
// Solidity: function insert(uint256 k, uint256 v) returns(uint256)
func (dB *DB) PackInsert(k *big.Int, v *big.Int) []byte {
	enc, err := dB.abi.Pack("insert", k, v)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackInsert is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x1d834a1b.
//
// Solidity: function insert(uint256 k, uint256 v) returns(uint256)
func (dB *DB) UnpackInsert(data []byte) (*big.Int, error) {
	out, err := dB.abi.Unpack("insert", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// DBInsert represents a Insert event raised by the DB contract.
type DBInsert struct {
	Key    *big.Int
	Value  *big.Int
	Length *big.Int
	Raw    *types.Log // Blockchain specific contextual infos
}

const DBInsertEventName = "Insert"

// ContractEventName returns the user-defined event name.
func (DBInsert) ContractEventName() string {
	return DBInsertEventName
}

// UnpackInsertEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Insert(uint256 key, uint256 value, uint256 length)
func (dB *DB) UnpackInsertEvent(log *types.Log) (*DBInsert, error) {
	event := "Insert"
	if log.Topics[0] != dB.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DBInsert)
	if len(log.Data) > 0 {
		if err := dB.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dB.abi.Events[event].Inputs {
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

// DBKeyedInsert represents a KeyedInsert event raised by the DB contract.
type DBKeyedInsert struct {
	Key   *big.Int
	Value *big.Int
	Raw   *types.Log // Blockchain specific contextual infos
}

const DBKeyedInsertEventName = "KeyedInsert"

// ContractEventName returns the user-defined event name.
func (DBKeyedInsert) ContractEventName() string {
	return DBKeyedInsertEventName
}

// UnpackKeyedInsertEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event KeyedInsert(uint256 indexed key, uint256 value)
func (dB *DB) UnpackKeyedInsertEvent(log *types.Log) (*DBKeyedInsert, error) {
	event := "KeyedInsert"
	if log.Topics[0] != dB.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DBKeyedInsert)
	if len(log.Data) > 0 {
		if err := dB.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dB.abi.Events[event].Inputs {
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
