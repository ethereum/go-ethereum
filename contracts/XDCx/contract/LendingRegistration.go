// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"math/big"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
)

// LAbstractRegistrationABI is the input ABI used to generate the binding from.
const LAbstractRegistrationABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"RESIGN_REQUESTS\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"getRelayerByCoinbase\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"uint16\"},{\"name\":\"\",\"type\":\"address[]\"},{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// LAbstractRegistrationBin is the compiled bytecode used for deploying new contracts.
const LAbstractRegistrationBin = `0x`

// DeployLAbstractRegistration deploys a new Ethereum contract, binding an instance of LAbstractRegistration to it.
func DeployLAbstractRegistration(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LAbstractRegistration, error) {
	parsed, err := abi.JSON(strings.NewReader(LAbstractRegistrationABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(LAbstractRegistrationBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LAbstractRegistration{LAbstractRegistrationCaller: LAbstractRegistrationCaller{contract: contract}, LAbstractRegistrationTransactor: LAbstractRegistrationTransactor{contract: contract}, LAbstractRegistrationFilterer: LAbstractRegistrationFilterer{contract: contract}}, nil
}

// LAbstractRegistration is an auto generated Go binding around an Ethereum contract.
type LAbstractRegistration struct {
	LAbstractRegistrationCaller     // Read-only binding to the contract
	LAbstractRegistrationTransactor // Write-only binding to the contract
	LAbstractRegistrationFilterer   // Log filterer for contract events
}

// LAbstractRegistrationCaller is an auto generated read-only Go binding around an Ethereum contract.
type LAbstractRegistrationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractRegistrationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LAbstractRegistrationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractRegistrationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LAbstractRegistrationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractRegistrationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LAbstractRegistrationSession struct {
	Contract     *LAbstractRegistration // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// LAbstractRegistrationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LAbstractRegistrationCallerSession struct {
	Contract *LAbstractRegistrationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// LAbstractRegistrationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LAbstractRegistrationTransactorSession struct {
	Contract     *LAbstractRegistrationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// LAbstractRegistrationRaw is an auto generated low-level Go binding around an Ethereum contract.
type LAbstractRegistrationRaw struct {
	Contract *LAbstractRegistration // Generic contract binding to access the raw methods on
}

// LAbstractRegistrationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LAbstractRegistrationCallerRaw struct {
	Contract *LAbstractRegistrationCaller // Generic read-only contract binding to access the raw methods on
}

// LAbstractRegistrationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LAbstractRegistrationTransactorRaw struct {
	Contract *LAbstractRegistrationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLAbstractRegistration creates a new instance of LAbstractRegistration, bound to a specific deployed contract.
func NewLAbstractRegistration(address common.Address, backend bind.ContractBackend) (*LAbstractRegistration, error) {
	contract, err := bindLAbstractRegistration(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LAbstractRegistration{LAbstractRegistrationCaller: LAbstractRegistrationCaller{contract: contract}, LAbstractRegistrationTransactor: LAbstractRegistrationTransactor{contract: contract}, LAbstractRegistrationFilterer: LAbstractRegistrationFilterer{contract: contract}}, nil
}

// NewLAbstractRegistrationCaller creates a new read-only instance of LAbstractRegistration, bound to a specific deployed contract.
func NewLAbstractRegistrationCaller(address common.Address, caller bind.ContractCaller) (*LAbstractRegistrationCaller, error) {
	contract, err := bindLAbstractRegistration(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LAbstractRegistrationCaller{contract: contract}, nil
}

// NewLAbstractRegistrationTransactor creates a new write-only instance of LAbstractRegistration, bound to a specific deployed contract.
func NewLAbstractRegistrationTransactor(address common.Address, transactor bind.ContractTransactor) (*LAbstractRegistrationTransactor, error) {
	contract, err := bindLAbstractRegistration(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LAbstractRegistrationTransactor{contract: contract}, nil
}

// NewLAbstractRegistrationFilterer creates a new log filterer instance of LAbstractRegistration, bound to a specific deployed contract.
func NewLAbstractRegistrationFilterer(address common.Address, filterer bind.ContractFilterer) (*LAbstractRegistrationFilterer, error) {
	contract, err := bindLAbstractRegistration(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LAbstractRegistrationFilterer{contract: contract}, nil
}

// bindLAbstractRegistration binds a generic wrapper to an already deployed contract.
func bindLAbstractRegistration(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LAbstractRegistrationABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LAbstractRegistration *LAbstractRegistrationRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _LAbstractRegistration.Contract.LAbstractRegistrationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LAbstractRegistration *LAbstractRegistrationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LAbstractRegistration.Contract.LAbstractRegistrationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LAbstractRegistration *LAbstractRegistrationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LAbstractRegistration.Contract.LAbstractRegistrationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LAbstractRegistration *LAbstractRegistrationCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _LAbstractRegistration.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LAbstractRegistration *LAbstractRegistrationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LAbstractRegistration.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LAbstractRegistration *LAbstractRegistrationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LAbstractRegistration.Contract.contract.Transact(opts, method, params...)
}

// RESIGNREQUESTS is a free data retrieval call binding the contract method 0x500f99f7.
//
// Solidity: function RESIGN_REQUESTS( address) constant returns(uint256)
func (_LAbstractRegistration *LAbstractRegistrationCaller) RESIGNREQUESTS(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _LAbstractRegistration.contract.Call(opts, out, "RESIGN_REQUESTS", arg0)
	return *ret0, err
}

// RESIGNREQUESTS is a free data retrieval call binding the contract method 0x500f99f7.
//
// Solidity: function RESIGN_REQUESTS( address) constant returns(uint256)
func (_LAbstractRegistration *LAbstractRegistrationSession) RESIGNREQUESTS(arg0 common.Address) (*big.Int, error) {
	return _LAbstractRegistration.Contract.RESIGNREQUESTS(&_LAbstractRegistration.CallOpts, arg0)
}

// RESIGNREQUESTS is a free data retrieval call binding the contract method 0x500f99f7.
//
// Solidity: function RESIGN_REQUESTS( address) constant returns(uint256)
func (_LAbstractRegistration *LAbstractRegistrationCallerSession) RESIGNREQUESTS(arg0 common.Address) (*big.Int, error) {
	return _LAbstractRegistration.Contract.RESIGNREQUESTS(&_LAbstractRegistration.CallOpts, arg0)
}

// GetRelayerByCoinbase is a free data retrieval call binding the contract method 0x540105c7.
//
// Solidity: function getRelayerByCoinbase( address) constant returns(uint256, address, uint256, uint16, address[], address[])
func (_LAbstractRegistration *LAbstractRegistrationCaller) GetRelayerByCoinbase(opts *bind.CallOpts, arg0 common.Address) (*big.Int, common.Address, *big.Int, uint16, []common.Address, []common.Address, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new(common.Address)
		ret2 = new(*big.Int)
		ret3 = new(uint16)
		ret4 = new([]common.Address)
		ret5 = new([]common.Address)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
		ret3,
		ret4,
		ret5,
	}
	err := _LAbstractRegistration.contract.Call(opts, out, "getRelayerByCoinbase", arg0)
	return *ret0, *ret1, *ret2, *ret3, *ret4, *ret5, err
}

// GetRelayerByCoinbase is a free data retrieval call binding the contract method 0x540105c7.
//
// Solidity: function getRelayerByCoinbase( address) constant returns(uint256, address, uint256, uint16, address[], address[])
func (_LAbstractRegistration *LAbstractRegistrationSession) GetRelayerByCoinbase(arg0 common.Address) (*big.Int, common.Address, *big.Int, uint16, []common.Address, []common.Address, error) {
	return _LAbstractRegistration.Contract.GetRelayerByCoinbase(&_LAbstractRegistration.CallOpts, arg0)
}

// GetRelayerByCoinbase is a free data retrieval call binding the contract method 0x540105c7.
//
// Solidity: function getRelayerByCoinbase( address) constant returns(uint256, address, uint256, uint16, address[], address[])
func (_LAbstractRegistration *LAbstractRegistrationCallerSession) GetRelayerByCoinbase(arg0 common.Address) (*big.Int, common.Address, *big.Int, uint16, []common.Address, []common.Address, error) {
	return _LAbstractRegistration.Contract.GetRelayerByCoinbase(&_LAbstractRegistration.CallOpts, arg0)
}

// LAbstractXDCXListingABI is the input ABI used to generate the binding from.
const LAbstractXDCXListingABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"getTokenStatus\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// LAbstractXDCXListingBin is the compiled bytecode used for deploying new contracts.
const LAbstractXDCXListingBin = `0x`

// DeployLAbstractXDCXListing deploys a new Ethereum contract, binding an instance of LAbstractXDCXListing to it.
func DeployLAbstractXDCXListing(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LAbstractXDCXListing, error) {
	parsed, err := abi.JSON(strings.NewReader(LAbstractXDCXListingABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(LAbstractXDCXListingBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LAbstractXDCXListing{LAbstractXDCXListingCaller: LAbstractXDCXListingCaller{contract: contract}, LAbstractXDCXListingTransactor: LAbstractXDCXListingTransactor{contract: contract}, LAbstractXDCXListingFilterer: LAbstractXDCXListingFilterer{contract: contract}}, nil
}

// LAbstractXDCXListing is an auto generated Go binding around an Ethereum contract.
type LAbstractXDCXListing struct {
	LAbstractXDCXListingCaller     // Read-only binding to the contract
	LAbstractXDCXListingTransactor // Write-only binding to the contract
	LAbstractXDCXListingFilterer   // Log filterer for contract events
}

// LAbstractXDCXListingCaller is an auto generated read-only Go binding around an Ethereum contract.
type LAbstractXDCXListingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractXDCXListingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LAbstractXDCXListingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractXDCXListingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LAbstractXDCXListingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractXDCXListingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LAbstractXDCXListingSession struct {
	Contract     *LAbstractXDCXListing // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// LAbstractXDCXListingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LAbstractXDCXListingCallerSession struct {
	Contract *LAbstractXDCXListingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// LAbstractXDCXListingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LAbstractXDCXListingTransactorSession struct {
	Contract     *LAbstractXDCXListingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// LAbstractXDCXListingRaw is an auto generated low-level Go binding around an Ethereum contract.
type LAbstractXDCXListingRaw struct {
	Contract *LAbstractXDCXListing // Generic contract binding to access the raw methods on
}

// LAbstractXDCXListingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LAbstractXDCXListingCallerRaw struct {
	Contract *LAbstractXDCXListingCaller // Generic read-only contract binding to access the raw methods on
}

// LAbstractXDCXListingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LAbstractXDCXListingTransactorRaw struct {
	Contract *LAbstractXDCXListingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLAbstractXDCXListing creates a new instance of LAbstractXDCXListing, bound to a specific deployed contract.
func NewLAbstractXDCXListing(address common.Address, backend bind.ContractBackend) (*LAbstractXDCXListing, error) {
	contract, err := bindLAbstractXDCXListing(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LAbstractXDCXListing{LAbstractXDCXListingCaller: LAbstractXDCXListingCaller{contract: contract}, LAbstractXDCXListingTransactor: LAbstractXDCXListingTransactor{contract: contract}, LAbstractXDCXListingFilterer: LAbstractXDCXListingFilterer{contract: contract}}, nil
}

// NewLAbstractXDCXListingCaller creates a new read-only instance of LAbstractXDCXListing, bound to a specific deployed contract.
func NewLAbstractXDCXListingCaller(address common.Address, caller bind.ContractCaller) (*LAbstractXDCXListingCaller, error) {
	contract, err := bindLAbstractXDCXListing(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LAbstractXDCXListingCaller{contract: contract}, nil
}

// NewLAbstractXDCXListingTransactor creates a new write-only instance of LAbstractXDCXListing, bound to a specific deployed contract.
func NewLAbstractXDCXListingTransactor(address common.Address, transactor bind.ContractTransactor) (*LAbstractXDCXListingTransactor, error) {
	contract, err := bindLAbstractXDCXListing(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LAbstractXDCXListingTransactor{contract: contract}, nil
}

// NewLAbstractXDCXListingFilterer creates a new log filterer instance of LAbstractXDCXListing, bound to a specific deployed contract.
func NewLAbstractXDCXListingFilterer(address common.Address, filterer bind.ContractFilterer) (*LAbstractXDCXListingFilterer, error) {
	contract, err := bindLAbstractXDCXListing(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LAbstractXDCXListingFilterer{contract: contract}, nil
}

// bindLAbstractXDCXListing binds a generic wrapper to an already deployed contract.
func bindLAbstractXDCXListing(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LAbstractXDCXListingABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LAbstractXDCXListing *LAbstractXDCXListingRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _LAbstractXDCXListing.Contract.LAbstractXDCXListingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LAbstractXDCXListing *LAbstractXDCXListingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LAbstractXDCXListing.Contract.LAbstractXDCXListingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LAbstractXDCXListing *LAbstractXDCXListingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LAbstractXDCXListing.Contract.LAbstractXDCXListingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LAbstractXDCXListing *LAbstractXDCXListingCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _LAbstractXDCXListing.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LAbstractXDCXListing *LAbstractXDCXListingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LAbstractXDCXListing.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LAbstractXDCXListing *LAbstractXDCXListingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LAbstractXDCXListing.Contract.contract.Transact(opts, method, params...)
}

// GetTokenStatus is a free data retrieval call binding the contract method 0xa3ff31b5.
//
// Solidity: function getTokenStatus( address) constant returns(bool)
func (_LAbstractXDCXListing *LAbstractXDCXListingCaller) GetTokenStatus(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _LAbstractXDCXListing.contract.Call(opts, out, "getTokenStatus", arg0)
	return *ret0, err
}

// GetTokenStatus is a free data retrieval call binding the contract method 0xa3ff31b5.
//
// Solidity: function getTokenStatus( address) constant returns(bool)
func (_LAbstractXDCXListing *LAbstractXDCXListingSession) GetTokenStatus(arg0 common.Address) (bool, error) {
	return _LAbstractXDCXListing.Contract.GetTokenStatus(&_LAbstractXDCXListing.CallOpts, arg0)
}

// GetTokenStatus is a free data retrieval call binding the contract method 0xa3ff31b5.
//
// Solidity: function getTokenStatus( address) constant returns(bool)
func (_LAbstractXDCXListing *LAbstractXDCXListingCallerSession) GetTokenStatus(arg0 common.Address) (bool, error) {
	return _LAbstractXDCXListing.Contract.GetTokenStatus(&_LAbstractXDCXListing.CallOpts, arg0)
}

// LAbstractTokenTRC21ABI is the input ABI used to generate the binding from.
const LAbstractTokenTRC21ABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"issuer\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// LAbstractTokenTRC21Bin is the compiled bytecode used for deploying new contracts.
const LAbstractTokenTRC21Bin = `0x`

// DeployLAbstractTokenTRC21 deploys a new Ethereum contract, binding an instance of LAbstractTokenTRC21 to it.
func DeployLAbstractTokenTRC21(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LAbstractTokenTRC21, error) {
	parsed, err := abi.JSON(strings.NewReader(LAbstractTokenTRC21ABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(LAbstractTokenTRC21Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LAbstractTokenTRC21{LAbstractTokenTRC21Caller: LAbstractTokenTRC21Caller{contract: contract}, LAbstractTokenTRC21Transactor: LAbstractTokenTRC21Transactor{contract: contract}, LAbstractTokenTRC21Filterer: LAbstractTokenTRC21Filterer{contract: contract}}, nil
}

// LAbstractTokenTRC21 is an auto generated Go binding around an Ethereum contract.
type LAbstractTokenTRC21 struct {
	LAbstractTokenTRC21Caller     // Read-only binding to the contract
	LAbstractTokenTRC21Transactor // Write-only binding to the contract
	LAbstractTokenTRC21Filterer   // Log filterer for contract events
}

// LAbstractTokenTRC21Caller is an auto generated read-only Go binding around an Ethereum contract.
type LAbstractTokenTRC21Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractTokenTRC21Transactor is an auto generated write-only Go binding around an Ethereum contract.
type LAbstractTokenTRC21Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractTokenTRC21Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LAbstractTokenTRC21Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LAbstractTokenTRC21Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LAbstractTokenTRC21Session struct {
	Contract     *LAbstractTokenTRC21 // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// LAbstractTokenTRC21CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LAbstractTokenTRC21CallerSession struct {
	Contract *LAbstractTokenTRC21Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// LAbstractTokenTRC21TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LAbstractTokenTRC21TransactorSession struct {
	Contract     *LAbstractTokenTRC21Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// LAbstractTokenTRC21Raw is an auto generated low-level Go binding around an Ethereum contract.
type LAbstractTokenTRC21Raw struct {
	Contract *LAbstractTokenTRC21 // Generic contract binding to access the raw methods on
}

// LAbstractTokenTRC21CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LAbstractTokenTRC21CallerRaw struct {
	Contract *LAbstractTokenTRC21Caller // Generic read-only contract binding to access the raw methods on
}

// LAbstractTokenTRC21TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LAbstractTokenTRC21TransactorRaw struct {
	Contract *LAbstractTokenTRC21Transactor // Generic write-only contract binding to access the raw methods on
}

// NewLAbstractTokenTRC21 creates a new instance of LAbstractTokenTRC21, bound to a specific deployed contract.
func NewLAbstractTokenTRC21(address common.Address, backend bind.ContractBackend) (*LAbstractTokenTRC21, error) {
	contract, err := bindLAbstractTokenTRC21(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LAbstractTokenTRC21{LAbstractTokenTRC21Caller: LAbstractTokenTRC21Caller{contract: contract}, LAbstractTokenTRC21Transactor: LAbstractTokenTRC21Transactor{contract: contract}, LAbstractTokenTRC21Filterer: LAbstractTokenTRC21Filterer{contract: contract}}, nil
}

// NewLAbstractTokenTRC21Caller creates a new read-only instance of LAbstractTokenTRC21, bound to a specific deployed contract.
func NewLAbstractTokenTRC21Caller(address common.Address, caller bind.ContractCaller) (*LAbstractTokenTRC21Caller, error) {
	contract, err := bindLAbstractTokenTRC21(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LAbstractTokenTRC21Caller{contract: contract}, nil
}

// NewLAbstractTokenTRC21Transactor creates a new write-only instance of LAbstractTokenTRC21, bound to a specific deployed contract.
func NewLAbstractTokenTRC21Transactor(address common.Address, transactor bind.ContractTransactor) (*LAbstractTokenTRC21Transactor, error) {
	contract, err := bindLAbstractTokenTRC21(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LAbstractTokenTRC21Transactor{contract: contract}, nil
}

// NewLAbstractTokenTRC21Filterer creates a new log filterer instance of LAbstractTokenTRC21, bound to a specific deployed contract.
func NewLAbstractTokenTRC21Filterer(address common.Address, filterer bind.ContractFilterer) (*LAbstractTokenTRC21Filterer, error) {
	contract, err := bindLAbstractTokenTRC21(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LAbstractTokenTRC21Filterer{contract: contract}, nil
}

// bindLAbstractTokenTRC21 binds a generic wrapper to an already deployed contract.
func bindLAbstractTokenTRC21(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LAbstractTokenTRC21ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21Raw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _LAbstractTokenTRC21.Contract.LAbstractTokenTRC21Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LAbstractTokenTRC21.Contract.LAbstractTokenTRC21Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LAbstractTokenTRC21.Contract.LAbstractTokenTRC21Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21CallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _LAbstractTokenTRC21.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LAbstractTokenTRC21.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LAbstractTokenTRC21.Contract.contract.Transact(opts, method, params...)
}

// Issuer is a free data retrieval call binding the contract method 0x1d143848.
//
// Solidity: function issuer() constant returns(address)
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21Caller) Issuer(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _LAbstractTokenTRC21.contract.Call(opts, out, "issuer")
	return *ret0, err
}

// Issuer is a free data retrieval call binding the contract method 0x1d143848.
//
// Solidity: function issuer() constant returns(address)
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21Session) Issuer() (common.Address, error) {
	return _LAbstractTokenTRC21.Contract.Issuer(&_LAbstractTokenTRC21.CallOpts)
}

// Issuer is a free data retrieval call binding the contract method 0x1d143848.
//
// Solidity: function issuer() constant returns(address)
func (_LAbstractTokenTRC21 *LAbstractTokenTRC21CallerSession) Issuer() (common.Address, error) {
	return _LAbstractTokenTRC21.Contract.Issuer(&_LAbstractTokenTRC21.CallOpts)
}

// LendingABI is the input ABI used to generate the binding from.
const LendingABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"COLLATERALS\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"ORACLE_PRICE_FEEDER\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"term\",\"type\":\"uint256\"}],\"name\":\"addTerm\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"LENDINGRELAYER_LIST\",\"outputs\":[{\"name\":\"_tradeFee\",\"type\":\"uint16\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"Relayer\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"XDCXListing\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"coinbase\",\"type\":\"address\"},{\"name\":\"tradeFee\",\"type\":\"uint16\"},{\"name\":\"baseTokens\",\"type\":\"address[]\"},{\"name\":\"terms\",\"type\":\"uint256[]\"},{\"name\":\"collaterals\",\"type\":\"address[]\"}],\"name\":\"update\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"MODERATOR\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"},{\"name\":\"depositRate\",\"type\":\"uint256\"},{\"name\":\"liquidationRate\",\"type\":\"uint256\"},{\"name\":\"recallRate\",\"type\":\"uint256\"}],\"name\":\"addILOCollateral\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"coinbase\",\"type\":\"address\"},{\"name\":\"tradeFee\",\"type\":\"uint16\"}],\"name\":\"updateFee\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"moderator\",\"type\":\"address\"}],\"name\":\"changeModerator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"TERMS\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"BASES\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"COLLATERAL_LIST\",\"outputs\":[{\"name\":\"_depositRate\",\"type\":\"uint256\"},{\"name\":\"_liquidationRate\",\"type\":\"uint256\"},{\"name\":\"_recallRate\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"addBaseToken\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"},{\"name\":\"lendingToken\",\"type\":\"address\"},{\"name\":\"price\",\"type\":\"uint256\"}],\"name\":\"setCollateralPrice\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"ILO_COLLATERALS\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"feeder\",\"type\":\"address\"}],\"name\":\"changeOraclePriceFeeder\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"},{\"name\":\"depositRate\",\"type\":\"uint256\"},{\"name\":\"liquidationRate\",\"type\":\"uint256\"},{\"name\":\"recallRate\",\"type\":\"uint256\"}],\"name\":\"addCollateral\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"},{\"name\":\"lendingToken\",\"type\":\"address\"}],\"name\":\"getCollateralPrice\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"coinbase\",\"type\":\"address\"}],\"name\":\"getLendingRelayerByCoinbase\",\"outputs\":[{\"name\":\"\",\"type\":\"uint16\"},{\"name\":\"\",\"type\":\"address[]\"},{\"name\":\"\",\"type\":\"uint256[]\"},{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"r\",\"type\":\"address\"},{\"name\":\"t\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// LendingBin is the compiled bytecode used for deploying new contracts.
const LendingBin = `0x608060405234801561001057600080fd5b506040516040806124e183398101604052805160209091015160068054600160a060020a0319908116600160a060020a03948516179091556008805482169390921692909217905560098054339083168117909155600780549092161790556124638061007e6000396000f3006080604052600436106101035763ffffffff60e060020a6000350416630811f05a81146101085780630c4c2cbb1461013c5780630c655955146101515780630faf292c1461016b578063264949d8146101a357806329a4ddec146101b85780632ddada4c146101cd57806334b4e625146102aa5780633b874827146102bf5780633ea2391f146102e9578063466429211461031157806356327f57146103325780636d1dc42a1461035c578063822507011461037457806383e280d9146103b3578063acb8cd92146103d4578063b8687ec4146103fe578063c38f473f14610416578063e5eecf6814610437578063f2dbd07014610461578063fe824700146104a1575b600080fd5b34801561011457600080fd5b506101206004356105af565b60408051600160a060020a039092168252519081900360200190f35b34801561014857600080fd5b506101206105d7565b34801561015d57600080fd5b506101696004356105e6565b005b34801561017757600080fd5b5061018c600160a060020a0360043516610728565b6040805161ffff9092168252519081900360200190f35b3480156101af57600080fd5b5061012061073e565b3480156101c457600080fd5b5061012061074d565b3480156101d957600080fd5b506040805160206004604435818101358381028086018501909652808552610169958335600160a060020a0316956024803561ffff1696369695606495939492019291829185019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a99890198929750908201955093508392508501908490808284375094975061075c9650505050505050565b3480156102b657600080fd5b50610120610e9b565b3480156102cb57600080fd5b50610169600160a060020a0360043516602435604435606435610eaa565b3480156102f557600080fd5b50610169600160a060020a036004351661ffff60243516611313565b34801561031d57600080fd5b50610169600160a060020a0360043516611664565b34801561033e57600080fd5b5061034a6004356116eb565b60408051918252519081900360200190f35b34801561036857600080fd5b5061012060043561170a565b34801561038057600080fd5b50610395600160a060020a0360043516611718565b60408051938452602084019290925282820152519081900360600190f35b3480156103bf57600080fd5b50610169600160a060020a0360043516611738565b3480156103e057600080fd5b50610169600160a060020a0360043581169060243516604435611930565b34801561040a57600080fd5b50610120600435611d11565b34801561042257600080fd5b50610169600160a060020a0360043516611d1f565b34801561044357600080fd5b50610169600160a060020a0360043516602435604435606435611db8565b34801561046d57600080fd5b50610488600160a060020a03600435811690602435166120f2565b6040805192835260208301919091528051918290030190f35b3480156104ad57600080fd5b506104c2600160a060020a0360043516612129565b604051808561ffff1661ffff168152602001806020018060200180602001848103845287818151815260200191508051906020019060200280838360005b83811015610518578181015183820152602001610500565b50505050905001848103835286818151815260200191508051906020019060200280838360005b8381101561055757818101518382015260200161053f565b50505050905001848103825285818151815260200191508051906020019060200280838360005b8381101561059657818101518382015260200161057e565b5050505090500197505050505050505060405180910390f35b60028054829081106105bd57fe5b600091825260209091200154600160a060020a0316905081565b600954600160a060020a031681565b600754600160a060020a03163314610636576040805160e560020a62461bcd02815260206004820152600f60248201526000805160206123f8833981519152604482015290519081900360640190fd5b603c81101561068f576040805160e560020a62461bcd02815260206004820152600c60248201527f496e76616c6964207465726d0000000000000000000000000000000000000000604482015290519081900360640190fd5b6106e960048054806020026020016040519081016040528092919081815260200182805480156106de57602002820191906000526020600020905b8154815260200190600101908083116106ca575b505050505082612272565b151561072557600480546001810182556000919091527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b018190555b50565b60006020819052908152604090205461ffff1681565b600654600160a060020a031681565b600854600160a060020a031681565b600654604080517f540105c7000000000000000000000000000000000000000000000000000000008152600160a060020a03888116600483015291516000938493849391169163540105c791602480820192869290919082900301818387803b1580156107c857600080fd5b505af11580156107dc573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405260c081101561080557600080fd5b81516020830151604084015160608501516080860180519496939592949193928301929164010000000081111561083b57600080fd5b8201602081018481111561084e57600080fd5b815185602082028301116401000000008211171561086b57600080fd5b5050929190602001805164010000000081111561088757600080fd5b8201602081018481111561089a57600080fd5b81518560208202830111640100000000821117156108b757600080fd5b50979b505050600160a060020a038a163314965061092695505050505050576040805160e560020a62461bcd02815260206004820152601660248201527f52656c61796572206f776e657220726571756972656400000000000000000000604482015290519081900360640190fd5b600654604080517f500f99f7000000000000000000000000000000000000000000000000000000008152600160a060020a038b811660048301529151919092169163500f99f79160248083019260209291908290030181600087803b15801561098e57600080fd5b505af11580156109a2573d6000803e3d6000fd5b505050506040513d60208110156109b857600080fd5b505115610a0f576040805160e560020a62461bcd02815260206004820152601960248201527f52656c6179657220726571756972656420746f20636c6f736500000000000000604482015290519081900360640190fd5b60008761ffff1610158015610a2957506103e88761ffff16105b1515610a7f576040805160e560020a62461bcd02815260206004820152601160248201527f496e76616c696420747261646520466565000000000000000000000000000000604482015290519081900360640190fd5b8451865114610ad8576040805160e560020a62461bcd02815260206004820152601960248201527f4e6f742076616c6964206e756d626572206f66207465726d7300000000000000604482015290519081900360640190fd5b8351865114610b31576040805160e560020a62461bcd02815260206004820152601f60248201527f4e6f742076616c6964206e756d626572206f6620636f6c6c61746572616c7300604482015290519081900360640190fd5b5060009050805b8551811015610c2057610bbc6003805480602002602001604051908101604052809291908181526020018280548015610b9a57602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610b7c575b50505050508783815181101515610bad57fe5b906020019060200201516122bb565b9150600182151514610c18576040805160e560020a62461bcd02815260206004820152601560248201527f496e76616c6964206c656e64696e6720746f6b656e0000000000000000000000604482015290519081900360640190fd5b600101610b38565b5060005b8451811015610d0257610c9e6004805480602002602001604051908101604052809291908181526020018280548015610c7c57602002820191906000526020600020905b815481526020019060010190808311610c68575b50505050508683815181101515610c8f57fe5b90602001906020020151612272565b9150600182151514610cfa576040805160e560020a62461bcd02815260206004820152600c60248201527f496e76616c6964207465726d0000000000000000000000000000000000000000604482015290519081900360640190fd5b600101610c24565b5060005b8351811015610df0578351600090859083908110610d2057fe5b60209081029091010151600160a060020a031614610de857610da46005805480602002602001604051908101604052809291908181526020018280548015610d9157602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610d73575b50505050508583815181101515610bad57fe5b1515610de8576040805160e560020a62461bcd0281526020600482015260126024820152600080516020612418833981519152604482015290519081900360640190fd5b600101610d06565b6040805160808101825261ffff898116825260208083018a81528385018a905260608401899052600160a060020a038d166000908152808352949094208351815461ffff191693169290921782559251805192939192610e56926001850192019061230a565b5060408201518051610e7291600284019160209091019061236f565b5060608201518051610e8e91600384019160209091019061230a565b5050505050505050505050565b600754600160a060020a031681565b60008060648510158015610ebe5750606484115b1515610f14576040805160e560020a62461bcd02815260206004820152600d60248201527f496e76616c696420726174657300000000000000000000000000000000000000604482015290519081900360640190fd5b838511610f6b576040805160e560020a62461bcd02815260206004820152601560248201527f496e76616c6964206465706f7369742072617465730000000000000000000000604482015290519081900360640190fd5b848311610fc2576040805160e560020a62461bcd02815260206004820152601460248201527f496e76616c696420726563616c6c207261746573000000000000000000000000604482015290519081900360640190fd5b611026600280548060200260200160405190810160405280929190818152602001828054801561101b57602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610ffd575b5050505050876122bb565b1561107b576040805160e560020a62461bcd02815260206004820152601660248201527f496e76616c696420494c4f20636f6c6c61746572616c00000000000000000000604482015290519081900360640190fd5b6008546040805160e060020a63a3ff31b5028152600160a060020a0389811660048301529151919092169163a3ff31b59160248083019260209291908290030181600087803b1580156110cd57600080fd5b505af11580156110e1573d6000803e3d6000fd5b505050506040513d60208110156110f757600080fd5b50519150811515611140576040805160e560020a62461bcd0281526020600482015260126024820152600080516020612418833981519152604482015290519081900360640190fd5b85905033600160a060020a031681600160a060020a0316631d1438486040518163ffffffff1660e060020a028152600401602060405180830381600087803b15801561118b57600080fd5b505af115801561119f573d6000803e3d6000fd5b505050506040513d60208110156111b557600080fd5b5051600160a060020a031614611215576040805160e560020a62461bcd02815260206004820152601560248201527f526571756972656420746f6b656e206973737565720000000000000000000000604482015290519081900360640190fd5b604080516060810182528681526020808201878152828401878152600160a060020a038b1660009081526001808552908690209451855591519184019190915551600290920191909155600580548351818402810184019094528084526112b9939283018282801561101b57602002820191906000526020600020908154600160a060020a03168152600190910190602001808311610ffd575050505050876122bb565b151561130b57600580546001810182556000919091527f036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db0018054600160a060020a031916600160a060020a0388161790555b505050505050565b600654604080517f540105c7000000000000000000000000000000000000000000000000000000008152600160a060020a0385811660048301529151600093929092169163540105c791602480820192869290919082900301818387803b15801561137d57600080fd5b505af1158015611391573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405260c08110156113ba57600080fd5b8151602083015160408401516060850151608086018051949693959294919392830192916401000000008111156113f057600080fd5b8201602081018481111561140357600080fd5b815185602082028301116401000000008211171561142057600080fd5b5050929190602001805164010000000081111561143c57600080fd5b8201602081018481111561144f57600080fd5b815185602082028301116401000000008211171561146c57600080fd5b509799505050600160a060020a038816331496506114db95505050505050576040805160e560020a62461bcd02815260206004820152601660248201527f52656c61796572206f776e657220726571756972656400000000000000000000604482015290519081900360640190fd5b600654604080517f500f99f7000000000000000000000000000000000000000000000000000000008152600160a060020a0386811660048301529151919092169163500f99f79160248083019260209291908290030181600087803b15801561154357600080fd5b505af1158015611557573d6000803e3d6000fd5b505050506040513d602081101561156d57600080fd5b5051156115c4576040805160e560020a62461bcd02815260206004820152601960248201527f52656c6179657220726571756972656420746f20636c6f736500000000000000604482015290519081900360640190fd5b60008261ffff16101580156115de57506103e88261ffff16105b1515611634576040805160e560020a62461bcd02815260206004820152601160248201527f496e76616c696420747261646520466565000000000000000000000000000000604482015290519081900360640190fd5b50600160a060020a03919091166000908152602081905260409020805461ffff191661ffff909216919091179055565b600754600160a060020a031633146116b4576040805160e560020a62461bcd02815260206004820152600f60248201526000805160206123f8833981519152604482015290519081900360640190fd5b600160a060020a03811615156116c957600080fd5b60078054600160a060020a031916600160a060020a0392909216919091179055565b60048054829081106116f957fe5b600091825260209091200154905081565b60038054829081106105bd57fe5b600160208190526000918252604090912080549181015460029091015483565b600754600090600160a060020a0316331461178b576040805160e560020a62461bcd02815260206004820152600f60248201526000805160206123f8833981519152604482015290519081900360640190fd5b6008546040805160e060020a63a3ff31b5028152600160a060020a0385811660048301529151919092169163a3ff31b59160248083019260209291908290030181600087803b1580156117dd57600080fd5b505af11580156117f1573d6000803e3d6000fd5b505050506040513d602081101561180757600080fd5b50518061181d5750600160a060020a0382166001145b9050801515611876576040805160e560020a62461bcd02815260206004820152601260248201527f496e76616c6964206261736520746f6b656e0000000000000000000000000000604482015290519081900360640190fd5b6118da60038054806020026020016040519081016040528092919081815260200182805480156118cf57602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116118b1575b5050505050836122bb565b151561192c57600380546001810182556000919091527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b018054600160a060020a031916600160a060020a0384161790555b5050565b6008546040805160e060020a63a3ff31b5028152600160a060020a03868116600483015291516000938493169163a3ff31b591602480830192602092919082900301818787803b15801561198357600080fd5b505af1158015611997573d6000803e3d6000fd5b505050506040513d60208110156119ad57600080fd5b5051806119c35750600160a060020a0385166001145b9150811515611a0a576040805160e560020a62461bcd0281526020600482015260126024820152600080516020612418833981519152604482015290519081900360640190fd5b611a6e6003805480602002602001604051908101604052809291908181526020018280548015611a6357602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311611a45575b5050505050856122bb565b1515611ac4576040805160e560020a62461bcd02815260206004820152601560248201527f496e76616c6964206c656e64696e6720746f6b656e0000000000000000000000604482015290519081900360640190fd5b600160a060020a03851660009081526001602052604090205460641115611b23576040805160e560020a62461bcd0281526020600482015260126024820152600080516020612418833981519152604482015290519081900360640190fd5b611b876002805480602002602001604051908101604052809291908181526020018280548015611b7c57602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311611b5e575b5050505050866122bb565b15611bf357600954600160a060020a03163314611bee576040805160e560020a62461bcd02815260206004820152601c60248201527f4f7261636c652050726963652046656564657220726571756972656400000000604482015290519081900360640190fd5b611cc8565b84905033600160a060020a031681600160a060020a0316631d1438486040518163ffffffff1660e060020a028152600401602060405180830381600087803b158015611c3e57600080fd5b505af1158015611c52573d6000803e3d6000fd5b505050506040513d6020811015611c6857600080fd5b5051600160a060020a031614611cc8576040805160e560020a62461bcd02815260206004820152601560248201527f526571756972656420746f6b656e206973737565720000000000000000000000604482015290519081900360640190fd5b5050604080518082018252918252436020808401918252600160a060020a0395861660009081526001808352848220969097168152600390950190529220905181559051910155565b60058054829081106105bd57fe5b600954600160a060020a03163314611d81576040805160e560020a62461bcd02815260206004820152601960248201527f4f7261636c6520707269636520666565646572206f6e6c792e00000000000000604482015290519081900360640190fd5b600160a060020a0381161515611d9657600080fd5b60098054600160a060020a031916600160a060020a0392909216919091179055565b600754600090600160a060020a03163314611e0b576040805160e560020a62461bcd02815260206004820152600f60248201526000805160206123f8833981519152604482015290519081900360640190fd5b60648410158015611e1c5750606483115b1515611e72576040805160e560020a62461bcd02815260206004820152600d60248201527f496e76616c696420726174657300000000000000000000000000000000000000604482015290519081900360640190fd5b828411611ec9576040805160e560020a62461bcd02815260206004820152601560248201527f496e76616c6964206465706f7369742072617465730000000000000000000000604482015290519081900360640190fd5b838211611f20576040805160e560020a62461bcd02815260206004820152601460248201527f496e76616c696420726563616c6c207261746573000000000000000000000000604482015290519081900360640190fd5b6008546040805160e060020a63a3ff31b5028152600160a060020a0388811660048301529151919092169163a3ff31b59160248083019260209291908290030181600087803b158015611f7257600080fd5b505af1158015611f86573d6000803e3d6000fd5b505050506040513d6020811015611f9c57600080fd5b505180611fb25750600160a060020a0385166001145b9050801515611ff9576040805160e560020a62461bcd0281526020600482015260126024820152600080516020612418833981519152604482015290519081900360640190fd5b604080516060810182528581526020808201868152828401868152600160a060020a038a16600090815260018085529086902094518555915191840191909155516002928301558154835181830281018301909452808452612099939291830182828015611b7c57602002820191906000526020600020908154600160a060020a03168152600190910190602001808311611b5e575050505050866122bb565b15156120eb57600280546001810182556000919091527f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace018054600160a060020a031916600160a060020a0387161790555b5050505050565b600160a060020a03918216600090815260016020818152604080842094909516835260039093019092529190912080549101549091565b600160a060020a03811660009081526020818152604080832080546001820180548451818702810187019095528085526060958695869561ffff9095169460028101936003909101928591908301828280156121ae57602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311612190575b505050505092508180548060200260200160405190810160405280929190818152602001828054801561220057602002820191906000526020600020905b8154815260200190600101908083116121ec575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561225c57602002820191906000526020600020905b8154600160a060020a0316815260019091019060200180831161223e575b5050505050905093509350935093509193509193565b6000805b83518110156122af5782848281518110151561228e57fe5b9060200190602002015114156122a757600191506122b4565b600101612276565b600091505b5092915050565b6000805b83518110156122af5782600160a060020a031684828151811015156122e057fe5b90602001906020020151600160a060020a0316141561230257600191506122b4565b6001016122bf565b82805482825590600052602060002090810192821561235f579160200282015b8281111561235f5782518254600160a060020a031916600160a060020a0390911617825560209092019160019091019061232a565b5061236b9291506123b6565b5090565b8280548282559060005260206000209081019282156123aa579160200282015b828111156123aa57825182559160200191906001019061238f565b5061236b9291506123dd565b6123da91905b8082111561236b578054600160a060020a03191681556001016123bc565b90565b6123da91905b8082111561236b57600081556001016123e356004d6f64657261746f72206f6e6c792e0000000000000000000000000000000000496e76616c696420636f6c6c61746572616c0000000000000000000000000000a165627a7a72305820c96a7844fbc99f6cd5124b4b98c05fdfa83bae82c294c9ecae7cce94364056950029`

// DeployLending deploys a new Ethereum contract, binding an instance of Lending to it.
func DeployLending(auth *bind.TransactOpts, backend bind.ContractBackend, r common.Address, t common.Address) (common.Address, *types.Transaction, *Lending, error) {
	parsed, err := abi.JSON(strings.NewReader(LendingABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(LendingBin), backend, r, t)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Lending{LendingCaller: LendingCaller{contract: contract}, LendingTransactor: LendingTransactor{contract: contract}, LendingFilterer: LendingFilterer{contract: contract}}, nil
}

// Lending is an auto generated Go binding around an Ethereum contract.
type Lending struct {
	LendingCaller     // Read-only binding to the contract
	LendingTransactor // Write-only binding to the contract
	LendingFilterer   // Log filterer for contract events
}

// LendingCaller is an auto generated read-only Go binding around an Ethereum contract.
type LendingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LendingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LendingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LendingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LendingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LendingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LendingSession struct {
	Contract     *Lending          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LendingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LendingCallerSession struct {
	Contract *LendingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// LendingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LendingTransactorSession struct {
	Contract     *LendingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// LendingRaw is an auto generated low-level Go binding around an Ethereum contract.
type LendingRaw struct {
	Contract *Lending // Generic contract binding to access the raw methods on
}

// LendingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LendingCallerRaw struct {
	Contract *LendingCaller // Generic read-only contract binding to access the raw methods on
}

// LendingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LendingTransactorRaw struct {
	Contract *LendingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLending creates a new instance of Lending, bound to a specific deployed contract.
func NewLending(address common.Address, backend bind.ContractBackend) (*Lending, error) {
	contract, err := bindLending(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Lending{LendingCaller: LendingCaller{contract: contract}, LendingTransactor: LendingTransactor{contract: contract}, LendingFilterer: LendingFilterer{contract: contract}}, nil
}

// NewLendingCaller creates a new read-only instance of Lending, bound to a specific deployed contract.
func NewLendingCaller(address common.Address, caller bind.ContractCaller) (*LendingCaller, error) {
	contract, err := bindLending(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LendingCaller{contract: contract}, nil
}

// NewLendingTransactor creates a new write-only instance of Lending, bound to a specific deployed contract.
func NewLendingTransactor(address common.Address, transactor bind.ContractTransactor) (*LendingTransactor, error) {
	contract, err := bindLending(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LendingTransactor{contract: contract}, nil
}

// NewLendingFilterer creates a new log filterer instance of Lending, bound to a specific deployed contract.
func NewLendingFilterer(address common.Address, filterer bind.ContractFilterer) (*LendingFilterer, error) {
	contract, err := bindLending(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LendingFilterer{contract: contract}, nil
}

// bindLending binds a generic wrapper to an already deployed contract.
func bindLending(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LendingABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Lending *LendingRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Lending.Contract.LendingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Lending *LendingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Lending.Contract.LendingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Lending *LendingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Lending.Contract.LendingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Lending *LendingCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Lending.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Lending *LendingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Lending.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Lending *LendingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Lending.Contract.contract.Transact(opts, method, params...)
}

// BASES is a free data retrieval call binding the contract method 0x6d1dc42a.
//
// Solidity: function BASES( uint256) constant returns(address)
func (_Lending *LendingCaller) BASES(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "BASES", arg0)
	return *ret0, err
}

// BASES is a free data retrieval call binding the contract method 0x6d1dc42a.
//
// Solidity: function BASES( uint256) constant returns(address)
func (_Lending *LendingSession) BASES(arg0 *big.Int) (common.Address, error) {
	return _Lending.Contract.BASES(&_Lending.CallOpts, arg0)
}

// BASES is a free data retrieval call binding the contract method 0x6d1dc42a.
//
// Solidity: function BASES( uint256) constant returns(address)
func (_Lending *LendingCallerSession) BASES(arg0 *big.Int) (common.Address, error) {
	return _Lending.Contract.BASES(&_Lending.CallOpts, arg0)
}

// COLLATERALS is a free data retrieval call binding the contract method 0x0811f05a.
//
// Solidity: function COLLATERALS( uint256) constant returns(address)
func (_Lending *LendingCaller) COLLATERALS(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "COLLATERALS", arg0)
	return *ret0, err
}

// COLLATERALS is a free data retrieval call binding the contract method 0x0811f05a.
//
// Solidity: function COLLATERALS( uint256) constant returns(address)
func (_Lending *LendingSession) COLLATERALS(arg0 *big.Int) (common.Address, error) {
	return _Lending.Contract.COLLATERALS(&_Lending.CallOpts, arg0)
}

// COLLATERALS is a free data retrieval call binding the contract method 0x0811f05a.
//
// Solidity: function COLLATERALS( uint256) constant returns(address)
func (_Lending *LendingCallerSession) COLLATERALS(arg0 *big.Int) (common.Address, error) {
	return _Lending.Contract.COLLATERALS(&_Lending.CallOpts, arg0)
}

// COLLATERALLIST is a free data retrieval call binding the contract method 0x82250701.
//
// Solidity: function COLLATERAL_LIST( address) constant returns(_depositRate uint256, _liquidationRate uint256, _recallRate uint256)
func (_Lending *LendingCaller) COLLATERALLIST(opts *bind.CallOpts, arg0 common.Address) (struct {
	DepositRate     *big.Int
	LiquidationRate *big.Int
	RecallRate      *big.Int
}, error) {
	ret := new(struct {
		DepositRate     *big.Int
		LiquidationRate *big.Int
		RecallRate      *big.Int
	})
	out := ret
	err := _Lending.contract.Call(opts, out, "COLLATERAL_LIST", arg0)
	return *ret, err
}

// COLLATERALLIST is a free data retrieval call binding the contract method 0x82250701.
//
// Solidity: function COLLATERAL_LIST( address) constant returns(_depositRate uint256, _liquidationRate uint256, _recallRate uint256)
func (_Lending *LendingSession) COLLATERALLIST(arg0 common.Address) (struct {
	DepositRate     *big.Int
	LiquidationRate *big.Int
	RecallRate      *big.Int
}, error) {
	return _Lending.Contract.COLLATERALLIST(&_Lending.CallOpts, arg0)
}

// COLLATERALLIST is a free data retrieval call binding the contract method 0x82250701.
//
// Solidity: function COLLATERAL_LIST( address) constant returns(_depositRate uint256, _liquidationRate uint256, _recallRate uint256)
func (_Lending *LendingCallerSession) COLLATERALLIST(arg0 common.Address) (struct {
	DepositRate     *big.Int
	LiquidationRate *big.Int
	RecallRate      *big.Int
}, error) {
	return _Lending.Contract.COLLATERALLIST(&_Lending.CallOpts, arg0)
}

// ILOCOLLATERALS is a free data retrieval call binding the contract method 0xb8687ec4.
//
// Solidity: function ILO_COLLATERALS( uint256) constant returns(address)
func (_Lending *LendingCaller) ILOCOLLATERALS(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "ILO_COLLATERALS", arg0)
	return *ret0, err
}

// ILOCOLLATERALS is a free data retrieval call binding the contract method 0xb8687ec4.
//
// Solidity: function ILO_COLLATERALS( uint256) constant returns(address)
func (_Lending *LendingSession) ILOCOLLATERALS(arg0 *big.Int) (common.Address, error) {
	return _Lending.Contract.ILOCOLLATERALS(&_Lending.CallOpts, arg0)
}

// ILOCOLLATERALS is a free data retrieval call binding the contract method 0xb8687ec4.
//
// Solidity: function ILO_COLLATERALS( uint256) constant returns(address)
func (_Lending *LendingCallerSession) ILOCOLLATERALS(arg0 *big.Int) (common.Address, error) {
	return _Lending.Contract.ILOCOLLATERALS(&_Lending.CallOpts, arg0)
}

// LENDINGRELAYERLIST is a free data retrieval call binding the contract method 0x0faf292c.
//
// Solidity: function LENDINGRELAYER_LIST( address) constant returns(_tradeFee uint16)
func (_Lending *LendingCaller) LENDINGRELAYERLIST(opts *bind.CallOpts, arg0 common.Address) (uint16, error) {
	var (
		ret0 = new(uint16)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "LENDINGRELAYER_LIST", arg0)
	return *ret0, err
}

// LENDINGRELAYERLIST is a free data retrieval call binding the contract method 0x0faf292c.
//
// Solidity: function LENDINGRELAYER_LIST( address) constant returns(_tradeFee uint16)
func (_Lending *LendingSession) LENDINGRELAYERLIST(arg0 common.Address) (uint16, error) {
	return _Lending.Contract.LENDINGRELAYERLIST(&_Lending.CallOpts, arg0)
}

// LENDINGRELAYERLIST is a free data retrieval call binding the contract method 0x0faf292c.
//
// Solidity: function LENDINGRELAYER_LIST( address) constant returns(_tradeFee uint16)
func (_Lending *LendingCallerSession) LENDINGRELAYERLIST(arg0 common.Address) (uint16, error) {
	return _Lending.Contract.LENDINGRELAYERLIST(&_Lending.CallOpts, arg0)
}

// MODERATOR is a free data retrieval call binding the contract method 0x34b4e625.
//
// Solidity: function MODERATOR() constant returns(address)
func (_Lending *LendingCaller) MODERATOR(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "MODERATOR")
	return *ret0, err
}

// MODERATOR is a free data retrieval call binding the contract method 0x34b4e625.
//
// Solidity: function MODERATOR() constant returns(address)
func (_Lending *LendingSession) MODERATOR() (common.Address, error) {
	return _Lending.Contract.MODERATOR(&_Lending.CallOpts)
}

// MODERATOR is a free data retrieval call binding the contract method 0x34b4e625.
//
// Solidity: function MODERATOR() constant returns(address)
func (_Lending *LendingCallerSession) MODERATOR() (common.Address, error) {
	return _Lending.Contract.MODERATOR(&_Lending.CallOpts)
}

// ORACLEPRICEFEEDER is a free data retrieval call binding the contract method 0x0c4c2cbb.
//
// Solidity: function ORACLE_PRICE_FEEDER() constant returns(address)
func (_Lending *LendingCaller) ORACLEPRICEFEEDER(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "ORACLE_PRICE_FEEDER")
	return *ret0, err
}

// ORACLEPRICEFEEDER is a free data retrieval call binding the contract method 0x0c4c2cbb.
//
// Solidity: function ORACLE_PRICE_FEEDER() constant returns(address)
func (_Lending *LendingSession) ORACLEPRICEFEEDER() (common.Address, error) {
	return _Lending.Contract.ORACLEPRICEFEEDER(&_Lending.CallOpts)
}

// ORACLEPRICEFEEDER is a free data retrieval call binding the contract method 0x0c4c2cbb.
//
// Solidity: function ORACLE_PRICE_FEEDER() constant returns(address)
func (_Lending *LendingCallerSession) ORACLEPRICEFEEDER() (common.Address, error) {
	return _Lending.Contract.ORACLEPRICEFEEDER(&_Lending.CallOpts)
}

// Relayer is a free data retrieval call binding the contract method 0x264949d8.
//
// Solidity: function Relayer() constant returns(address)
func (_Lending *LendingCaller) Relayer(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "Relayer")
	return *ret0, err
}

// Relayer is a free data retrieval call binding the contract method 0x264949d8.
//
// Solidity: function Relayer() constant returns(address)
func (_Lending *LendingSession) Relayer() (common.Address, error) {
	return _Lending.Contract.Relayer(&_Lending.CallOpts)
}

// Relayer is a free data retrieval call binding the contract method 0x264949d8.
//
// Solidity: function Relayer() constant returns(address)
func (_Lending *LendingCallerSession) Relayer() (common.Address, error) {
	return _Lending.Contract.Relayer(&_Lending.CallOpts)
}

// TERMS is a free data retrieval call binding the contract method 0x56327f57.
//
// Solidity: function TERMS( uint256) constant returns(uint256)
func (_Lending *LendingCaller) TERMS(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "TERMS", arg0)
	return *ret0, err
}

// TERMS is a free data retrieval call binding the contract method 0x56327f57.
//
// Solidity: function TERMS( uint256) constant returns(uint256)
func (_Lending *LendingSession) TERMS(arg0 *big.Int) (*big.Int, error) {
	return _Lending.Contract.TERMS(&_Lending.CallOpts, arg0)
}

// TERMS is a free data retrieval call binding the contract method 0x56327f57.
//
// Solidity: function TERMS( uint256) constant returns(uint256)
func (_Lending *LendingCallerSession) TERMS(arg0 *big.Int) (*big.Int, error) {
	return _Lending.Contract.TERMS(&_Lending.CallOpts, arg0)
}

// XDCXListing is a free data retrieval call binding the contract method 0x29a4ddec.
//
// Solidity: function XDCXListing() constant returns(address)
func (_Lending *LendingCaller) XDCXListing(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Lending.contract.Call(opts, out, "XDCXListing")
	return *ret0, err
}

// XDCXListing is a free data retrieval call binding the contract method 0x29a4ddec.
//
// Solidity: function XDCXListing() constant returns(address)
func (_Lending *LendingSession) XDCXListing() (common.Address, error) {
	return _Lending.Contract.XDCXListing(&_Lending.CallOpts)
}

// XDCXListing is a free data retrieval call binding the contract method 0x29a4ddec.
//
// Solidity: function XDCXListing() constant returns(address)
func (_Lending *LendingCallerSession) XDCXListing() (common.Address, error) {
	return _Lending.Contract.XDCXListing(&_Lending.CallOpts)
}

// GetCollateralPrice is a free data retrieval call binding the contract method 0xf2dbd070.
//
// Solidity: function getCollateralPrice(token address, lendingToken address) constant returns(uint256, uint256)
func (_Lending *LendingCaller) GetCollateralPrice(opts *bind.CallOpts, token common.Address, lendingToken common.Address) (*big.Int, *big.Int, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new(*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _Lending.contract.Call(opts, out, "getCollateralPrice", token, lendingToken)
	return *ret0, *ret1, err
}

// GetCollateralPrice is a free data retrieval call binding the contract method 0xf2dbd070.
//
// Solidity: function getCollateralPrice(token address, lendingToken address) constant returns(uint256, uint256)
func (_Lending *LendingSession) GetCollateralPrice(token common.Address, lendingToken common.Address) (*big.Int, *big.Int, error) {
	return _Lending.Contract.GetCollateralPrice(&_Lending.CallOpts, token, lendingToken)
}

// GetCollateralPrice is a free data retrieval call binding the contract method 0xf2dbd070.
//
// Solidity: function getCollateralPrice(token address, lendingToken address) constant returns(uint256, uint256)
func (_Lending *LendingCallerSession) GetCollateralPrice(token common.Address, lendingToken common.Address) (*big.Int, *big.Int, error) {
	return _Lending.Contract.GetCollateralPrice(&_Lending.CallOpts, token, lendingToken)
}

// GetLendingRelayerByCoinbase is a free data retrieval call binding the contract method 0xfe824700.
//
// Solidity: function getLendingRelayerByCoinbase(coinbase address) constant returns(uint16, address[], uint256[], address[])
func (_Lending *LendingCaller) GetLendingRelayerByCoinbase(opts *bind.CallOpts, coinbase common.Address) (uint16, []common.Address, []*big.Int, []common.Address, error) {
	var (
		ret0 = new(uint16)
		ret1 = new([]common.Address)
		ret2 = new([]*big.Int)
		ret3 = new([]common.Address)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
		ret3,
	}
	err := _Lending.contract.Call(opts, out, "getLendingRelayerByCoinbase", coinbase)
	return *ret0, *ret1, *ret2, *ret3, err
}

// GetLendingRelayerByCoinbase is a free data retrieval call binding the contract method 0xfe824700.
//
// Solidity: function getLendingRelayerByCoinbase(coinbase address) constant returns(uint16, address[], uint256[], address[])
func (_Lending *LendingSession) GetLendingRelayerByCoinbase(coinbase common.Address) (uint16, []common.Address, []*big.Int, []common.Address, error) {
	return _Lending.Contract.GetLendingRelayerByCoinbase(&_Lending.CallOpts, coinbase)
}

// GetLendingRelayerByCoinbase is a free data retrieval call binding the contract method 0xfe824700.
//
// Solidity: function getLendingRelayerByCoinbase(coinbase address) constant returns(uint16, address[], uint256[], address[])
func (_Lending *LendingCallerSession) GetLendingRelayerByCoinbase(coinbase common.Address) (uint16, []common.Address, []*big.Int, []common.Address, error) {
	return _Lending.Contract.GetLendingRelayerByCoinbase(&_Lending.CallOpts, coinbase)
}

// AddBaseToken is a paid mutator transaction binding the contract method 0x83e280d9.
//
// Solidity: function addBaseToken(token address) returns()
func (_Lending *LendingTransactor) AddBaseToken(opts *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "addBaseToken", token)
}

// AddBaseToken is a paid mutator transaction binding the contract method 0x83e280d9.
//
// Solidity: function addBaseToken(token address) returns()
func (_Lending *LendingSession) AddBaseToken(token common.Address) (*types.Transaction, error) {
	return _Lending.Contract.AddBaseToken(&_Lending.TransactOpts, token)
}

// AddBaseToken is a paid mutator transaction binding the contract method 0x83e280d9.
//
// Solidity: function addBaseToken(token address) returns()
func (_Lending *LendingTransactorSession) AddBaseToken(token common.Address) (*types.Transaction, error) {
	return _Lending.Contract.AddBaseToken(&_Lending.TransactOpts, token)
}

// AddCollateral is a paid mutator transaction binding the contract method 0xe5eecf68.
//
// Solidity: function addCollateral(token address, depositRate uint256, liquidationRate uint256, recallRate uint256) returns()
func (_Lending *LendingTransactor) AddCollateral(opts *bind.TransactOpts, token common.Address, depositRate *big.Int, liquidationRate *big.Int, recallRate *big.Int) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "addCollateral", token, depositRate, liquidationRate, recallRate)
}

// AddCollateral is a paid mutator transaction binding the contract method 0xe5eecf68.
//
// Solidity: function addCollateral(token address, depositRate uint256, liquidationRate uint256, recallRate uint256) returns()
func (_Lending *LendingSession) AddCollateral(token common.Address, depositRate *big.Int, liquidationRate *big.Int, recallRate *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.AddCollateral(&_Lending.TransactOpts, token, depositRate, liquidationRate, recallRate)
}

// AddCollateral is a paid mutator transaction binding the contract method 0xe5eecf68.
//
// Solidity: function addCollateral(token address, depositRate uint256, liquidationRate uint256, recallRate uint256) returns()
func (_Lending *LendingTransactorSession) AddCollateral(token common.Address, depositRate *big.Int, liquidationRate *big.Int, recallRate *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.AddCollateral(&_Lending.TransactOpts, token, depositRate, liquidationRate, recallRate)
}

// AddILOCollateral is a paid mutator transaction binding the contract method 0x3b874827.
//
// Solidity: function addILOCollateral(token address, depositRate uint256, liquidationRate uint256, recallRate uint256) returns()
func (_Lending *LendingTransactor) AddILOCollateral(opts *bind.TransactOpts, token common.Address, depositRate *big.Int, liquidationRate *big.Int, recallRate *big.Int) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "addILOCollateral", token, depositRate, liquidationRate, recallRate)
}

// AddILOCollateral is a paid mutator transaction binding the contract method 0x3b874827.
//
// Solidity: function addILOCollateral(token address, depositRate uint256, liquidationRate uint256, recallRate uint256) returns()
func (_Lending *LendingSession) AddILOCollateral(token common.Address, depositRate *big.Int, liquidationRate *big.Int, recallRate *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.AddILOCollateral(&_Lending.TransactOpts, token, depositRate, liquidationRate, recallRate)
}

// AddILOCollateral is a paid mutator transaction binding the contract method 0x3b874827.
//
// Solidity: function addILOCollateral(token address, depositRate uint256, liquidationRate uint256, recallRate uint256) returns()
func (_Lending *LendingTransactorSession) AddILOCollateral(token common.Address, depositRate *big.Int, liquidationRate *big.Int, recallRate *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.AddILOCollateral(&_Lending.TransactOpts, token, depositRate, liquidationRate, recallRate)
}

// AddTerm is a paid mutator transaction binding the contract method 0x0c655955.
//
// Solidity: function addTerm(term uint256) returns()
func (_Lending *LendingTransactor) AddTerm(opts *bind.TransactOpts, term *big.Int) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "addTerm", term)
}

// AddTerm is a paid mutator transaction binding the contract method 0x0c655955.
//
// Solidity: function addTerm(term uint256) returns()
func (_Lending *LendingSession) AddTerm(term *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.AddTerm(&_Lending.TransactOpts, term)
}

// AddTerm is a paid mutator transaction binding the contract method 0x0c655955.
//
// Solidity: function addTerm(term uint256) returns()
func (_Lending *LendingTransactorSession) AddTerm(term *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.AddTerm(&_Lending.TransactOpts, term)
}

// ChangeModerator is a paid mutator transaction binding the contract method 0x46642921.
//
// Solidity: function changeModerator(moderator address) returns()
func (_Lending *LendingTransactor) ChangeModerator(opts *bind.TransactOpts, moderator common.Address) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "changeModerator", moderator)
}

// ChangeModerator is a paid mutator transaction binding the contract method 0x46642921.
//
// Solidity: function changeModerator(moderator address) returns()
func (_Lending *LendingSession) ChangeModerator(moderator common.Address) (*types.Transaction, error) {
	return _Lending.Contract.ChangeModerator(&_Lending.TransactOpts, moderator)
}

// ChangeModerator is a paid mutator transaction binding the contract method 0x46642921.
//
// Solidity: function changeModerator(moderator address) returns()
func (_Lending *LendingTransactorSession) ChangeModerator(moderator common.Address) (*types.Transaction, error) {
	return _Lending.Contract.ChangeModerator(&_Lending.TransactOpts, moderator)
}

// ChangeOraclePriceFeeder is a paid mutator transaction binding the contract method 0xc38f473f.
//
// Solidity: function changeOraclePriceFeeder(feeder address) returns()
func (_Lending *LendingTransactor) ChangeOraclePriceFeeder(opts *bind.TransactOpts, feeder common.Address) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "changeOraclePriceFeeder", feeder)
}

// ChangeOraclePriceFeeder is a paid mutator transaction binding the contract method 0xc38f473f.
//
// Solidity: function changeOraclePriceFeeder(feeder address) returns()
func (_Lending *LendingSession) ChangeOraclePriceFeeder(feeder common.Address) (*types.Transaction, error) {
	return _Lending.Contract.ChangeOraclePriceFeeder(&_Lending.TransactOpts, feeder)
}

// ChangeOraclePriceFeeder is a paid mutator transaction binding the contract method 0xc38f473f.
//
// Solidity: function changeOraclePriceFeeder(feeder address) returns()
func (_Lending *LendingTransactorSession) ChangeOraclePriceFeeder(feeder common.Address) (*types.Transaction, error) {
	return _Lending.Contract.ChangeOraclePriceFeeder(&_Lending.TransactOpts, feeder)
}

// SetCollateralPrice is a paid mutator transaction binding the contract method 0xacb8cd92.
//
// Solidity: function setCollateralPrice(token address, lendingToken address, price uint256) returns()
func (_Lending *LendingTransactor) SetCollateralPrice(opts *bind.TransactOpts, token common.Address, lendingToken common.Address, price *big.Int) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "setCollateralPrice", token, lendingToken, price)
}

// SetCollateralPrice is a paid mutator transaction binding the contract method 0xacb8cd92.
//
// Solidity: function setCollateralPrice(token address, lendingToken address, price uint256) returns()
func (_Lending *LendingSession) SetCollateralPrice(token common.Address, lendingToken common.Address, price *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.SetCollateralPrice(&_Lending.TransactOpts, token, lendingToken, price)
}

// SetCollateralPrice is a paid mutator transaction binding the contract method 0xacb8cd92.
//
// Solidity: function setCollateralPrice(token address, lendingToken address, price uint256) returns()
func (_Lending *LendingTransactorSession) SetCollateralPrice(token common.Address, lendingToken common.Address, price *big.Int) (*types.Transaction, error) {
	return _Lending.Contract.SetCollateralPrice(&_Lending.TransactOpts, token, lendingToken, price)
}

// Update is a paid mutator transaction binding the contract method 0x2ddada4c.
//
// Solidity: function update(coinbase address, tradeFee uint16, baseTokens address[], terms uint256[], collaterals address[]) returns()
func (_Lending *LendingTransactor) Update(opts *bind.TransactOpts, coinbase common.Address, tradeFee uint16, baseTokens []common.Address, terms []*big.Int, collaterals []common.Address) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "update", coinbase, tradeFee, baseTokens, terms, collaterals)
}

// Update is a paid mutator transaction binding the contract method 0x2ddada4c.
//
// Solidity: function update(coinbase address, tradeFee uint16, baseTokens address[], terms uint256[], collaterals address[]) returns()
func (_Lending *LendingSession) Update(coinbase common.Address, tradeFee uint16, baseTokens []common.Address, terms []*big.Int, collaterals []common.Address) (*types.Transaction, error) {
	return _Lending.Contract.Update(&_Lending.TransactOpts, coinbase, tradeFee, baseTokens, terms, collaterals)
}

// Update is a paid mutator transaction binding the contract method 0x2ddada4c.
//
// Solidity: function update(coinbase address, tradeFee uint16, baseTokens address[], terms uint256[], collaterals address[]) returns()
func (_Lending *LendingTransactorSession) Update(coinbase common.Address, tradeFee uint16, baseTokens []common.Address, terms []*big.Int, collaterals []common.Address) (*types.Transaction, error) {
	return _Lending.Contract.Update(&_Lending.TransactOpts, coinbase, tradeFee, baseTokens, terms, collaterals)
}

// UpdateFee is a paid mutator transaction binding the contract method 0x3ea2391f.
//
// Solidity: function updateFee(coinbase address, tradeFee uint16) returns()
func (_Lending *LendingTransactor) UpdateFee(opts *bind.TransactOpts, coinbase common.Address, tradeFee uint16) (*types.Transaction, error) {
	return _Lending.contract.Transact(opts, "updateFee", coinbase, tradeFee)
}

// UpdateFee is a paid mutator transaction binding the contract method 0x3ea2391f.
//
// Solidity: function updateFee(coinbase address, tradeFee uint16) returns()
func (_Lending *LendingSession) UpdateFee(coinbase common.Address, tradeFee uint16) (*types.Transaction, error) {
	return _Lending.Contract.UpdateFee(&_Lending.TransactOpts, coinbase, tradeFee)
}

// UpdateFee is a paid mutator transaction binding the contract method 0x3ea2391f.
//
// Solidity: function updateFee(coinbase address, tradeFee uint16) returns()
func (_Lending *LendingTransactorSession) UpdateFee(coinbase common.Address, tradeFee uint16) (*types.Transaction, error) {
	return _Lending.Contract.UpdateFee(&_Lending.TransactOpts, coinbase, tradeFee)
}
