// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package core

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
	_ = abi.ConvertType
)

// TicketAllocatorMetaData contains all meta data concerning the TicketAllocator contract.
var TicketAllocatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"InvalidBlockNumber\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidPayment\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotEnoughTickets\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotSystemContract\",\"type\":\"error\"},{\"stateMutability\":\"nonpayable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"GetBalance\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"},{\"internalType\":\"uint16[]\",\"name\":\"\",\"type\":\"uint16[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"numTickets\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"bidPerTicket\",\"type\":\"uint256\"}],\"name\":\"RequestTickets\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TIMEOUT\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TOTAL_TICKETS\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"balance\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"head\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"pendingBids\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"requestor\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"amount\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"bidPerTicket\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"queue\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"amount\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"bidPerTicket\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"requestor\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"senders\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"withdrawable\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// TicketAllocatorABI is the input ABI used to generate the binding from.
// Deprecated: Use TicketAllocatorMetaData.ABI instead.
var TicketAllocatorABI = TicketAllocatorMetaData.ABI

// TicketAllocator is an auto generated Go binding around an Ethereum contract.
type TicketAllocator struct {
	TicketAllocatorCaller     // Read-only binding to the contract
	TicketAllocatorTransactor // Write-only binding to the contract
	TicketAllocatorFilterer   // Log filterer for contract events
}

// TicketAllocatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type TicketAllocatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TicketAllocatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TicketAllocatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TicketAllocatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TicketAllocatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TicketAllocatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TicketAllocatorSession struct {
	Contract     *TicketAllocator  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TicketAllocatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TicketAllocatorCallerSession struct {
	Contract *TicketAllocatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// TicketAllocatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TicketAllocatorTransactorSession struct {
	Contract     *TicketAllocatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// TicketAllocatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type TicketAllocatorRaw struct {
	Contract *TicketAllocator // Generic contract binding to access the raw methods on
}

// TicketAllocatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TicketAllocatorCallerRaw struct {
	Contract *TicketAllocatorCaller // Generic read-only contract binding to access the raw methods on
}

// TicketAllocatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TicketAllocatorTransactorRaw struct {
	Contract *TicketAllocatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTicketAllocator creates a new instance of TicketAllocator, bound to a specific deployed contract.
func NewTicketAllocator(address common.Address, backend bind.ContractBackend) (*TicketAllocator, error) {
	contract, err := bindTicketAllocator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TicketAllocator{TicketAllocatorCaller: TicketAllocatorCaller{contract: contract}, TicketAllocatorTransactor: TicketAllocatorTransactor{contract: contract}, TicketAllocatorFilterer: TicketAllocatorFilterer{contract: contract}}, nil
}

// NewTicketAllocatorCaller creates a new read-only instance of TicketAllocator, bound to a specific deployed contract.
func NewTicketAllocatorCaller(address common.Address, caller bind.ContractCaller) (*TicketAllocatorCaller, error) {
	contract, err := bindTicketAllocator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TicketAllocatorCaller{contract: contract}, nil
}

// NewTicketAllocatorTransactor creates a new write-only instance of TicketAllocator, bound to a specific deployed contract.
func NewTicketAllocatorTransactor(address common.Address, transactor bind.ContractTransactor) (*TicketAllocatorTransactor, error) {
	contract, err := bindTicketAllocator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TicketAllocatorTransactor{contract: contract}, nil
}

// NewTicketAllocatorFilterer creates a new log filterer instance of TicketAllocator, bound to a specific deployed contract.
func NewTicketAllocatorFilterer(address common.Address, filterer bind.ContractFilterer) (*TicketAllocatorFilterer, error) {
	contract, err := bindTicketAllocator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TicketAllocatorFilterer{contract: contract}, nil
}

// bindTicketAllocator binds a generic wrapper to an already deployed contract.
func bindTicketAllocator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TicketAllocatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TicketAllocator *TicketAllocatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TicketAllocator.Contract.TicketAllocatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TicketAllocator *TicketAllocatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TicketAllocator.Contract.TicketAllocatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TicketAllocator *TicketAllocatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TicketAllocator.Contract.TicketAllocatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TicketAllocator *TicketAllocatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TicketAllocator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TicketAllocator *TicketAllocatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TicketAllocator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TicketAllocator *TicketAllocatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TicketAllocator.Contract.contract.Transact(opts, method, params...)
}

// GetBalance is a free data retrieval call binding the contract method 0xf8f8a912.
//
// Solidity: function GetBalance() view returns(address[], uint16[])
func (_TicketAllocator *TicketAllocatorCaller) GetBalance(opts *bind.CallOpts) ([]common.Address, []uint16, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "GetBalance")

	if err != nil {
		return *new([]common.Address), *new([]uint16), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	out1 := *abi.ConvertType(out[1], new([]uint16)).(*[]uint16)

	return out0, out1, err

}

// GetBalance is a free data retrieval call binding the contract method 0xf8f8a912.
//
// Solidity: function GetBalance() view returns(address[], uint16[])
func (_TicketAllocator *TicketAllocatorSession) GetBalance() ([]common.Address, []uint16, error) {
	return _TicketAllocator.Contract.GetBalance(&_TicketAllocator.CallOpts)
}

// GetBalance is a free data retrieval call binding the contract method 0xf8f8a912.
//
// Solidity: function GetBalance() view returns(address[], uint16[])
func (_TicketAllocator *TicketAllocatorCallerSession) GetBalance() ([]common.Address, []uint16, error) {
	return _TicketAllocator.Contract.GetBalance(&_TicketAllocator.CallOpts)
}

// TIMEOUT is a free data retrieval call binding the contract method 0xf56f48f2.
//
// Solidity: function TIMEOUT() view returns(uint16)
func (_TicketAllocator *TicketAllocatorCaller) TIMEOUT(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "TIMEOUT")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// TIMEOUT is a free data retrieval call binding the contract method 0xf56f48f2.
//
// Solidity: function TIMEOUT() view returns(uint16)
func (_TicketAllocator *TicketAllocatorSession) TIMEOUT() (uint16, error) {
	return _TicketAllocator.Contract.TIMEOUT(&_TicketAllocator.CallOpts)
}

// TIMEOUT is a free data retrieval call binding the contract method 0xf56f48f2.
//
// Solidity: function TIMEOUT() view returns(uint16)
func (_TicketAllocator *TicketAllocatorCallerSession) TIMEOUT() (uint16, error) {
	return _TicketAllocator.Contract.TIMEOUT(&_TicketAllocator.CallOpts)
}

// TOTALTICKETS is a free data retrieval call binding the contract method 0xafbd1782.
//
// Solidity: function TOTAL_TICKETS() view returns(uint16)
func (_TicketAllocator *TicketAllocatorCaller) TOTALTICKETS(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "TOTAL_TICKETS")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// TOTALTICKETS is a free data retrieval call binding the contract method 0xafbd1782.
//
// Solidity: function TOTAL_TICKETS() view returns(uint16)
func (_TicketAllocator *TicketAllocatorSession) TOTALTICKETS() (uint16, error) {
	return _TicketAllocator.Contract.TOTALTICKETS(&_TicketAllocator.CallOpts)
}

// TOTALTICKETS is a free data retrieval call binding the contract method 0xafbd1782.
//
// Solidity: function TOTAL_TICKETS() view returns(uint16)
func (_TicketAllocator *TicketAllocatorCallerSession) TOTALTICKETS() (uint16, error) {
	return _TicketAllocator.Contract.TOTALTICKETS(&_TicketAllocator.CallOpts)
}

// Balance is a free data retrieval call binding the contract method 0xe3d670d7.
//
// Solidity: function balance(address ) view returns(uint16)
func (_TicketAllocator *TicketAllocatorCaller) Balance(opts *bind.CallOpts, arg0 common.Address) (uint16, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "balance", arg0)

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// Balance is a free data retrieval call binding the contract method 0xe3d670d7.
//
// Solidity: function balance(address ) view returns(uint16)
func (_TicketAllocator *TicketAllocatorSession) Balance(arg0 common.Address) (uint16, error) {
	return _TicketAllocator.Contract.Balance(&_TicketAllocator.CallOpts, arg0)
}

// Balance is a free data retrieval call binding the contract method 0xe3d670d7.
//
// Solidity: function balance(address ) view returns(uint16)
func (_TicketAllocator *TicketAllocatorCallerSession) Balance(arg0 common.Address) (uint16, error) {
	return _TicketAllocator.Contract.Balance(&_TicketAllocator.CallOpts, arg0)
}

// Head is a free data retrieval call binding the contract method 0xbda4fec5.
//
// Solidity: function head(address ) view returns(uint256)
func (_TicketAllocator *TicketAllocatorCaller) Head(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "head", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Head is a free data retrieval call binding the contract method 0xbda4fec5.
//
// Solidity: function head(address ) view returns(uint256)
func (_TicketAllocator *TicketAllocatorSession) Head(arg0 common.Address) (*big.Int, error) {
	return _TicketAllocator.Contract.Head(&_TicketAllocator.CallOpts, arg0)
}

// Head is a free data retrieval call binding the contract method 0xbda4fec5.
//
// Solidity: function head(address ) view returns(uint256)
func (_TicketAllocator *TicketAllocatorCallerSession) Head(arg0 common.Address) (*big.Int, error) {
	return _TicketAllocator.Contract.Head(&_TicketAllocator.CallOpts, arg0)
}

// PendingBids is a free data retrieval call binding the contract method 0xcf8589b9.
//
// Solidity: function pendingBids(uint256 ) view returns(address sender, address requestor, uint16 amount, uint256 bidPerTicket)
func (_TicketAllocator *TicketAllocatorCaller) PendingBids(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Sender       common.Address
	Requestor    common.Address
	Amount       uint16
	BidPerTicket *big.Int
}, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "pendingBids", arg0)

	outstruct := new(struct {
		Sender       common.Address
		Requestor    common.Address
		Amount       uint16
		BidPerTicket *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Sender = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Requestor = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Amount = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.BidPerTicket = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PendingBids is a free data retrieval call binding the contract method 0xcf8589b9.
//
// Solidity: function pendingBids(uint256 ) view returns(address sender, address requestor, uint16 amount, uint256 bidPerTicket)
func (_TicketAllocator *TicketAllocatorSession) PendingBids(arg0 *big.Int) (struct {
	Sender       common.Address
	Requestor    common.Address
	Amount       uint16
	BidPerTicket *big.Int
}, error) {
	return _TicketAllocator.Contract.PendingBids(&_TicketAllocator.CallOpts, arg0)
}

// PendingBids is a free data retrieval call binding the contract method 0xcf8589b9.
//
// Solidity: function pendingBids(uint256 ) view returns(address sender, address requestor, uint16 amount, uint256 bidPerTicket)
func (_TicketAllocator *TicketAllocatorCallerSession) PendingBids(arg0 *big.Int) (struct {
	Sender       common.Address
	Requestor    common.Address
	Amount       uint16
	BidPerTicket *big.Int
}, error) {
	return _TicketAllocator.Contract.PendingBids(&_TicketAllocator.CallOpts, arg0)
}

// PendingBlock is a free data retrieval call binding the contract method 0xd33d9969.
//
// Solidity: function pendingBlock() view returns(uint256)
func (_TicketAllocator *TicketAllocatorCaller) PendingBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "pendingBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PendingBlock is a free data retrieval call binding the contract method 0xd33d9969.
//
// Solidity: function pendingBlock() view returns(uint256)
func (_TicketAllocator *TicketAllocatorSession) PendingBlock() (*big.Int, error) {
	return _TicketAllocator.Contract.PendingBlock(&_TicketAllocator.CallOpts)
}

// PendingBlock is a free data retrieval call binding the contract method 0xd33d9969.
//
// Solidity: function pendingBlock() view returns(uint256)
func (_TicketAllocator *TicketAllocatorCallerSession) PendingBlock() (*big.Int, error) {
	return _TicketAllocator.Contract.PendingBlock(&_TicketAllocator.CallOpts)
}

// Queue is a free data retrieval call binding the contract method 0x44287113.
//
// Solidity: function queue(address , uint256 ) view returns(uint16 amount, uint256 blockNumber, uint256 bidPerTicket, address requestor)
func (_TicketAllocator *TicketAllocatorCaller) Queue(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (struct {
	Amount       uint16
	BlockNumber  *big.Int
	BidPerTicket *big.Int
	Requestor    common.Address
}, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "queue", arg0, arg1)

	outstruct := new(struct {
		Amount       uint16
		BlockNumber  *big.Int
		BidPerTicket *big.Int
		Requestor    common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Amount = *abi.ConvertType(out[0], new(uint16)).(*uint16)
	outstruct.BlockNumber = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.BidPerTicket = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Requestor = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// Queue is a free data retrieval call binding the contract method 0x44287113.
//
// Solidity: function queue(address , uint256 ) view returns(uint16 amount, uint256 blockNumber, uint256 bidPerTicket, address requestor)
func (_TicketAllocator *TicketAllocatorSession) Queue(arg0 common.Address, arg1 *big.Int) (struct {
	Amount       uint16
	BlockNumber  *big.Int
	BidPerTicket *big.Int
	Requestor    common.Address
}, error) {
	return _TicketAllocator.Contract.Queue(&_TicketAllocator.CallOpts, arg0, arg1)
}

// Queue is a free data retrieval call binding the contract method 0x44287113.
//
// Solidity: function queue(address , uint256 ) view returns(uint16 amount, uint256 blockNumber, uint256 bidPerTicket, address requestor)
func (_TicketAllocator *TicketAllocatorCallerSession) Queue(arg0 common.Address, arg1 *big.Int) (struct {
	Amount       uint16
	BlockNumber  *big.Int
	BidPerTicket *big.Int
	Requestor    common.Address
}, error) {
	return _TicketAllocator.Contract.Queue(&_TicketAllocator.CallOpts, arg0, arg1)
}

// Senders is a free data retrieval call binding the contract method 0x9977c78a.
//
// Solidity: function senders(uint256 ) view returns(address)
func (_TicketAllocator *TicketAllocatorCaller) Senders(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "senders", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Senders is a free data retrieval call binding the contract method 0x9977c78a.
//
// Solidity: function senders(uint256 ) view returns(address)
func (_TicketAllocator *TicketAllocatorSession) Senders(arg0 *big.Int) (common.Address, error) {
	return _TicketAllocator.Contract.Senders(&_TicketAllocator.CallOpts, arg0)
}

// Senders is a free data retrieval call binding the contract method 0x9977c78a.
//
// Solidity: function senders(uint256 ) view returns(address)
func (_TicketAllocator *TicketAllocatorCallerSession) Senders(arg0 *big.Int) (common.Address, error) {
	return _TicketAllocator.Contract.Senders(&_TicketAllocator.CallOpts, arg0)
}

// Withdrawable is a free data retrieval call binding the contract method 0xce513b6f.
//
// Solidity: function withdrawable(address ) view returns(uint256)
func (_TicketAllocator *TicketAllocatorCaller) Withdrawable(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TicketAllocator.contract.Call(opts, &out, "withdrawable", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Withdrawable is a free data retrieval call binding the contract method 0xce513b6f.
//
// Solidity: function withdrawable(address ) view returns(uint256)
func (_TicketAllocator *TicketAllocatorSession) Withdrawable(arg0 common.Address) (*big.Int, error) {
	return _TicketAllocator.Contract.Withdrawable(&_TicketAllocator.CallOpts, arg0)
}

// Withdrawable is a free data retrieval call binding the contract method 0xce513b6f.
//
// Solidity: function withdrawable(address ) view returns(uint256)
func (_TicketAllocator *TicketAllocatorCallerSession) Withdrawable(arg0 common.Address) (*big.Int, error) {
	return _TicketAllocator.Contract.Withdrawable(&_TicketAllocator.CallOpts, arg0)
}

// RequestTickets is a paid mutator transaction binding the contract method 0x91ef4c9d.
//
// Solidity: function RequestTickets(address sender, uint16 numTickets, uint256 bidPerTicket) payable returns()
func (_TicketAllocator *TicketAllocatorTransactor) RequestTickets(opts *bind.TransactOpts, sender common.Address, numTickets uint16, bidPerTicket *big.Int) (*types.Transaction, error) {
	return _TicketAllocator.contract.Transact(opts, "RequestTickets", sender, numTickets, bidPerTicket)
}

// RequestTickets is a paid mutator transaction binding the contract method 0x91ef4c9d.
//
// Solidity: function RequestTickets(address sender, uint16 numTickets, uint256 bidPerTicket) payable returns()
func (_TicketAllocator *TicketAllocatorSession) RequestTickets(sender common.Address, numTickets uint16, bidPerTicket *big.Int) (*types.Transaction, error) {
	return _TicketAllocator.Contract.RequestTickets(&_TicketAllocator.TransactOpts, sender, numTickets, bidPerTicket)
}

// RequestTickets is a paid mutator transaction binding the contract method 0x91ef4c9d.
//
// Solidity: function RequestTickets(address sender, uint16 numTickets, uint256 bidPerTicket) payable returns()
func (_TicketAllocator *TicketAllocatorTransactorSession) RequestTickets(sender common.Address, numTickets uint16, bidPerTicket *big.Int) (*types.Transaction, error) {
	return _TicketAllocator.Contract.RequestTickets(&_TicketAllocator.TransactOpts, sender, numTickets, bidPerTicket)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_TicketAllocator *TicketAllocatorTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TicketAllocator.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_TicketAllocator *TicketAllocatorSession) Withdraw() (*types.Transaction, error) {
	return _TicketAllocator.Contract.Withdraw(&_TicketAllocator.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_TicketAllocator *TicketAllocatorTransactorSession) Withdraw() (*types.Transaction, error) {
	return _TicketAllocator.Contract.Withdraw(&_TicketAllocator.TransactOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_TicketAllocator *TicketAllocatorTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _TicketAllocator.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_TicketAllocator *TicketAllocatorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _TicketAllocator.Contract.Fallback(&_TicketAllocator.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_TicketAllocator *TicketAllocatorTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _TicketAllocator.Contract.Fallback(&_TicketAllocator.TransactOpts, calldata)
}
