// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SafeMathABI is the input ABI used to generate the binding from.
const SafeMathABI = "[]"

// SafeMathBin is the compiled bytecode used for deploying new contracts.
const SafeMathBin = `0x604c602c600b82828239805160001a60731460008114601c57601e565bfe5b5030600052607381538281f30073000000000000000000000000000000000000000030146060604052600080fd00a165627a7a72305820b9407d48ebc7efee5c9f08b3b3a957df2939281f5913225e8c1291f069b900490029`

// DeploySafeMath deploys a new Ethereum contract, binding an instance of SafeMath to it.
func DeploySafeMath(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SafeMath, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeMathABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SafeMathBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SafeMath{SafeMathCaller: SafeMathCaller{contract: contract}, SafeMathTransactor: SafeMathTransactor{contract: contract}, SafeMathFilterer: SafeMathFilterer{contract: contract}}, nil
}

// SafeMath is an auto generated Go binding around an Ethereum contract.
type SafeMath struct {
	SafeMathCaller     // Read-only binding to the contract
	SafeMathTransactor // Write-only binding to the contract
	SafeMathFilterer   // Log filterer for contract events
}

// SafeMathCaller is an auto generated read-only Go binding around an Ethereum contract.
type SafeMathCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SafeMathTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SafeMathFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SafeMathSession struct {
	Contract     *SafeMath         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SafeMathCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SafeMathCallerSession struct {
	Contract *SafeMathCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// SafeMathTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SafeMathTransactorSession struct {
	Contract     *SafeMathTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// SafeMathRaw is an auto generated low-level Go binding around an Ethereum contract.
type SafeMathRaw struct {
	Contract *SafeMath // Generic contract binding to access the raw methods on
}

// SafeMathCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SafeMathCallerRaw struct {
	Contract *SafeMathCaller // Generic read-only contract binding to access the raw methods on
}

// SafeMathTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SafeMathTransactorRaw struct {
	Contract *SafeMathTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSafeMath creates a new instance of SafeMath, bound to a specific deployed contract.
func NewSafeMath(address common.Address, backend bind.ContractBackend) (*SafeMath, error) {
	contract, err := bindSafeMath(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SafeMath{SafeMathCaller: SafeMathCaller{contract: contract}, SafeMathTransactor: SafeMathTransactor{contract: contract}, SafeMathFilterer: SafeMathFilterer{contract: contract}}, nil
}

// NewSafeMathCaller creates a new read-only instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathCaller(address common.Address, caller bind.ContractCaller) (*SafeMathCaller, error) {
	contract, err := bindSafeMath(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SafeMathCaller{contract: contract}, nil
}

// NewSafeMathTransactor creates a new write-only instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathTransactor(address common.Address, transactor bind.ContractTransactor) (*SafeMathTransactor, error) {
	contract, err := bindSafeMath(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SafeMathTransactor{contract: contract}, nil
}

// NewSafeMathFilterer creates a new log filterer instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathFilterer(address common.Address, filterer bind.ContractFilterer) (*SafeMathFilterer, error) {
	contract, err := bindSafeMath(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SafeMathFilterer{contract: contract}, nil
}

// bindSafeMath binds a generic wrapper to an already deployed contract.
func bindSafeMath(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeMathABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeMath *SafeMathRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SafeMath.Contract.SafeMathCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeMath *SafeMathRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeMath.Contract.SafeMathTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeMath *SafeMathRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeMath.Contract.SafeMathTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeMath *SafeMathCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SafeMath.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeMath *SafeMathTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeMath.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeMath *SafeMathTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeMath.Contract.contract.Transact(opts, method, params...)
}

// TomoRandomizeABI is the input ABI used to generate the binding from.
const TomoRandomizeABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"_validator\",\"type\":\"address\"}],\"name\":\"getSecret\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_secret\",\"type\":\"bytes32[]\"}],\"name\":\"setSecret\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"randomNumber\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_validator\",\"type\":\"address\"}],\"name\":\"getOpening\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_opening\",\"type\":\"bytes32\"}],\"name\":\"setOpening\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_randomNumber\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// TomoRandomizeBin is the compiled bytecode used for deploying new contracts.
const TomoRandomizeBin = `0x6060604052341561000f57600080fd5b604051602080610359833981016040528080516000555050610323806100366000396000f30060606040526004361061006c5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663284180fc811461007157806334d38600146100e3578063ccbac9f514610134578063d442d6cc14610159578063e11f5ba214610178575b600080fd5b341561007c57600080fd5b610090600160a060020a036004351661018e565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156100cf5780820151838201526020016100b7565b505050509050019250505060405180910390f35b34156100ee57600080fd5b610132600460248135818101908301358060208181020160405190810160405280939291908181526020018383602002808284375094965061021295505050505050565b005b341561013f57600080fd5b61014761023f565b60405190815260200160405180910390f35b341561016457600080fd5b610147600160a060020a0360043516610245565b341561018357600080fd5b610132600435610260565b61019661027b565b6001600083600160a060020a0316600160a060020a0316815260200190815260200160002080548060200260200160405190810160405280929190818152602001828054801561020657602002820191906000526020600020905b815481526001909101906020018083116101f1575b50505050509050919050565b600160a060020a033316600090815260016020526040902081805161023b92916020019061028d565b5050565b60005481565b600160a060020a031660009081526002602052604090205490565b600160a060020a033316600090815260026020526040902055565b60206040519081016040526000815290565b8280548282559060005260206000209081019282156102ca579160200282015b828111156102ca57825182556020909201916001909101906102ad565b506102d69291506102da565b5090565b6102f491905b808211156102d657600081556001016102e0565b905600a165627a7a7230582043a345787b9942e3361515ce1a0a4dbaab0ed6ce42c6ca0344119134351d9ffe0029`

// DeployTomoRandomize deploys a new Ethereum contract, binding an instance of TomoRandomize to it.
func DeployTomoRandomize(auth *bind.TransactOpts, backend bind.ContractBackend, _randomNumber *big.Int) (common.Address, *types.Transaction, *TomoRandomize, error) {
	parsed, err := abi.JSON(strings.NewReader(TomoRandomizeABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(TomoRandomizeBin), backend, _randomNumber)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TomoRandomize{TomoRandomizeCaller: TomoRandomizeCaller{contract: contract}, TomoRandomizeTransactor: TomoRandomizeTransactor{contract: contract}, TomoRandomizeFilterer: TomoRandomizeFilterer{contract: contract}}, nil
}

// TomoRandomize is an auto generated Go binding around an Ethereum contract.
type TomoRandomize struct {
	TomoRandomizeCaller     // Read-only binding to the contract
	TomoRandomizeTransactor // Write-only binding to the contract
	TomoRandomizeFilterer   // Log filterer for contract events
}

// TomoRandomizeCaller is an auto generated read-only Go binding around an Ethereum contract.
type TomoRandomizeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TomoRandomizeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TomoRandomizeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TomoRandomizeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TomoRandomizeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TomoRandomizeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TomoRandomizeSession struct {
	Contract     *TomoRandomize    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TomoRandomizeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TomoRandomizeCallerSession struct {
	Contract *TomoRandomizeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// TomoRandomizeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TomoRandomizeTransactorSession struct {
	Contract     *TomoRandomizeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// TomoRandomizeRaw is an auto generated low-level Go binding around an Ethereum contract.
type TomoRandomizeRaw struct {
	Contract *TomoRandomize // Generic contract binding to access the raw methods on
}

// TomoRandomizeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TomoRandomizeCallerRaw struct {
	Contract *TomoRandomizeCaller // Generic read-only contract binding to access the raw methods on
}

// TomoRandomizeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TomoRandomizeTransactorRaw struct {
	Contract *TomoRandomizeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTomoRandomize creates a new instance of TomoRandomize, bound to a specific deployed contract.
func NewTomoRandomize(address common.Address, backend bind.ContractBackend) (*TomoRandomize, error) {
	contract, err := bindTomoRandomize(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TomoRandomize{TomoRandomizeCaller: TomoRandomizeCaller{contract: contract}, TomoRandomizeTransactor: TomoRandomizeTransactor{contract: contract}, TomoRandomizeFilterer: TomoRandomizeFilterer{contract: contract}}, nil
}

// NewTomoRandomizeCaller creates a new read-only instance of TomoRandomize, bound to a specific deployed contract.
func NewTomoRandomizeCaller(address common.Address, caller bind.ContractCaller) (*TomoRandomizeCaller, error) {
	contract, err := bindTomoRandomize(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TomoRandomizeCaller{contract: contract}, nil
}

// NewTomoRandomizeTransactor creates a new write-only instance of TomoRandomize, bound to a specific deployed contract.
func NewTomoRandomizeTransactor(address common.Address, transactor bind.ContractTransactor) (*TomoRandomizeTransactor, error) {
	contract, err := bindTomoRandomize(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TomoRandomizeTransactor{contract: contract}, nil
}

// NewTomoRandomizeFilterer creates a new log filterer instance of TomoRandomize, bound to a specific deployed contract.
func NewTomoRandomizeFilterer(address common.Address, filterer bind.ContractFilterer) (*TomoRandomizeFilterer, error) {
	contract, err := bindTomoRandomize(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TomoRandomizeFilterer{contract: contract}, nil
}

// bindTomoRandomize binds a generic wrapper to an already deployed contract.
func bindTomoRandomize(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TomoRandomizeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TomoRandomize *TomoRandomizeRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _TomoRandomize.Contract.TomoRandomizeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TomoRandomize *TomoRandomizeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TomoRandomize.Contract.TomoRandomizeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TomoRandomize *TomoRandomizeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TomoRandomize.Contract.TomoRandomizeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TomoRandomize *TomoRandomizeCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _TomoRandomize.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TomoRandomize *TomoRandomizeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TomoRandomize.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TomoRandomize *TomoRandomizeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TomoRandomize.Contract.contract.Transact(opts, method, params...)
}

// GetOpening is a free data retrieval call binding the contract method 0xd442d6cc.
//
// Solidity: function getOpening(_validator address) constant returns(bytes32)
func (_TomoRandomize *TomoRandomizeCaller) GetOpening(opts *bind.CallOpts, _validator common.Address) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _TomoRandomize.contract.Call(opts, out, "getOpening", _validator)
	return *ret0, err
}

// GetOpening is a free data retrieval call binding the contract method 0xd442d6cc.
//
// Solidity: function getOpening(_validator address) constant returns(bytes32)
func (_TomoRandomize *TomoRandomizeSession) GetOpening(_validator common.Address) ([32]byte, error) {
	return _TomoRandomize.Contract.GetOpening(&_TomoRandomize.CallOpts, _validator)
}

// GetOpening is a free data retrieval call binding the contract method 0xd442d6cc.
//
// Solidity: function getOpening(_validator address) constant returns(bytes32)
func (_TomoRandomize *TomoRandomizeCallerSession) GetOpening(_validator common.Address) ([32]byte, error) {
	return _TomoRandomize.Contract.GetOpening(&_TomoRandomize.CallOpts, _validator)
}

// GetSecret is a free data retrieval call binding the contract method 0x284180fc.
//
// Solidity: function getSecret(_validator address) constant returns(bytes32[])
func (_TomoRandomize *TomoRandomizeCaller) GetSecret(opts *bind.CallOpts, _validator common.Address) ([][32]byte, error) {
	var (
		ret0 = new([][32]byte)
	)
	out := ret0
	err := _TomoRandomize.contract.Call(opts, out, "getSecret", _validator)
	return *ret0, err
}

// GetSecret is a free data retrieval call binding the contract method 0x284180fc.
//
// Solidity: function getSecret(_validator address) constant returns(bytes32[])
func (_TomoRandomize *TomoRandomizeSession) GetSecret(_validator common.Address) ([][32]byte, error) {
	return _TomoRandomize.Contract.GetSecret(&_TomoRandomize.CallOpts, _validator)
}

// GetSecret is a free data retrieval call binding the contract method 0x284180fc.
//
// Solidity: function getSecret(_validator address) constant returns(bytes32[])
func (_TomoRandomize *TomoRandomizeCallerSession) GetSecret(_validator common.Address) ([][32]byte, error) {
	return _TomoRandomize.Contract.GetSecret(&_TomoRandomize.CallOpts, _validator)
}

// RandomNumber is a free data retrieval call binding the contract method 0xccbac9f5.
//
// Solidity: function randomNumber() constant returns(uint256)
func (_TomoRandomize *TomoRandomizeCaller) RandomNumber(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _TomoRandomize.contract.Call(opts, out, "randomNumber")
	return *ret0, err
}

// RandomNumber is a free data retrieval call binding the contract method 0xccbac9f5.
//
// Solidity: function randomNumber() constant returns(uint256)
func (_TomoRandomize *TomoRandomizeSession) RandomNumber() (*big.Int, error) {
	return _TomoRandomize.Contract.RandomNumber(&_TomoRandomize.CallOpts)
}

// RandomNumber is a free data retrieval call binding the contract method 0xccbac9f5.
//
// Solidity: function randomNumber() constant returns(uint256)
func (_TomoRandomize *TomoRandomizeCallerSession) RandomNumber() (*big.Int, error) {
	return _TomoRandomize.Contract.RandomNumber(&_TomoRandomize.CallOpts)
}

// SetOpening is a paid mutator transaction binding the contract method 0xe11f5ba2.
//
// Solidity: function setOpening(_opening bytes32) returns()
func (_TomoRandomize *TomoRandomizeTransactor) SetOpening(opts *bind.TransactOpts, _opening [32]byte) (*types.Transaction, error) {
	return _TomoRandomize.contract.Transact(opts, "setOpening", _opening)
}

// SetOpening is a paid mutator transaction binding the contract method 0xe11f5ba2.
//
// Solidity: function setOpening(_opening bytes32) returns()
func (_TomoRandomize *TomoRandomizeSession) SetOpening(_opening [32]byte) (*types.Transaction, error) {
	return _TomoRandomize.Contract.SetOpening(&_TomoRandomize.TransactOpts, _opening)
}

// SetOpening is a paid mutator transaction binding the contract method 0xe11f5ba2.
//
// Solidity: function setOpening(_opening bytes32) returns()
func (_TomoRandomize *TomoRandomizeTransactorSession) SetOpening(_opening [32]byte) (*types.Transaction, error) {
	return _TomoRandomize.Contract.SetOpening(&_TomoRandomize.TransactOpts, _opening)
}

// SetSecret is a paid mutator transaction binding the contract method 0x34d38600.
//
// Solidity: function setSecret(_secret bytes32[]) returns()
func (_TomoRandomize *TomoRandomizeTransactor) SetSecret(opts *bind.TransactOpts, _secret [][32]byte) (*types.Transaction, error) {
	return _TomoRandomize.contract.Transact(opts, "setSecret", _secret)
}

// SetSecret is a paid mutator transaction binding the contract method 0x34d38600.
//
// Solidity: function setSecret(_secret bytes32[]) returns()
func (_TomoRandomize *TomoRandomizeSession) SetSecret(_secret [][32]byte) (*types.Transaction, error) {
	return _TomoRandomize.Contract.SetSecret(&_TomoRandomize.TransactOpts, _secret)
}

// SetSecret is a paid mutator transaction binding the contract method 0x34d38600.
//
// Solidity: function setSecret(_secret bytes32[]) returns()
func (_TomoRandomize *TomoRandomizeTransactorSession) SetSecret(_secret [][32]byte) (*types.Transaction, error) {
	return _TomoRandomize.Contract.SetSecret(&_TomoRandomize.TransactOpts, _secret)
}
