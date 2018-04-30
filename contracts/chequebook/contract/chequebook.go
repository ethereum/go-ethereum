// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

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

// ChequebookABI is the input ABI used to generate the binding from.
const ChequebookABI = "[{\"constant\":false,\"inputs\":[],\"name\":\"kill\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"sent\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"beneficiary\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"sig_v\",\"type\":\"uint8\"},{\"name\":\"sig_r\",\"type\":\"bytes32\"},{\"name\":\"sig_s\",\"type\":\"bytes32\"}],\"name\":\"cash\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"deadbeat\",\"type\":\"address\"}],\"name\":\"Overdraft\",\"type\":\"event\"}]"

// ChequebookBin is the compiled bytecode used for deploying new contracts.
const ChequebookBin = `0x606060405260008054600160a060020a033316600160a060020a03199091161790556102ec806100306000396000f3006060604052600436106100565763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166341c0e1b581146100585780637bf786f81461006b578063fbf788d61461009c575b005b341561006357600080fd5b6100566100ca565b341561007657600080fd5b61008a600160a060020a03600435166100f1565b60405190815260200160405180910390f35b34156100a757600080fd5b610056600160a060020a036004351660243560ff60443516606435608435610103565b60005433600160a060020a03908116911614156100ef57600054600160a060020a0316ff5b565b60016020526000908152604090205481565b600160a060020a0385166000908152600160205260408120548190861161012957600080fd5b3087876040516c01000000000000000000000000600160a060020a03948516810282529290931690910260148301526028820152604801604051809103902091506001828686866040516000815260200160405260006040516020015260405193845260ff90921660208085019190915260408085019290925260608401929092526080909201915160208103908084039060008661646e5a03f115156101cf57600080fd5b505060206040510351600054600160a060020a039081169116146101f257600080fd5b50600160a060020a03808716600090815260016020526040902054860390301631811161026257600160a060020a0387166000818152600160205260409081902088905582156108fc0290839051600060405180830381858888f19350505050151561025d57600080fd5b6102b7565b6000547f2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f97890600160a060020a0316604051600160a060020a03909116815260200160405180910390a186600160a060020a0316ff5b505050505050505600a165627a7a72305820533e856fc37e3d64d1706bcc7dfb6b1d490c8d566ea498d9d01ec08965a896ca0029`

// DeployChequebook deploys a new Ethereum contract, binding an instance of Chequebook to it.
func DeployChequebook(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Chequebook, error) {
	parsed, err := abi.JSON(strings.NewReader(ChequebookABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ChequebookBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Chequebook{ChequebookCaller: ChequebookCaller{contract: contract}, ChequebookTransactor: ChequebookTransactor{contract: contract}, ChequebookFilterer: ChequebookFilterer{contract: contract}}, nil
}

// Chequebook is an auto generated Go binding around an Ethereum contract.
type Chequebook struct {
	ChequebookCaller     // Read-only binding to the contract
	ChequebookTransactor // Write-only binding to the contract
	ChequebookFilterer   // Log filterer for contract events
}

// ChequebookCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChequebookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChequebookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChequebookTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChequebookFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChequebookFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChequebookSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChequebookSession struct {
	Contract     *Chequebook       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ChequebookCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChequebookCallerSession struct {
	Contract *ChequebookCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ChequebookTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChequebookTransactorSession struct {
	Contract     *ChequebookTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ChequebookRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChequebookRaw struct {
	Contract *Chequebook // Generic contract binding to access the raw methods on
}

// ChequebookCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChequebookCallerRaw struct {
	Contract *ChequebookCaller // Generic read-only contract binding to access the raw methods on
}

// ChequebookTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChequebookTransactorRaw struct {
	Contract *ChequebookTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChequebook creates a new instance of Chequebook, bound to a specific deployed contract.
func NewChequebook(address common.Address, backend bind.ContractBackend) (*Chequebook, error) {
	contract, err := bindChequebook(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Chequebook{ChequebookCaller: ChequebookCaller{contract: contract}, ChequebookTransactor: ChequebookTransactor{contract: contract}, ChequebookFilterer: ChequebookFilterer{contract: contract}}, nil
}

// NewChequebookCaller creates a new read-only instance of Chequebook, bound to a specific deployed contract.
func NewChequebookCaller(address common.Address, caller bind.ContractCaller) (*ChequebookCaller, error) {
	contract, err := bindChequebook(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChequebookCaller{contract: contract}, nil
}

// NewChequebookTransactor creates a new write-only instance of Chequebook, bound to a specific deployed contract.
func NewChequebookTransactor(address common.Address, transactor bind.ContractTransactor) (*ChequebookTransactor, error) {
	contract, err := bindChequebook(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChequebookTransactor{contract: contract}, nil
}

// NewChequebookFilterer creates a new log filterer instance of Chequebook, bound to a specific deployed contract.
func NewChequebookFilterer(address common.Address, filterer bind.ContractFilterer) (*ChequebookFilterer, error) {
	contract, err := bindChequebook(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChequebookFilterer{contract: contract}, nil
}

// bindChequebook binds a generic wrapper to an already deployed contract.
func bindChequebook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ChequebookABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Chequebook *ChequebookRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Chequebook.Contract.ChequebookCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Chequebook *ChequebookRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Chequebook.Contract.ChequebookTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Chequebook *ChequebookRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Chequebook.Contract.ChequebookTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Chequebook *ChequebookCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Chequebook.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Chequebook *ChequebookTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Chequebook.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Chequebook *ChequebookTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Chequebook.Contract.contract.Transact(opts, method, params...)
}

// Sent is a free data retrieval call binding the contract method 0x7bf786f8.
//
// Solidity: function sent( address) constant returns(uint256)
func (_Chequebook *ChequebookCaller) Sent(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Chequebook.contract.Call(opts, out, "sent", arg0)
	return *ret0, err
}

// Sent is a free data retrieval call binding the contract method 0x7bf786f8.
//
// Solidity: function sent( address) constant returns(uint256)
func (_Chequebook *ChequebookSession) Sent(arg0 common.Address) (*big.Int, error) {
	return _Chequebook.Contract.Sent(&_Chequebook.CallOpts, arg0)
}

// Sent is a free data retrieval call binding the contract method 0x7bf786f8.
//
// Solidity: function sent( address) constant returns(uint256)
func (_Chequebook *ChequebookCallerSession) Sent(arg0 common.Address) (*big.Int, error) {
	return _Chequebook.Contract.Sent(&_Chequebook.CallOpts, arg0)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequebook *ChequebookTransactor) Cash(opts *bind.TransactOpts, beneficiary common.Address, amount *big.Int, sigV uint8, sigR [32]byte, sigS [32]byte) (*types.Transaction, error) {
	return _Chequebook.contract.Transact(opts, "cash", beneficiary, amount, sigV, sigR, sigS)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequebook *ChequebookSession) Cash(beneficiary common.Address, amount *big.Int, sigV uint8, sigR [32]byte, sigS [32]byte) (*types.Transaction, error) {
	return _Chequebook.Contract.Cash(&_Chequebook.TransactOpts, beneficiary, amount, sigV, sigR, sigS)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequebook *ChequebookTransactorSession) Cash(beneficiary common.Address, amount *big.Int, sigV uint8, sigR [32]byte, sigS [32]byte) (*types.Transaction, error) {
	return _Chequebook.Contract.Cash(&_Chequebook.TransactOpts, beneficiary, amount, sigV, sigR, sigS)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Chequebook *ChequebookTransactor) Kill(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Chequebook.contract.Transact(opts, "kill")
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Chequebook *ChequebookSession) Kill() (*types.Transaction, error) {
	return _Chequebook.Contract.Kill(&_Chequebook.TransactOpts)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Chequebook *ChequebookTransactorSession) Kill() (*types.Transaction, error) {
	return _Chequebook.Contract.Kill(&_Chequebook.TransactOpts)
}

// ChequebookOverdraftIterator is returned from FilterOverdraft and is used to iterate over the raw logs and unpacked data for Overdraft events raised by the Chequebook contract.
type ChequebookOverdraftIterator struct {
	Event *ChequebookOverdraft // Event containing the contract specifics and raw log

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
func (it *ChequebookOverdraftIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChequebookOverdraft)
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
		it.Event = new(ChequebookOverdraft)
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

// Error retruned any retrieval or parsing error occurred during filtering.
func (it *ChequebookOverdraftIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChequebookOverdraftIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChequebookOverdraft represents a Overdraft event raised by the Chequebook contract.
type ChequebookOverdraft struct {
	Deadbeat common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOverdraft is a free log retrieval operation binding the contract event 0x2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f978.
//
// Solidity: event Overdraft(deadbeat address)
func (_Chequebook *ChequebookFilterer) FilterOverdraft(opts *bind.FilterOpts) (*ChequebookOverdraftIterator, error) {

	logs, sub, err := _Chequebook.contract.FilterLogs(opts, "Overdraft")
	if err != nil {
		return nil, err
	}
	return &ChequebookOverdraftIterator{contract: _Chequebook.contract, event: "Overdraft", logs: logs, sub: sub}, nil
}

// WatchOverdraft is a free log subscription operation binding the contract event 0x2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f978.
//
// Solidity: event Overdraft(deadbeat address)
func (_Chequebook *ChequebookFilterer) WatchOverdraft(opts *bind.WatchOpts, sink chan<- *ChequebookOverdraft) (event.Subscription, error) {

	logs, sub, err := _Chequebook.contract.WatchLogs(opts, "Overdraft")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChequebookOverdraft)
				if err := _Chequebook.contract.UnpackLog(event, "Overdraft", log); err != nil {
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
