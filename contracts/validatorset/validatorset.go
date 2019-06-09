// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package validatorset

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ValidatorsetABI is the input ABI used to generate the binding from.
const ValidatorsetABI = "[{\"constant\":false,\"inputs\":[],\"name\":\"finalizeChange\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getValidators\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"},{\"name\":\"\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"validator\",\"type\":\"address\"},{\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"reportMalicious\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"validator\",\"type\":\"address\"},{\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"reportBenign\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_parentHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"_newSet\",\"type\":\"address[]\"}],\"name\":\"InitiateChange\",\"type\":\"event\"}]"

// Validatorset is an auto generated Go binding around an Ethereum contract.
type Validatorset struct {
	ValidatorsetCaller     // Read-only binding to the contract
	ValidatorsetTransactor // Write-only binding to the contract
	ValidatorsetFilterer   // Log filterer for contract events
}

// ValidatorsetCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValidatorsetCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorsetTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValidatorsetTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorsetFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValidatorsetFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorsetSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValidatorsetSession struct {
	Contract     *Validatorset     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ValidatorsetCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValidatorsetCallerSession struct {
	Contract *ValidatorsetCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// ValidatorsetTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValidatorsetTransactorSession struct {
	Contract     *ValidatorsetTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ValidatorsetRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValidatorsetRaw struct {
	Contract *Validatorset // Generic contract binding to access the raw methods on
}

// ValidatorsetCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValidatorsetCallerRaw struct {
	Contract *ValidatorsetCaller // Generic read-only contract binding to access the raw methods on
}

// ValidatorsetTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValidatorsetTransactorRaw struct {
	Contract *ValidatorsetTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValidatorset creates a new instance of Validatorset, bound to a specific deployed contract.
func NewValidatorset(address common.Address, backend bind.ContractBackend) (*Validatorset, error) {
	contract, err := bindValidatorset(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Validatorset{ValidatorsetCaller: ValidatorsetCaller{contract: contract}, ValidatorsetTransactor: ValidatorsetTransactor{contract: contract}, ValidatorsetFilterer: ValidatorsetFilterer{contract: contract}}, nil
}

// NewValidatorsetCaller creates a new read-only instance of Validatorset, bound to a specific deployed contract.
func NewValidatorsetCaller(address common.Address, caller bind.ContractCaller) (*ValidatorsetCaller, error) {
	contract, err := bindValidatorset(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorsetCaller{contract: contract}, nil
}

// NewValidatorsetTransactor creates a new write-only instance of Validatorset, bound to a specific deployed contract.
func NewValidatorsetTransactor(address common.Address, transactor bind.ContractTransactor) (*ValidatorsetTransactor, error) {
	contract, err := bindValidatorset(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorsetTransactor{contract: contract}, nil
}

// NewValidatorsetFilterer creates a new log filterer instance of Validatorset, bound to a specific deployed contract.
func NewValidatorsetFilterer(address common.Address, filterer bind.ContractFilterer) (*ValidatorsetFilterer, error) {
	contract, err := bindValidatorset(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValidatorsetFilterer{contract: contract}, nil
}

// bindValidatorset binds a generic wrapper to an already deployed contract.
func bindValidatorset(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorsetABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Validatorset *ValidatorsetRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Validatorset.Contract.ValidatorsetCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Validatorset *ValidatorsetRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Validatorset.Contract.ValidatorsetTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Validatorset *ValidatorsetRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Validatorset.Contract.ValidatorsetTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Validatorset *ValidatorsetCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Validatorset.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Validatorset *ValidatorsetTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Validatorset.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Validatorset *ValidatorsetTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Validatorset.Contract.contract.Transact(opts, method, params...)
}

// GetValidators is a free data retrieval call binding the contract method 0xb7ab4db5.
//
// Solidity: function getValidators() constant returns(address[], uint256[])
func (_Validatorset *ValidatorsetCaller) GetValidators(opts *bind.CallOpts) ([]common.Address, []*big.Int, error) {
	var (
		ret0 = new([]common.Address)
		ret1 = new([]*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _Validatorset.contract.Call(opts, out, "getValidators")
	return *ret0, *ret1, err
}

// GetValidators is a free data retrieval call binding the contract method 0xb7ab4db5.
//
// Solidity: function getValidators() constant returns(address[], uint256[])
func (_Validatorset *ValidatorsetSession) GetValidators() ([]common.Address, []*big.Int, error) {
	return _Validatorset.Contract.GetValidators(&_Validatorset.CallOpts)
}

// GetValidators is a free data retrieval call binding the contract method 0xb7ab4db5.
//
// Solidity: function getValidators() constant returns(address[], uint256[])
func (_Validatorset *ValidatorsetCallerSession) GetValidators() ([]common.Address, []*big.Int, error) {
	return _Validatorset.Contract.GetValidators(&_Validatorset.CallOpts)
}

// FinalizeChange is a paid mutator transaction binding the contract method 0x75286211.
//
// Solidity: function finalizeChange() returns()
func (_Validatorset *ValidatorsetTransactor) FinalizeChange(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Validatorset.contract.Transact(opts, "finalizeChange")
}

// FinalizeChange is a paid mutator transaction binding the contract method 0x75286211.
//
// Solidity: function finalizeChange() returns()
func (_Validatorset *ValidatorsetSession) FinalizeChange() (*types.Transaction, error) {
	return _Validatorset.Contract.FinalizeChange(&_Validatorset.TransactOpts)
}

// FinalizeChange is a paid mutator transaction binding the contract method 0x75286211.
//
// Solidity: function finalizeChange() returns()
func (_Validatorset *ValidatorsetTransactorSession) FinalizeChange() (*types.Transaction, error) {
	return _Validatorset.Contract.FinalizeChange(&_Validatorset.TransactOpts)
}

// ReportBenign is a paid mutator transaction binding the contract method 0xd69f13bb.
//
// Solidity: function reportBenign(address validator, uint256 blockNumber) returns()
func (_Validatorset *ValidatorsetTransactor) ReportBenign(opts *bind.TransactOpts, validator common.Address, blockNumber *big.Int) (*types.Transaction, error) {
	return _Validatorset.contract.Transact(opts, "reportBenign", validator, blockNumber)
}

// ReportBenign is a paid mutator transaction binding the contract method 0xd69f13bb.
//
// Solidity: function reportBenign(address validator, uint256 blockNumber) returns()
func (_Validatorset *ValidatorsetSession) ReportBenign(validator common.Address, blockNumber *big.Int) (*types.Transaction, error) {
	return _Validatorset.Contract.ReportBenign(&_Validatorset.TransactOpts, validator, blockNumber)
}

// ReportBenign is a paid mutator transaction binding the contract method 0xd69f13bb.
//
// Solidity: function reportBenign(address validator, uint256 blockNumber) returns()
func (_Validatorset *ValidatorsetTransactorSession) ReportBenign(validator common.Address, blockNumber *big.Int) (*types.Transaction, error) {
	return _Validatorset.Contract.ReportBenign(&_Validatorset.TransactOpts, validator, blockNumber)
}

// ReportMalicious is a paid mutator transaction binding the contract method 0xc476dd40.
//
// Solidity: function reportMalicious(address validator, uint256 blockNumber, bytes proof) returns()
func (_Validatorset *ValidatorsetTransactor) ReportMalicious(opts *bind.TransactOpts, validator common.Address, blockNumber *big.Int, proof []byte) (*types.Transaction, error) {
	return _Validatorset.contract.Transact(opts, "reportMalicious", validator, blockNumber, proof)
}

// ReportMalicious is a paid mutator transaction binding the contract method 0xc476dd40.
//
// Solidity: function reportMalicious(address validator, uint256 blockNumber, bytes proof) returns()
func (_Validatorset *ValidatorsetSession) ReportMalicious(validator common.Address, blockNumber *big.Int, proof []byte) (*types.Transaction, error) {
	return _Validatorset.Contract.ReportMalicious(&_Validatorset.TransactOpts, validator, blockNumber, proof)
}

// ReportMalicious is a paid mutator transaction binding the contract method 0xc476dd40.
//
// Solidity: function reportMalicious(address validator, uint256 blockNumber, bytes proof) returns()
func (_Validatorset *ValidatorsetTransactorSession) ReportMalicious(validator common.Address, blockNumber *big.Int, proof []byte) (*types.Transaction, error) {
	return _Validatorset.Contract.ReportMalicious(&_Validatorset.TransactOpts, validator, blockNumber, proof)
}

// ValidatorsetInitiateChangeIterator is returned from FilterInitiateChange and is used to iterate over the raw logs and unpacked data for InitiateChange events raised by the Validatorset contract.
type ValidatorsetInitiateChangeIterator struct {
	Event *ValidatorsetInitiateChange // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidatorsetInitiateChangeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorsetInitiateChange)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidatorsetInitiateChange)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidatorsetInitiateChangeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorsetInitiateChangeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorsetInitiateChange represents a InitiateChange event raised by the Validatorset contract.
type ValidatorsetInitiateChange struct {
	ParentHash [32]byte
	NewSet     []common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInitiateChange is a free log retrieval operation binding the contract event 0x55252fa6eee4741b4e24a74a70e9c11fd2c2281df8d6ea13126ff845f7825c89.
//
// Solidity: event InitiateChange(bytes32 indexed _parentHash, address[] _newSet)
func (_Validatorset *ValidatorsetFilterer) FilterInitiateChange(opts *bind.FilterOpts, _parentHash [][32]byte) (*ValidatorsetInitiateChangeIterator, error) {

	var _parentHashRule []interface{}
	for _, _parentHashItem := range _parentHash {
		_parentHashRule = append(_parentHashRule, _parentHashItem)
	}

	logs, sub, err := _Validatorset.contract.FilterLogs(opts, "InitiateChange", _parentHashRule)
	if err != nil {
		return nil, err
	}
	return &ValidatorsetInitiateChangeIterator{contract: _Validatorset.contract, event: "InitiateChange", logs: logs, sub: sub}, nil
}

// WatchInitiateChange is a free log subscription operation binding the contract event 0x55252fa6eee4741b4e24a74a70e9c11fd2c2281df8d6ea13126ff845f7825c89.
//
// Solidity: event InitiateChange(bytes32 indexed _parentHash, address[] _newSet)
func (_Validatorset *ValidatorsetFilterer) WatchInitiateChange(opts *bind.WatchOpts, sink chan<- *ValidatorsetInitiateChange, _parentHash [][32]byte) (event.Subscription, error) {

	var _parentHashRule []interface{}
	for _, _parentHashItem := range _parentHash {
		_parentHashRule = append(_parentHashRule, _parentHashItem)
	}

	logs, sub, err := _Validatorset.contract.WatchLogs(opts, "InitiateChange", _parentHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorsetInitiateChange)
				if err := _Validatorset.contract.UnpackLog(event, "InitiateChange", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
