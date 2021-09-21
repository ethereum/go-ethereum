// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"strings"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
)

// Base1ABI is the input ABI used to generate the binding from.
const Base1ABI = "[{\"inputs\":[],\"name\":\"foo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Base1Bin is the compiled bytecode used for deploying new contracts.
const Base1Bin = `0x6080604052348015600f57600080fd5b50606d80601d6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063c298557814602d575b600080fd5b60336035565b005b56fea2646970667358221220861aecb7678c5118a12be7047c66ea27b4e28b80d5d5d4ad1813a7b08983adb664736f6c634300060a0033`

// DeployBase1 deploys a new Ethereum contract, binding an instance of Base1 to it.
func DeployBase1(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Base1, error) {
	parsed, err := abi.JSON(strings.NewReader(Base1ABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(Base1Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Base1{Base1Caller: Base1Caller{contract: contract}, Base1Transactor: Base1Transactor{contract: contract}, Base1Filterer: Base1Filterer{contract: contract}}, nil
}

// Base1 is an auto generated Go binding around an Ethereum contract.
type Base1 struct {
	Base1Caller     // Read-only binding to the contract
	Base1Transactor // Write-only binding to the contract
	Base1Filterer   // Log filterer for contract events
}

// Base1Caller is an auto generated read-only Go binding around an Ethereum contract.
type Base1Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Base1Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Base1Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Base1Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Base1Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Base1Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Base1Session struct {
	Contract     *Base1            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Base1CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Base1CallerSession struct {
	Contract *Base1Caller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// Base1TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Base1TransactorSession struct {
	Contract     *Base1Transactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Base1Raw is an auto generated low-level Go binding around an Ethereum contract.
type Base1Raw struct {
	Contract *Base1 // Generic contract binding to access the raw methods on
}

// Base1CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Base1CallerRaw struct {
	Contract *Base1Caller // Generic read-only contract binding to access the raw methods on
}

// Base1TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Base1TransactorRaw struct {
	Contract *Base1Transactor // Generic write-only contract binding to access the raw methods on
}

// NewBase1 creates a new instance of Base1, bound to a specific deployed contract.
func NewBase1(address common.Address, backend bind.ContractBackend) (*Base1, error) {
	contract, err := bindBase1(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Base1{Base1Caller: Base1Caller{contract: contract}, Base1Transactor: Base1Transactor{contract: contract}, Base1Filterer: Base1Filterer{contract: contract}}, nil
}

// NewBase1Caller creates a new read-only instance of Base1, bound to a specific deployed contract.
func NewBase1Caller(address common.Address, caller bind.ContractCaller) (*Base1Caller, error) {
	contract, err := bindBase1(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Base1Caller{contract: contract}, nil
}

// NewBase1Transactor creates a new write-only instance of Base1, bound to a specific deployed contract.
func NewBase1Transactor(address common.Address, transactor bind.ContractTransactor) (*Base1Transactor, error) {
	contract, err := bindBase1(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Base1Transactor{contract: contract}, nil
}

// NewBase1Filterer creates a new log filterer instance of Base1, bound to a specific deployed contract.
func NewBase1Filterer(address common.Address, filterer bind.ContractFilterer) (*Base1Filterer, error) {
	contract, err := bindBase1(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Base1Filterer{contract: contract}, nil
}

// bindBase1 binds a generic wrapper to an already deployed contract.
func bindBase1(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(Base1ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Base1 *Base1Raw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Base1.Contract.Base1Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Base1 *Base1Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Base1.Contract.Base1Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Base1 *Base1Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Base1.Contract.Base1Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Base1 *Base1CallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Base1.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Base1 *Base1TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Base1.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Base1 *Base1TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Base1.Contract.contract.Transact(opts, method, params...)
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Base1 *Base1Transactor) Foo(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Base1.contract.Transact(opts, "foo")
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Base1 *Base1Session) Foo() (*types.Transaction, error) {
	return _Base1.Contract.Foo(&_Base1.TransactOpts)
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Base1 *Base1TransactorSession) Foo() (*types.Transaction, error) {
	return _Base1.Contract.Foo(&_Base1.TransactOpts)
}

// Base2ABI is the input ABI used to generate the binding from.
const Base2ABI = "[{\"inputs\":[],\"name\":\"foo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Base2Bin is the compiled bytecode used for deploying new contracts.
const Base2Bin = `0x6080604052348015600f57600080fd5b50606d80601d6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063c298557814602d575b600080fd5b60336035565b005b56fea26469706673582212205b8cfeb4357fea7b0f1d1c30727a20d5e54b63b328315e480e275c07ca89189564736f6c634300060a0033`

// DeployBase2 deploys a new Ethereum contract, binding an instance of Base2 to it.
func DeployBase2(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Base2, error) {
	parsed, err := abi.JSON(strings.NewReader(Base2ABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(Base2Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Base2{Base2Caller: Base2Caller{contract: contract}, Base2Transactor: Base2Transactor{contract: contract}, Base2Filterer: Base2Filterer{contract: contract}}, nil
}

// Base2 is an auto generated Go binding around an Ethereum contract.
type Base2 struct {
	Base2Caller     // Read-only binding to the contract
	Base2Transactor // Write-only binding to the contract
	Base2Filterer   // Log filterer for contract events
}

// Base2Caller is an auto generated read-only Go binding around an Ethereum contract.
type Base2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Base2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Base2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Base2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Base2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Base2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Base2Session struct {
	Contract     *Base2            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Base2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Base2CallerSession struct {
	Contract *Base2Caller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// Base2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Base2TransactorSession struct {
	Contract     *Base2Transactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Base2Raw is an auto generated low-level Go binding around an Ethereum contract.
type Base2Raw struct {
	Contract *Base2 // Generic contract binding to access the raw methods on
}

// Base2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Base2CallerRaw struct {
	Contract *Base2Caller // Generic read-only contract binding to access the raw methods on
}

// Base2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Base2TransactorRaw struct {
	Contract *Base2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewBase2 creates a new instance of Base2, bound to a specific deployed contract.
func NewBase2(address common.Address, backend bind.ContractBackend) (*Base2, error) {
	contract, err := bindBase2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Base2{Base2Caller: Base2Caller{contract: contract}, Base2Transactor: Base2Transactor{contract: contract}, Base2Filterer: Base2Filterer{contract: contract}}, nil
}

// NewBase2Caller creates a new read-only instance of Base2, bound to a specific deployed contract.
func NewBase2Caller(address common.Address, caller bind.ContractCaller) (*Base2Caller, error) {
	contract, err := bindBase2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Base2Caller{contract: contract}, nil
}

// NewBase2Transactor creates a new write-only instance of Base2, bound to a specific deployed contract.
func NewBase2Transactor(address common.Address, transactor bind.ContractTransactor) (*Base2Transactor, error) {
	contract, err := bindBase2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Base2Transactor{contract: contract}, nil
}

// NewBase2Filterer creates a new log filterer instance of Base2, bound to a specific deployed contract.
func NewBase2Filterer(address common.Address, filterer bind.ContractFilterer) (*Base2Filterer, error) {
	contract, err := bindBase2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Base2Filterer{contract: contract}, nil
}

// bindBase2 binds a generic wrapper to an already deployed contract.
func bindBase2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(Base2ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Base2 *Base2Raw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Base2.Contract.Base2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Base2 *Base2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Base2.Contract.Base2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Base2 *Base2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Base2.Contract.Base2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Base2 *Base2CallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Base2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Base2 *Base2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Base2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Base2 *Base2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Base2.Contract.contract.Transact(opts, method, params...)
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Base2 *Base2Transactor) Foo(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Base2.contract.Transact(opts, "foo")
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Base2 *Base2Session) Foo() (*types.Transaction, error) {
	return _Base2.Contract.Foo(&_Base2.TransactOpts)
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Base2 *Base2TransactorSession) Foo() (*types.Transaction, error) {
	return _Base2.Contract.Foo(&_Base2.TransactOpts)
}

// InheritedABI is the input ABI used to generate the binding from.
const InheritedABI = "[{\"inputs\":[],\"name\":\"foo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// InheritedBin is the compiled bytecode used for deploying new contracts.
const InheritedBin = `0x6080604052348015600f57600080fd5b50606d80601d6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063c298557814602d575b600080fd5b60336035565b005b56fea2646970667358221220bbe212c6ad2a1b1546352d1975164c6f0fb7b6d29285b29b7d444d7bc2d8198d64736f6c634300060a0033`

// DeployInherited deploys a new Ethereum contract, binding an instance of Inherited to it.
func DeployInherited(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Inherited, error) {
	parsed, err := abi.JSON(strings.NewReader(InheritedABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(InheritedBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Inherited{InheritedCaller: InheritedCaller{contract: contract}, InheritedTransactor: InheritedTransactor{contract: contract}, InheritedFilterer: InheritedFilterer{contract: contract}}, nil
}

// Inherited is an auto generated Go binding around an Ethereum contract.
type Inherited struct {
	InheritedCaller     // Read-only binding to the contract
	InheritedTransactor // Write-only binding to the contract
	InheritedFilterer   // Log filterer for contract events
}

// InheritedCaller is an auto generated read-only Go binding around an Ethereum contract.
type InheritedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InheritedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InheritedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InheritedFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InheritedFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InheritedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InheritedSession struct {
	Contract     *Inherited        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InheritedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InheritedCallerSession struct {
	Contract *InheritedCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// InheritedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InheritedTransactorSession struct {
	Contract     *InheritedTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// InheritedRaw is an auto generated low-level Go binding around an Ethereum contract.
type InheritedRaw struct {
	Contract *Inherited // Generic contract binding to access the raw methods on
}

// InheritedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InheritedCallerRaw struct {
	Contract *InheritedCaller // Generic read-only contract binding to access the raw methods on
}

// InheritedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InheritedTransactorRaw struct {
	Contract *InheritedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInherited creates a new instance of Inherited, bound to a specific deployed contract.
func NewInherited(address common.Address, backend bind.ContractBackend) (*Inherited, error) {
	contract, err := bindInherited(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Inherited{InheritedCaller: InheritedCaller{contract: contract}, InheritedTransactor: InheritedTransactor{contract: contract}, InheritedFilterer: InheritedFilterer{contract: contract}}, nil
}

// NewInheritedCaller creates a new read-only instance of Inherited, bound to a specific deployed contract.
func NewInheritedCaller(address common.Address, caller bind.ContractCaller) (*InheritedCaller, error) {
	contract, err := bindInherited(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InheritedCaller{contract: contract}, nil
}

// NewInheritedTransactor creates a new write-only instance of Inherited, bound to a specific deployed contract.
func NewInheritedTransactor(address common.Address, transactor bind.ContractTransactor) (*InheritedTransactor, error) {
	contract, err := bindInherited(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InheritedTransactor{contract: contract}, nil
}

// NewInheritedFilterer creates a new log filterer instance of Inherited, bound to a specific deployed contract.
func NewInheritedFilterer(address common.Address, filterer bind.ContractFilterer) (*InheritedFilterer, error) {
	contract, err := bindInherited(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InheritedFilterer{contract: contract}, nil
}

// bindInherited binds a generic wrapper to an already deployed contract.
func bindInherited(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(InheritedABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Inherited *InheritedRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Inherited.Contract.InheritedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Inherited *InheritedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inherited.Contract.InheritedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Inherited *InheritedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Inherited.Contract.InheritedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Inherited *InheritedCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Inherited.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Inherited *InheritedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inherited.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Inherited *InheritedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Inherited.Contract.contract.Transact(opts, method, params...)
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Inherited *InheritedTransactor) Foo(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inherited.contract.Transact(opts, "foo")
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Inherited *InheritedSession) Foo() (*types.Transaction, error) {
	return _Inherited.Contract.Foo(&_Inherited.TransactOpts)
}

// Foo is a paid mutator transaction binding the contract method 0xc2985578.
//
// Solidity: function foo() returns()
func (_Inherited *InheritedTransactorSession) Foo() (*types.Transaction, error) {
	return _Inherited.Contract.Foo(&_Inherited.TransactOpts)
}
