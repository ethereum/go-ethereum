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
const ContractABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"GetAllAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GetLatestCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_sectionIndex\",\"type\":\"uint256\"},{\"name\":\"_sectionHead\",\"type\":\"bytes32\"},{\"name\":\"_chtRoot\",\"type\":\"bytes32\"},{\"name\":\"_bloomTrieRoot\",\"type\":\"bytes32\"}],\"name\":\"SetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_sectionIndex\",\"type\":\"uint256\"}],\"name\":\"GetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\"}],\"name\":\"RemoveAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\"}],\"name\":\"AddAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_adminlist\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"sectionHead\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"chtRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"bloomTrieRoot\",\"type\":\"bytes32\"}],\"name\":\"NewCheckpointEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"AddAdminEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"RemoveAdminEvent\",\"type\":\"event\"}]"

// ContractBin is the compiled bytecode used for deploying new contracts.
const ContractBin = `608060405234801561001057600080fd5b50604051610d06380380610d0683398101806040528101908080518201929190505050600060016000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060013390806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050600090505b81518110156101d2576001600080848481518110151561010057fe5b9060200190602002015173ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506001828281518110151561015857fe5b9060200190602002015190806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505080806001019150506100e4565b5050610b23806101e36000396000f300608060405260043610610078576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806345848dfc1461007d5780634d6a304c146100e9578063651c7f4914610141578063710aeac8146101b0578063753f40ca14610217578063ad6de44514610272575b600080fd5b34801561008957600080fd5b506100926102cd565b6040518080602001828103825283818151815260200191508051906020019060200280838360005b838110156100d55780820151818401526020810190506100ba565b505050509050019250505060405180910390f35b3480156100f557600080fd5b506100fe6103b4565b6040518085815260200184600019166000191681526020018360001916600019168152602001826000191660001916815260200194505050505060405180910390f35b34801561014d57600080fd5b50610196600480360381019080803590602001909291908035600019169060200190929190803560001916906020019092919080356000191690602001909291905050506103e7565b604051808215151515815260200191505060405180910390f35b3480156101bc57600080fd5b506101db60048036038101908080359060200190929190505050610555565b60405180846000191660001916815260200183600019166000191681526020018260001916600019168152602001935050505060405180910390f35b34801561022357600080fd5b50610258600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506105d6565b604051808215151515815260200191505060405180910390f35b34801561027e57600080fd5b506102b3600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506108c6565b604051808215151515815260200191505060405180910390f35b60608060006001805490506040519080825280602002602001820160405280156103065781602001602082028038833980820191505090505b509150600090505b6001805490508110156103ac5760018181548110151561032a57fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16828281518110151561036357fe5b9060200190602002019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff1681525050808060010191505061030e565b819250505090565b60008060008060008060006103ca600354610555565b925092509250600354838383965096509650965050505090919293565b6000806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411151561043557600080fd5b600160035401851415801561044d5750600060035414155b1561045b576000905061054d565b612710618000600187010201431015610477576000905061054d565b606060405190810160405280856000191681526020018460001916815260200183600019168152506002600087815260200190815260200160002060008201518160000190600019169055602082015181600101906000191690556040820151816002019060001916905590505084600381905550847fa9b81895520c1feeea26e91ca03e455eb4be8dc9ecb34071c362e67835a6fd4285858560405180846000191660001916815260200183600019166000191681526020018260001916600019168152602001935050505060405180910390a25b949350505050565b6000806000610562610a7b565b600260008681526020019081526020016000206060604051908101604052908160008201546000191660001916815260200160018201546000191660001916815260200160028201546000191660001916815250509050806000015181602001518260400151935093509350509193909250565b6000806000806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411151561062757600080fd5b60008060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054141561067757600092506108bf565b6000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009055600091505b600180549050821015610857578373ffffffffffffffffffffffffffffffffffffffff166001838154811015156106f157fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561084a578190505b60018080549050038110156107eb57600180820181548110151561075c57fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1660018281548110151561079657fe5b9060005260206000200160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550808060010191505061073c565b60018080805490500381548110151561080057fe5b9060005260206000200160006101000a81549073ffffffffffffffffffffffffffffffffffffffff0219169055600180818180549050039150816108449190610aa6565b50610857565b81806001019250506106be565b7f16f0165239516629af323911340c1c906029142cb0dc3c32c93ce6808d98a4ee84604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a1600192505b5050919050565b6000806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411151561091457600080fd5b60008060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411156109645760009050610a76565b60016000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060018290806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550507f68377f3d45858cbcc9e95998fb55b9c0f9e16b6ebf5f3602a40e2fcd0d7b8c0082604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a1600190505b919050565b6060604051908101604052806000801916815260200160008019168152602001600080191681525090565b815481835581811115610acd57818360005260206000209182019101610acc9190610ad2565b5b505050565b610af491905b80821115610af0576000816000905550600101610ad8565b5090565b905600a165627a7a72305820de9e3481e0a4d931d0b0a0a34b53b4315479499d7deb6724e5de6d1f439d44e70029`

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
// Solidity: function GetCheckpoint(_sectionIndex uint256) constant returns(bytes32, bytes32, bytes32)
func (_Contract *ContractCaller) GetCheckpoint(opts *bind.CallOpts, _sectionIndex *big.Int) ([32]byte, [32]byte, [32]byte, error) {
	var (
		ret0 = new([32]byte)
		ret1 = new([32]byte)
		ret2 = new([32]byte)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
	}
	err := _Contract.contract.Call(opts, out, "GetCheckpoint", _sectionIndex)
	return *ret0, *ret1, *ret2, err
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(_sectionIndex uint256) constant returns(bytes32, bytes32, bytes32)
func (_Contract *ContractSession) GetCheckpoint(_sectionIndex *big.Int) ([32]byte, [32]byte, [32]byte, error) {
	return _Contract.Contract.GetCheckpoint(&_Contract.CallOpts, _sectionIndex)
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(_sectionIndex uint256) constant returns(bytes32, bytes32, bytes32)
func (_Contract *ContractCallerSession) GetCheckpoint(_sectionIndex *big.Int) ([32]byte, [32]byte, [32]byte, error) {
	return _Contract.Contract.GetCheckpoint(&_Contract.CallOpts, _sectionIndex)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32, bytes32, bytes32)
func (_Contract *ContractCaller) GetLatestCheckpoint(opts *bind.CallOpts) (*big.Int, [32]byte, [32]byte, [32]byte, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new([32]byte)
		ret2 = new([32]byte)
		ret3 = new([32]byte)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
		ret3,
	}
	err := _Contract.contract.Call(opts, out, "GetLatestCheckpoint")
	return *ret0, *ret1, *ret2, *ret3, err
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32, bytes32, bytes32)
func (_Contract *ContractSession) GetLatestCheckpoint() (*big.Int, [32]byte, [32]byte, [32]byte, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32, bytes32, bytes32)
func (_Contract *ContractCallerSession) GetLatestCheckpoint() (*big.Int, [32]byte, [32]byte, [32]byte, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// AddAdmin is a paid mutator transaction binding the contract method 0xad6de445.
//
// Solidity: function AddAdmin(_addr address) returns(bool)
func (_Contract *ContractTransactor) AddAdmin(opts *bind.TransactOpts, _addr common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "AddAdmin", _addr)
}

// AddAdmin is a paid mutator transaction binding the contract method 0xad6de445.
//
// Solidity: function AddAdmin(_addr address) returns(bool)
func (_Contract *ContractSession) AddAdmin(_addr common.Address) (*types.Transaction, error) {
	return _Contract.Contract.AddAdmin(&_Contract.TransactOpts, _addr)
}

// AddAdmin is a paid mutator transaction binding the contract method 0xad6de445.
//
// Solidity: function AddAdmin(_addr address) returns(bool)
func (_Contract *ContractTransactorSession) AddAdmin(_addr common.Address) (*types.Transaction, error) {
	return _Contract.Contract.AddAdmin(&_Contract.TransactOpts, _addr)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x753f40ca.
//
// Solidity: function RemoveAdmin(_addr address) returns(bool)
func (_Contract *ContractTransactor) RemoveAdmin(opts *bind.TransactOpts, _addr common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "RemoveAdmin", _addr)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x753f40ca.
//
// Solidity: function RemoveAdmin(_addr address) returns(bool)
func (_Contract *ContractSession) RemoveAdmin(_addr common.Address) (*types.Transaction, error) {
	return _Contract.Contract.RemoveAdmin(&_Contract.TransactOpts, _addr)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x753f40ca.
//
// Solidity: function RemoveAdmin(_addr address) returns(bool)
func (_Contract *ContractTransactorSession) RemoveAdmin(_addr common.Address) (*types.Transaction, error) {
	return _Contract.Contract.RemoveAdmin(&_Contract.TransactOpts, _addr)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0x651c7f49.
//
// Solidity: function SetCheckpoint(_sectionIndex uint256, _sectionHead bytes32, _chtRoot bytes32, _bloomTrieRoot bytes32) returns(bool)
func (_Contract *ContractTransactor) SetCheckpoint(opts *bind.TransactOpts, _sectionIndex *big.Int, _sectionHead [32]byte, _chtRoot [32]byte, _bloomTrieRoot [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "SetCheckpoint", _sectionIndex, _sectionHead, _chtRoot, _bloomTrieRoot)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0x651c7f49.
//
// Solidity: function SetCheckpoint(_sectionIndex uint256, _sectionHead bytes32, _chtRoot bytes32, _bloomTrieRoot bytes32) returns(bool)
func (_Contract *ContractSession) SetCheckpoint(_sectionIndex *big.Int, _sectionHead [32]byte, _chtRoot [32]byte, _bloomTrieRoot [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _sectionIndex, _sectionHead, _chtRoot, _bloomTrieRoot)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0x651c7f49.
//
// Solidity: function SetCheckpoint(_sectionIndex uint256, _sectionHead bytes32, _chtRoot bytes32, _bloomTrieRoot bytes32) returns(bool)
func (_Contract *ContractTransactorSession) SetCheckpoint(_sectionIndex *big.Int, _sectionHead [32]byte, _chtRoot [32]byte, _bloomTrieRoot [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _sectionIndex, _sectionHead, _chtRoot, _bloomTrieRoot)
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
	Addr common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterAddAdminEvent is a free log retrieval operation binding the contract event 0x68377f3d45858cbcc9e95998fb55b9c0f9e16b6ebf5f3602a40e2fcd0d7b8c00.
//
// Solidity: e AddAdminEvent(addr address)
func (_Contract *ContractFilterer) FilterAddAdminEvent(opts *bind.FilterOpts) (*ContractAddAdminEventIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "AddAdminEvent")
	if err != nil {
		return nil, err
	}
	return &ContractAddAdminEventIterator{contract: _Contract.contract, event: "AddAdminEvent", logs: logs, sub: sub}, nil
}

// WatchAddAdminEvent is a free log subscription operation binding the contract event 0x68377f3d45858cbcc9e95998fb55b9c0f9e16b6ebf5f3602a40e2fcd0d7b8c00.
//
// Solidity: e AddAdminEvent(addr address)
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
	Index         *big.Int
	SectionHead   [32]byte
	ChtRoot       [32]byte
	BloomTrieRoot [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterNewCheckpointEvent is a free log retrieval operation binding the contract event 0xa9b81895520c1feeea26e91ca03e455eb4be8dc9ecb34071c362e67835a6fd42.
//
// Solidity: e NewCheckpointEvent(index indexed uint256, sectionHead bytes32, chtRoot bytes32, bloomTrieRoot bytes32)
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

// WatchNewCheckpointEvent is a free log subscription operation binding the contract event 0xa9b81895520c1feeea26e91ca03e455eb4be8dc9ecb34071c362e67835a6fd42.
//
// Solidity: e NewCheckpointEvent(index indexed uint256, sectionHead bytes32, chtRoot bytes32, bloomTrieRoot bytes32)
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
	Addr common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterRemoveAdminEvent is a free log retrieval operation binding the contract event 0x16f0165239516629af323911340c1c906029142cb0dc3c32c93ce6808d98a4ee.
//
// Solidity: e RemoveAdminEvent(addr address)
func (_Contract *ContractFilterer) FilterRemoveAdminEvent(opts *bind.FilterOpts) (*ContractRemoveAdminEventIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "RemoveAdminEvent")
	if err != nil {
		return nil, err
	}
	return &ContractRemoveAdminEventIterator{contract: _Contract.contract, event: "RemoveAdminEvent", logs: logs, sub: sub}, nil
}

// WatchRemoveAdminEvent is a free log subscription operation binding the contract event 0x16f0165239516629af323911340c1c906029142cb0dc3c32c93ce6808d98a4ee.
//
// Solidity: e RemoveAdminEvent(addr address)
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
