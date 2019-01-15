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

// KycABI is the input ABI used to generate the binding from.
const KycABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"propose\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"owners\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"},{\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"unvote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCandidates\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"}],\"name\":\"hasVotedInvalid\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"getWithdrawCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"ownerToCandidate\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getVoters\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getWithdrawBlockNumbers\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"},{\"name\":\"_voter\",\"type\":\"address\"}],\"name\":\"getVoterCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"candidates\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getCandidateCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"vote\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"invalidKYCCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_invalidMasternode\",\"type\":\"address\"}],\"name\":\"InvalidPercent\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_invalidMasternode\",\"type\":\"address\"}],\"name\":\"VoteInvalidKYC\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"candidateCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"voterWithdrawDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"resign\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getCandidateOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxValidatorNumber\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"candidateWithdrawDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"isCandidate\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minCandidateCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_kycdata\",\"type\":\"bytes32\"}],\"name\":\"uploadKYC\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getOwnerCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minVoterCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_masternode\",\"type\":\"address\"}],\"name\":\"getOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_candidates\",\"type\":\"address[]\"},{\"name\":\"_caps\",\"type\":\"uint256[]\"},{\"name\":\"_firstOwner\",\"type\":\"address\"},{\"name\":\"_minCandidateCap\",\"type\":\"uint256\"},{\"name\":\"_minVoterCap\",\"type\":\"uint256\"},{\"name\":\"_maxValidatorNumber\",\"type\":\"uint256\"},{\"name\":\"_candidateWithdrawDelay\",\"type\":\"uint256\"},{\"name\":\"_voterWithdrawDelay\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Vote\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Unvote\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Propose\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"Resign\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Withdraw\",\"type\":\"event\"}]"

// Kyc is an auto generated Go binding around an Ethereum contract.
type Kyc struct {
	KycCaller     // Read-only binding to the contract
	KycTransactor // Write-only binding to the contract
	KycFilterer   // Log filterer for contract events
}

// KycCaller is an auto generated read-only Go binding around an Ethereum contract.
type KycCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// KycTransactor is an auto generated write-only Go binding around an Ethereum contract.
type KycTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// KycFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type KycFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// KycSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type KycSession struct {
	Contract     *Kyc              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// KycCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type KycCallerSession struct {
	Contract *KycCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// KycTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type KycTransactorSession struct {
	Contract     *KycTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// KycRaw is an auto generated low-level Go binding around an Ethereum contract.
type KycRaw struct {
	Contract *Kyc // Generic contract binding to access the raw methods on
}

// KycCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type KycCallerRaw struct {
	Contract *KycCaller // Generic read-only contract binding to access the raw methods on
}

// KycTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type KycTransactorRaw struct {
	Contract *KycTransactor // Generic write-only contract binding to access the raw methods on
}

// NewKyc creates a new instance of Kyc, bound to a specific deployed contract.
func NewKyc(address common.Address, backend bind.ContractBackend) (*Kyc, error) {
	contract, err := bindKyc(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Kyc{KycCaller: KycCaller{contract: contract}, KycTransactor: KycTransactor{contract: contract}, KycFilterer: KycFilterer{contract: contract}}, nil
}

// NewKycCaller creates a new read-only instance of Kyc, bound to a specific deployed contract.
func NewKycCaller(address common.Address, caller bind.ContractCaller) (*KycCaller, error) {
	contract, err := bindKyc(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &KycCaller{contract: contract}, nil
}

// NewKycTransactor creates a new write-only instance of Kyc, bound to a specific deployed contract.
func NewKycTransactor(address common.Address, transactor bind.ContractTransactor) (*KycTransactor, error) {
	contract, err := bindKyc(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &KycTransactor{contract: contract}, nil
}

// NewKycFilterer creates a new log filterer instance of Kyc, bound to a specific deployed contract.
func NewKycFilterer(address common.Address, filterer bind.ContractFilterer) (*KycFilterer, error) {
	contract, err := bindKyc(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &KycFilterer{contract: contract}, nil
}

// bindKyc binds a generic wrapper to an already deployed contract.
func bindKyc(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(KycABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Kyc *KycRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Kyc.Contract.KycCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Kyc *KycRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Kyc.Contract.KycTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Kyc *KycRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Kyc.Contract.KycTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Kyc *KycCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Kyc.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Kyc *KycTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Kyc.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Kyc *KycTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Kyc.Contract.contract.Transact(opts, method, params...)
}

// InvalidPercent is a free data retrieval call binding the contract method 0x7ea80829.
//
// Solidity: function InvalidPercent(address _invalidMasternode) constant returns(uint256)
func (_Kyc *KycCaller) InvalidPercent(opts *bind.CallOpts, _invalidMasternode common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "InvalidPercent", _invalidMasternode)
	return *ret0, err
}

// InvalidPercent is a free data retrieval call binding the contract method 0x7ea80829.
//
// Solidity: function InvalidPercent(address _invalidMasternode) constant returns(uint256)
func (_Kyc *KycSession) InvalidPercent(_invalidMasternode common.Address) (*big.Int, error) {
	return _Kyc.Contract.InvalidPercent(&_Kyc.CallOpts, _invalidMasternode)
}

// InvalidPercent is a free data retrieval call binding the contract method 0x7ea80829.
//
// Solidity: function InvalidPercent(address _invalidMasternode) constant returns(uint256)
func (_Kyc *KycCallerSession) InvalidPercent(_invalidMasternode common.Address) (*big.Int, error) {
	return _Kyc.Contract.InvalidPercent(&_Kyc.CallOpts, _invalidMasternode)
}

// CandidateCount is a free data retrieval call binding the contract method 0xa9a981a3.
//
// Solidity: function candidateCount() constant returns(uint256)
func (_Kyc *KycCaller) CandidateCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "candidateCount")
	return *ret0, err
}

// CandidateCount is a free data retrieval call binding the contract method 0xa9a981a3.
//
// Solidity: function candidateCount() constant returns(uint256)
func (_Kyc *KycSession) CandidateCount() (*big.Int, error) {
	return _Kyc.Contract.CandidateCount(&_Kyc.CallOpts)
}

// CandidateCount is a free data retrieval call binding the contract method 0xa9a981a3.
//
// Solidity: function candidateCount() constant returns(uint256)
func (_Kyc *KycCallerSession) CandidateCount() (*big.Int, error) {
	return _Kyc.Contract.CandidateCount(&_Kyc.CallOpts)
}

// CandidateWithdrawDelay is a free data retrieval call binding the contract method 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() constant returns(uint256)
func (_Kyc *KycCaller) CandidateWithdrawDelay(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "candidateWithdrawDelay")
	return *ret0, err
}

// CandidateWithdrawDelay is a free data retrieval call binding the contract method 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() constant returns(uint256)
func (_Kyc *KycSession) CandidateWithdrawDelay() (*big.Int, error) {
	return _Kyc.Contract.CandidateWithdrawDelay(&_Kyc.CallOpts)
}

// CandidateWithdrawDelay is a free data retrieval call binding the contract method 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() constant returns(uint256)
func (_Kyc *KycCallerSession) CandidateWithdrawDelay() (*big.Int, error) {
	return _Kyc.Contract.CandidateWithdrawDelay(&_Kyc.CallOpts)
}

// Candidates is a free data retrieval call binding the contract method 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) constant returns(address)
func (_Kyc *KycCaller) Candidates(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "candidates", arg0)
	return *ret0, err
}

// Candidates is a free data retrieval call binding the contract method 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) constant returns(address)
func (_Kyc *KycSession) Candidates(arg0 *big.Int) (common.Address, error) {
	return _Kyc.Contract.Candidates(&_Kyc.CallOpts, arg0)
}

// Candidates is a free data retrieval call binding the contract method 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) constant returns(address)
func (_Kyc *KycCallerSession) Candidates(arg0 *big.Int) (common.Address, error) {
	return _Kyc.Contract.Candidates(&_Kyc.CallOpts, arg0)
}

// GetCandidateCap is a free data retrieval call binding the contract method 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) constant returns(uint256)
func (_Kyc *KycCaller) GetCandidateCap(opts *bind.CallOpts, _candidate common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getCandidateCap", _candidate)
	return *ret0, err
}

// GetCandidateCap is a free data retrieval call binding the contract method 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) constant returns(uint256)
func (_Kyc *KycSession) GetCandidateCap(_candidate common.Address) (*big.Int, error) {
	return _Kyc.Contract.GetCandidateCap(&_Kyc.CallOpts, _candidate)
}

// GetCandidateCap is a free data retrieval call binding the contract method 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) constant returns(uint256)
func (_Kyc *KycCallerSession) GetCandidateCap(_candidate common.Address) (*big.Int, error) {
	return _Kyc.Contract.GetCandidateCap(&_Kyc.CallOpts, _candidate)
}

// GetCandidateOwner is a free data retrieval call binding the contract method 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) constant returns(address)
func (_Kyc *KycCaller) GetCandidateOwner(opts *bind.CallOpts, _candidate common.Address) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getCandidateOwner", _candidate)
	return *ret0, err
}

// GetCandidateOwner is a free data retrieval call binding the contract method 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) constant returns(address)
func (_Kyc *KycSession) GetCandidateOwner(_candidate common.Address) (common.Address, error) {
	return _Kyc.Contract.GetCandidateOwner(&_Kyc.CallOpts, _candidate)
}

// GetCandidateOwner is a free data retrieval call binding the contract method 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) constant returns(address)
func (_Kyc *KycCallerSession) GetCandidateOwner(_candidate common.Address) (common.Address, error) {
	return _Kyc.Contract.GetCandidateOwner(&_Kyc.CallOpts, _candidate)
}

// GetCandidates is a free data retrieval call binding the contract method 0x06a49fce.
//
// Solidity: function getCandidates() constant returns(address[])
func (_Kyc *KycCaller) GetCandidates(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getCandidates")
	return *ret0, err
}

// GetCandidates is a free data retrieval call binding the contract method 0x06a49fce.
//
// Solidity: function getCandidates() constant returns(address[])
func (_Kyc *KycSession) GetCandidates() ([]common.Address, error) {
	return _Kyc.Contract.GetCandidates(&_Kyc.CallOpts)
}

// GetCandidates is a free data retrieval call binding the contract method 0x06a49fce.
//
// Solidity: function getCandidates() constant returns(address[])
func (_Kyc *KycCallerSession) GetCandidates() ([]common.Address, error) {
	return _Kyc.Contract.GetCandidates(&_Kyc.CallOpts)
}

// GetOwner is a free data retrieval call binding the contract method 0xfa544161.
//
// Solidity: function getOwner(address _masternode) constant returns(address)
func (_Kyc *KycCaller) GetOwner(opts *bind.CallOpts, _masternode common.Address) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getOwner", _masternode)
	return *ret0, err
}

// GetOwner is a free data retrieval call binding the contract method 0xfa544161.
//
// Solidity: function getOwner(address _masternode) constant returns(address)
func (_Kyc *KycSession) GetOwner(_masternode common.Address) (common.Address, error) {
	return _Kyc.Contract.GetOwner(&_Kyc.CallOpts, _masternode)
}

// GetOwner is a free data retrieval call binding the contract method 0xfa544161.
//
// Solidity: function getOwner(address _masternode) constant returns(address)
func (_Kyc *KycCallerSession) GetOwner(_masternode common.Address) (common.Address, error) {
	return _Kyc.Contract.GetOwner(&_Kyc.CallOpts, _masternode)
}

// GetOwnerCount is a free data retrieval call binding the contract method 0xef18374a.
//
// Solidity: function getOwnerCount() constant returns(uint256)
func (_Kyc *KycCaller) GetOwnerCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getOwnerCount")
	return *ret0, err
}

// GetOwnerCount is a free data retrieval call binding the contract method 0xef18374a.
//
// Solidity: function getOwnerCount() constant returns(uint256)
func (_Kyc *KycSession) GetOwnerCount() (*big.Int, error) {
	return _Kyc.Contract.GetOwnerCount(&_Kyc.CallOpts)
}

// GetOwnerCount is a free data retrieval call binding the contract method 0xef18374a.
//
// Solidity: function getOwnerCount() constant returns(uint256)
func (_Kyc *KycCallerSession) GetOwnerCount() (*big.Int, error) {
	return _Kyc.Contract.GetOwnerCount(&_Kyc.CallOpts)
}

// GetVoterCap is a free data retrieval call binding the contract method 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) constant returns(uint256)
func (_Kyc *KycCaller) GetVoterCap(opts *bind.CallOpts, _candidate common.Address, _voter common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getVoterCap", _candidate, _voter)
	return *ret0, err
}

// GetVoterCap is a free data retrieval call binding the contract method 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) constant returns(uint256)
func (_Kyc *KycSession) GetVoterCap(_candidate common.Address, _voter common.Address) (*big.Int, error) {
	return _Kyc.Contract.GetVoterCap(&_Kyc.CallOpts, _candidate, _voter)
}

// GetVoterCap is a free data retrieval call binding the contract method 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) constant returns(uint256)
func (_Kyc *KycCallerSession) GetVoterCap(_candidate common.Address, _voter common.Address) (*big.Int, error) {
	return _Kyc.Contract.GetVoterCap(&_Kyc.CallOpts, _candidate, _voter)
}

// GetVoters is a free data retrieval call binding the contract method 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) constant returns(address[])
func (_Kyc *KycCaller) GetVoters(opts *bind.CallOpts, _candidate common.Address) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getVoters", _candidate)
	return *ret0, err
}

// GetVoters is a free data retrieval call binding the contract method 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) constant returns(address[])
func (_Kyc *KycSession) GetVoters(_candidate common.Address) ([]common.Address, error) {
	return _Kyc.Contract.GetVoters(&_Kyc.CallOpts, _candidate)
}

// GetVoters is a free data retrieval call binding the contract method 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) constant returns(address[])
func (_Kyc *KycCallerSession) GetVoters(_candidate common.Address) ([]common.Address, error) {
	return _Kyc.Contract.GetVoters(&_Kyc.CallOpts, _candidate)
}

// GetWithdrawBlockNumbers is a free data retrieval call binding the contract method 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() constant returns(uint256[])
func (_Kyc *KycCaller) GetWithdrawBlockNumbers(opts *bind.CallOpts) ([]*big.Int, error) {
	var (
		ret0 = new([]*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getWithdrawBlockNumbers")
	return *ret0, err
}

// GetWithdrawBlockNumbers is a free data retrieval call binding the contract method 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() constant returns(uint256[])
func (_Kyc *KycSession) GetWithdrawBlockNumbers() ([]*big.Int, error) {
	return _Kyc.Contract.GetWithdrawBlockNumbers(&_Kyc.CallOpts)
}

// GetWithdrawBlockNumbers is a free data retrieval call binding the contract method 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() constant returns(uint256[])
func (_Kyc *KycCallerSession) GetWithdrawBlockNumbers() ([]*big.Int, error) {
	return _Kyc.Contract.GetWithdrawBlockNumbers(&_Kyc.CallOpts)
}

// GetWithdrawCap is a free data retrieval call binding the contract method 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) constant returns(uint256)
func (_Kyc *KycCaller) GetWithdrawCap(opts *bind.CallOpts, _blockNumber *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "getWithdrawCap", _blockNumber)
	return *ret0, err
}

// GetWithdrawCap is a free data retrieval call binding the contract method 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) constant returns(uint256)
func (_Kyc *KycSession) GetWithdrawCap(_blockNumber *big.Int) (*big.Int, error) {
	return _Kyc.Contract.GetWithdrawCap(&_Kyc.CallOpts, _blockNumber)
}

// GetWithdrawCap is a free data retrieval call binding the contract method 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) constant returns(uint256)
func (_Kyc *KycCallerSession) GetWithdrawCap(_blockNumber *big.Int) (*big.Int, error) {
	return _Kyc.Contract.GetWithdrawCap(&_Kyc.CallOpts, _blockNumber)
}

// HasVotedInvalid is a free data retrieval call binding the contract method 0x0e3e4fb8.
//
// Solidity: function hasVotedInvalid(address , address ) constant returns(bool)
func (_Kyc *KycCaller) HasVotedInvalid(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "hasVotedInvalid", arg0, arg1)
	return *ret0, err
}

// HasVotedInvalid is a free data retrieval call binding the contract method 0x0e3e4fb8.
//
// Solidity: function hasVotedInvalid(address , address ) constant returns(bool)
func (_Kyc *KycSession) HasVotedInvalid(arg0 common.Address, arg1 common.Address) (bool, error) {
	return _Kyc.Contract.HasVotedInvalid(&_Kyc.CallOpts, arg0, arg1)
}

// HasVotedInvalid is a free data retrieval call binding the contract method 0x0e3e4fb8.
//
// Solidity: function hasVotedInvalid(address , address ) constant returns(bool)
func (_Kyc *KycCallerSession) HasVotedInvalid(arg0 common.Address, arg1 common.Address) (bool, error) {
	return _Kyc.Contract.HasVotedInvalid(&_Kyc.CallOpts, arg0, arg1)
}

// InvalidKYCCount is a free data retrieval call binding the contract method 0x72e44a38.
//
// Solidity: function invalidKYCCount(address ) constant returns(uint256)
func (_Kyc *KycCaller) InvalidKYCCount(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "invalidKYCCount", arg0)
	return *ret0, err
}

// InvalidKYCCount is a free data retrieval call binding the contract method 0x72e44a38.
//
// Solidity: function invalidKYCCount(address ) constant returns(uint256)
func (_Kyc *KycSession) InvalidKYCCount(arg0 common.Address) (*big.Int, error) {
	return _Kyc.Contract.InvalidKYCCount(&_Kyc.CallOpts, arg0)
}

// InvalidKYCCount is a free data retrieval call binding the contract method 0x72e44a38.
//
// Solidity: function invalidKYCCount(address ) constant returns(uint256)
func (_Kyc *KycCallerSession) InvalidKYCCount(arg0 common.Address) (*big.Int, error) {
	return _Kyc.Contract.InvalidKYCCount(&_Kyc.CallOpts, arg0)
}

// IsCandidate is a free data retrieval call binding the contract method 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) constant returns(bool)
func (_Kyc *KycCaller) IsCandidate(opts *bind.CallOpts, _candidate common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "isCandidate", _candidate)
	return *ret0, err
}

// IsCandidate is a free data retrieval call binding the contract method 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) constant returns(bool)
func (_Kyc *KycSession) IsCandidate(_candidate common.Address) (bool, error) {
	return _Kyc.Contract.IsCandidate(&_Kyc.CallOpts, _candidate)
}

// IsCandidate is a free data retrieval call binding the contract method 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) constant returns(bool)
func (_Kyc *KycCallerSession) IsCandidate(_candidate common.Address) (bool, error) {
	return _Kyc.Contract.IsCandidate(&_Kyc.CallOpts, _candidate)
}

// MaxValidatorNumber is a free data retrieval call binding the contract method 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() constant returns(uint256)
func (_Kyc *KycCaller) MaxValidatorNumber(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "maxValidatorNumber")
	return *ret0, err
}

// MaxValidatorNumber is a free data retrieval call binding the contract method 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() constant returns(uint256)
func (_Kyc *KycSession) MaxValidatorNumber() (*big.Int, error) {
	return _Kyc.Contract.MaxValidatorNumber(&_Kyc.CallOpts)
}

// MaxValidatorNumber is a free data retrieval call binding the contract method 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() constant returns(uint256)
func (_Kyc *KycCallerSession) MaxValidatorNumber() (*big.Int, error) {
	return _Kyc.Contract.MaxValidatorNumber(&_Kyc.CallOpts)
}

// MinCandidateCap is a free data retrieval call binding the contract method 0xd55b7dff.
//
// Solidity: function minCandidateCap() constant returns(uint256)
func (_Kyc *KycCaller) MinCandidateCap(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "minCandidateCap")
	return *ret0, err
}

// MinCandidateCap is a free data retrieval call binding the contract method 0xd55b7dff.
//
// Solidity: function minCandidateCap() constant returns(uint256)
func (_Kyc *KycSession) MinCandidateCap() (*big.Int, error) {
	return _Kyc.Contract.MinCandidateCap(&_Kyc.CallOpts)
}

// MinCandidateCap is a free data retrieval call binding the contract method 0xd55b7dff.
//
// Solidity: function minCandidateCap() constant returns(uint256)
func (_Kyc *KycCallerSession) MinCandidateCap() (*big.Int, error) {
	return _Kyc.Contract.MinCandidateCap(&_Kyc.CallOpts)
}

// MinVoterCap is a free data retrieval call binding the contract method 0xf8ac9dd5.
//
// Solidity: function minVoterCap() constant returns(uint256)
func (_Kyc *KycCaller) MinVoterCap(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "minVoterCap")
	return *ret0, err
}

// MinVoterCap is a free data retrieval call binding the contract method 0xf8ac9dd5.
//
// Solidity: function minVoterCap() constant returns(uint256)
func (_Kyc *KycSession) MinVoterCap() (*big.Int, error) {
	return _Kyc.Contract.MinVoterCap(&_Kyc.CallOpts)
}

// MinVoterCap is a free data retrieval call binding the contract method 0xf8ac9dd5.
//
// Solidity: function minVoterCap() constant returns(uint256)
func (_Kyc *KycCallerSession) MinVoterCap() (*big.Int, error) {
	return _Kyc.Contract.MinVoterCap(&_Kyc.CallOpts)
}

// OwnerToCandidate is a free data retrieval call binding the contract method 0x2a3640b1.
//
// Solidity: function ownerToCandidate(address , uint256 ) constant returns(address)
func (_Kyc *KycCaller) OwnerToCandidate(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "ownerToCandidate", arg0, arg1)
	return *ret0, err
}

// OwnerToCandidate is a free data retrieval call binding the contract method 0x2a3640b1.
//
// Solidity: function ownerToCandidate(address , uint256 ) constant returns(address)
func (_Kyc *KycSession) OwnerToCandidate(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _Kyc.Contract.OwnerToCandidate(&_Kyc.CallOpts, arg0, arg1)
}

// OwnerToCandidate is a free data retrieval call binding the contract method 0x2a3640b1.
//
// Solidity: function ownerToCandidate(address , uint256 ) constant returns(address)
func (_Kyc *KycCallerSession) OwnerToCandidate(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _Kyc.Contract.OwnerToCandidate(&_Kyc.CallOpts, arg0, arg1)
}

// Owners is a free data retrieval call binding the contract method 0x025e7c27.
//
// Solidity: function owners(uint256 ) constant returns(address)
func (_Kyc *KycCaller) Owners(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "owners", arg0)
	return *ret0, err
}

// Owners is a free data retrieval call binding the contract method 0x025e7c27.
//
// Solidity: function owners(uint256 ) constant returns(address)
func (_Kyc *KycSession) Owners(arg0 *big.Int) (common.Address, error) {
	return _Kyc.Contract.Owners(&_Kyc.CallOpts, arg0)
}

// Owners is a free data retrieval call binding the contract method 0x025e7c27.
//
// Solidity: function owners(uint256 ) constant returns(address)
func (_Kyc *KycCallerSession) Owners(arg0 *big.Int) (common.Address, error) {
	return _Kyc.Contract.Owners(&_Kyc.CallOpts, arg0)
}

// VoterWithdrawDelay is a free data retrieval call binding the contract method 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() constant returns(uint256)
func (_Kyc *KycCaller) VoterWithdrawDelay(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Kyc.contract.Call(opts, out, "voterWithdrawDelay")
	return *ret0, err
}

// VoterWithdrawDelay is a free data retrieval call binding the contract method 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() constant returns(uint256)
func (_Kyc *KycSession) VoterWithdrawDelay() (*big.Int, error) {
	return _Kyc.Contract.VoterWithdrawDelay(&_Kyc.CallOpts)
}

// VoterWithdrawDelay is a free data retrieval call binding the contract method 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() constant returns(uint256)
func (_Kyc *KycCallerSession) VoterWithdrawDelay() (*big.Int, error) {
	return _Kyc.Contract.VoterWithdrawDelay(&_Kyc.CallOpts)
}

// VoteInvalidKYC is a paid mutator transaction binding the contract method 0xa0f56aa1.
//
// Solidity: function VoteInvalidKYC(address _invalidMasternode) returns()
func (_Kyc *KycTransactor) VoteInvalidKYC(opts *bind.TransactOpts, _invalidMasternode common.Address) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "VoteInvalidKYC", _invalidMasternode)
}

// VoteInvalidKYC is a paid mutator transaction binding the contract method 0xa0f56aa1.
//
// Solidity: function VoteInvalidKYC(address _invalidMasternode) returns()
func (_Kyc *KycSession) VoteInvalidKYC(_invalidMasternode common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.VoteInvalidKYC(&_Kyc.TransactOpts, _invalidMasternode)
}

// VoteInvalidKYC is a paid mutator transaction binding the contract method 0xa0f56aa1.
//
// Solidity: function VoteInvalidKYC(address _invalidMasternode) returns()
func (_Kyc *KycTransactorSession) VoteInvalidKYC(_invalidMasternode common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.VoteInvalidKYC(&_Kyc.TransactOpts, _invalidMasternode)
}

// Propose is a paid mutator transaction binding the contract method 0x01267951.
//
// Solidity: function propose(address _candidate) returns()
func (_Kyc *KycTransactor) Propose(opts *bind.TransactOpts, _candidate common.Address) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "propose", _candidate)
}

// Propose is a paid mutator transaction binding the contract method 0x01267951.
//
// Solidity: function propose(address _candidate) returns()
func (_Kyc *KycSession) Propose(_candidate common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.Propose(&_Kyc.TransactOpts, _candidate)
}

// Propose is a paid mutator transaction binding the contract method 0x01267951.
//
// Solidity: function propose(address _candidate) returns()
func (_Kyc *KycTransactorSession) Propose(_candidate common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.Propose(&_Kyc.TransactOpts, _candidate)
}

// Resign is a paid mutator transaction binding the contract method 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (_Kyc *KycTransactor) Resign(opts *bind.TransactOpts, _candidate common.Address) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "resign", _candidate)
}

// Resign is a paid mutator transaction binding the contract method 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (_Kyc *KycSession) Resign(_candidate common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.Resign(&_Kyc.TransactOpts, _candidate)
}

// Resign is a paid mutator transaction binding the contract method 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (_Kyc *KycTransactorSession) Resign(_candidate common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.Resign(&_Kyc.TransactOpts, _candidate)
}

// Unvote is a paid mutator transaction binding the contract method 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (_Kyc *KycTransactor) Unvote(opts *bind.TransactOpts, _candidate common.Address, _cap *big.Int) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "unvote", _candidate, _cap)
}

// Unvote is a paid mutator transaction binding the contract method 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (_Kyc *KycSession) Unvote(_candidate common.Address, _cap *big.Int) (*types.Transaction, error) {
	return _Kyc.Contract.Unvote(&_Kyc.TransactOpts, _candidate, _cap)
}

// Unvote is a paid mutator transaction binding the contract method 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (_Kyc *KycTransactorSession) Unvote(_candidate common.Address, _cap *big.Int) (*types.Transaction, error) {
	return _Kyc.Contract.Unvote(&_Kyc.TransactOpts, _candidate, _cap)
}

// UploadKYC is a paid mutator transaction binding the contract method 0xe8fd6927.
//
// Solidity: function uploadKYC(bytes32 _kycdata) returns()
func (_Kyc *KycTransactor) UploadKYC(opts *bind.TransactOpts, _kycdata [32]byte) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "uploadKYC", _kycdata)
}

// UploadKYC is a paid mutator transaction binding the contract method 0xe8fd6927.
//
// Solidity: function uploadKYC(bytes32 _kycdata) returns()
func (_Kyc *KycSession) UploadKYC(_kycdata [32]byte) (*types.Transaction, error) {
	return _Kyc.Contract.UploadKYC(&_Kyc.TransactOpts, _kycdata)
}

// UploadKYC is a paid mutator transaction binding the contract method 0xe8fd6927.
//
// Solidity: function uploadKYC(bytes32 _kycdata) returns()
func (_Kyc *KycTransactorSession) UploadKYC(_kycdata [32]byte) (*types.Transaction, error) {
	return _Kyc.Contract.UploadKYC(&_Kyc.TransactOpts, _kycdata)
}

// Vote is a paid mutator transaction binding the contract method 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) returns()
func (_Kyc *KycTransactor) Vote(opts *bind.TransactOpts, _candidate common.Address) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "vote", _candidate)
}

// Vote is a paid mutator transaction binding the contract method 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) returns()
func (_Kyc *KycSession) Vote(_candidate common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.Vote(&_Kyc.TransactOpts, _candidate)
}

// Vote is a paid mutator transaction binding the contract method 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) returns()
func (_Kyc *KycTransactorSession) Vote(_candidate common.Address) (*types.Transaction, error) {
	return _Kyc.Contract.Vote(&_Kyc.TransactOpts, _candidate)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (_Kyc *KycTransactor) Withdraw(opts *bind.TransactOpts, _blockNumber *big.Int, _index *big.Int) (*types.Transaction, error) {
	return _Kyc.contract.Transact(opts, "withdraw", _blockNumber, _index)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (_Kyc *KycSession) Withdraw(_blockNumber *big.Int, _index *big.Int) (*types.Transaction, error) {
	return _Kyc.Contract.Withdraw(&_Kyc.TransactOpts, _blockNumber, _index)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (_Kyc *KycTransactorSession) Withdraw(_blockNumber *big.Int, _index *big.Int) (*types.Transaction, error) {
	return _Kyc.Contract.Withdraw(&_Kyc.TransactOpts, _blockNumber, _index)
}

// KycProposeIterator is returned from FilterPropose and is used to iterate over the raw logs and unpacked data for Propose events raised by the Kyc contract.
type KycProposeIterator struct {
	Event *KycPropose // Event containing the contract specifics and raw log

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
func (it *KycProposeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(KycPropose)
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
		it.Event = new(KycPropose)
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
func (it *KycProposeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *KycProposeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// KycPropose represents a Propose event raised by the Kyc contract.
type KycPropose struct {
	Owner     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPropose is a free log retrieval operation binding the contract event 0x7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1.
//
// Solidity: event Propose(address _owner, address _candidate, uint256 _cap)
func (_Kyc *KycFilterer) FilterPropose(opts *bind.FilterOpts) (*KycProposeIterator, error) {

	logs, sub, err := _Kyc.contract.FilterLogs(opts, "Propose")
	if err != nil {
		return nil, err
	}
	return &KycProposeIterator{contract: _Kyc.contract, event: "Propose", logs: logs, sub: sub}, nil
}

// WatchPropose is a free log subscription operation binding the contract event 0x7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1.
//
// Solidity: event Propose(address _owner, address _candidate, uint256 _cap)
func (_Kyc *KycFilterer) WatchPropose(opts *bind.WatchOpts, sink chan<- *KycPropose) (event.Subscription, error) {

	logs, sub, err := _Kyc.contract.WatchLogs(opts, "Propose")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(KycPropose)
				if err := _Kyc.contract.UnpackLog(event, "Propose", log); err != nil {
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

// KycResignIterator is returned from FilterResign and is used to iterate over the raw logs and unpacked data for Resign events raised by the Kyc contract.
type KycResignIterator struct {
	Event *KycResign // Event containing the contract specifics and raw log

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
func (it *KycResignIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(KycResign)
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
		it.Event = new(KycResign)
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
func (it *KycResignIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *KycResignIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// KycResign represents a Resign event raised by the Kyc contract.
type KycResign struct {
	Owner     common.Address
	Candidate common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterResign is a free log retrieval operation binding the contract event 0x4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d3.
//
// Solidity: event Resign(address _owner, address _candidate)
func (_Kyc *KycFilterer) FilterResign(opts *bind.FilterOpts) (*KycResignIterator, error) {

	logs, sub, err := _Kyc.contract.FilterLogs(opts, "Resign")
	if err != nil {
		return nil, err
	}
	return &KycResignIterator{contract: _Kyc.contract, event: "Resign", logs: logs, sub: sub}, nil
}

// WatchResign is a free log subscription operation binding the contract event 0x4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d3.
//
// Solidity: event Resign(address _owner, address _candidate)
func (_Kyc *KycFilterer) WatchResign(opts *bind.WatchOpts, sink chan<- *KycResign) (event.Subscription, error) {

	logs, sub, err := _Kyc.contract.WatchLogs(opts, "Resign")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(KycResign)
				if err := _Kyc.contract.UnpackLog(event, "Resign", log); err != nil {
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

// KycUnvoteIterator is returned from FilterUnvote and is used to iterate over the raw logs and unpacked data for Unvote events raised by the Kyc contract.
type KycUnvoteIterator struct {
	Event *KycUnvote // Event containing the contract specifics and raw log

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
func (it *KycUnvoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(KycUnvote)
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
		it.Event = new(KycUnvote)
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
func (it *KycUnvoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *KycUnvoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// KycUnvote represents a Unvote event raised by the Kyc contract.
type KycUnvote struct {
	Voter     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterUnvote is a free log retrieval operation binding the contract event 0xaa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2.
//
// Solidity: event Unvote(address _voter, address _candidate, uint256 _cap)
func (_Kyc *KycFilterer) FilterUnvote(opts *bind.FilterOpts) (*KycUnvoteIterator, error) {

	logs, sub, err := _Kyc.contract.FilterLogs(opts, "Unvote")
	if err != nil {
		return nil, err
	}
	return &KycUnvoteIterator{contract: _Kyc.contract, event: "Unvote", logs: logs, sub: sub}, nil
}

// WatchUnvote is a free log subscription operation binding the contract event 0xaa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2.
//
// Solidity: event Unvote(address _voter, address _candidate, uint256 _cap)
func (_Kyc *KycFilterer) WatchUnvote(opts *bind.WatchOpts, sink chan<- *KycUnvote) (event.Subscription, error) {

	logs, sub, err := _Kyc.contract.WatchLogs(opts, "Unvote")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(KycUnvote)
				if err := _Kyc.contract.UnpackLog(event, "Unvote", log); err != nil {
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

// KycVoteIterator is returned from FilterVote and is used to iterate over the raw logs and unpacked data for Vote events raised by the Kyc contract.
type KycVoteIterator struct {
	Event *KycVote // Event containing the contract specifics and raw log

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
func (it *KycVoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(KycVote)
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
		it.Event = new(KycVote)
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
func (it *KycVoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *KycVoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// KycVote represents a Vote event raised by the Kyc contract.
type KycVote struct {
	Voter     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterVote is a free log retrieval operation binding the contract event 0x66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc.
//
// Solidity: event Vote(address _voter, address _candidate, uint256 _cap)
func (_Kyc *KycFilterer) FilterVote(opts *bind.FilterOpts) (*KycVoteIterator, error) {

	logs, sub, err := _Kyc.contract.FilterLogs(opts, "Vote")
	if err != nil {
		return nil, err
	}
	return &KycVoteIterator{contract: _Kyc.contract, event: "Vote", logs: logs, sub: sub}, nil
}

// WatchVote is a free log subscription operation binding the contract event 0x66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc.
//
// Solidity: event Vote(address _voter, address _candidate, uint256 _cap)
func (_Kyc *KycFilterer) WatchVote(opts *bind.WatchOpts, sink chan<- *KycVote) (event.Subscription, error) {

	logs, sub, err := _Kyc.contract.WatchLogs(opts, "Vote")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(KycVote)
				if err := _Kyc.contract.UnpackLog(event, "Vote", log); err != nil {
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

// KycWithdrawIterator is returned from FilterWithdraw and is used to iterate over the raw logs and unpacked data for Withdraw events raised by the Kyc contract.
type KycWithdrawIterator struct {
	Event *KycWithdraw // Event containing the contract specifics and raw log

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
func (it *KycWithdrawIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(KycWithdraw)
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
		it.Event = new(KycWithdraw)
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
func (it *KycWithdrawIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *KycWithdrawIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// KycWithdraw represents a Withdraw event raised by the Kyc contract.
type KycWithdraw struct {
	Owner       common.Address
	BlockNumber *big.Int
	Cap         *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterWithdraw is a free log retrieval operation binding the contract event 0xf279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b568.
//
// Solidity: event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap)
func (_Kyc *KycFilterer) FilterWithdraw(opts *bind.FilterOpts) (*KycWithdrawIterator, error) {

	logs, sub, err := _Kyc.contract.FilterLogs(opts, "Withdraw")
	if err != nil {
		return nil, err
	}
	return &KycWithdrawIterator{contract: _Kyc.contract, event: "Withdraw", logs: logs, sub: sub}, nil
}

// WatchWithdraw is a free log subscription operation binding the contract event 0xf279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b568.
//
// Solidity: event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap)
func (_Kyc *KycFilterer) WatchWithdraw(opts *bind.WatchOpts, sink chan<- *KycWithdraw) (event.Subscription, error) {

	logs, sub, err := _Kyc.contract.WatchLogs(opts, "Withdraw")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(KycWithdraw)
				if err := _Kyc.contract.UnpackLog(event, "Withdraw", log); err != nil {
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
