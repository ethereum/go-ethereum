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

// ContractABI is the input ABI used to generate the binding from.
const ContractABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"GetAllAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GetLatestCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"uint64\"},{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_recentNumber\",\"type\":\"uint256\"},{\"name\":\"_recentHash\",\"type\":\"bytes32\"},{\"name\":\"_hash\",\"type\":\"bytes32\"},{\"name\":\"_sectionIndex\",\"type\":\"uint64\"},{\"name\":\"v\",\"type\":\"uint8[]\"},{\"name\":\"r\",\"type\":\"bytes32[]\"},{\"name\":\"s\",\"type\":\"bytes32[]\"}],\"name\":\"SetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_adminlist\",\"type\":\"address[]\"},{\"name\":\"_sectionSize\",\"type\":\"uint256\"},{\"name\":\"_processConfirms\",\"type\":\"uint256\"},{\"name\":\"_threshold\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"index\",\"type\":\"uint64\"},{\"indexed\":false,\"name\":\"checkpointHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"v\",\"type\":\"uint8\"},{\"indexed\":false,\"name\":\"r\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"NewCheckpointVote\",\"type\":\"event\"}]"

// ContractBin is the compiled bytecode used for deploying new contracts.
const ContractBin = `608060405234801561001057600080fd5b506040516108153803806108158339818101604052608081101561003357600080fd5b81019080805164010000000081111561004b57600080fd5b8201602081018481111561005e57600080fd5b815185602082028301116401000000008211171561007b57600080fd5b505060208201516040830151606090930151919450925060005b84518110156101415760016000808784815181106100af57fe5b60200260200101516001600160a01b03166001600160a01b0316815260200190815260200160002060006101000a81548160ff02191690831515021790555060018582815181106100fc57fe5b60209081029190910181015182546001808201855560009485529290932090920180546001600160a01b0319166001600160a01b039093169290921790915501610095565b50600592909255600655600755506106b78061015e6000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c806345848dfc146100465780634d6a304c1461009e578063d459fc46146100cf575b600080fd5b61004e6102b0565b60408051602080825283518183015283519192839290830191858101910280838360005b8381101561008a578181015183820152602001610072565b505050509050019250505060405180910390f35b6100a661034f565b6040805167ffffffffffffffff9094168452602084019290925282820152519081900360600190f35b61029c600480360360e08110156100e557600080fd5b81359160208101359160408201359167ffffffffffffffff6060820135169181019060a08101608082013564010000000081111561012257600080fd5b82018360208201111561013457600080fd5b8035906020019184602083028401116401000000008311171561015657600080fd5b91908080602002602001604051908101604052809392919081815260200183836020028082843760009201919091525092959493602081019350359150506401000000008111156101a657600080fd5b8201836020820111156101b857600080fd5b803590602001918460208302840111640100000000831117156101da57600080fd5b919080806020026020016040519081016040528093929190818152602001838360200280828437600092019190915250929594936020810193503591505064010000000081111561022a57600080fd5b82018360208201111561023c57600080fd5b8035906020019184602083028401116401000000008311171561025e57600080fd5b91908080602002602001604051908101604052809392919081815260200183836020028082843760009201919091525092955061036a945050505050565b604080519115158252519081900360200190f35b6060806001805490506040519080825280602002602001820160405280156102e2578160200160208202803883390190505b50905060005b60015481101561034957600181815481106102ff57fe5b9060005260206000200160009054906101000a90046001600160a01b031682828151811061032957fe5b6001600160a01b03909216602092830291909101909101526001016102e8565b50905090565b60025460045460035467ffffffffffffffff90921691909192565b3360009081526020819052604081205460ff1661038657600080fd5b8688401461039357600080fd5b82518451146103a157600080fd5b81518451146103af57600080fd5b6006546005548660010167ffffffffffffffff1602014310156103d457506000610677565b60025467ffffffffffffffff90811690861610156103f457506000610677565b60025467ffffffffffffffff8681169116148015610426575067ffffffffffffffff8516151580610426575060035415155b1561043357506000610677565b8561044057506000610677565b60408051601960f81b6020808301919091526000602183018190523060601b60228401526001600160c01b031960c08a901b166036840152603e8084018b905284518085039091018152605e909301909352815191012090805b86518110156106715760006001848984815181106104b457fe5b60200260200101518985815181106104c857fe5b60200260200101518986815181106104dc57fe5b602002602001015160405160008152602001604052604051808581526020018460ff1660ff1681526020018381526020018281526020019450505050506020604051602081039080840390855afa15801561053b573d6000803e3d6000fd5b505060408051601f1901516001600160a01b03811660009081526020819052919091205490925060ff16905061057057600080fd5b826001600160a01b0316816001600160a01b03161161058e57600080fd5b8092508867ffffffffffffffff167fce51ffa16246bcaf0899f6504f473cd0114f430f566cef71ab7e03d3dde42a418b8a85815181106105ca57fe5b60200260200101518a86815181106105de57fe5b60200260200101518a87815181106105f257fe5b6020026020010151604051808581526020018460ff1660ff16815260200183815260200182815260200194505050505060405180910390a260075482600101106106685750505060048790555050436003556002805467ffffffffffffffff191667ffffffffffffffff86161790556001610677565b5060010161049a565b50600080fd5b97965050505050505056fea265627a7a723058208677c14fda2c1f741620e42301a9c1d509cdc26b5bffc651bfe32fa11112990e64736f6c634300050a0032`

// DeployContract deploys a new Ethereum contract, binding an instance of Contract to it.
func DeployContract(auth *bind.TransactOpts, backend bind.ContractBackend, _adminlist []common.Address, _sectionSize *big.Int, _processConfirms *big.Int, _threshold *big.Int) (common.Address, *types.Transaction, *Contract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ContractBin), backend, _adminlist, _sectionSize, _processConfirms, _threshold)
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

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint64, bytes32, uint256)
func (_Contract *ContractCaller) GetLatestCheckpoint(opts *bind.CallOpts) (uint64, [32]byte, *big.Int, error) {
	var (
		ret0 = new(uint64)
		ret1 = new([32]byte)
		ret2 = new(*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
	}
	err := _Contract.contract.Call(opts, out, "GetLatestCheckpoint")
	return *ret0, *ret1, *ret2, err
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint64, bytes32, uint256)
func (_Contract *ContractSession) GetLatestCheckpoint() (uint64, [32]byte, *big.Int, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint64, bytes32, uint256)
func (_Contract *ContractCallerSession) GetLatestCheckpoint() (uint64, [32]byte, *big.Int, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0xd459fc46.
//
// Solidity: function SetCheckpoint(uint256 _recentNumber, bytes32 _recentHash, bytes32 _hash, uint64 _sectionIndex, uint8[] v, bytes32[] r, bytes32[] s) returns(bool)
func (_Contract *ContractTransactor) SetCheckpoint(opts *bind.TransactOpts, _recentNumber *big.Int, _recentHash [32]byte, _hash [32]byte, _sectionIndex uint64, v []uint8, r [][32]byte, s [][32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "SetCheckpoint", _recentNumber, _recentHash, _hash, _sectionIndex, v, r, s)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0xd459fc46.
//
// Solidity: function SetCheckpoint(uint256 _recentNumber, bytes32 _recentHash, bytes32 _hash, uint64 _sectionIndex, uint8[] v, bytes32[] r, bytes32[] s) returns(bool)
func (_Contract *ContractSession) SetCheckpoint(_recentNumber *big.Int, _recentHash [32]byte, _hash [32]byte, _sectionIndex uint64, v []uint8, r [][32]byte, s [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _recentNumber, _recentHash, _hash, _sectionIndex, v, r, s)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0xd459fc46.
//
// Solidity: function SetCheckpoint(uint256 _recentNumber, bytes32 _recentHash, bytes32 _hash, uint64 _sectionIndex, uint8[] v, bytes32[] r, bytes32[] s) returns(bool)
func (_Contract *ContractTransactorSession) SetCheckpoint(_recentNumber *big.Int, _recentHash [32]byte, _hash [32]byte, _sectionIndex uint64, v []uint8, r [][32]byte, s [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _recentNumber, _recentHash, _hash, _sectionIndex, v, r, s)
}

// ContractNewCheckpointVoteIterator is returned from FilterNewCheckpointVote and is used to iterate over the raw logs and unpacked data for NewCheckpointVote events raised by the Contract contract.
type ContractNewCheckpointVoteIterator struct {
	Event *ContractNewCheckpointVote // Event containing the contract specifics and raw log

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
func (it *ContractNewCheckpointVoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractNewCheckpointVote)
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
		it.Event = new(ContractNewCheckpointVote)
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
func (it *ContractNewCheckpointVoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractNewCheckpointVoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractNewCheckpointVote represents a NewCheckpointVote event raised by the Contract contract.
type ContractNewCheckpointVote struct {
	Index          uint64
	CheckpointHash [32]byte
	V              uint8
	R              [32]byte
	S              [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterNewCheckpointVote is a free log retrieval operation binding the contract event 0xce51ffa16246bcaf0899f6504f473cd0114f430f566cef71ab7e03d3dde42a41.
//
// Solidity: event NewCheckpointVote(uint64 indexed index, bytes32 checkpointHash, uint8 v, bytes32 r, bytes32 s)
func (_Contract *ContractFilterer) FilterNewCheckpointVote(opts *bind.FilterOpts, index []uint64) (*ContractNewCheckpointVoteIterator, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "NewCheckpointVote", indexRule)
	if err != nil {
		return nil, err
	}
	return &ContractNewCheckpointVoteIterator{contract: _Contract.contract, event: "NewCheckpointVote", logs: logs, sub: sub}, nil
}

// WatchNewCheckpointVote is a free log subscription operation binding the contract event 0xce51ffa16246bcaf0899f6504f473cd0114f430f566cef71ab7e03d3dde42a41.
//
// Solidity: event NewCheckpointVote(uint64 indexed index, bytes32 checkpointHash, uint8 v, bytes32 r, bytes32 s)
func (_Contract *ContractFilterer) WatchNewCheckpointVote(opts *bind.WatchOpts, sink chan<- *ContractNewCheckpointVote, index []uint64) (event.Subscription, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "NewCheckpointVote", indexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractNewCheckpointVote)
				if err := _Contract.contract.UnpackLog(event, "NewCheckpointVote", log); err != nil {
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

// ParseNewCheckpointVote is a log parse operation binding the contract event 0xce51ffa16246bcaf0899f6504f473cd0114f430f566cef71ab7e03d3dde42a41.
//
// Solidity: event NewCheckpointVote(uint64 indexed index, bytes32 checkpointHash, uint8 v, bytes32 r, bytes32 s)
func (_Contract *ContractFilterer) ParseNewCheckpointVote(log types.Log) (*ContractNewCheckpointVote, error) {
	event := new(ContractNewCheckpointVote)
	if err := _Contract.contract.UnpackLog(event, "NewCheckpointVote", log); err != nil {
		return nil, err
	}
	return event, nil
}
