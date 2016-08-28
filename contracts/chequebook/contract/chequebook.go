// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ChequebookABI is the input ABI used to generate the binding from.
const ChequebookABI = `[{"constant":false,"inputs":[],"name":"kill","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"sent","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"sig_v","type":"uint8"},{"name":"sig_r","type":"bytes32"},{"name":"sig_s","type":"bytes32"}],"name":"cash","outputs":[],"type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"deadbeat","type":"address"}],"name":"Overdraft","type":"event"}]`

// ChequebookBin is the compiled bytecode used for deploying new contracts.
const ChequebookBin = `0x606060405260008054600160a060020a03191633179055610223806100246000396000f3606060405260e060020a600035046341c0e1b581146100315780637bf786f814610059578063fbf788d614610071575b005b61002f60005433600160a060020a03908116911614156100b957600054600160a060020a0316ff5b6100a760043560016020526000908152604090205481565b61002f600435602435604435606435608435600160a060020a03851660009081526001602052604081205485116100bb576101cf565b60408051918252519081900360200190f35b565b50604080516c0100000000000000000000000030600160a060020a0390811682028352881602601482015260288101869052815190819003604801812080825260ff861660208381019190915282840186905260608301859052925190926001926080818101939182900301816000866161da5a03f11561000257505060405151600054600160a060020a03908116911614610156576101cf565b600160a060020a038681166000908152600160205260409020543090911631908603106101d75760406000818120549151600160a060020a0389169288039082818181858883f19350505050156101cf57846001600050600088600160a060020a03168152602001908152602001600020600050819055505b505050505050565b60005460408051600160a060020a03929092168252517f2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f9789181900360200190a185600160a060020a0316ff`

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
	return address, tx, &Chequebook{ChequebookCaller: ChequebookCaller{contract: contract}, ChequebookTransactor: ChequebookTransactor{contract: contract}}, nil
}

// Chequebook is an auto generated Go binding around an Ethereum contract.
type Chequebook struct {
	ChequebookCaller     // Read-only binding to the contract
	ChequebookTransactor // Write-only binding to the contract
}

// ChequebookCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChequebookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChequebookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChequebookTransactor struct {
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
	contract, err := bindChequebook(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Chequebook{ChequebookCaller: ChequebookCaller{contract: contract}, ChequebookTransactor: ChequebookTransactor{contract: contract}}, nil
}

// NewChequebookCaller creates a new read-only instance of Chequebook, bound to a specific deployed contract.
func NewChequebookCaller(address common.Address, caller bind.ContractCaller) (*ChequebookCaller, error) {
	contract, err := bindChequebook(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &ChequebookCaller{contract: contract}, nil
}

// NewChequebookTransactor creates a new write-only instance of Chequebook, bound to a specific deployed contract.
func NewChequebookTransactor(address common.Address, transactor bind.ContractTransactor) (*ChequebookTransactor, error) {
	contract, err := bindChequebook(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ChequebookTransactor{contract: contract}, nil
}

// bindChequebook binds a generic wrapper to an already deployed contract.
func bindChequebook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ChequebookABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
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
func (_Chequebook *ChequebookTransactor) Cash(opts *bind.TransactOpts, beneficiary common.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _Chequebook.contract.Transact(opts, "cash", beneficiary, amount, sig_v, sig_r, sig_s)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequebook *ChequebookSession) Cash(beneficiary common.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _Chequebook.Contract.Cash(&_Chequebook.TransactOpts, beneficiary, amount, sig_v, sig_r, sig_s)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(beneficiary address, amount uint256, sig_v uint8, sig_r bytes32, sig_s bytes32) returns()
func (_Chequebook *ChequebookTransactorSession) Cash(beneficiary common.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _Chequebook.Contract.Cash(&_Chequebook.TransactOpts, beneficiary, amount, sig_v, sig_r, sig_s)
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

// MortalABI is the input ABI used to generate the binding from.
const MortalABI = `[{"constant":false,"inputs":[],"name":"kill","outputs":[],"type":"function"}]`

// MortalBin is the compiled bytecode used for deploying new contracts.
const MortalBin = `0x606060405260008054600160a060020a03191633179055605c8060226000396000f3606060405260e060020a600035046341c0e1b58114601a575b005b60186000543373ffffffffffffffffffffffffffffffffffffffff90811691161415605a5760005473ffffffffffffffffffffffffffffffffffffffff16ff5b56`

// DeployMortal deploys a new Ethereum contract, binding an instance of Mortal to it.
func DeployMortal(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Mortal, error) {
	parsed, err := abi.JSON(strings.NewReader(MortalABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MortalBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Mortal{MortalCaller: MortalCaller{contract: contract}, MortalTransactor: MortalTransactor{contract: contract}}, nil
}

// Mortal is an auto generated Go binding around an Ethereum contract.
type Mortal struct {
	MortalCaller     // Read-only binding to the contract
	MortalTransactor // Write-only binding to the contract
}

// MortalCaller is an auto generated read-only Go binding around an Ethereum contract.
type MortalCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MortalTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MortalTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MortalSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MortalSession struct {
	Contract     *Mortal           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MortalCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MortalCallerSession struct {
	Contract *MortalCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// MortalTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MortalTransactorSession struct {
	Contract     *MortalTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MortalRaw is an auto generated low-level Go binding around an Ethereum contract.
type MortalRaw struct {
	Contract *Mortal // Generic contract binding to access the raw methods on
}

// MortalCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MortalCallerRaw struct {
	Contract *MortalCaller // Generic read-only contract binding to access the raw methods on
}

// MortalTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MortalTransactorRaw struct {
	Contract *MortalTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMortal creates a new instance of Mortal, bound to a specific deployed contract.
func NewMortal(address common.Address, backend bind.ContractBackend) (*Mortal, error) {
	contract, err := bindMortal(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Mortal{MortalCaller: MortalCaller{contract: contract}, MortalTransactor: MortalTransactor{contract: contract}}, nil
}

// NewMortalCaller creates a new read-only instance of Mortal, bound to a specific deployed contract.
func NewMortalCaller(address common.Address, caller bind.ContractCaller) (*MortalCaller, error) {
	contract, err := bindMortal(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &MortalCaller{contract: contract}, nil
}

// NewMortalTransactor creates a new write-only instance of Mortal, bound to a specific deployed contract.
func NewMortalTransactor(address common.Address, transactor bind.ContractTransactor) (*MortalTransactor, error) {
	contract, err := bindMortal(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &MortalTransactor{contract: contract}, nil
}

// bindMortal binds a generic wrapper to an already deployed contract.
func bindMortal(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MortalABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Mortal *MortalRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Mortal.Contract.MortalCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Mortal *MortalRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Mortal.Contract.MortalTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Mortal *MortalRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Mortal.Contract.MortalTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Mortal *MortalCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Mortal.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Mortal *MortalTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Mortal.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Mortal *MortalTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Mortal.Contract.contract.Transact(opts, method, params...)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Mortal *MortalTransactor) Kill(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Mortal.contract.Transact(opts, "kill")
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Mortal *MortalSession) Kill() (*types.Transaction, error) {
	return _Mortal.Contract.Kill(&_Mortal.TransactOpts)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Mortal *MortalTransactorSession) Kill() (*types.Transaction, error) {
	return _Mortal.Contract.Kill(&_Mortal.TransactOpts)
}

// OwnedABI is the input ABI used to generate the binding from.
const OwnedABI = `[{"inputs":[],"type":"constructor"}]`

// OwnedBin is the compiled bytecode used for deploying new contracts.
const OwnedBin = `0x606060405260008054600160a060020a0319163317905560068060226000396000f3606060405200`

// DeployOwned deploys a new Ethereum contract, binding an instance of Owned to it.
func DeployOwned(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Owned, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnedABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(OwnedBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Owned{OwnedCaller: OwnedCaller{contract: contract}, OwnedTransactor: OwnedTransactor{contract: contract}}, nil
}

// Owned is an auto generated Go binding around an Ethereum contract.
type Owned struct {
	OwnedCaller     // Read-only binding to the contract
	OwnedTransactor // Write-only binding to the contract
}

// OwnedCaller is an auto generated read-only Go binding around an Ethereum contract.
type OwnedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OwnedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OwnedSession struct {
	Contract     *Owned            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OwnedCallerSession struct {
	Contract *OwnedCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OwnedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OwnedTransactorSession struct {
	Contract     *OwnedTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnedRaw is an auto generated low-level Go binding around an Ethereum contract.
type OwnedRaw struct {
	Contract *Owned // Generic contract binding to access the raw methods on
}

// OwnedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OwnedCallerRaw struct {
	Contract *OwnedCaller // Generic read-only contract binding to access the raw methods on
}

// OwnedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OwnedTransactorRaw struct {
	Contract *OwnedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOwned creates a new instance of Owned, bound to a specific deployed contract.
func NewOwned(address common.Address, backend bind.ContractBackend) (*Owned, error) {
	contract, err := bindOwned(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Owned{OwnedCaller: OwnedCaller{contract: contract}, OwnedTransactor: OwnedTransactor{contract: contract}}, nil
}

// NewOwnedCaller creates a new read-only instance of Owned, bound to a specific deployed contract.
func NewOwnedCaller(address common.Address, caller bind.ContractCaller) (*OwnedCaller, error) {
	contract, err := bindOwned(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &OwnedCaller{contract: contract}, nil
}

// NewOwnedTransactor creates a new write-only instance of Owned, bound to a specific deployed contract.
func NewOwnedTransactor(address common.Address, transactor bind.ContractTransactor) (*OwnedTransactor, error) {
	contract, err := bindOwned(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &OwnedTransactor{contract: contract}, nil
}

// bindOwned binds a generic wrapper to an already deployed contract.
func bindOwned(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnedABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Owned *OwnedRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Owned.Contract.OwnedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Owned *OwnedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Owned.Contract.OwnedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Owned *OwnedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Owned.Contract.OwnedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Owned *OwnedCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Owned.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Owned *OwnedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Owned.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Owned *OwnedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Owned.Contract.contract.Transact(opts, method, params...)
}
