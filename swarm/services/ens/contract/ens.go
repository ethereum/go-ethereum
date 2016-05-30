// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package contract

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ResolverABI is the input ABI used to generate the binding from.
const ResolverABI = `[{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"bytes32[]"}],"name":"deletePrivateRR","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"isPersonalResolver","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":false,"inputs":[{"name":"label","type":"bytes32"},{"name":"newOwner","type":"address"}],"name":"setOwner","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"id","type":"bytes32"}],"name":"getExtended","outputs":[{"name":"data","type":"bytes"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"string"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"name":"setRR","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"bytes32[]"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"name":"setPrivateRR","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"qtype","type":"bytes32"},{"name":"index","type":"uint16"}],"name":"resolve","outputs":[{"name":"rcode","type":"uint16"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"label","type":"bytes32"},{"name":"resolver","type":"address"},{"name":"nodeId","type":"bytes12"}],"name":"register","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"label","type":"bytes32"},{"name":"resolver","type":"address"},{"name":"nodeId","type":"bytes12"}],"name":"setResolver","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"string"}],"name":"deleteRR","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"label","type":"bytes32"}],"name":"getOwner","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"label","type":"bytes32"}],"name":"findResolver","outputs":[{"name":"rcode","type":"uint16"},{"name":"ttl","type":"uint32"},{"name":"rnode","type":"bytes12"},{"name":"raddress","type":"address"}],"type":"function"}]`

// ResolverBin is the compiled bytecode used for deploying new contracts.
const ResolverBin = `0x`

// DeployResolver deploys a new Ethereum contract, binding an instance of Resolver to it.
func DeployResolver(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Resolver, error) {
	parsed, err := abi.JSON(strings.NewReader(ResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ResolverBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Resolver{ResolverCaller: ResolverCaller{contract: contract}, ResolverTransactor: ResolverTransactor{contract: contract}}, nil
}

// Resolver is an auto generated Go binding around an Ethereum contract.
type Resolver struct {
	ResolverCaller     // Read-only binding to the contract
	ResolverTransactor // Write-only binding to the contract
}

// ResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type ResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ResolverSession struct {
	Contract     *Resolver         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ResolverCallerSession struct {
	Contract *ResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ResolverTransactorSession struct {
	Contract     *ResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type ResolverRaw struct {
	Contract *Resolver // Generic contract binding to access the raw methods on
}

// ResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ResolverCallerRaw struct {
	Contract *ResolverCaller // Generic read-only contract binding to access the raw methods on
}

// ResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ResolverTransactorRaw struct {
	Contract *ResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewResolver creates a new instance of Resolver, bound to a specific deployed contract.
func NewResolver(address common.Address, backend bind.ContractBackend) (*Resolver, error) {
	contract, err := bindResolver(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))
	if err != nil {
		return nil, err
	}
	return &Resolver{ResolverCaller: ResolverCaller{contract: contract}, ResolverTransactor: ResolverTransactor{contract: contract}}, nil
}

// NewResolverCaller creates a new read-only instance of Resolver, bound to a specific deployed contract.
func NewResolverCaller(address common.Address, caller bind.ContractCaller) (*ResolverCaller, error) {
	contract, err := bindResolver(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &ResolverCaller{contract: contract}, nil
}

// NewResolverTransactor creates a new write-only instance of Resolver, bound to a specific deployed contract.
func NewResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*ResolverTransactor, error) {
	contract, err := bindResolver(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ResolverTransactor{contract: contract}, nil
}

// bindResolver binds a generic wrapper to an already deployed contract.
func bindResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Resolver *ResolverRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Resolver.Contract.ResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Resolver *ResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Resolver.Contract.ResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Resolver *ResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Resolver.Contract.ResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Resolver *ResolverCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Resolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Resolver *ResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Resolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Resolver *ResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Resolver.Contract.contract.Transact(opts, method, params...)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_Resolver *ResolverCaller) FindResolver(opts *bind.CallOpts, nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	ret := new(struct {
		Rcode    uint16
		Ttl      uint32
		Rnode    [12]byte
		Raddress common.Address
	})
	out := ret
	err := _Resolver.contract.Call(opts, out, "findResolver", nodeId, label)
	return *ret, err
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_Resolver *ResolverSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _Resolver.Contract.FindResolver(&_Resolver.CallOpts, nodeId, label)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_Resolver *ResolverCallerSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _Resolver.Contract.FindResolver(&_Resolver.CallOpts, nodeId, label)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_Resolver *ResolverCaller) GetExtended(opts *bind.CallOpts, id [32]byte) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _Resolver.contract.Call(opts, out, "getExtended", id)
	return *ret0, err
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_Resolver *ResolverSession) GetExtended(id [32]byte) ([]byte, error) {
	return _Resolver.Contract.GetExtended(&_Resolver.CallOpts, id)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_Resolver *ResolverCallerSession) GetExtended(id [32]byte) ([]byte, error) {
	return _Resolver.Contract.GetExtended(&_Resolver.CallOpts, id)
}

// GetOwner is a free data retrieval call binding the contract method 0xdeb931a2.
//
// Solidity: function getOwner(label bytes32) constant returns(address)
func (_Resolver *ResolverCaller) GetOwner(opts *bind.CallOpts, label [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Resolver.contract.Call(opts, out, "getOwner", label)
	return *ret0, err
}

// GetOwner is a free data retrieval call binding the contract method 0xdeb931a2.
//
// Solidity: function getOwner(label bytes32) constant returns(address)
func (_Resolver *ResolverSession) GetOwner(label [32]byte) (common.Address, error) {
	return _Resolver.Contract.GetOwner(&_Resolver.CallOpts, label)
}

// GetOwner is a free data retrieval call binding the contract method 0xdeb931a2.
//
// Solidity: function getOwner(label bytes32) constant returns(address)
func (_Resolver *ResolverCallerSession) GetOwner(label [32]byte) (common.Address, error) {
	return _Resolver.Contract.GetOwner(&_Resolver.CallOpts, label)
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_Resolver *ResolverCaller) IsPersonalResolver(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Resolver.contract.Call(opts, out, "isPersonalResolver")
	return *ret0, err
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_Resolver *ResolverSession) IsPersonalResolver() (bool, error) {
	return _Resolver.Contract.IsPersonalResolver(&_Resolver.CallOpts)
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_Resolver *ResolverCallerSession) IsPersonalResolver() (bool, error) {
	return _Resolver.Contract.IsPersonalResolver(&_Resolver.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_Resolver *ResolverCaller) Resolve(opts *bind.CallOpts, nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	ret := new(struct {
		Rcode uint16
		Rtype [16]byte
		Ttl   uint32
		Len   uint16
		Data  [32]byte
	})
	out := ret
	err := _Resolver.contract.Call(opts, out, "resolve", nodeId, qtype, index)
	return *ret, err
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_Resolver *ResolverSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _Resolver.Contract.Resolve(&_Resolver.CallOpts, nodeId, qtype, index)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_Resolver *ResolverCallerSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _Resolver.Contract.Resolve(&_Resolver.CallOpts, nodeId, qtype, index)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_Resolver *ResolverTransactor) DeletePrivateRR(opts *bind.TransactOpts, rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "deletePrivateRR", rootNodeId, name)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_Resolver *ResolverSession) DeletePrivateRR(rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _Resolver.Contract.DeletePrivateRR(&_Resolver.TransactOpts, rootNodeId, name)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_Resolver *ResolverTransactorSession) DeletePrivateRR(rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _Resolver.Contract.DeletePrivateRR(&_Resolver.TransactOpts, rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_Resolver *ResolverTransactor) DeleteRR(opts *bind.TransactOpts, rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "deleteRR", rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_Resolver *ResolverSession) DeleteRR(rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _Resolver.Contract.DeleteRR(&_Resolver.TransactOpts, rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_Resolver *ResolverTransactorSession) DeleteRR(rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _Resolver.Contract.DeleteRR(&_Resolver.TransactOpts, rootNodeId, name)
}

// Register is a paid mutator transaction binding the contract method 0xa1f8f8f0.
//
// Solidity: function register(label bytes32, resolver address, nodeId bytes12) returns()
func (_Resolver *ResolverTransactor) Register(opts *bind.TransactOpts, label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "register", label, resolver, nodeId)
}

// Register is a paid mutator transaction binding the contract method 0xa1f8f8f0.
//
// Solidity: function register(label bytes32, resolver address, nodeId bytes12) returns()
func (_Resolver *ResolverSession) Register(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _Resolver.Contract.Register(&_Resolver.TransactOpts, label, resolver, nodeId)
}

// Register is a paid mutator transaction binding the contract method 0xa1f8f8f0.
//
// Solidity: function register(label bytes32, resolver address, nodeId bytes12) returns()
func (_Resolver *ResolverTransactorSession) Register(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _Resolver.Contract.Register(&_Resolver.TransactOpts, label, resolver, nodeId)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(label bytes32, newOwner address) returns()
func (_Resolver *ResolverTransactor) SetOwner(opts *bind.TransactOpts, label [32]byte, newOwner common.Address) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "setOwner", label, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(label bytes32, newOwner address) returns()
func (_Resolver *ResolverSession) SetOwner(label [32]byte, newOwner common.Address) (*types.Transaction, error) {
	return _Resolver.Contract.SetOwner(&_Resolver.TransactOpts, label, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(label bytes32, newOwner address) returns()
func (_Resolver *ResolverTransactorSession) SetOwner(label [32]byte, newOwner common.Address) (*types.Transaction, error) {
	return _Resolver.Contract.SetOwner(&_Resolver.TransactOpts, label, newOwner)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_Resolver *ResolverTransactor) SetPrivateRR(opts *bind.TransactOpts, rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "setPrivateRR", rootNodeId, name, rtype, ttl, len, data)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_Resolver *ResolverSession) SetPrivateRR(rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _Resolver.Contract.SetPrivateRR(&_Resolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_Resolver *ResolverTransactorSession) SetPrivateRR(rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _Resolver.Contract.SetPrivateRR(&_Resolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_Resolver *ResolverTransactor) SetRR(opts *bind.TransactOpts, rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "setRR", rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_Resolver *ResolverSession) SetRR(rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _Resolver.Contract.SetRR(&_Resolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_Resolver *ResolverTransactorSession) SetRR(rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _Resolver.Contract.SetRR(&_Resolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetResolver is a paid mutator transaction binding the contract method 0xa9f2a1b2.
//
// Solidity: function setResolver(label bytes32, resolver address, nodeId bytes12) returns()
func (_Resolver *ResolverTransactor) SetResolver(opts *bind.TransactOpts, label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _Resolver.contract.Transact(opts, "setResolver", label, resolver, nodeId)
}

// SetResolver is a paid mutator transaction binding the contract method 0xa9f2a1b2.
//
// Solidity: function setResolver(label bytes32, resolver address, nodeId bytes12) returns()
func (_Resolver *ResolverSession) SetResolver(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _Resolver.Contract.SetResolver(&_Resolver.TransactOpts, label, resolver, nodeId)
}

// SetResolver is a paid mutator transaction binding the contract method 0xa9f2a1b2.
//
// Solidity: function setResolver(label bytes32, resolver address, nodeId bytes12) returns()
func (_Resolver *ResolverTransactorSession) SetResolver(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _Resolver.Contract.SetResolver(&_Resolver.TransactOpts, label, resolver, nodeId)
}
