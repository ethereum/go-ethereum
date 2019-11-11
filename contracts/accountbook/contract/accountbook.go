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

// AccountBookABI is the input ABI used to generate the binding from.
const AccountBookABI = "[{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_challengeTimeWindow\",\"type\":\"uint64\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldBalance\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newBalance\",\"type\":\"uint256\"}],\"name\":\"balanceChangedEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawEvent\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"payer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"sig_v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"sig_r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sig_s\",\"type\":\"bytes32\"}],\"name\":\"cash\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"challengeTimeWindow\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"claim\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"deposit\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"addresspayable\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"paids\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"withdrawRequests\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"createdAt\",\"type\":\"uint128\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// AccountBookBin is the compiled bytecode used for deploying new contracts.
var AccountBookBin = "0x608060405234801561001057600080fd5b506040516108ff3803806108ff8339818101604052602081101561003357600080fd5b5051600380546001600160401b039092167401000000000000000000000000000000000000000002600160a01b600160e01b03196001600160a01b03199093163317929092169190911790556108718061008e6000396000f3fe6080604052600436106100865760003560e01c806352df49ec1161005957806352df49ec146101435780638da5cb5b1461019c578063d0e30db0146101cd578063fbf788d6146101d5578063fc7e286d1461022357610086565b80630c18c9ac1461008b5780632e1a7d4d146100d0578063481acb8f146100fc5780634e71d92d1461012e575b600080fd5b34801561009757600080fd5b506100be600480360360208110156100ae57600080fd5b50356001600160a01b0316610256565b60408051918252519081900360200190f35b3480156100dc57600080fd5b506100fa600480360360208110156100f357600080fd5b5035610268565b005b34801561010857600080fd5b50610111610340565b6040805167ffffffffffffffff9092168252519081900360200190f35b34801561013a57600080fd5b506100fa610357565b34801561014f57600080fd5b506101766004803603602081101561016657600080fd5b50356001600160a01b0316610450565b604080516001600160801b03938416815291909216602082015281519081900390910190f35b3480156101a857600080fd5b506101b1610476565b604080516001600160a01b039092168252519081900360200190f35b6100fa610485565b3480156101e157600080fd5b506100fa600480360360a08110156101f857600080fd5b506001600160a01b038135169060208101359060ff604082013516906060810135906080013561051e565b34801561022f57600080fd5b506100be6004803603602081101561024657600080fd5b50356001600160a01b03166107e9565b60006020819052908152604090205481565b806102725761033d565b3360009081526001602052604090205481111561028e5761033d565b336000908152600260205260409020546001600160801b0316156102b15761033d565b6040805180820182526001600160801b0380841682524381166020808401918252336000818152600283528690209451855493518516600160801b029085166001600160801b031990941693909317909316919091179092558251848152925190927f87d5f4772963d1f9b76047158b4ae97c420a1b3bff2a746c828beffd9e7c3e2692908290030190a25b50565b600354600160a01b900467ffffffffffffffff1681565b336000908152600260205260409020546001600160801b03168061037b575061044e565b60035433600090815260026020526040902054600160a01b90910467ffffffffffffffff16600160801b9091046001600160801b0316430310156103bf575061044e565b33600081815260016020908152604080832080546001600160801b0387168082039092556002909352818420849055905191939281156108fc029290818181858888f19350505050158015610418573d6000803e3d6000fd5b50604080518281526001600160801b038416830360208201528151339260008051602061081d833981519152928290030190a250505b565b6002602052600090815260409020546001600160801b0380821691600160801b90041682565b6003546001600160a01b031681565b3360009081526001602052604090205434810181106104d55760405162461bcd60e51b81526004018080602001828103825260218152602001806107fc6021913960400191505060405180910390fd5b33600081815260016020908152604091829020805434908101909155825185815290850191810191909152815160008051602061081d833981519152929181900390910190a250565b6003546001600160a01b0316331461053557600080fd5b6001600160a01b038516600090815260208190526040902054841161055957600080fd5b60408051601960f81b6020808301919091526000602183018190523060601b6022840152603680840189905284518085039091018152605684018086528151918401919091209190526076830180855281905260ff8716609684015260b6830186905260d68301859052925160019260f68082019392601f1981019281900390910190855afa1580156105f0573d6000803e3d6000fd5b505050602060405103516001600160a01b0316866001600160a01b03161461061757600080fd5b6001600160a01b0386166000908152600160209081526040808320549183905282205490919087038083106106d0576001600160a01b03808a1660009081526001602052604080822084870390819055600354915190955092169183156108fc0291849190818181858888f19350505050158015610699573d6000803e3d6000fd5b50604080518481526020810184905281516001600160a01b038c169260008051602061081d833981519152928290030190a2610755565b8215610755576001600160a01b03808a16600090815260016020526040808220829055600354905192169185156108fc0291869190818181858888f19350505050158015610722573d6000803e3d6000fd5b50604080518481526000602082015281516001600160a01b038c169260008051602061081d833981519152928290030190a25b6001600160a01b0389166000908152602081815260408083208b905560029091529020546001600160801b03168210156107de57816107ac576001600160a01b0389166000908152600260205260408120556107de565b6001600160a01b038916600090815260026020526040902080546001600160801b0319166001600160801b0384161790555b505050505050505050565b6001602052600090815260409020548156fe6164646974696f6e206f766572666c6f77206f72207a65726f206465706f7369744decf22c5bd17fc02ce062b1d086abebc7516d810e214d9ad784fc1a023fbba6a265627a7a723158207177277e626e0106cb630fb83ae0319188ba7f2a583b1f8846d7c25dbb3daa4164736f6c634300050c0032"

// DeployAccountBook deploys a new Ethereum contract, binding an instance of AccountBook to it.
func DeployAccountBook(auth *bind.TransactOpts, backend bind.ContractBackend, _challengeTimeWindow uint64) (common.Address, *types.Transaction, *AccountBook, error) {
	parsed, err := abi.JSON(strings.NewReader(AccountBookABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(AccountBookBin), backend, _challengeTimeWindow)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AccountBook{AccountBookCaller: AccountBookCaller{contract: contract}, AccountBookTransactor: AccountBookTransactor{contract: contract}, AccountBookFilterer: AccountBookFilterer{contract: contract}}, nil
}

// AccountBook is an auto generated Go binding around an Ethereum contract.
type AccountBook struct {
	AccountBookCaller     // Read-only binding to the contract
	AccountBookTransactor // Write-only binding to the contract
	AccountBookFilterer   // Log filterer for contract events
}

// AccountBookCaller is an auto generated read-only Go binding around an Ethereum contract.
type AccountBookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountBookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AccountBookTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountBookFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AccountBookFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountBookSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AccountBookSession struct {
	Contract     *AccountBook      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AccountBookCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AccountBookCallerSession struct {
	Contract *AccountBookCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// AccountBookTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AccountBookTransactorSession struct {
	Contract     *AccountBookTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// AccountBookRaw is an auto generated low-level Go binding around an Ethereum contract.
type AccountBookRaw struct {
	Contract *AccountBook // Generic contract binding to access the raw methods on
}

// AccountBookCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AccountBookCallerRaw struct {
	Contract *AccountBookCaller // Generic read-only contract binding to access the raw methods on
}

// AccountBookTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AccountBookTransactorRaw struct {
	Contract *AccountBookTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAccountBook creates a new instance of AccountBook, bound to a specific deployed contract.
func NewAccountBook(address common.Address, backend bind.ContractBackend) (*AccountBook, error) {
	contract, err := bindAccountBook(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AccountBook{AccountBookCaller: AccountBookCaller{contract: contract}, AccountBookTransactor: AccountBookTransactor{contract: contract}, AccountBookFilterer: AccountBookFilterer{contract: contract}}, nil
}

// NewAccountBookCaller creates a new read-only instance of AccountBook, bound to a specific deployed contract.
func NewAccountBookCaller(address common.Address, caller bind.ContractCaller) (*AccountBookCaller, error) {
	contract, err := bindAccountBook(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AccountBookCaller{contract: contract}, nil
}

// NewAccountBookTransactor creates a new write-only instance of AccountBook, bound to a specific deployed contract.
func NewAccountBookTransactor(address common.Address, transactor bind.ContractTransactor) (*AccountBookTransactor, error) {
	contract, err := bindAccountBook(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AccountBookTransactor{contract: contract}, nil
}

// NewAccountBookFilterer creates a new log filterer instance of AccountBook, bound to a specific deployed contract.
func NewAccountBookFilterer(address common.Address, filterer bind.ContractFilterer) (*AccountBookFilterer, error) {
	contract, err := bindAccountBook(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AccountBookFilterer{contract: contract}, nil
}

// bindAccountBook binds a generic wrapper to an already deployed contract.
func bindAccountBook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AccountBookABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AccountBook *AccountBookRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _AccountBook.Contract.AccountBookCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AccountBook *AccountBookRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountBook.Contract.AccountBookTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AccountBook *AccountBookRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AccountBook.Contract.AccountBookTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AccountBook *AccountBookCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _AccountBook.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AccountBook *AccountBookTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountBook.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AccountBook *AccountBookTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AccountBook.Contract.contract.Transact(opts, method, params...)
}

// ChallengeTimeWindow is a free data retrieval call binding the contract method 0x481acb8f.
//
// Solidity: function challengeTimeWindow() constant returns(uint64)
func (_AccountBook *AccountBookCaller) ChallengeTimeWindow(opts *bind.CallOpts) (uint64, error) {
	var (
		ret0 = new(uint64)
	)
	out := ret0
	err := _AccountBook.contract.Call(opts, out, "challengeTimeWindow")
	return *ret0, err
}

// ChallengeTimeWindow is a free data retrieval call binding the contract method 0x481acb8f.
//
// Solidity: function challengeTimeWindow() constant returns(uint64)
func (_AccountBook *AccountBookSession) ChallengeTimeWindow() (uint64, error) {
	return _AccountBook.Contract.ChallengeTimeWindow(&_AccountBook.CallOpts)
}

// ChallengeTimeWindow is a free data retrieval call binding the contract method 0x481acb8f.
//
// Solidity: function challengeTimeWindow() constant returns(uint64)
func (_AccountBook *AccountBookCallerSession) ChallengeTimeWindow() (uint64, error) {
	return _AccountBook.Contract.ChallengeTimeWindow(&_AccountBook.CallOpts)
}

// Deposits is a free data retrieval call binding the contract method 0xfc7e286d.
//
// Solidity: function deposits(address ) constant returns(uint256)
func (_AccountBook *AccountBookCaller) Deposits(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _AccountBook.contract.Call(opts, out, "deposits", arg0)
	return *ret0, err
}

// Deposits is a free data retrieval call binding the contract method 0xfc7e286d.
//
// Solidity: function deposits(address ) constant returns(uint256)
func (_AccountBook *AccountBookSession) Deposits(arg0 common.Address) (*big.Int, error) {
	return _AccountBook.Contract.Deposits(&_AccountBook.CallOpts, arg0)
}

// Deposits is a free data retrieval call binding the contract method 0xfc7e286d.
//
// Solidity: function deposits(address ) constant returns(uint256)
func (_AccountBook *AccountBookCallerSession) Deposits(arg0 common.Address) (*big.Int, error) {
	return _AccountBook.Contract.Deposits(&_AccountBook.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_AccountBook *AccountBookCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _AccountBook.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_AccountBook *AccountBookSession) Owner() (common.Address, error) {
	return _AccountBook.Contract.Owner(&_AccountBook.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_AccountBook *AccountBookCallerSession) Owner() (common.Address, error) {
	return _AccountBook.Contract.Owner(&_AccountBook.CallOpts)
}

// Paids is a free data retrieval call binding the contract method 0x0c18c9ac.
//
// Solidity: function paids(address ) constant returns(uint256)
func (_AccountBook *AccountBookCaller) Paids(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _AccountBook.contract.Call(opts, out, "paids", arg0)
	return *ret0, err
}

// Paids is a free data retrieval call binding the contract method 0x0c18c9ac.
//
// Solidity: function paids(address ) constant returns(uint256)
func (_AccountBook *AccountBookSession) Paids(arg0 common.Address) (*big.Int, error) {
	return _AccountBook.Contract.Paids(&_AccountBook.CallOpts, arg0)
}

// Paids is a free data retrieval call binding the contract method 0x0c18c9ac.
//
// Solidity: function paids(address ) constant returns(uint256)
func (_AccountBook *AccountBookCallerSession) Paids(arg0 common.Address) (*big.Int, error) {
	return _AccountBook.Contract.Paids(&_AccountBook.CallOpts, arg0)
}

// WithdrawRequests is a free data retrieval call binding the contract method 0x52df49ec.
//
// Solidity: function withdrawRequests(address ) constant returns(uint128 amount, uint128 createdAt)
func (_AccountBook *AccountBookCaller) WithdrawRequests(opts *bind.CallOpts, arg0 common.Address) (struct {
	Amount    *big.Int
	CreatedAt *big.Int
}, error) {
	ret := new(struct {
		Amount    *big.Int
		CreatedAt *big.Int
	})
	out := ret
	err := _AccountBook.contract.Call(opts, out, "withdrawRequests", arg0)
	return *ret, err
}

// WithdrawRequests is a free data retrieval call binding the contract method 0x52df49ec.
//
// Solidity: function withdrawRequests(address ) constant returns(uint128 amount, uint128 createdAt)
func (_AccountBook *AccountBookSession) WithdrawRequests(arg0 common.Address) (struct {
	Amount    *big.Int
	CreatedAt *big.Int
}, error) {
	return _AccountBook.Contract.WithdrawRequests(&_AccountBook.CallOpts, arg0)
}

// WithdrawRequests is a free data retrieval call binding the contract method 0x52df49ec.
//
// Solidity: function withdrawRequests(address ) constant returns(uint128 amount, uint128 createdAt)
func (_AccountBook *AccountBookCallerSession) WithdrawRequests(arg0 common.Address) (struct {
	Amount    *big.Int
	CreatedAt *big.Int
}, error) {
	return _AccountBook.Contract.WithdrawRequests(&_AccountBook.CallOpts, arg0)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(address payer, uint256 amount, uint8 sig_v, bytes32 sig_r, bytes32 sig_s) returns()
func (_AccountBook *AccountBookTransactor) Cash(opts *bind.TransactOpts, payer common.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _AccountBook.contract.Transact(opts, "cash", payer, amount, sig_v, sig_r, sig_s)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(address payer, uint256 amount, uint8 sig_v, bytes32 sig_r, bytes32 sig_s) returns()
func (_AccountBook *AccountBookSession) Cash(payer common.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _AccountBook.Contract.Cash(&_AccountBook.TransactOpts, payer, amount, sig_v, sig_r, sig_s)
}

// Cash is a paid mutator transaction binding the contract method 0xfbf788d6.
//
// Solidity: function cash(address payer, uint256 amount, uint8 sig_v, bytes32 sig_r, bytes32 sig_s) returns()
func (_AccountBook *AccountBookTransactorSession) Cash(payer common.Address, amount *big.Int, sig_v uint8, sig_r [32]byte, sig_s [32]byte) (*types.Transaction, error) {
	return _AccountBook.Contract.Cash(&_AccountBook.TransactOpts, payer, amount, sig_v, sig_r, sig_s)
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_AccountBook *AccountBookTransactor) Claim(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountBook.contract.Transact(opts, "claim")
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_AccountBook *AccountBookSession) Claim() (*types.Transaction, error) {
	return _AccountBook.Contract.Claim(&_AccountBook.TransactOpts)
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_AccountBook *AccountBookTransactorSession) Claim() (*types.Transaction, error) {
	return _AccountBook.Contract.Claim(&_AccountBook.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() returns()
func (_AccountBook *AccountBookTransactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountBook.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() returns()
func (_AccountBook *AccountBookSession) Deposit() (*types.Transaction, error) {
	return _AccountBook.Contract.Deposit(&_AccountBook.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() returns()
func (_AccountBook *AccountBookTransactorSession) Deposit() (*types.Transaction, error) {
	return _AccountBook.Contract.Deposit(&_AccountBook.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_AccountBook *AccountBookTransactor) Withdraw(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _AccountBook.contract.Transact(opts, "withdraw", amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_AccountBook *AccountBookSession) Withdraw(amount *big.Int) (*types.Transaction, error) {
	return _AccountBook.Contract.Withdraw(&_AccountBook.TransactOpts, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_AccountBook *AccountBookTransactorSession) Withdraw(amount *big.Int) (*types.Transaction, error) {
	return _AccountBook.Contract.Withdraw(&_AccountBook.TransactOpts, amount)
}

// AccountBookBalanceChangedEventIterator is returned from FilterBalanceChangedEvent and is used to iterate over the raw logs and unpacked data for BalanceChangedEvent events raised by the AccountBook contract.
type AccountBookBalanceChangedEventIterator struct {
	Event *AccountBookBalanceChangedEvent // Event containing the contract specifics and raw log

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
func (it *AccountBookBalanceChangedEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountBookBalanceChangedEvent)
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
		it.Event = new(AccountBookBalanceChangedEvent)
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
func (it *AccountBookBalanceChangedEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountBookBalanceChangedEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountBookBalanceChangedEvent represents a BalanceChangedEvent event raised by the AccountBook contract.
type AccountBookBalanceChangedEvent struct {
	Addr       common.Address
	OldBalance *big.Int
	NewBalance *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterBalanceChangedEvent is a free log retrieval operation binding the contract event 0x4decf22c5bd17fc02ce062b1d086abebc7516d810e214d9ad784fc1a023fbba6.
//
// Solidity: event balanceChangedEvent(address indexed addr, uint256 oldBalance, uint256 newBalance)
func (_AccountBook *AccountBookFilterer) FilterBalanceChangedEvent(opts *bind.FilterOpts, addr []common.Address) (*AccountBookBalanceChangedEventIterator, error) {

	var addrRule []interface{}
	for _, addrItem := range addr {
		addrRule = append(addrRule, addrItem)
	}

	logs, sub, err := _AccountBook.contract.FilterLogs(opts, "balanceChangedEvent", addrRule)
	if err != nil {
		return nil, err
	}
	return &AccountBookBalanceChangedEventIterator{contract: _AccountBook.contract, event: "balanceChangedEvent", logs: logs, sub: sub}, nil
}

// WatchBalanceChangedEvent is a free log subscription operation binding the contract event 0x4decf22c5bd17fc02ce062b1d086abebc7516d810e214d9ad784fc1a023fbba6.
//
// Solidity: event balanceChangedEvent(address indexed addr, uint256 oldBalance, uint256 newBalance)
func (_AccountBook *AccountBookFilterer) WatchBalanceChangedEvent(opts *bind.WatchOpts, sink chan<- *AccountBookBalanceChangedEvent, addr []common.Address) (event.Subscription, error) {

	var addrRule []interface{}
	for _, addrItem := range addr {
		addrRule = append(addrRule, addrItem)
	}

	logs, sub, err := _AccountBook.contract.WatchLogs(opts, "balanceChangedEvent", addrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountBookBalanceChangedEvent)
				if err := _AccountBook.contract.UnpackLog(event, "balanceChangedEvent", log); err != nil {
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

// ParseBalanceChangedEvent is a log parse operation binding the contract event 0x4decf22c5bd17fc02ce062b1d086abebc7516d810e214d9ad784fc1a023fbba6.
//
// Solidity: event balanceChangedEvent(address indexed addr, uint256 oldBalance, uint256 newBalance)
func (_AccountBook *AccountBookFilterer) ParseBalanceChangedEvent(log types.Log) (*AccountBookBalanceChangedEvent, error) {
	event := new(AccountBookBalanceChangedEvent)
	if err := _AccountBook.contract.UnpackLog(event, "balanceChangedEvent", log); err != nil {
		return nil, err
	}
	return event, nil
}

// AccountBookWithdrawEventIterator is returned from FilterWithdrawEvent and is used to iterate over the raw logs and unpacked data for WithdrawEvent events raised by the AccountBook contract.
type AccountBookWithdrawEventIterator struct {
	Event *AccountBookWithdrawEvent // Event containing the contract specifics and raw log

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
func (it *AccountBookWithdrawEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountBookWithdrawEvent)
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
		it.Event = new(AccountBookWithdrawEvent)
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
func (it *AccountBookWithdrawEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountBookWithdrawEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountBookWithdrawEvent represents a WithdrawEvent event raised by the AccountBook contract.
type AccountBookWithdrawEvent struct {
	Addr   common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterWithdrawEvent is a free log retrieval operation binding the contract event 0x87d5f4772963d1f9b76047158b4ae97c420a1b3bff2a746c828beffd9e7c3e26.
//
// Solidity: event withdrawEvent(address indexed addr, uint256 amount)
func (_AccountBook *AccountBookFilterer) FilterWithdrawEvent(opts *bind.FilterOpts, addr []common.Address) (*AccountBookWithdrawEventIterator, error) {

	var addrRule []interface{}
	for _, addrItem := range addr {
		addrRule = append(addrRule, addrItem)
	}

	logs, sub, err := _AccountBook.contract.FilterLogs(opts, "withdrawEvent", addrRule)
	if err != nil {
		return nil, err
	}
	return &AccountBookWithdrawEventIterator{contract: _AccountBook.contract, event: "withdrawEvent", logs: logs, sub: sub}, nil
}

// WatchWithdrawEvent is a free log subscription operation binding the contract event 0x87d5f4772963d1f9b76047158b4ae97c420a1b3bff2a746c828beffd9e7c3e26.
//
// Solidity: event withdrawEvent(address indexed addr, uint256 amount)
func (_AccountBook *AccountBookFilterer) WatchWithdrawEvent(opts *bind.WatchOpts, sink chan<- *AccountBookWithdrawEvent, addr []common.Address) (event.Subscription, error) {

	var addrRule []interface{}
	for _, addrItem := range addr {
		addrRule = append(addrRule, addrItem)
	}

	logs, sub, err := _AccountBook.contract.WatchLogs(opts, "withdrawEvent", addrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountBookWithdrawEvent)
				if err := _AccountBook.contract.UnpackLog(event, "withdrawEvent", log); err != nil {
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

// ParseWithdrawEvent is a log parse operation binding the contract event 0x87d5f4772963d1f9b76047158b4ae97c420a1b3bff2a746c828beffd9e7c3e26.
//
// Solidity: event withdrawEvent(address indexed addr, uint256 amount)
func (_AccountBook *AccountBookFilterer) ParseWithdrawEvent(log types.Log) (*AccountBookWithdrawEvent, error) {
	event := new(AccountBookWithdrawEvent)
	if err := _AccountBook.contract.UnpackLog(event, "withdrawEvent", log); err != nil {
		return nil, err
	}
	return event, nil
}
