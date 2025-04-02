// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package wrappeda0gibase

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// Supply is an auto generated low-level Go binding around an user-defined struct.
type Supply struct {
	Cap           *big.Int
	InitialSupply *big.Int
	Supply        *big.Int
}

// Wrappeda0gibaseMetaData contains all meta data concerning the Wrappeda0gibase contract.
var Wrappeda0gibaseMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"minter\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getWA0GI\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"minter\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"minter\",\"type\":\"address\"}],\"name\":\"minterSupply\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"cap\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"initialSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"supply\",\"type\":\"uint256\"}],\"internalType\":\"structSupply\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"minter\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"cap\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"initialSupply\",\"type\":\"uint256\"}],\"name\":\"setMinterCap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// Wrappeda0gibaseABI is the input ABI used to generate the binding from.
// Deprecated: Use Wrappeda0gibaseMetaData.ABI instead.
var Wrappeda0gibaseABI = Wrappeda0gibaseMetaData.ABI

// Wrappeda0gibase is an auto generated Go binding around an Ethereum contract.
type Wrappeda0gibase struct {
	Wrappeda0gibaseCaller     // Read-only binding to the contract
	Wrappeda0gibaseTransactor // Write-only binding to the contract
	Wrappeda0gibaseFilterer   // Log filterer for contract events
}

// Wrappeda0gibaseCaller is an auto generated read-only Go binding around an Ethereum contract.
type Wrappeda0gibaseCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Wrappeda0gibaseTransactor is an auto generated write-only Go binding around an Ethereum contract.
type Wrappeda0gibaseTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Wrappeda0gibaseFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Wrappeda0gibaseFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Wrappeda0gibaseSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Wrappeda0gibaseSession struct {
	Contract     *Wrappeda0gibase  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Wrappeda0gibaseCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Wrappeda0gibaseCallerSession struct {
	Contract *Wrappeda0gibaseCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// Wrappeda0gibaseTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Wrappeda0gibaseTransactorSession struct {
	Contract     *Wrappeda0gibaseTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// Wrappeda0gibaseRaw is an auto generated low-level Go binding around an Ethereum contract.
type Wrappeda0gibaseRaw struct {
	Contract *Wrappeda0gibase // Generic contract binding to access the raw methods on
}

// Wrappeda0gibaseCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Wrappeda0gibaseCallerRaw struct {
	Contract *Wrappeda0gibaseCaller // Generic read-only contract binding to access the raw methods on
}

// Wrappeda0gibaseTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Wrappeda0gibaseTransactorRaw struct {
	Contract *Wrappeda0gibaseTransactor // Generic write-only contract binding to access the raw methods on
}

// NewWrappeda0gibase creates a new instance of Wrappeda0gibase, bound to a specific deployed contract.
func NewWrappeda0gibase(address common.Address, backend bind.ContractBackend) (*Wrappeda0gibase, error) {
	contract, err := bindWrappeda0gibase(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Wrappeda0gibase{Wrappeda0gibaseCaller: Wrappeda0gibaseCaller{contract: contract}, Wrappeda0gibaseTransactor: Wrappeda0gibaseTransactor{contract: contract}, Wrappeda0gibaseFilterer: Wrappeda0gibaseFilterer{contract: contract}}, nil
}

// NewWrappeda0gibaseCaller creates a new read-only instance of Wrappeda0gibase, bound to a specific deployed contract.
func NewWrappeda0gibaseCaller(address common.Address, caller bind.ContractCaller) (*Wrappeda0gibaseCaller, error) {
	contract, err := bindWrappeda0gibase(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Wrappeda0gibaseCaller{contract: contract}, nil
}

// NewWrappeda0gibaseTransactor creates a new write-only instance of Wrappeda0gibase, bound to a specific deployed contract.
func NewWrappeda0gibaseTransactor(address common.Address, transactor bind.ContractTransactor) (*Wrappeda0gibaseTransactor, error) {
	contract, err := bindWrappeda0gibase(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Wrappeda0gibaseTransactor{contract: contract}, nil
}

// NewWrappeda0gibaseFilterer creates a new log filterer instance of Wrappeda0gibase, bound to a specific deployed contract.
func NewWrappeda0gibaseFilterer(address common.Address, filterer bind.ContractFilterer) (*Wrappeda0gibaseFilterer, error) {
	contract, err := bindWrappeda0gibase(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Wrappeda0gibaseFilterer{contract: contract}, nil
}

// bindWrappeda0gibase binds a generic wrapper to an already deployed contract.
func bindWrappeda0gibase(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(Wrappeda0gibaseABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Wrappeda0gibase *Wrappeda0gibaseRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Wrappeda0gibase.Contract.Wrappeda0gibaseCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Wrappeda0gibase *Wrappeda0gibaseRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.Wrappeda0gibaseTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Wrappeda0gibase *Wrappeda0gibaseRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.Wrappeda0gibaseTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Wrappeda0gibase *Wrappeda0gibaseCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Wrappeda0gibase.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Wrappeda0gibase *Wrappeda0gibaseTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Wrappeda0gibase *Wrappeda0gibaseTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.contract.Transact(opts, method, params...)
}

// GetWA0GI is a free data retrieval call binding the contract method 0xa9283a7a.
//
// Solidity: function getWA0GI() view returns(address)
func (_Wrappeda0gibase *Wrappeda0gibaseCaller) GetWA0GI(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Wrappeda0gibase.contract.Call(opts, &out, "getWA0GI")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetWA0GI is a free data retrieval call binding the contract method 0xa9283a7a.
//
// Solidity: function getWA0GI() view returns(address)
func (_Wrappeda0gibase *Wrappeda0gibaseSession) GetWA0GI() (common.Address, error) {
	return _Wrappeda0gibase.Contract.GetWA0GI(&_Wrappeda0gibase.CallOpts)
}

// GetWA0GI is a free data retrieval call binding the contract method 0xa9283a7a.
//
// Solidity: function getWA0GI() view returns(address)
func (_Wrappeda0gibase *Wrappeda0gibaseCallerSession) GetWA0GI() (common.Address, error) {
	return _Wrappeda0gibase.Contract.GetWA0GI(&_Wrappeda0gibase.CallOpts)
}

// MinterSupply is a free data retrieval call binding the contract method 0x95609212.
//
// Solidity: function minterSupply(address minter) view returns((uint256,uint256,uint256))
func (_Wrappeda0gibase *Wrappeda0gibaseCaller) MinterSupply(opts *bind.CallOpts, minter common.Address) (Supply, error) {
	var out []interface{}
	err := _Wrappeda0gibase.contract.Call(opts, &out, "minterSupply", minter)

	if err != nil {
		return *new(Supply), err
	}

	out0 := *abi.ConvertType(out[0], new(Supply)).(*Supply)

	return out0, err

}

// MinterSupply is a free data retrieval call binding the contract method 0x95609212.
//
// Solidity: function minterSupply(address minter) view returns((uint256,uint256,uint256))
func (_Wrappeda0gibase *Wrappeda0gibaseSession) MinterSupply(minter common.Address) (Supply, error) {
	return _Wrappeda0gibase.Contract.MinterSupply(&_Wrappeda0gibase.CallOpts, minter)
}

// MinterSupply is a free data retrieval call binding the contract method 0x95609212.
//
// Solidity: function minterSupply(address minter) view returns((uint256,uint256,uint256))
func (_Wrappeda0gibase *Wrappeda0gibaseCallerSession) MinterSupply(minter common.Address) (Supply, error) {
	return _Wrappeda0gibase.Contract.MinterSupply(&_Wrappeda0gibase.CallOpts, minter)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address minter, uint256 amount) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseTransactor) Burn(opts *bind.TransactOpts, minter common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.contract.Transact(opts, "burn", minter, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address minter, uint256 amount) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseSession) Burn(minter common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.Burn(&_Wrappeda0gibase.TransactOpts, minter, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address minter, uint256 amount) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseTransactorSession) Burn(minter common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.Burn(&_Wrappeda0gibase.TransactOpts, minter, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address minter, uint256 amount) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseTransactor) Mint(opts *bind.TransactOpts, minter common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.contract.Transact(opts, "mint", minter, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address minter, uint256 amount) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseSession) Mint(minter common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.Mint(&_Wrappeda0gibase.TransactOpts, minter, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address minter, uint256 amount) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseTransactorSession) Mint(minter common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.Mint(&_Wrappeda0gibase.TransactOpts, minter, amount)
}

// SetMinterCap is a paid mutator transaction binding the contract method 0xdddba6c8.
//
// Solidity: function setMinterCap(address minter, uint256 cap, uint256 initialSupply) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseTransactor) SetMinterCap(opts *bind.TransactOpts, minter common.Address, cap *big.Int, initialSupply *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.contract.Transact(opts, "setMinterCap", minter, cap, initialSupply)
}

// SetMinterCap is a paid mutator transaction binding the contract method 0xdddba6c8.
//
// Solidity: function setMinterCap(address minter, uint256 cap, uint256 initialSupply) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseSession) SetMinterCap(minter common.Address, cap *big.Int, initialSupply *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.SetMinterCap(&_Wrappeda0gibase.TransactOpts, minter, cap, initialSupply)
}

// SetMinterCap is a paid mutator transaction binding the contract method 0xdddba6c8.
//
// Solidity: function setMinterCap(address minter, uint256 cap, uint256 initialSupply) returns()
func (_Wrappeda0gibase *Wrappeda0gibaseTransactorSession) SetMinterCap(minter common.Address, cap *big.Int, initialSupply *big.Int) (*types.Transaction, error) {
	return _Wrappeda0gibase.Contract.SetMinterCap(&_Wrappeda0gibase.TransactOpts, minter, cap, initialSupply)
}
