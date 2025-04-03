// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package dasigners

import (
	"errors"
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
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// DASignersMetaData contains all meta data concerning the DASigners contract.
var DASignersMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"X\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Y\",\"type\":\"uint256\"}],\"indexed\":false,\"internalType\":\"structBN254.G1Point\",\"name\":\"pkG1\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256[2]\",\"name\":\"X\",\"type\":\"uint256[2]\"},{\"internalType\":\"uint256[2]\",\"name\":\"Y\",\"type\":\"uint256[2]\"}],\"indexed\":false,\"internalType\":\"structBN254.G2Point\",\"name\":\"pkG2\",\"type\":\"tuple\"}],\"name\":\"NewSigner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"socket\",\"type\":\"string\"}],\"name\":\"SocketUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"epochNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_quorumId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_quorumBitmap\",\"type\":\"bytes\"}],\"name\":\"getAggPkG1\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"X\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Y\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G1Point\",\"name\":\"aggPkG1\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"total\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"hit\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_quorumId\",\"type\":\"uint256\"}],\"name\":\"getQuorum\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_quorumId\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_rowIndex\",\"type\":\"uint32\"}],\"name\":\"getQuorumRow\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_account\",\"type\":\"address[]\"}],\"name\":\"getSigner\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"socket\",\"type\":\"string\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"X\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Y\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G1Point\",\"name\":\"pkG1\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256[2]\",\"name\":\"X\",\"type\":\"uint256[2]\"},{\"internalType\":\"uint256[2]\",\"name\":\"Y\",\"type\":\"uint256[2]\"}],\"internalType\":\"structBN254.G2Point\",\"name\":\"pkG2\",\"type\":\"tuple\"}],\"internalType\":\"structIDASigners.SignerDetail[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"name\":\"isSigner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"makeEpoch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"params\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokensPerVote\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxVotesPerSigner\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxQuorums\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"epochBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"encodedSlices\",\"type\":\"uint256\"}],\"internalType\":\"structIDASigners.Params\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_epoch\",\"type\":\"uint256\"}],\"name\":\"quorumCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"X\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Y\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G1Point\",\"name\":\"_signature\",\"type\":\"tuple\"}],\"name\":\"registerNextEpoch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"socket\",\"type\":\"string\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"X\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Y\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G1Point\",\"name\":\"pkG1\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256[2]\",\"name\":\"X\",\"type\":\"uint256[2]\"},{\"internalType\":\"uint256[2]\",\"name\":\"Y\",\"type\":\"uint256[2]\"}],\"internalType\":\"structBN254.G2Point\",\"name\":\"pkG2\",\"type\":\"tuple\"}],\"internalType\":\"structIDASigners.SignerDetail\",\"name\":\"_signer\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"X\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Y\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G1Point\",\"name\":\"_signature\",\"type\":\"tuple\"}],\"name\":\"registerSigner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_epoch\",\"type\":\"uint256\"}],\"name\":\"registeredEpoch\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_socket\",\"type\":\"string\"}],\"name\":\"updateSocket\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// DASignersABI is the input ABI used to generate the binding from.
// Deprecated: Use DASignersMetaData.ABI instead.
var DASignersABI = DASignersMetaData.ABI

// DASigners is an auto generated Go binding around an Ethereum contract.
type DASigners struct {
	DASignersCaller     // Read-only binding to the contract
	DASignersTransactor // Write-only binding to the contract
	DASignersFilterer   // Log filterer for contract events
}

// DASignersCaller is an auto generated read-only Go binding around an Ethereum contract.
type DASignersCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DASignersTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DASignersTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DASignersFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DASignersFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DASignersSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DASignersSession struct {
	Contract     *DASigners        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DASignersCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DASignersCallerSession struct {
	Contract *DASignersCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// DASignersTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DASignersTransactorSession struct {
	Contract     *DASignersTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// DASignersRaw is an auto generated low-level Go binding around an Ethereum contract.
type DASignersRaw struct {
	Contract *DASigners // Generic contract binding to access the raw methods on
}

// DASignersCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DASignersCallerRaw struct {
	Contract *DASignersCaller // Generic read-only contract binding to access the raw methods on
}

// DASignersTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DASignersTransactorRaw struct {
	Contract *DASignersTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDASigners creates a new instance of DASigners, bound to a specific deployed contract.
func NewDASigners(address common.Address, backend bind.ContractBackend) (*DASigners, error) {
	contract, err := bindDASigners(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DASigners{DASignersCaller: DASignersCaller{contract: contract}, DASignersTransactor: DASignersTransactor{contract: contract}, DASignersFilterer: DASignersFilterer{contract: contract}}, nil
}

// NewDASignersCaller creates a new read-only instance of DASigners, bound to a specific deployed contract.
func NewDASignersCaller(address common.Address, caller bind.ContractCaller) (*DASignersCaller, error) {
	contract, err := bindDASigners(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DASignersCaller{contract: contract}, nil
}

// NewDASignersTransactor creates a new write-only instance of DASigners, bound to a specific deployed contract.
func NewDASignersTransactor(address common.Address, transactor bind.ContractTransactor) (*DASignersTransactor, error) {
	contract, err := bindDASigners(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DASignersTransactor{contract: contract}, nil
}

// NewDASignersFilterer creates a new log filterer instance of DASigners, bound to a specific deployed contract.
func NewDASignersFilterer(address common.Address, filterer bind.ContractFilterer) (*DASignersFilterer, error) {
	contract, err := bindDASigners(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DASignersFilterer{contract: contract}, nil
}

// bindDASigners binds a generic wrapper to an already deployed contract.
func bindDASigners(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DASignersABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DASigners *DASignersRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DASigners.Contract.DASignersCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DASigners *DASignersRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DASigners.Contract.DASignersTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DASigners *DASignersRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DASigners.Contract.DASignersTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DASigners *DASignersCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DASigners.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DASigners *DASignersTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DASigners.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DASigners *DASignersTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DASigners.Contract.contract.Transact(opts, method, params...)
}

// EpochNumber is a free data retrieval call binding the contract method 0xf4145a83.
//
// Solidity: function epochNumber() view returns(uint256)
func (_DASigners *DASignersCaller) EpochNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "epochNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EpochNumber is a free data retrieval call binding the contract method 0xf4145a83.
//
// Solidity: function epochNumber() view returns(uint256)
func (_DASigners *DASignersSession) EpochNumber() (*big.Int, error) {
	return _DASigners.Contract.EpochNumber(&_DASigners.CallOpts)
}

// EpochNumber is a free data retrieval call binding the contract method 0xf4145a83.
//
// Solidity: function epochNumber() view returns(uint256)
func (_DASigners *DASignersCallerSession) EpochNumber() (*big.Int, error) {
	return _DASigners.Contract.EpochNumber(&_DASigners.CallOpts)
}

// GetAggPkG1 is a free data retrieval call binding the contract method 0x50b73739.
//
// Solidity: function getAggPkG1(uint256 _epoch, uint256 _quorumId, bytes _quorumBitmap) view returns((uint256,uint256) aggPkG1, uint256 total, uint256 hit)
func (_DASigners *DASignersCaller) GetAggPkG1(opts *bind.CallOpts, _epoch *big.Int, _quorumId *big.Int, _quorumBitmap []byte) (struct {
	AggPkG1 BN254G1Point
	Total   *big.Int
	Hit     *big.Int
}, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "getAggPkG1", _epoch, _quorumId, _quorumBitmap)

	outstruct := new(struct {
		AggPkG1 BN254G1Point
		Total   *big.Int
		Hit     *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AggPkG1 = *abi.ConvertType(out[0], new(BN254G1Point)).(*BN254G1Point)
	outstruct.Total = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Hit = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetAggPkG1 is a free data retrieval call binding the contract method 0x50b73739.
//
// Solidity: function getAggPkG1(uint256 _epoch, uint256 _quorumId, bytes _quorumBitmap) view returns((uint256,uint256) aggPkG1, uint256 total, uint256 hit)
func (_DASigners *DASignersSession) GetAggPkG1(_epoch *big.Int, _quorumId *big.Int, _quorumBitmap []byte) (struct {
	AggPkG1 BN254G1Point
	Total   *big.Int
	Hit     *big.Int
}, error) {
	return _DASigners.Contract.GetAggPkG1(&_DASigners.CallOpts, _epoch, _quorumId, _quorumBitmap)
}

// GetAggPkG1 is a free data retrieval call binding the contract method 0x50b73739.
//
// Solidity: function getAggPkG1(uint256 _epoch, uint256 _quorumId, bytes _quorumBitmap) view returns((uint256,uint256) aggPkG1, uint256 total, uint256 hit)
func (_DASigners *DASignersCallerSession) GetAggPkG1(_epoch *big.Int, _quorumId *big.Int, _quorumBitmap []byte) (struct {
	AggPkG1 BN254G1Point
	Total   *big.Int
	Hit     *big.Int
}, error) {
	return _DASigners.Contract.GetAggPkG1(&_DASigners.CallOpts, _epoch, _quorumId, _quorumBitmap)
}

// GetQuorum is a free data retrieval call binding the contract method 0x6ab6f654.
//
// Solidity: function getQuorum(uint256 _epoch, uint256 _quorumId) view returns(address[])
func (_DASigners *DASignersCaller) GetQuorum(opts *bind.CallOpts, _epoch *big.Int, _quorumId *big.Int) ([]common.Address, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "getQuorum", _epoch, _quorumId)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetQuorum is a free data retrieval call binding the contract method 0x6ab6f654.
//
// Solidity: function getQuorum(uint256 _epoch, uint256 _quorumId) view returns(address[])
func (_DASigners *DASignersSession) GetQuorum(_epoch *big.Int, _quorumId *big.Int) ([]common.Address, error) {
	return _DASigners.Contract.GetQuorum(&_DASigners.CallOpts, _epoch, _quorumId)
}

// GetQuorum is a free data retrieval call binding the contract method 0x6ab6f654.
//
// Solidity: function getQuorum(uint256 _epoch, uint256 _quorumId) view returns(address[])
func (_DASigners *DASignersCallerSession) GetQuorum(_epoch *big.Int, _quorumId *big.Int) ([]common.Address, error) {
	return _DASigners.Contract.GetQuorum(&_DASigners.CallOpts, _epoch, _quorumId)
}

// GetQuorumRow is a free data retrieval call binding the contract method 0xfa6fcba6.
//
// Solidity: function getQuorumRow(uint256 _epoch, uint256 _quorumId, uint32 _rowIndex) view returns(address)
func (_DASigners *DASignersCaller) GetQuorumRow(opts *bind.CallOpts, _epoch *big.Int, _quorumId *big.Int, _rowIndex uint32) (common.Address, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "getQuorumRow", _epoch, _quorumId, _rowIndex)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetQuorumRow is a free data retrieval call binding the contract method 0xfa6fcba6.
//
// Solidity: function getQuorumRow(uint256 _epoch, uint256 _quorumId, uint32 _rowIndex) view returns(address)
func (_DASigners *DASignersSession) GetQuorumRow(_epoch *big.Int, _quorumId *big.Int, _rowIndex uint32) (common.Address, error) {
	return _DASigners.Contract.GetQuorumRow(&_DASigners.CallOpts, _epoch, _quorumId, _rowIndex)
}

// GetQuorumRow is a free data retrieval call binding the contract method 0xfa6fcba6.
//
// Solidity: function getQuorumRow(uint256 _epoch, uint256 _quorumId, uint32 _rowIndex) view returns(address)
func (_DASigners *DASignersCallerSession) GetQuorumRow(_epoch *big.Int, _quorumId *big.Int, _rowIndex uint32) (common.Address, error) {
	return _DASigners.Contract.GetQuorumRow(&_DASigners.CallOpts, _epoch, _quorumId, _rowIndex)
}

// GetSigner is a free data retrieval call binding the contract method 0xd1f5e5f8.
//
// Solidity: function getSigner(address[] _account) view returns((address,string,(uint256,uint256),(uint256[2],uint256[2]))[])
func (_DASigners *DASignersCaller) GetSigner(opts *bind.CallOpts, _account []common.Address) ([]IDASignersSignerDetail, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "getSigner", _account)

	if err != nil {
		return *new([]IDASignersSignerDetail), err
	}

	out0 := *abi.ConvertType(out[0], new([]IDASignersSignerDetail)).(*[]IDASignersSignerDetail)

	return out0, err

}

// GetSigner is a free data retrieval call binding the contract method 0xd1f5e5f8.
//
// Solidity: function getSigner(address[] _account) view returns((address,string,(uint256,uint256),(uint256[2],uint256[2]))[])
func (_DASigners *DASignersSession) GetSigner(_account []common.Address) ([]IDASignersSignerDetail, error) {
	return _DASigners.Contract.GetSigner(&_DASigners.CallOpts, _account)
}

// GetSigner is a free data retrieval call binding the contract method 0xd1f5e5f8.
//
// Solidity: function getSigner(address[] _account) view returns((address,string,(uint256,uint256),(uint256[2],uint256[2]))[])
func (_DASigners *DASignersCallerSession) GetSigner(_account []common.Address) ([]IDASignersSignerDetail, error) {
	return _DASigners.Contract.GetSigner(&_DASigners.CallOpts, _account)
}

// IsSigner is a free data retrieval call binding the contract method 0x7df73e27.
//
// Solidity: function isSigner(address _account) view returns(bool)
func (_DASigners *DASignersCaller) IsSigner(opts *bind.CallOpts, _account common.Address) (bool, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "isSigner", _account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSigner is a free data retrieval call binding the contract method 0x7df73e27.
//
// Solidity: function isSigner(address _account) view returns(bool)
func (_DASigners *DASignersSession) IsSigner(_account common.Address) (bool, error) {
	return _DASigners.Contract.IsSigner(&_DASigners.CallOpts, _account)
}

// IsSigner is a free data retrieval call binding the contract method 0x7df73e27.
//
// Solidity: function isSigner(address _account) view returns(bool)
func (_DASigners *DASignersCallerSession) IsSigner(_account common.Address) (bool, error) {
	return _DASigners.Contract.IsSigner(&_DASigners.CallOpts, _account)
}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns((uint256,uint256,uint256,uint256,uint256))
func (_DASigners *DASignersCaller) Params(opts *bind.CallOpts) (IDASignersParams, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "params")

	if err != nil {
		return *new(IDASignersParams), err
	}

	out0 := *abi.ConvertType(out[0], new(IDASignersParams)).(*IDASignersParams)

	return out0, err

}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns((uint256,uint256,uint256,uint256,uint256))
func (_DASigners *DASignersSession) Params() (IDASignersParams, error) {
	return _DASigners.Contract.Params(&_DASigners.CallOpts)
}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns((uint256,uint256,uint256,uint256,uint256))
func (_DASigners *DASignersCallerSession) Params() (IDASignersParams, error) {
	return _DASigners.Contract.Params(&_DASigners.CallOpts)
}

// QuorumCount is a free data retrieval call binding the contract method 0x5ecba503.
//
// Solidity: function quorumCount(uint256 _epoch) view returns(uint256)
func (_DASigners *DASignersCaller) QuorumCount(opts *bind.CallOpts, _epoch *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "quorumCount", _epoch)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuorumCount is a free data retrieval call binding the contract method 0x5ecba503.
//
// Solidity: function quorumCount(uint256 _epoch) view returns(uint256)
func (_DASigners *DASignersSession) QuorumCount(_epoch *big.Int) (*big.Int, error) {
	return _DASigners.Contract.QuorumCount(&_DASigners.CallOpts, _epoch)
}

// QuorumCount is a free data retrieval call binding the contract method 0x5ecba503.
//
// Solidity: function quorumCount(uint256 _epoch) view returns(uint256)
func (_DASigners *DASignersCallerSession) QuorumCount(_epoch *big.Int) (*big.Int, error) {
	return _DASigners.Contract.QuorumCount(&_DASigners.CallOpts, _epoch)
}

// RegisteredEpoch is a free data retrieval call binding the contract method 0x6c9e560c.
//
// Solidity: function registeredEpoch(address _account, uint256 _epoch) view returns(bool)
func (_DASigners *DASignersCaller) RegisteredEpoch(opts *bind.CallOpts, _account common.Address, _epoch *big.Int) (bool, error) {
	var out []interface{}
	err := _DASigners.contract.Call(opts, &out, "registeredEpoch", _account, _epoch)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// RegisteredEpoch is a free data retrieval call binding the contract method 0x6c9e560c.
//
// Solidity: function registeredEpoch(address _account, uint256 _epoch) view returns(bool)
func (_DASigners *DASignersSession) RegisteredEpoch(_account common.Address, _epoch *big.Int) (bool, error) {
	return _DASigners.Contract.RegisteredEpoch(&_DASigners.CallOpts, _account, _epoch)
}

// RegisteredEpoch is a free data retrieval call binding the contract method 0x6c9e560c.
//
// Solidity: function registeredEpoch(address _account, uint256 _epoch) view returns(bool)
func (_DASigners *DASignersCallerSession) RegisteredEpoch(_account common.Address, _epoch *big.Int) (bool, error) {
	return _DASigners.Contract.RegisteredEpoch(&_DASigners.CallOpts, _account, _epoch)
}

// MakeEpoch is a paid mutator transaction binding the contract method 0x5a889f0c.
//
// Solidity: function makeEpoch() returns()
func (_DASigners *DASignersTransactor) MakeEpoch(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DASigners.contract.Transact(opts, "makeEpoch")
}

// MakeEpoch is a paid mutator transaction binding the contract method 0x5a889f0c.
//
// Solidity: function makeEpoch() returns()
func (_DASigners *DASignersSession) MakeEpoch() (*types.Transaction, error) {
	return _DASigners.Contract.MakeEpoch(&_DASigners.TransactOpts)
}

// MakeEpoch is a paid mutator transaction binding the contract method 0x5a889f0c.
//
// Solidity: function makeEpoch() returns()
func (_DASigners *DASignersTransactorSession) MakeEpoch() (*types.Transaction, error) {
	return _DASigners.Contract.MakeEpoch(&_DASigners.TransactOpts)
}

// RegisterNextEpoch is a paid mutator transaction binding the contract method 0x56a32372.
//
// Solidity: function registerNextEpoch((uint256,uint256) _signature) returns()
func (_DASigners *DASignersTransactor) RegisterNextEpoch(opts *bind.TransactOpts, _signature BN254G1Point) (*types.Transaction, error) {
	return _DASigners.contract.Transact(opts, "registerNextEpoch", _signature)
}

// RegisterNextEpoch is a paid mutator transaction binding the contract method 0x56a32372.
//
// Solidity: function registerNextEpoch((uint256,uint256) _signature) returns()
func (_DASigners *DASignersSession) RegisterNextEpoch(_signature BN254G1Point) (*types.Transaction, error) {
	return _DASigners.Contract.RegisterNextEpoch(&_DASigners.TransactOpts, _signature)
}

// RegisterNextEpoch is a paid mutator transaction binding the contract method 0x56a32372.
//
// Solidity: function registerNextEpoch((uint256,uint256) _signature) returns()
func (_DASigners *DASignersTransactorSession) RegisterNextEpoch(_signature BN254G1Point) (*types.Transaction, error) {
	return _DASigners.Contract.RegisterNextEpoch(&_DASigners.TransactOpts, _signature)
}

// RegisterSigner is a paid mutator transaction binding the contract method 0x7ca4dd5e.
//
// Solidity: function registerSigner((address,string,(uint256,uint256),(uint256[2],uint256[2])) _signer, (uint256,uint256) _signature) returns()
func (_DASigners *DASignersTransactor) RegisterSigner(opts *bind.TransactOpts, _signer IDASignersSignerDetail, _signature BN254G1Point) (*types.Transaction, error) {
	return _DASigners.contract.Transact(opts, "registerSigner", _signer, _signature)
}

// RegisterSigner is a paid mutator transaction binding the contract method 0x7ca4dd5e.
//
// Solidity: function registerSigner((address,string,(uint256,uint256),(uint256[2],uint256[2])) _signer, (uint256,uint256) _signature) returns()
func (_DASigners *DASignersSession) RegisterSigner(_signer IDASignersSignerDetail, _signature BN254G1Point) (*types.Transaction, error) {
	return _DASigners.Contract.RegisterSigner(&_DASigners.TransactOpts, _signer, _signature)
}

// RegisterSigner is a paid mutator transaction binding the contract method 0x7ca4dd5e.
//
// Solidity: function registerSigner((address,string,(uint256,uint256),(uint256[2],uint256[2])) _signer, (uint256,uint256) _signature) returns()
func (_DASigners *DASignersTransactorSession) RegisterSigner(_signer IDASignersSignerDetail, _signature BN254G1Point) (*types.Transaction, error) {
	return _DASigners.Contract.RegisterSigner(&_DASigners.TransactOpts, _signer, _signature)
}

// UpdateSocket is a paid mutator transaction binding the contract method 0x0cf4b767.
//
// Solidity: function updateSocket(string _socket) returns()
func (_DASigners *DASignersTransactor) UpdateSocket(opts *bind.TransactOpts, _socket string) (*types.Transaction, error) {
	return _DASigners.contract.Transact(opts, "updateSocket", _socket)
}

// UpdateSocket is a paid mutator transaction binding the contract method 0x0cf4b767.
//
// Solidity: function updateSocket(string _socket) returns()
func (_DASigners *DASignersSession) UpdateSocket(_socket string) (*types.Transaction, error) {
	return _DASigners.Contract.UpdateSocket(&_DASigners.TransactOpts, _socket)
}

// UpdateSocket is a paid mutator transaction binding the contract method 0x0cf4b767.
//
// Solidity: function updateSocket(string _socket) returns()
func (_DASigners *DASignersTransactorSession) UpdateSocket(_socket string) (*types.Transaction, error) {
	return _DASigners.Contract.UpdateSocket(&_DASigners.TransactOpts, _socket)
}

// DASignersNewSignerIterator is returned from FilterNewSigner and is used to iterate over the raw logs and unpacked data for NewSigner events raised by the DASigners contract.
type DASignersNewSignerIterator struct {
	Event *DASignersNewSigner // Event containing the contract specifics and raw log

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
func (it *DASignersNewSignerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DASignersNewSigner)
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
		it.Event = new(DASignersNewSigner)
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
func (it *DASignersNewSignerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DASignersNewSignerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DASignersNewSigner represents a NewSigner event raised by the DASigners contract.
type DASignersNewSigner struct {
	Signer common.Address
	PkG1   BN254G1Point
	PkG2   BN254G2Point
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterNewSigner is a free log retrieval operation binding the contract event 0x679917c2006df1daaa987a56bf1d66e99764d5ad317892d9e83a6eb4e3f051e7.
//
// Solidity: event NewSigner(address indexed signer, (uint256,uint256) pkG1, (uint256[2],uint256[2]) pkG2)
func (_DASigners *DASignersFilterer) FilterNewSigner(opts *bind.FilterOpts, signer []common.Address) (*DASignersNewSignerIterator, error) {

	var signerRule []interface{}
	for _, signerItem := range signer {
		signerRule = append(signerRule, signerItem)
	}

	logs, sub, err := _DASigners.contract.FilterLogs(opts, "NewSigner", signerRule)
	if err != nil {
		return nil, err
	}
	return &DASignersNewSignerIterator{contract: _DASigners.contract, event: "NewSigner", logs: logs, sub: sub}, nil
}

// WatchNewSigner is a free log subscription operation binding the contract event 0x679917c2006df1daaa987a56bf1d66e99764d5ad317892d9e83a6eb4e3f051e7.
//
// Solidity: event NewSigner(address indexed signer, (uint256,uint256) pkG1, (uint256[2],uint256[2]) pkG2)
func (_DASigners *DASignersFilterer) WatchNewSigner(opts *bind.WatchOpts, sink chan<- *DASignersNewSigner, signer []common.Address) (event.Subscription, error) {

	var signerRule []interface{}
	for _, signerItem := range signer {
		signerRule = append(signerRule, signerItem)
	}

	logs, sub, err := _DASigners.contract.WatchLogs(opts, "NewSigner", signerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DASignersNewSigner)
				if err := _DASigners.contract.UnpackLog(event, "NewSigner", log); err != nil {
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

// ParseNewSigner is a log parse operation binding the contract event 0x679917c2006df1daaa987a56bf1d66e99764d5ad317892d9e83a6eb4e3f051e7.
//
// Solidity: event NewSigner(address indexed signer, (uint256,uint256) pkG1, (uint256[2],uint256[2]) pkG2)
func (_DASigners *DASignersFilterer) ParseNewSigner(log types.Log) (*DASignersNewSigner, error) {
	event := new(DASignersNewSigner)
	if err := _DASigners.contract.UnpackLog(event, "NewSigner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DASignersSocketUpdatedIterator is returned from FilterSocketUpdated and is used to iterate over the raw logs and unpacked data for SocketUpdated events raised by the DASigners contract.
type DASignersSocketUpdatedIterator struct {
	Event *DASignersSocketUpdated // Event containing the contract specifics and raw log

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
func (it *DASignersSocketUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DASignersSocketUpdated)
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
		it.Event = new(DASignersSocketUpdated)
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
func (it *DASignersSocketUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DASignersSocketUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DASignersSocketUpdated represents a SocketUpdated event raised by the DASigners contract.
type DASignersSocketUpdated struct {
	Signer common.Address
	Socket string
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSocketUpdated is a free log retrieval operation binding the contract event 0x09617a966176a40f8f1410768b118506db0096484acd5811064fcc12038798de.
//
// Solidity: event SocketUpdated(address indexed signer, string socket)
func (_DASigners *DASignersFilterer) FilterSocketUpdated(opts *bind.FilterOpts, signer []common.Address) (*DASignersSocketUpdatedIterator, error) {

	var signerRule []interface{}
	for _, signerItem := range signer {
		signerRule = append(signerRule, signerItem)
	}

	logs, sub, err := _DASigners.contract.FilterLogs(opts, "SocketUpdated", signerRule)
	if err != nil {
		return nil, err
	}
	return &DASignersSocketUpdatedIterator{contract: _DASigners.contract, event: "SocketUpdated", logs: logs, sub: sub}, nil
}

// WatchSocketUpdated is a free log subscription operation binding the contract event 0x09617a966176a40f8f1410768b118506db0096484acd5811064fcc12038798de.
//
// Solidity: event SocketUpdated(address indexed signer, string socket)
func (_DASigners *DASignersFilterer) WatchSocketUpdated(opts *bind.WatchOpts, sink chan<- *DASignersSocketUpdated, signer []common.Address) (event.Subscription, error) {

	var signerRule []interface{}
	for _, signerItem := range signer {
		signerRule = append(signerRule, signerItem)
	}

	logs, sub, err := _DASigners.contract.WatchLogs(opts, "SocketUpdated", signerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DASignersSocketUpdated)
				if err := _DASigners.contract.UnpackLog(event, "SocketUpdated", log); err != nil {
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

// ParseSocketUpdated is a log parse operation binding the contract event 0x09617a966176a40f8f1410768b118506db0096484acd5811064fcc12038798de.
//
// Solidity: event SocketUpdated(address indexed signer, string socket)
func (_DASigners *DASignersFilterer) ParseSocketUpdated(log types.Log) (*DASignersSocketUpdated, error) {
	event := new(DASignersSocketUpdated)
	if err := _DASigners.contract.UnpackLog(event, "SocketUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
