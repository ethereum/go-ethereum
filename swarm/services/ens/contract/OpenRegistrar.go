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

// OpenRegistrarABI is the input ABI used to generate the binding from.
const OpenRegistrarABI = `[{"constant":false,"inputs":[{"name":"label","type":"bytes32"},{"name":"newOwner","type":"address"}],"name":"setOwner","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"id","type":"bytes32"}],"name":"getExtended","outputs":[{"name":"data","type":"bytes"}],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"qtype","type":"bytes32"},{"name":"index","type":"uint16"}],"name":"resolve","outputs":[{"name":"rcode","type":"uint16"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"label","type":"bytes32"},{"name":"resolver","type":"address"},{"name":"nodeId","type":"bytes12"}],"name":"register","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"label","type":"bytes32"},{"name":"resolver","type":"address"},{"name":"nodeId","type":"bytes12"}],"name":"setResolver","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"label","type":"bytes32"}],"name":"getOwner","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"label","type":"bytes32"}],"name":"findResolver","outputs":[{"name":"rcode","type":"uint16"},{"name":"ttl","type":"uint32"},{"name":"rnode","type":"bytes12"},{"name":"raddress","type":"address"}],"type":"function"}]`

// OpenRegistrarBin is the compiled bytecode used for deploying new contracts.
const OpenRegistrarBin = `606060405261083b806100126000396000f36060604052361561007f576000357c0100000000000000000000000000000000000000000000000000000000900480635b0fc9c3146100815780638021061c146100a2578063a16fdafa14610126578063a1f8f8f0146101a5578063a9f2a1b2146101cf578063deb931a2146101f9578063edc0277c1461023b5761007f565b005b6100a060048080359060200190919080359060200190919050506102bc565b005b6100b86004808035906020019091905050610392565b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156101185780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61014e60048080359060200190919080359060200190919080359060200190919050506103c6565b604051808661ffff168152602001856fffffffffffffffffffffffffffffffff191681526020018463ffffffff1681526020018361ffff168152602001826000191681526020019550505050505060405180910390f35b6101cd600480803590602001909190803590602001909190803590602001909190505061041f565b005b6101f760048080359060200190919080359060200190919080359060200190919050506105ac565b005b61020f60048080359060200190919050506106c0565b604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b61025a600480803590602001909190803590602001909190505061070f565b604051808561ffff1681526020018463ffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff191681526020018273ffffffffffffffffffffffffffffffffffffffff16815260200194505050505060405180910390f35b600060008273ffffffffffffffffffffffffffffffffffffffff1614156102e257610002565b600060005060008460001916815260200190815260200160002060005090503373ffffffffffffffffffffffffffffffffffffffff168160010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561035f57610002565b818160010160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908302179055505b505050565b6020604051908101604052806000815260200150602060405190810160405280600081526020015090506103c1565b919050565b60006000600060006000600074010000000000000000000000000000000000000000028873ffffffffffffffffffffffffffffffffffffffff191614151561041357600394508450610414565b5b939792965093509350565b600060008373ffffffffffffffffffffffffffffffffffffffff16141561044557610002565b60006000506000856000191681526020019081526020016000206000509050600073ffffffffffffffffffffffffffffffffffffffff168160010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff161415156104c357610002565b60606040519081016040528084815260200183815260200133815260200150600060005060008660001916815260200190815260200160002060005060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff0219169083021790555060208201518160000160146101000a8154816bffffffffffffffffffffffff0219169083740100000000000000000000000000000000000000009004021790555060408201518160010160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908302179055509050505b50505050565b600060008373ffffffffffffffffffffffffffffffffffffffff1614156105d257610002565b600060005060008560001916815260200190815260200160002060005090503373ffffffffffffffffffffffffffffffffffffffff168160010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561064f57610002565b828160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff02191690830217905550818160000160146101000a8154816bffffffffffffffffffffffff021916908374010000000000000000000000000000000000000000900402179055505b50505050565b6000600060005060008360001916815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905061070a565b919050565b6000600060006000600060006000506000876000191681526020019081526020016000206000509050600074010000000000000000000000000000000000000000028773ffffffffffffffffffffffffffffffffffffffff19161415806107c65750600073ffffffffffffffffffffffffffffffffffffffff168160010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16145b156107d657600394508450610831565b610e10935083508060000160149054906101000a90047401000000000000000000000000000000000000000002925082508060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16915081505b509295919450925056`

// DeployOpenRegistrar deploys a new Ethereum contract, binding an instance of OpenRegistrar to it.
func DeployOpenRegistrar(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OpenRegistrar, error) {
	parsed, err := abi.JSON(strings.NewReader(OpenRegistrarABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(OpenRegistrarBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OpenRegistrar{OpenRegistrarCaller: OpenRegistrarCaller{contract: contract}, OpenRegistrarTransactor: OpenRegistrarTransactor{contract: contract}}, nil
}

// OpenRegistrar is an auto generated Go binding around an Ethereum contract.
type OpenRegistrar struct {
	OpenRegistrarCaller     // Read-only binding to the contract
	OpenRegistrarTransactor // Write-only binding to the contract
}

// OpenRegistrarCaller is an auto generated read-only Go binding around an Ethereum contract.
type OpenRegistrarCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OpenRegistrarTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OpenRegistrarTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OpenRegistrarSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OpenRegistrarSession struct {
	Contract     *OpenRegistrar    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OpenRegistrarCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OpenRegistrarCallerSession struct {
	Contract *OpenRegistrarCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// OpenRegistrarTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OpenRegistrarTransactorSession struct {
	Contract     *OpenRegistrarTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// OpenRegistrarRaw is an auto generated low-level Go binding around an Ethereum contract.
type OpenRegistrarRaw struct {
	Contract *OpenRegistrar // Generic contract binding to access the raw methods on
}

// OpenRegistrarCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OpenRegistrarCallerRaw struct {
	Contract *OpenRegistrarCaller // Generic read-only contract binding to access the raw methods on
}

// OpenRegistrarTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OpenRegistrarTransactorRaw struct {
	Contract *OpenRegistrarTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOpenRegistrar creates a new instance of OpenRegistrar, bound to a specific deployed contract.
func NewOpenRegistrar(address common.Address, backend bind.ContractBackend) (*OpenRegistrar, error) {
	contract, err := bindOpenRegistrar(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))
	if err != nil {
		return nil, err
	}
	return &OpenRegistrar{OpenRegistrarCaller: OpenRegistrarCaller{contract: contract}, OpenRegistrarTransactor: OpenRegistrarTransactor{contract: contract}}, nil
}

// NewOpenRegistrarCaller creates a new read-only instance of OpenRegistrar, bound to a specific deployed contract.
func NewOpenRegistrarCaller(address common.Address, caller bind.ContractCaller) (*OpenRegistrarCaller, error) {
	contract, err := bindOpenRegistrar(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &OpenRegistrarCaller{contract: contract}, nil
}

// NewOpenRegistrarTransactor creates a new write-only instance of OpenRegistrar, bound to a specific deployed contract.
func NewOpenRegistrarTransactor(address common.Address, transactor bind.ContractTransactor) (*OpenRegistrarTransactor, error) {
	contract, err := bindOpenRegistrar(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &OpenRegistrarTransactor{contract: contract}, nil
}

// bindOpenRegistrar binds a generic wrapper to an already deployed contract.
func bindOpenRegistrar(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OpenRegistrarABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OpenRegistrar *OpenRegistrarRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _OpenRegistrar.Contract.OpenRegistrarCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OpenRegistrar *OpenRegistrarRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.OpenRegistrarTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OpenRegistrar *OpenRegistrarRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.OpenRegistrarTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OpenRegistrar *OpenRegistrarCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _OpenRegistrar.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OpenRegistrar *OpenRegistrarTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OpenRegistrar *OpenRegistrarTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.contract.Transact(opts, method, params...)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_OpenRegistrar *OpenRegistrarCaller) FindResolver(opts *bind.CallOpts, nodeId [12]byte, label [32]byte) (struct {
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
	err := _OpenRegistrar.contract.Call(opts, out, "findResolver", nodeId, label)
	return *ret, err
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_OpenRegistrar *OpenRegistrarSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _OpenRegistrar.Contract.FindResolver(&_OpenRegistrar.CallOpts, nodeId, label)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_OpenRegistrar *OpenRegistrarCallerSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _OpenRegistrar.Contract.FindResolver(&_OpenRegistrar.CallOpts, nodeId, label)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_OpenRegistrar *OpenRegistrarCaller) GetExtended(opts *bind.CallOpts, id [32]byte) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _OpenRegistrar.contract.Call(opts, out, "getExtended", id)
	return *ret0, err
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_OpenRegistrar *OpenRegistrarSession) GetExtended(id [32]byte) ([]byte, error) {
	return _OpenRegistrar.Contract.GetExtended(&_OpenRegistrar.CallOpts, id)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_OpenRegistrar *OpenRegistrarCallerSession) GetExtended(id [32]byte) ([]byte, error) {
	return _OpenRegistrar.Contract.GetExtended(&_OpenRegistrar.CallOpts, id)
}

// GetOwner is a free data retrieval call binding the contract method 0xdeb931a2.
//
// Solidity: function getOwner(label bytes32) constant returns(address)
func (_OpenRegistrar *OpenRegistrarCaller) GetOwner(opts *bind.CallOpts, label [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OpenRegistrar.contract.Call(opts, out, "getOwner", label)
	return *ret0, err
}

// GetOwner is a free data retrieval call binding the contract method 0xdeb931a2.
//
// Solidity: function getOwner(label bytes32) constant returns(address)
func (_OpenRegistrar *OpenRegistrarSession) GetOwner(label [32]byte) (common.Address, error) {
	return _OpenRegistrar.Contract.GetOwner(&_OpenRegistrar.CallOpts, label)
}

// GetOwner is a free data retrieval call binding the contract method 0xdeb931a2.
//
// Solidity: function getOwner(label bytes32) constant returns(address)
func (_OpenRegistrar *OpenRegistrarCallerSession) GetOwner(label [32]byte) (common.Address, error) {
	return _OpenRegistrar.Contract.GetOwner(&_OpenRegistrar.CallOpts, label)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_OpenRegistrar *OpenRegistrarCaller) Resolve(opts *bind.CallOpts, nodeId [12]byte, qtype [32]byte, index uint16) (struct {
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
	err := _OpenRegistrar.contract.Call(opts, out, "resolve", nodeId, qtype, index)
	return *ret, err
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_OpenRegistrar *OpenRegistrarSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _OpenRegistrar.Contract.Resolve(&_OpenRegistrar.CallOpts, nodeId, qtype, index)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_OpenRegistrar *OpenRegistrarCallerSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _OpenRegistrar.Contract.Resolve(&_OpenRegistrar.CallOpts, nodeId, qtype, index)
}

// Register is a paid mutator transaction binding the contract method 0xa1f8f8f0.
//
// Solidity: function register(label bytes32, resolver address, nodeId bytes12) returns()
func (_OpenRegistrar *OpenRegistrarTransactor) Register(opts *bind.TransactOpts, label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _OpenRegistrar.contract.Transact(opts, "register", label, resolver, nodeId)
}

// Register is a paid mutator transaction binding the contract method 0xa1f8f8f0.
//
// Solidity: function register(label bytes32, resolver address, nodeId bytes12) returns()
func (_OpenRegistrar *OpenRegistrarSession) Register(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.Register(&_OpenRegistrar.TransactOpts, label, resolver, nodeId)
}

// Register is a paid mutator transaction binding the contract method 0xa1f8f8f0.
//
// Solidity: function register(label bytes32, resolver address, nodeId bytes12) returns()
func (_OpenRegistrar *OpenRegistrarTransactorSession) Register(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.Register(&_OpenRegistrar.TransactOpts, label, resolver, nodeId)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(label bytes32, newOwner address) returns()
func (_OpenRegistrar *OpenRegistrarTransactor) SetOwner(opts *bind.TransactOpts, label [32]byte, newOwner common.Address) (*types.Transaction, error) {
	return _OpenRegistrar.contract.Transact(opts, "setOwner", label, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(label bytes32, newOwner address) returns()
func (_OpenRegistrar *OpenRegistrarSession) SetOwner(label [32]byte, newOwner common.Address) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.SetOwner(&_OpenRegistrar.TransactOpts, label, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(label bytes32, newOwner address) returns()
func (_OpenRegistrar *OpenRegistrarTransactorSession) SetOwner(label [32]byte, newOwner common.Address) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.SetOwner(&_OpenRegistrar.TransactOpts, label, newOwner)
}

// SetResolver is a paid mutator transaction binding the contract method 0xa9f2a1b2.
//
// Solidity: function setResolver(label bytes32, resolver address, nodeId bytes12) returns()
func (_OpenRegistrar *OpenRegistrarTransactor) SetResolver(opts *bind.TransactOpts, label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _OpenRegistrar.contract.Transact(opts, "setResolver", label, resolver, nodeId)
}

// SetResolver is a paid mutator transaction binding the contract method 0xa9f2a1b2.
//
// Solidity: function setResolver(label bytes32, resolver address, nodeId bytes12) returns()
func (_OpenRegistrar *OpenRegistrarSession) SetResolver(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.SetResolver(&_OpenRegistrar.TransactOpts, label, resolver, nodeId)
}

// SetResolver is a paid mutator transaction binding the contract method 0xa9f2a1b2.
//
// Solidity: function setResolver(label bytes32, resolver address, nodeId bytes12) returns()
func (_OpenRegistrar *OpenRegistrarTransactorSession) SetResolver(label [32]byte, resolver common.Address, nodeId [12]byte) (*types.Transaction, error) {
	return _OpenRegistrar.Contract.SetResolver(&_OpenRegistrar.TransactOpts, label, resolver, nodeId)
}
