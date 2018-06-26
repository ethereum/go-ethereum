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

// BlockSignerABI is the input ABI used to generate the binding from.
const BlockSignerABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"sign\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"getSigners\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_signer\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"Sign\",\"type\":\"event\"}]"

// BlockSignerBin is the compiled bytecode used for deploying new contracts.
const BlockSignerBin = `0x6060604052341561000f57600080fd5b6102d88061001e6000396000f30060606040526004361061004b5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416632fb1b25f8114610050578063dfceceae14610068575b600080fd5b341561005b57600080fd5b6100666004356100d1565b005b341561007357600080fd5b61007e6004356101b3565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156100bd5780820151838201526020016100a5565b505050509050019250505060405180910390f35b43819010156100df57600080fd5b6100f1816107bc63ffffffff61023a16565b4311156100fd57600080fd5b600081815260208190526040902080546001810161011b8382610250565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff19163373ffffffffffffffffffffffffffffffffffffffff8116919091179091557f9a10b6124411386407c4a174729b856d293832181c352e98b5cb316b96cd3059908260405173ffffffffffffffffffffffffffffffffffffffff909216825260208201526040908101905180910390a150565b6101bb610279565b60008083815260200190815260200160002080548060200260200160405190810160405280929190818152602001828054801561022e57602002820191906000526020600020905b815473ffffffffffffffffffffffffffffffffffffffff168152600190910190602001808311610203575b50505050509050919050565b60008282018381101561024957fe5b9392505050565b8154818355818115116102745760008381526020902061027491810190830161028b565b505050565b60206040519081016040526000815290565b6102a991905b808211156102a55760008155600101610291565b5090565b905600a165627a7a7230582072c605c43392422edd0a185ff1131c536a80cb5329c717d23fc954f2afb51b5e0029`

// DeployBlockSigner deploys a new Ethereum contract, binding an instance of BlockSigner to it.
func DeployBlockSigner(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BlockSigner, error) {
	parsed, err := abi.JSON(strings.NewReader(BlockSignerABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(BlockSignerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BlockSigner{BlockSignerCaller: BlockSignerCaller{contract: contract}, BlockSignerTransactor: BlockSignerTransactor{contract: contract}, BlockSignerFilterer: BlockSignerFilterer{contract: contract}}, nil
}

// BlockSigner is an auto generated Go binding around an Ethereum contract.
type BlockSigner struct {
	BlockSignerCaller     // Read-only binding to the contract
	BlockSignerTransactor // Write-only binding to the contract
	BlockSignerFilterer   // Log filterer for contract events
}

// BlockSignerCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlockSignerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockSignerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlockSignerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockSignerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlockSignerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockSignerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlockSignerSession struct {
	Contract     *BlockSigner      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlockSignerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlockSignerCallerSession struct {
	Contract *BlockSignerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// BlockSignerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlockSignerTransactorSession struct {
	Contract     *BlockSignerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// BlockSignerRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlockSignerRaw struct {
	Contract *BlockSigner // Generic contract binding to access the raw methods on
}

// BlockSignerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlockSignerCallerRaw struct {
	Contract *BlockSignerCaller // Generic read-only contract binding to access the raw methods on
}

// BlockSignerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlockSignerTransactorRaw struct {
	Contract *BlockSignerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlockSigner creates a new instance of BlockSigner, bound to a specific deployed contract.
func NewBlockSigner(address common.Address, backend bind.ContractBackend) (*BlockSigner, error) {
	contract, err := bindBlockSigner(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BlockSigner{BlockSignerCaller: BlockSignerCaller{contract: contract}, BlockSignerTransactor: BlockSignerTransactor{contract: contract}, BlockSignerFilterer: BlockSignerFilterer{contract: contract}}, nil
}

// NewBlockSignerCaller creates a new read-only instance of BlockSigner, bound to a specific deployed contract.
func NewBlockSignerCaller(address common.Address, caller bind.ContractCaller) (*BlockSignerCaller, error) {
	contract, err := bindBlockSigner(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlockSignerCaller{contract: contract}, nil
}

// NewBlockSignerTransactor creates a new write-only instance of BlockSigner, bound to a specific deployed contract.
func NewBlockSignerTransactor(address common.Address, transactor bind.ContractTransactor) (*BlockSignerTransactor, error) {
	contract, err := bindBlockSigner(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlockSignerTransactor{contract: contract}, nil
}

// NewBlockSignerFilterer creates a new log filterer instance of BlockSigner, bound to a specific deployed contract.
func NewBlockSignerFilterer(address common.Address, filterer bind.ContractFilterer) (*BlockSignerFilterer, error) {
	contract, err := bindBlockSigner(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlockSignerFilterer{contract: contract}, nil
}

// bindBlockSigner binds a generic wrapper to an already deployed contract.
func bindBlockSigner(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BlockSignerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockSigner *BlockSignerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _BlockSigner.Contract.BlockSignerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockSigner *BlockSignerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockSigner.Contract.BlockSignerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockSigner *BlockSignerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockSigner.Contract.BlockSignerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockSigner *BlockSignerCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _BlockSigner.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockSigner *BlockSignerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockSigner.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockSigner *BlockSignerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockSigner.Contract.contract.Transact(opts, method, params...)
}

// GetSigners is a free data retrieval call binding the contract method 0xdfceceae.
//
// Solidity: function getSigners(_blockNumber uint256) constant returns(address[])
func (_BlockSigner *BlockSignerCaller) GetSigners(opts *bind.CallOpts, _blockNumber *big.Int) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _BlockSigner.contract.Call(opts, out, "getSigners", _blockNumber)
	return *ret0, err
}

// GetSigners is a free data retrieval call binding the contract method 0xdfceceae.
//
// Solidity: function getSigners(_blockNumber uint256) constant returns(address[])
func (_BlockSigner *BlockSignerSession) GetSigners(_blockNumber *big.Int) ([]common.Address, error) {
	return _BlockSigner.Contract.GetSigners(&_BlockSigner.CallOpts, _blockNumber)
}

// GetSigners is a free data retrieval call binding the contract method 0xdfceceae.
//
// Solidity: function getSigners(_blockNumber uint256) constant returns(address[])
func (_BlockSigner *BlockSignerCallerSession) GetSigners(_blockNumber *big.Int) ([]common.Address, error) {
	return _BlockSigner.Contract.GetSigners(&_BlockSigner.CallOpts, _blockNumber)
}

// Sign is a paid mutator transaction binding the contract method 0x2fb1b25f.
//
// Solidity: function sign(_blockNumber uint256) returns()
func (_BlockSigner *BlockSignerTransactor) Sign(opts *bind.TransactOpts, _blockNumber *big.Int) (*types.Transaction, error) {
	return _BlockSigner.contract.Transact(opts, "sign", _blockNumber)
}

// Sign is a paid mutator transaction binding the contract method 0x2fb1b25f.
//
// Solidity: function sign(_blockNumber uint256) returns()
func (_BlockSigner *BlockSignerSession) Sign(_blockNumber *big.Int) (*types.Transaction, error) {
	return _BlockSigner.Contract.Sign(&_BlockSigner.TransactOpts, _blockNumber)
}

// Sign is a paid mutator transaction binding the contract method 0x2fb1b25f.
//
// Solidity: function sign(_blockNumber uint256) returns()
func (_BlockSigner *BlockSignerTransactorSession) Sign(_blockNumber *big.Int) (*types.Transaction, error) {
	return _BlockSigner.Contract.Sign(&_BlockSigner.TransactOpts, _blockNumber)
}

// BlockSignerSignIterator is returned from FilterSign and is used to iterate over the raw logs and unpacked data for Sign events raised by the BlockSigner contract.
type BlockSignerSignIterator struct {
	Event *BlockSignerSign // Event containing the contract specifics and raw log

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
func (it *BlockSignerSignIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlockSignerSign)
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
		it.Event = new(BlockSignerSign)
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
func (it *BlockSignerSignIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlockSignerSignIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlockSignerSign represents a Sign event raised by the BlockSigner contract.
type BlockSignerSign struct {
	Signer      common.Address
	BlockNumber *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSign is a free log retrieval operation binding the contract event 0x9a10b6124411386407c4a174729b856d293832181c352e98b5cb316b96cd3059.
//
// Solidity: event Sign(_signer address, _blockNumber uint256)
func (_BlockSigner *BlockSignerFilterer) FilterSign(opts *bind.FilterOpts) (*BlockSignerSignIterator, error) {

	logs, sub, err := _BlockSigner.contract.FilterLogs(opts, "Sign")
	if err != nil {
		return nil, err
	}
	return &BlockSignerSignIterator{contract: _BlockSigner.contract, event: "Sign", logs: logs, sub: sub}, nil
}

// WatchSign is a free log subscription operation binding the contract event 0x9a10b6124411386407c4a174729b856d293832181c352e98b5cb316b96cd3059.
//
// Solidity: event Sign(_signer address, _blockNumber uint256)
func (_BlockSigner *BlockSignerFilterer) WatchSign(opts *bind.WatchOpts, sink chan<- *BlockSignerSign) (event.Subscription, error) {

	logs, sub, err := _BlockSigner.contract.WatchLogs(opts, "Sign")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlockSignerSign)
				if err := _BlockSigner.contract.UnpackLog(event, "Sign", log); err != nil {
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
