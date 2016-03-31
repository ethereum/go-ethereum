// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package releases

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ReleaseOracleABI is the input ABI used to generate the binding from.
const ReleaseOracleABI = `[{"constant":true,"inputs":[],"name":"AuthProposals","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"Promote","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"},{"name":"authorize","type":"bool"}],"name":"updateStatus","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"Demote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"user","type":"address"}],"name":"AuthVotes","outputs":[{"name":"promote","type":"address[]"},{"name":"demote","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"Signers","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"inputs":[],"type":"constructor"}]`

// ReleaseOracleBin is the compiled bytecode used for deploying new contracts.
const ReleaseOracleBin = `0x6060604052600160a060020a0333166000908152602081905260409020805460ff1916600190811790915580548082018083558281838015829011606257818360005260206000209182019101606291905b808211156094576000815584016051565b505050919090600052602060002090016000508054600160a060020a03191633179055506108d3806100986000396000f35b509056606060405236156100565760e060020a6000350463282fe4e581146100585780636195db9c146100c45780637444144b146100d557806380bbbd4a1461015b578063a29226f21461016c578063f04c475814610266575b005b604080516020818101835260008252600380548451818402810184019095528085526102d294928301828280156102c757602002820191906000526020600020908154600160a060020a03168152600191909101906020018083116102a8575b505050505090506102cf565b6100566004356103a18160016100df565b6100566004356024355b33600160a060020a0316600090815260208190526040812054819060ff161561088557600160a060020a038416815260026020526040812091505b81548110156103a457815433600160a060020a0316908390839081101561000257600091825260209091200154600160a060020a031614156103ef57610885565b6100566004356103a18160006100df565b61031c6004356040805160208181018352600080835283518083018552818152600160a060020a03861682526002835290849020805485518185028101850190965280865293949193909260018401929184918301828280156101f957602002820191906000526020600020905b8154600160a060020a03168152600191909101906020018083116101da575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561025657602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610237575b5050505050905091509150915091565b604080516020818101835260008252600180548451818402810184019095528085526102d294928301828280156102c757602002820191906000526020600020905b8154600160a060020a03168152600191909101906020018083116102a8575b505050505090505b90565b60405180806020018281038252838181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019250505060405180910390f35b6040518080602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f15090500194505050505060405180910390f35b50565b5060005b60018201548110156103f75733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a0316141561044157610885565b60010161011a565b8154600014801561040c575060018201546000145b1561046957600380546001810180835582818380158290116104495781836000526020600020918201910161044991906104ef565b6001016103a8565b5050506000928352506020909120018054600160a060020a031916851790555b821561050757815460018101808455839190828183801582901161053c5760008381526020902061053c9181019083016104ef565b5050506000928352506020909120018054600160a060020a031916851790555b600160a060020a03841660009081526002602090815260408220805483825581845291832090929161077091908101905b8082111561050357600081556001016104ef565b5090565b8160010160005080548060010182818154818355818115116105f8578183600052602060002091820191016105f891906104ef565b5050506000928352506020909120018054600160a060020a0319163317905560015482546002909104901161057057610885565b8280156105965750600160a060020a03841660009081526020819052604090205460ff16155b1561062f57600160a060020a0384166000908152602081905260409020805460ff191660019081179091558054808201808355828183801582901161049e57600083905261049e906000805160206108b38339815191529081019083016104ef565b5050506000928352506020909120018054600160a060020a0319163317905560018054908301546002909104901161057057610885565b821580156106555750600160a060020a03841660009081526020819052604090205460ff165b156104be5750600160a060020a0383166000908152602081905260408120805460ff191690555b6001548110156104be5783600160a060020a03166001600050828154811015610002576000919091526000805160206108b38339815191520154600160a060020a0316141561075f57600180546000198101908110156100025760009182526000805160206108b383398151915201909054906101000a9004600160a060020a03166001600050828154811015610002576000805160206108b3833981519152018054600160a060020a031916909217909155805460001981018083559091908280158290116107675781836000526020600020918201910161076791906104ef565b60010161067c565b505050506104be565b5060018201805460008083559182526020909120610790918101906104ef565b506000925050505b6003548110156108855783600160a060020a03166003600050828154811015610002576000919091526000805160206108938339815191520154600160a060020a0316141561088b576003805460001981019081101561000257600091825260008051602061089383398151915201909054906101000a9004600160a060020a0316600360005082815481101561000257600080516020610893833981519152018054600160a060020a0319169092179091558054600019810180835590919082801582901161088057610880906000805160206108938339815191529081019083016104ef565b505050505b50505050565b60010161079856c2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85bb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6`

// DeployReleaseOracle deploys a new Ethereum contract, binding an instance of ReleaseOracle to it.
func DeployReleaseOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ReleaseOracle, error) {
	parsed, err := abi.JSON(strings.NewReader(ReleaseOracleABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ReleaseOracleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ReleaseOracle{ReleaseOracleCaller: ReleaseOracleCaller{contract: contract}, ReleaseOracleTransactor: ReleaseOracleTransactor{contract: contract}}, nil
}

// ReleaseOracle is an auto generated Go binding around an Ethereum contract.
type ReleaseOracle struct {
	ReleaseOracleCaller     // Read-only binding to the contract
	ReleaseOracleTransactor // Write-only binding to the contract
}

// ReleaseOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type ReleaseOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReleaseOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ReleaseOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReleaseOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ReleaseOracleSession struct {
	Contract     *ReleaseOracle    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ReleaseOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ReleaseOracleCallerSession struct {
	Contract *ReleaseOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ReleaseOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ReleaseOracleTransactorSession struct {
	Contract     *ReleaseOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ReleaseOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type ReleaseOracleRaw struct {
	Contract *ReleaseOracle // Generic contract binding to access the raw methods on
}

// ReleaseOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ReleaseOracleCallerRaw struct {
	Contract *ReleaseOracleCaller // Generic read-only contract binding to access the raw methods on
}

// ReleaseOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ReleaseOracleTransactorRaw struct {
	Contract *ReleaseOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewReleaseOracle creates a new instance of ReleaseOracle, bound to a specific deployed contract.
func NewReleaseOracle(address common.Address, backend bind.ContractBackend) (*ReleaseOracle, error) {
	contract, err := bindReleaseOracle(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))
	if err != nil {
		return nil, err
	}
	return &ReleaseOracle{ReleaseOracleCaller: ReleaseOracleCaller{contract: contract}, ReleaseOracleTransactor: ReleaseOracleTransactor{contract: contract}}, nil
}

// NewReleaseOracleCaller creates a new read-only instance of ReleaseOracle, bound to a specific deployed contract.
func NewReleaseOracleCaller(address common.Address, caller bind.ContractCaller) (*ReleaseOracleCaller, error) {
	contract, err := bindReleaseOracle(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &ReleaseOracleCaller{contract: contract}, nil
}

// NewReleaseOracleTransactor creates a new write-only instance of ReleaseOracle, bound to a specific deployed contract.
func NewReleaseOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*ReleaseOracleTransactor, error) {
	contract, err := bindReleaseOracle(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ReleaseOracleTransactor{contract: contract}, nil
}

// bindReleaseOracle binds a generic wrapper to an already deployed contract.
func bindReleaseOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ReleaseOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ReleaseOracle *ReleaseOracleRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ReleaseOracle.Contract.ReleaseOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ReleaseOracle *ReleaseOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.ReleaseOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ReleaseOracle *ReleaseOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.ReleaseOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ReleaseOracle *ReleaseOracleCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ReleaseOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ReleaseOracle *ReleaseOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ReleaseOracle *ReleaseOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.contract.Transact(opts, method, params...)
}

// AuthProposals is a free data retrieval call binding the contract method 0x282fe4e5.
//
// Solidity: function AuthProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCaller) AuthProposals(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _ReleaseOracle.contract.Call(opts, out, "AuthProposals")
	return *ret0, err
}

// AuthProposals is a free data retrieval call binding the contract method 0x282fe4e5.
//
// Solidity: function AuthProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleSession) AuthProposals() ([]common.Address, error) {
	return _ReleaseOracle.Contract.AuthProposals(&_ReleaseOracle.CallOpts)
}

// AuthProposals is a free data retrieval call binding the contract method 0x282fe4e5.
//
// Solidity: function AuthProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) AuthProposals() ([]common.Address, error) {
	return _ReleaseOracle.Contract.AuthProposals(&_ReleaseOracle.CallOpts)
}

// AuthVotes is a free data retrieval call binding the contract method 0xa29226f2.
//
// Solidity: function AuthVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleCaller) AuthVotes(opts *bind.CallOpts, user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	ret := new(struct {
		Promote []common.Address
		Demote  []common.Address
	})
	out := ret
	err := _ReleaseOracle.contract.Call(opts, out, "AuthVotes", user)
	return *ret, err
}

// AuthVotes is a free data retrieval call binding the contract method 0xa29226f2.
//
// Solidity: function AuthVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleSession) AuthVotes(user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	return _ReleaseOracle.Contract.AuthVotes(&_ReleaseOracle.CallOpts, user)
}

// AuthVotes is a free data retrieval call binding the contract method 0xa29226f2.
//
// Solidity: function AuthVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) AuthVotes(user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	return _ReleaseOracle.Contract.AuthVotes(&_ReleaseOracle.CallOpts, user)
}

// Signers is a free data retrieval call binding the contract method 0xf04c4758.
//
// Solidity: function Signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCaller) Signers(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _ReleaseOracle.contract.Call(opts, out, "Signers")
	return *ret0, err
}

// Signers is a free data retrieval call binding the contract method 0xf04c4758.
//
// Solidity: function Signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleSession) Signers() ([]common.Address, error) {
	return _ReleaseOracle.Contract.Signers(&_ReleaseOracle.CallOpts)
}

// Signers is a free data retrieval call binding the contract method 0xf04c4758.
//
// Solidity: function Signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) Signers() ([]common.Address, error) {
	return _ReleaseOracle.Contract.Signers(&_ReleaseOracle.CallOpts)
}

// Demote is a paid mutator transaction binding the contract method 0x80bbbd4a.
//
// Solidity: function Demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Demote(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "Demote", user)
}

// Demote is a paid mutator transaction binding the contract method 0x80bbbd4a.
//
// Solidity: function Demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleSession) Demote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Demote(&_ReleaseOracle.TransactOpts, user)
}

// Demote is a paid mutator transaction binding the contract method 0x80bbbd4a.
//
// Solidity: function Demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Demote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Demote(&_ReleaseOracle.TransactOpts, user)
}

// Promote is a paid mutator transaction binding the contract method 0x6195db9c.
//
// Solidity: function Promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Promote(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "Promote", user)
}

// Promote is a paid mutator transaction binding the contract method 0x6195db9c.
//
// Solidity: function Promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleSession) Promote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Promote(&_ReleaseOracle.TransactOpts, user)
}

// Promote is a paid mutator transaction binding the contract method 0x6195db9c.
//
// Solidity: function Promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Promote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Promote(&_ReleaseOracle.TransactOpts, user)
}

// UpdateStatus is a paid mutator transaction binding the contract method 0x7444144b.
//
// Solidity: function updateStatus(user address, authorize bool) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) UpdateStatus(opts *bind.TransactOpts, user common.Address, authorize bool) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "updateStatus", user, authorize)
}

// UpdateStatus is a paid mutator transaction binding the contract method 0x7444144b.
//
// Solidity: function updateStatus(user address, authorize bool) returns()
func (_ReleaseOracle *ReleaseOracleSession) UpdateStatus(user common.Address, authorize bool) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.UpdateStatus(&_ReleaseOracle.TransactOpts, user, authorize)
}

// UpdateStatus is a paid mutator transaction binding the contract method 0x7444144b.
//
// Solidity: function updateStatus(user address, authorize bool) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) UpdateStatus(user common.Address, authorize bool) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.UpdateStatus(&_ReleaseOracle.TransactOpts, user, authorize)
}
