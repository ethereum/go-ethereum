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

// ContractABI is the input ABI used to generate the binding from.
const ContractABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\"},{\"name\":\"_description\",\"type\":\"string\"}],\"name\":\"AddAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GetAllAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GetLatestCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_sectionIndex\",\"type\":\"uint256\"}],\"name\":\"GetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\"},{\"name\":\"_reason\",\"type\":\"string\"}],\"name\":\"RemoveAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_sectionIndex\",\"type\":\"uint256\"},{\"name\":\"_hash\",\"type\":\"bytes32\"}],\"name\":\"SetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_adminlist\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"grantor\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"checkpointHash\",\"type\":\"bytes32\"}],\"name\":\"NewCheckpointEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"grantor\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"description\",\"type\":\"string\"}],\"name\":\"AddAdminEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"grantor\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"RemoveAdminEvent\",\"type\":\"event\"}]"

// ContractBin is the compiled bytecode used for deploying new contracts.
const ContractBin = `608060405234801561001057600080fd5b50604051610de2380380610de283398101806040528101908080518201929190505050600060016000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060013390806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050600090505b81518110156101d2576001600080848481518110151561010057fe5b9060200190602002015173ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506001828281518110151561015857fe5b9060200190602002015190806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505080806001019150506100e4565b5050610bff806101e36000396000f300608060405260043610610078576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633561247d1461007d57806345848dfc1461011e5780634d6a304c1461018a578063710aeac8146101c4578063a5ba0be21461020d578063c4885d57146102ae575b600080fd5b34801561008957600080fd5b50610104600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290505050610301565b604051808215151515815260200191505060405180910390f35b34801561012a57600080fd5b50610133610558565b6040518080602001828103825283818151815260200191508051906020019060200280838360005b8381101561017657808201518184015260208101905061015b565b505050509050019250505060405180910390f35b34801561019657600080fd5b5061019f61063f565b6040518083815260200182600019166000191681526020019250505060405180910390f35b3480156101d057600080fd5b506101ef6004803603810190808035906020019092919050505061065e565b60405180826000191660001916815260200191505060405180910390f35b34801561021957600080fd5b50610294600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001908201803590602001908080601f016020809104026020016040519081016040528093929190818152602001838380828437820191505050505050919291929050505061067b565b604051808215151515815260200191505060405180910390f35b3480156102ba57600080fd5b506102e7600480360381019080803590602001909291908035600019169060200190929190505050610a0d565b604051808215151515815260200191505060405180910390f35b6000806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411151561034f57600080fd5b60008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054111561039f5760009050610552565b60016000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060018390806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550507fecd88a3d10808aac9f879bbae95f9e4f9687f6c9f0615b88af2085f226d3cf93833384604051808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200180602001828103825283818151815260200191508051906020019080838360005b838110156105115780820151818401526020810190506104f6565b50505050905090810190601f16801561053e5780820380516001836020036101000a031916815260200191505b5094505050505060405180910390a1600190505b92915050565b60608060006001805490506040519080825280602002602001820160405280156105915781602001602082028038833980820191505090505b509150600090505b600180549050811015610637576001818154811015156105b557fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1682828151811015156105ee57fe5b9060200190602002019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250508080600101915050610599565b819250505090565b600080600061064f60035461065e565b90506003548192509250509091565b600060026000838152602001908152602001600020549050919050565b6000806000806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541115156106cc57600080fd5b60008060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054141561071c5760009250610a05565b6000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009055600091505b6001805490508210156108fc578473ffffffffffffffffffffffffffffffffffffffff1660018381548110151561079657fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156108ef578190505b600180805490500381101561089057600180820181548110151561080157fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1660018281548110151561083b57fe5b9060005260206000200160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555080806001019150506107e1565b6001808080549050038154811015156108a557fe5b9060005260206000200160006101000a81549073ffffffffffffffffffffffffffffffffffffffff0219169055600180818180549050039150816108e99190610b82565b506108fc565b8180600101925050610763565b7fe7d6ab069cde0507dc9c5971bf93e9fd183db04690725946e8cae907367e7c1f853386604051808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200180602001828103825283818151815260200191508051906020019080838360005b838110156109c45780820151818401526020810190506109a9565b50505050905090810190601f1680156109f15780820380516001836020036101000a031916815260200191505b5094505050505060405180910390a1600192505b505092915050565b6000806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054111515610a5b57600080fd5b6003548314158015610a7257506001600354018314155b8015610a815750600060035414155b15610a8f5760009050610b7c565b610100618000600185010201431015610aab5760009050610b7c565b600060035414158015610abf575060035483145b8015610ad657506120006180006001850102014310155b15610ae45760009050610b7c565b8160026000858152602001908152602001600020816000191690555082600381905550827f5b3eab050ccf1a983a26b4f6acb13c199093c7d2e7523a0ff036519f7ddaf5603384604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182600019166000191681526020019250505060405180910390a25b92915050565b815481835581811115610ba957818360005260206000209182019101610ba89190610bae565b5b505050565b610bd091905b80821115610bcc576000816000905550600101610bb4565b5090565b905600a165627a7a723058206af229b85ceb246e4b06af39c8058d670504df652a68b19dad6b2a883f3028580029`

// DeployContract deploys a new Ethereum contract, binding an instance of Contract to it.
func DeployContract(auth *bind.TransactOpts, backend bind.ContractBackend, _adminlist []common.Address) (common.Address, *types.Transaction, *Contract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ContractBin), backend, _adminlist)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// Contract is an auto generated Go binding around an Ethereum contract.
type Contract struct {
	ContractCaller     // Read-only binding to the contract
	ContractTransactor // Write-only binding to the contract
	ContractFilterer   // Log filterer for contract events
}

// ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractSession struct {
	Contract     *Contract         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractCallerSession struct {
	Contract *ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractTransactorSession struct {
	Contract     *ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractRaw struct {
	Contract *Contract // Generic contract binding to access the raw methods on
}

// ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractCallerRaw struct {
	Contract *ContractCaller // Generic read-only contract binding to access the raw methods on
}

// ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractTransactorRaw struct {
	Contract *ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address common.Address, backend bind.ContractBackend) (*Contract, error) {
	contract, err := bindContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// NewContractCaller creates a new read-only instance of Contract, bound to a specific deployed contract.
func NewContractCaller(address common.Address, caller bind.ContractCaller) (*ContractCaller, error) {
	contract, err := bindContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractCaller{contract: contract}, nil
}

// NewContractTransactor creates a new write-only instance of Contract, bound to a specific deployed contract.
func NewContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractTransactor, error) {
	contract, err := bindContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{contract: contract}, nil
}

// NewContractFilterer creates a new log filterer instance of Contract, bound to a specific deployed contract.
func NewContractFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractFilterer, error) {
	contract, err := bindContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractFilterer{contract: contract}, nil
}

// bindContract binds a generic wrapper to an already deployed contract.
func bindContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transact(opts, method, params...)
}

// GetAllAdmin is a free data retrieval call binding the contract method 0x45848dfc.
//
// Solidity: function GetAllAdmin() constant returns(address[])
func (_Contract *ContractCaller) GetAllAdmin(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _Contract.contract.Call(opts, out, "GetAllAdmin")
	return *ret0, err
}

// GetAllAdmin is a free data retrieval call binding the contract method 0x45848dfc.
//
// Solidity: function GetAllAdmin() constant returns(address[])
func (_Contract *ContractSession) GetAllAdmin() ([]common.Address, error) {
	return _Contract.Contract.GetAllAdmin(&_Contract.CallOpts)
}

// GetAllAdmin is a free data retrieval call binding the contract method 0x45848dfc.
//
// Solidity: function GetAllAdmin() constant returns(address[])
func (_Contract *ContractCallerSession) GetAllAdmin() ([]common.Address, error) {
	return _Contract.Contract.GetAllAdmin(&_Contract.CallOpts)
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(_sectionIndex uint256) constant returns(bytes32)
func (_Contract *ContractCaller) GetCheckpoint(opts *bind.CallOpts, _sectionIndex *big.Int) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Contract.contract.Call(opts, out, "GetCheckpoint", _sectionIndex)
	return *ret0, err
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(_sectionIndex uint256) constant returns(bytes32)
func (_Contract *ContractSession) GetCheckpoint(_sectionIndex *big.Int) ([32]byte, error) {
	return _Contract.Contract.GetCheckpoint(&_Contract.CallOpts, _sectionIndex)
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(_sectionIndex uint256) constant returns(bytes32)
func (_Contract *ContractCallerSession) GetCheckpoint(_sectionIndex *big.Int) ([32]byte, error) {
	return _Contract.Contract.GetCheckpoint(&_Contract.CallOpts, _sectionIndex)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32)
func (_Contract *ContractCaller) GetLatestCheckpoint(opts *bind.CallOpts) (*big.Int, [32]byte, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new([32]byte)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _Contract.contract.Call(opts, out, "GetLatestCheckpoint")
	return *ret0, *ret1, err
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32)
func (_Contract *ContractSession) GetLatestCheckpoint() (*big.Int, [32]byte, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32)
func (_Contract *ContractCallerSession) GetLatestCheckpoint() (*big.Int, [32]byte, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x3561247d.
//
// Solidity: function AddAdmin(_addr address, _description string) returns(bool)
func (_Contract *ContractTransactor) AddAdmin(opts *bind.TransactOpts, _addr common.Address, _description string) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "AddAdmin", _addr, _description)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x3561247d.
//
// Solidity: function AddAdmin(_addr address, _description string) returns(bool)
func (_Contract *ContractSession) AddAdmin(_addr common.Address, _description string) (*types.Transaction, error) {
	return _Contract.Contract.AddAdmin(&_Contract.TransactOpts, _addr, _description)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x3561247d.
//
// Solidity: function AddAdmin(_addr address, _description string) returns(bool)
func (_Contract *ContractTransactorSession) AddAdmin(_addr common.Address, _description string) (*types.Transaction, error) {
	return _Contract.Contract.AddAdmin(&_Contract.TransactOpts, _addr, _description)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0xa5ba0be2.
//
// Solidity: function RemoveAdmin(_addr address, _reason string) returns(bool)
func (_Contract *ContractTransactor) RemoveAdmin(opts *bind.TransactOpts, _addr common.Address, _reason string) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "RemoveAdmin", _addr, _reason)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0xa5ba0be2.
//
// Solidity: function RemoveAdmin(_addr address, _reason string) returns(bool)
func (_Contract *ContractSession) RemoveAdmin(_addr common.Address, _reason string) (*types.Transaction, error) {
	return _Contract.Contract.RemoveAdmin(&_Contract.TransactOpts, _addr, _reason)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0xa5ba0be2.
//
// Solidity: function RemoveAdmin(_addr address, _reason string) returns(bool)
func (_Contract *ContractTransactorSession) RemoveAdmin(_addr common.Address, _reason string) (*types.Transaction, error) {
	return _Contract.Contract.RemoveAdmin(&_Contract.TransactOpts, _addr, _reason)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0xc4885d57.
//
// Solidity: function SetCheckpoint(_sectionIndex uint256, _hash bytes32) returns(bool)
func (_Contract *ContractTransactor) SetCheckpoint(opts *bind.TransactOpts, _sectionIndex *big.Int, _hash [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "SetCheckpoint", _sectionIndex, _hash)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0xc4885d57.
//
// Solidity: function SetCheckpoint(_sectionIndex uint256, _hash bytes32) returns(bool)
func (_Contract *ContractSession) SetCheckpoint(_sectionIndex *big.Int, _hash [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _sectionIndex, _hash)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0xc4885d57.
//
// Solidity: function SetCheckpoint(_sectionIndex uint256, _hash bytes32) returns(bool)
func (_Contract *ContractTransactorSession) SetCheckpoint(_sectionIndex *big.Int, _hash [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _sectionIndex, _hash)
}

// ContractAddAdminEventIterator is returned from FilterAddAdminEvent and is used to iterate over the raw logs and unpacked data for AddAdminEvent events raised by the Contract contract.
type ContractAddAdminEventIterator struct {
	Event *ContractAddAdminEvent // Event containing the contract specifics and raw log

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
func (it *ContractAddAdminEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractAddAdminEvent)
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
		it.Event = new(ContractAddAdminEvent)
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
func (it *ContractAddAdminEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractAddAdminEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractAddAdminEvent represents a AddAdminEvent event raised by the Contract contract.
type ContractAddAdminEvent struct {
	Addr        common.Address
	Grantor     common.Address
	Description string
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAddAdminEvent is a free log retrieval operation binding the contract event 0xecd88a3d10808aac9f879bbae95f9e4f9687f6c9f0615b88af2085f226d3cf93.
//
// Solidity: e AddAdminEvent(addr address, grantor address, description string)
func (_Contract *ContractFilterer) FilterAddAdminEvent(opts *bind.FilterOpts) (*ContractAddAdminEventIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "AddAdminEvent")
	if err != nil {
		return nil, err
	}
	return &ContractAddAdminEventIterator{contract: _Contract.contract, event: "AddAdminEvent", logs: logs, sub: sub}, nil
}

// WatchAddAdminEvent is a free log subscription operation binding the contract event 0xecd88a3d10808aac9f879bbae95f9e4f9687f6c9f0615b88af2085f226d3cf93.
//
// Solidity: e AddAdminEvent(addr address, grantor address, description string)
func (_Contract *ContractFilterer) WatchAddAdminEvent(opts *bind.WatchOpts, sink chan<- *ContractAddAdminEvent) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "AddAdminEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractAddAdminEvent)
				if err := _Contract.contract.UnpackLog(event, "AddAdminEvent", log); err != nil {
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

// ContractNewCheckpointEventIterator is returned from FilterNewCheckpointEvent and is used to iterate over the raw logs and unpacked data for NewCheckpointEvent events raised by the Contract contract.
type ContractNewCheckpointEventIterator struct {
	Event *ContractNewCheckpointEvent // Event containing the contract specifics and raw log

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
func (it *ContractNewCheckpointEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractNewCheckpointEvent)
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
		it.Event = new(ContractNewCheckpointEvent)
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
func (it *ContractNewCheckpointEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractNewCheckpointEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractNewCheckpointEvent represents a NewCheckpointEvent event raised by the Contract contract.
type ContractNewCheckpointEvent struct {
	Index          *big.Int
	Grantor        common.Address
	CheckpointHash [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterNewCheckpointEvent is a free log retrieval operation binding the contract event 0x5b3eab050ccf1a983a26b4f6acb13c199093c7d2e7523a0ff036519f7ddaf560.
//
// Solidity: e NewCheckpointEvent(index indexed uint256, grantor address, checkpointHash bytes32)
func (_Contract *ContractFilterer) FilterNewCheckpointEvent(opts *bind.FilterOpts, index []*big.Int) (*ContractNewCheckpointEventIterator, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "NewCheckpointEvent", indexRule)
	if err != nil {
		return nil, err
	}
	return &ContractNewCheckpointEventIterator{contract: _Contract.contract, event: "NewCheckpointEvent", logs: logs, sub: sub}, nil
}

// WatchNewCheckpointEvent is a free log subscription operation binding the contract event 0x5b3eab050ccf1a983a26b4f6acb13c199093c7d2e7523a0ff036519f7ddaf560.
//
// Solidity: e NewCheckpointEvent(index indexed uint256, grantor address, checkpointHash bytes32)
func (_Contract *ContractFilterer) WatchNewCheckpointEvent(opts *bind.WatchOpts, sink chan<- *ContractNewCheckpointEvent, index []*big.Int) (event.Subscription, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "NewCheckpointEvent", indexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractNewCheckpointEvent)
				if err := _Contract.contract.UnpackLog(event, "NewCheckpointEvent", log); err != nil {
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

// ContractRemoveAdminEventIterator is returned from FilterRemoveAdminEvent and is used to iterate over the raw logs and unpacked data for RemoveAdminEvent events raised by the Contract contract.
type ContractRemoveAdminEventIterator struct {
	Event *ContractRemoveAdminEvent // Event containing the contract specifics and raw log

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
func (it *ContractRemoveAdminEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractRemoveAdminEvent)
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
		it.Event = new(ContractRemoveAdminEvent)
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
func (it *ContractRemoveAdminEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractRemoveAdminEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractRemoveAdminEvent represents a RemoveAdminEvent event raised by the Contract contract.
type ContractRemoveAdminEvent struct {
	Addr    common.Address
	Grantor common.Address
	Reason  string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRemoveAdminEvent is a free log retrieval operation binding the contract event 0xe7d6ab069cde0507dc9c5971bf93e9fd183db04690725946e8cae907367e7c1f.
//
// Solidity: e RemoveAdminEvent(addr address, grantor address, reason string)
func (_Contract *ContractFilterer) FilterRemoveAdminEvent(opts *bind.FilterOpts) (*ContractRemoveAdminEventIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "RemoveAdminEvent")
	if err != nil {
		return nil, err
	}
	return &ContractRemoveAdminEventIterator{contract: _Contract.contract, event: "RemoveAdminEvent", logs: logs, sub: sub}, nil
}

// WatchRemoveAdminEvent is a free log subscription operation binding the contract event 0xe7d6ab069cde0507dc9c5971bf93e9fd183db04690725946e8cae907367e7c1f.
//
// Solidity: e RemoveAdminEvent(addr address, grantor address, reason string)
func (_Contract *ContractFilterer) WatchRemoveAdminEvent(opts *bind.WatchOpts, sink chan<- *ContractRemoveAdminEvent) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "RemoveAdminEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractRemoveAdminEvent)
				if err := _Contract.contract.UnpackLog(event, "RemoveAdminEvent", log); err != nil {
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
