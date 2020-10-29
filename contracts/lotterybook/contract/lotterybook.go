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
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// LotteryBookABI is the input ABI used to generate the binding from.
const LotteryBookABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"lotteryClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"lotteryCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bytes4\",\"name\":\"revealRange\",\"type\":\"bytes4\"},{\"internalType\":\"uint8\",\"name\":\"sig_v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"sig_r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sig_s\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"receiverSalt\",\"type\":\"uint64\"},{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"}],\"name\":\"claim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"destroyLottery\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"lotteries\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"amount\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"revealNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"salt\",\"type\":\"uint64\"},{\"internalType\":\"addresspayable\",\"name\":\"owner\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"salt\",\"type\":\"uint64\"}],\"name\":\"newLottery\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"newid\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"newRevealNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"newSalt\",\"type\":\"uint64\"}],\"name\":\"resetLottery\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"visibleBlocks\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// LotteryBookBin is the compiled bytecode used for deploying new contracts.
var LotteryBookBin = "0x60806040523480156100115760006000fd5b50610017565b611be3806100266000396000f3fe6080604052600436106100745760003560e01c8063915c72c71161004e578063915c72c714610265578063a4f603d514610337578063ac209f2114610377578063add6cadb146103d257610074565b806338f7e2961461007a578063531809dc146100e357806354fd4d501461022557610074565b60006000fd5b6100e1600480360360808110156100915760006000fd5b8101908080356000191690602001909291908035600019169060200190929190803567ffffffffffffffff169060200190929190803567ffffffffffffffff169060200190929190505050610413565b005b3480156100f05760006000fd5b50610223600480360360e08110156101085760006000fd5b81019080803560001916906020019092919080357bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19169060200190929190803560ff16906020019092919080356000191690602001909291908035600019169060200190929190803567ffffffffffffffff1690602001909291908035906020019064010000000081111561019a5760006000fd5b8201836020820111156101ad5760006000fd5b803590602001918460208302840111640100000000831117156101d05760006000fd5b919080806020026020016040519081016040528093929190818152602001838360200280828437600081840152601f19601f82011690508083019250505050505050909091929090919290505050610b02565b005b3480156102325760006000fd5b5061023b6114d4565b604051808267ffffffffffffffff1667ffffffffffffffff16815260200191505060405180910390f35b3480156102725760006000fd5b506102a46004803603602081101561028a5760006000fd5b8101908080356000191690602001909291905050506114d9565b604051808567ffffffffffffffff1667ffffffffffffffff1681526020018467ffffffffffffffff1667ffffffffffffffff1681526020018367ffffffffffffffff1667ffffffffffffffff1681526020018273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200194505050505060405180910390f35b3480156103445760006000fd5b5061034d611568565b604051808267ffffffffffffffff1667ffffffffffffffff16815260200191505060405180910390f35b6103d06004803603606081101561038e5760006000fd5b810190808035600019169060200190929190803567ffffffffffffffff169060200190929190803567ffffffffffffffff16906020019092919050505061156e565b005b3480156103df5760006000fd5b50610411600480360360208110156103f75760006000fd5b81019080803560001916906020019092919050505061187b565b005b438267ffffffffffffffff16111515610497576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601f8152602001807f696e76616c6964206c6f7474657279207265736574206f7065726174696f6e0081526020015060200191505060405180910390fd5b600060006000506000866000191660001916815260200190815260200160002060005060000160089054906101000a900467ffffffffffffffff16905060008167ffffffffffffffff16141580156104fc575043610100820167ffffffffffffffff16105b1515610553576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526023815260200180611b5d6023913960400191505060405180910390fd5b3373ffffffffffffffffffffffffffffffffffffffff1660006000506000876000191660001916815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561061d576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526026815260200180611b376026913960400191505060405180910390fd5b600060006000506000866000191660001916815260200190815260200160002060005060000160089054906101000a900467ffffffffffffffff1667ffffffffffffffff161415156106da576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260128152602001807f6475706c696361746564206c6f7474657279000000000000000000000000000081526020015060200191505060405180910390fd5b8260006000506000866000191660001916815260200190815260200160002060005060000160086101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055508160006000506000866000191660001916815260200190815260200160002060005060000160106101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555060006000506000866000191660001916815260200190815260200160002060005060000160009054906101000a900467ffffffffffffffff1660006000506000866000191660001916815260200190815260200160002060005060000160006101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055503360006000506000866000191660001916815260200190815260200160002060005060010160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000341115610a0a57600060006000506000866000191660001916815260200190815260200160002060005060000160009054906101000a900467ffffffffffffffff1690508067ffffffffffffffff1634820167ffffffffffffffff1611151561092c576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260118152602001807f6164646974696f6e206f766572666c6f7700000000000000000000000000000081526020015060200191505060405180910390fd5b670de0b6b3a764000034820167ffffffffffffffff16111515156109bb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601f8152602001807f65786365656473206d6178696d756d206c6f7474657279206465706f7369740081526020015060200191505060405180910390fd5b34810160006000506000876000191660001916815260200190815260200160002060005060000160006101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550505b60006000506000866000191660001916815260200190815260200160002060006000820160006101000a81549067ffffffffffffffff02191690556000820160086101000a81549067ffffffffffffffff02191690556000820160106101000a81549067ffffffffffffffff02191690556001820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff021916905550503373ffffffffffffffffffffffffffffffffffffffff167f741e16afb90ee258a4466d76efeb566820839a26f04fe2a8ce4f4733d5dfefcf8560405180826000191660001916815260200191505060405180910390a2505b50505050565b600060006000506000896000191660001916815260200190815260200160002060005060000160089054906101000a900467ffffffffffffffff16905060008167ffffffffffffffff1614151515610bc5576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f6e6f6e2d6578697374656e74206c6f747465727900000000000000000000000081526020015060200191505060405180910390fd5b438167ffffffffffffffff16108015610bec575043610100820167ffffffffffffffff1610155b1515610c43576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252602e815260200180611b80602e913960400191505060405180910390fd5b60003384604051602001808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1660601b81526014018267ffffffffffffffff1667ffffffffffffffff1660c01b81526008019250505060405160208183030381529060405280519060200120905060006000600090505b84518160ff161015610dad576000858260ff16815181101515610ce457fe5b60200260200101519050806000191684600019161015610d465783816040516020018083600019166000191681526020018260001916600019168152602001925050506040516020818303038152906040528051906020012093508350610d9e565b8160ff16600160ff16901b60ff16830192508250808460405160200180836000191660001916815260200182600019166000191681526020019250505060405160208183030381529060405280519060200120935083505b505b8080600101915050610cc5565b5081600060005060008c6000191660001916815260200190815260200160002060005060000160109054906101000a900467ffffffffffffffff166040516020018083600019166000191681526020018267ffffffffffffffff1667ffffffffffffffff1660c01b815260080192505050604051602081830303815290604052805190602001209150815089600019168260001916141515610eba576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601d8152602001807f696e76616c696420706f736974696f6e206d65726b6c652070726f6f6600000081526020015060200191505060405180910390fd5b8860e01c63ffffffff168367ffffffffffffffff164060001c63ffffffff1611151515610f52576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f696e76616c69642077696e6e65722070726f6f6600000000000000000000000081526020015060200191505060405180910390fd5b8267ffffffffffffffff164060001c63ffffffff1681855164010000000067ffffffffffffffff16901c67ffffffffffffffff160211151515611000576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f696e76616c69642077696e6e65722070726f6f6600000000000000000000000081526020015060200191505060405180910390fd5b600060018201855164010000000067ffffffffffffffff16901c02905060008163ffffffff16148061104057508960e01c63ffffffff168163ffffffff16115b15156110b7576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f696e76616c69642077696e6e65722070726f6f6600000000000000000000000081526020015060200191505060405180910390fd5b6000601960f81b600060f81b308e8e60405160200180867effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168152600101857effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191681526001018473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1660601b81526014018360001916600019168152602001827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168152600401955050505050506040516020818303038152906040528051906020012090506001818b8b8b604051600081526020016040526040518085600019166000191681526020018460ff1660ff168152602001836000191660001916815260200182600019166000191681526020019450505050506020604051602081039080840390855afa15801561127d573d600060003e3d6000fd5b5050506020604051035173ffffffffffffffffffffffffffffffffffffffff16600060005060008e6000191660001916815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141515611370576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260118152602001807f696e76616c6964207369676e617475726500000000000000000000000000000081526020015060200191505060405180910390fd5b3373ffffffffffffffffffffffffffffffffffffffff166108fc600060005060008f6000191660001916815260200190815260200160002060005060000160009054906101000a900467ffffffffffffffff1667ffffffffffffffff169081150290604051600060405180830381858888f193505050501580156113f9573d600060003e3d6000fd5b50600060005060008d6000191660001916815260200190815260200160002060006000820160006101000a81549067ffffffffffffffff02191690556000820160086101000a81549067ffffffffffffffff02191690556000820160106101000a81549067ffffffffffffffff02191690556001820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff021916905550508b600019167f4c02162f394fb7efbecba1d186e234f1fe96b1f5f5b4fe67591b4b3e87c1881f60405160405180910390a250505050505b50505050505050565b600081565b60006000506020528060005260406000206000915090508060000160009054906101000a900467ffffffffffffffff16908060000160089054906101000a900467ffffffffffffffff16908060000160109054906101000a900467ffffffffffffffff16908060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905084565b61010081565b438267ffffffffffffffff161180156115875750600034115b801561159b5750670de0b6b3a76400003411155b1515611612576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f696e76616c6964206c6f74746572792073657474696e6773000000000000000081526020015060200191505060405180910390fd5b600060006000506000856000191660001916815260200190815260200160002060005060000160089054906101000a900467ffffffffffffffff1667ffffffffffffffff161415156116cf576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260128152602001807f6475706c696361746564206c6f7474657279000000000000000000000000000081526020015060200191505060405180910390fd5b60405180608001604052803467ffffffffffffffff1681526020018367ffffffffffffffff1681526020018267ffffffffffffffff1681526020013373ffffffffffffffffffffffffffffffffffffffff1681526020015060006000506000856000191660001916815260200190815260200160002060005060008201518160000160006101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555060208201518160000160086101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555060408201518160000160106101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555060608201518160010160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055509050503373ffffffffffffffffffffffffffffffffffffffff167f741e16afb90ee258a4466d76efeb566820839a26f04fe2a8ce4f4733d5dfefcf8460405180826000191660001916815260200191505060405180910390a25b505050565b600060006000506000836000191660001916815260200190815260200160002060005060000160089054906101000a900467ffffffffffffffff16905060008167ffffffffffffffff16141580156118e0575043610100820167ffffffffffffffff16105b1515611937576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526023815260200180611b5d6023913960400191505060405180910390fd5b600060006000506000846000191660001916815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690503373ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515611a06576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526026815260200180611b376026913960400191505060405180910390fd5b600060006000506000856000191660001916815260200190815260200160002060005060000160009054906101000a900467ffffffffffffffff16905060006000506000856000191660001916815260200190815260200160002060006000820160006101000a81549067ffffffffffffffff02191690556000820160086101000a81549067ffffffffffffffff02191690556000820160106101000a81549067ffffffffffffffff02191690556001820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff021916905550508173ffffffffffffffffffffffffffffffffffffffff166108fc8267ffffffffffffffff169081150290604051600060405180830381858888f19350505050158015611b2e573d600060003e3d6000fd5b505050505b5056fe6f6e6c79206f776e657220697320616c6c6f77656420746f207265736574206c6f74746572796e6f6e2d6578697374656e74206f72206e6f6e2d65787069726564206c6f74746572796c6f74746572792069736e277420636c61696d6561626c65206f72206974277320616c7265616479207374616c65a264697066735822122087a3698200877fde4c85eb0d4d6a9dcef3549cd77aebe18b9b20321a7488bb7e64736f6c63430006090033"

// DeployLotteryBook deploys a new Ethereum contract, binding an instance of LotteryBook to it.
func DeployLotteryBook(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LotteryBook, error) {
	parsed, err := abi.JSON(strings.NewReader(LotteryBookABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(LotteryBookBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LotteryBook{LotteryBookCaller: LotteryBookCaller{contract: contract}, LotteryBookTransactor: LotteryBookTransactor{contract: contract}, LotteryBookFilterer: LotteryBookFilterer{contract: contract}}, nil
}

// LotteryBook is an auto generated Go binding around an Ethereum contract.
type LotteryBook struct {
	LotteryBookCaller     // Read-only binding to the contract
	LotteryBookTransactor // Write-only binding to the contract
	LotteryBookFilterer   // Log filterer for contract events
}

// LotteryBookCaller is an auto generated read-only Go binding around an Ethereum contract.
type LotteryBookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LotteryBookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LotteryBookTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LotteryBookFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LotteryBookFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LotteryBookSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LotteryBookSession struct {
	Contract     *LotteryBook      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LotteryBookCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LotteryBookCallerSession struct {
	Contract *LotteryBookCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// LotteryBookTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LotteryBookTransactorSession struct {
	Contract     *LotteryBookTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// LotteryBookRaw is an auto generated low-level Go binding around an Ethereum contract.
type LotteryBookRaw struct {
	Contract *LotteryBook // Generic contract binding to access the raw methods on
}

// LotteryBookCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LotteryBookCallerRaw struct {
	Contract *LotteryBookCaller // Generic read-only contract binding to access the raw methods on
}

// LotteryBookTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LotteryBookTransactorRaw struct {
	Contract *LotteryBookTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLotteryBook creates a new instance of LotteryBook, bound to a specific deployed contract.
func NewLotteryBook(address common.Address, backend bind.ContractBackend) (*LotteryBook, error) {
	contract, err := bindLotteryBook(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LotteryBook{LotteryBookCaller: LotteryBookCaller{contract: contract}, LotteryBookTransactor: LotteryBookTransactor{contract: contract}, LotteryBookFilterer: LotteryBookFilterer{contract: contract}}, nil
}

// NewLotteryBookCaller creates a new read-only instance of LotteryBook, bound to a specific deployed contract.
func NewLotteryBookCaller(address common.Address, caller bind.ContractCaller) (*LotteryBookCaller, error) {
	contract, err := bindLotteryBook(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LotteryBookCaller{contract: contract}, nil
}

// NewLotteryBookTransactor creates a new write-only instance of LotteryBook, bound to a specific deployed contract.
func NewLotteryBookTransactor(address common.Address, transactor bind.ContractTransactor) (*LotteryBookTransactor, error) {
	contract, err := bindLotteryBook(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LotteryBookTransactor{contract: contract}, nil
}

// NewLotteryBookFilterer creates a new log filterer instance of LotteryBook, bound to a specific deployed contract.
func NewLotteryBookFilterer(address common.Address, filterer bind.ContractFilterer) (*LotteryBookFilterer, error) {
	contract, err := bindLotteryBook(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LotteryBookFilterer{contract: contract}, nil
}

// bindLotteryBook binds a generic wrapper to an already deployed contract.
func bindLotteryBook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LotteryBookABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LotteryBook *LotteryBookRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LotteryBook.Contract.LotteryBookCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LotteryBook *LotteryBookRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LotteryBook.Contract.LotteryBookTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LotteryBook *LotteryBookRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LotteryBook.Contract.LotteryBookTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LotteryBook *LotteryBookCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LotteryBook.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LotteryBook *LotteryBookTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LotteryBook.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LotteryBook *LotteryBookTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LotteryBook.Contract.contract.Transact(opts, method, params...)
}

// Lotteries is a free data retrieval call binding the contract method 0x915c72c7.
//
// Solidity: function lotteries(bytes32 ) view returns(uint64 amount, uint64 revealNumber, uint64 salt, address owner)
func (_LotteryBook *LotteryBookCaller) Lotteries(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Amount       uint64
	RevealNumber uint64
	Salt         uint64
	Owner        common.Address
}, error) {
	var out []interface{}
	err := _LotteryBook.contract.Call(opts, &out, "lotteries", arg0)

	outstruct := new(struct {
		Amount       uint64
		RevealNumber uint64
		Salt         uint64
		Owner        common.Address
	})

	outstruct.Amount = out[0].(uint64)
	outstruct.RevealNumber = out[1].(uint64)
	outstruct.Salt = out[2].(uint64)
	outstruct.Owner = out[3].(common.Address)

	return *outstruct, err

}

// Lotteries is a free data retrieval call binding the contract method 0x915c72c7.
//
// Solidity: function lotteries(bytes32 ) view returns(uint64 amount, uint64 revealNumber, uint64 salt, address owner)
func (_LotteryBook *LotteryBookSession) Lotteries(arg0 [32]byte) (struct {
	Amount       uint64
	RevealNumber uint64
	Salt         uint64
	Owner        common.Address
}, error) {
	return _LotteryBook.Contract.Lotteries(&_LotteryBook.CallOpts, arg0)
}

// Lotteries is a free data retrieval call binding the contract method 0x915c72c7.
//
// Solidity: function lotteries(bytes32 ) view returns(uint64 amount, uint64 revealNumber, uint64 salt, address owner)
func (_LotteryBook *LotteryBookCallerSession) Lotteries(arg0 [32]byte) (struct {
	Amount       uint64
	RevealNumber uint64
	Salt         uint64
	Owner        common.Address
}, error) {
	return _LotteryBook.Contract.Lotteries(&_LotteryBook.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint64)
func (_LotteryBook *LotteryBookCaller) Version(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _LotteryBook.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint64)
func (_LotteryBook *LotteryBookSession) Version() (uint64, error) {
	return _LotteryBook.Contract.Version(&_LotteryBook.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint64)
func (_LotteryBook *LotteryBookCallerSession) Version() (uint64, error) {
	return _LotteryBook.Contract.Version(&_LotteryBook.CallOpts)
}

// VisibleBlocks is a free data retrieval call binding the contract method 0xa4f603d5.
//
// Solidity: function visibleBlocks() view returns(uint64)
func (_LotteryBook *LotteryBookCaller) VisibleBlocks(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _LotteryBook.contract.Call(opts, &out, "visibleBlocks")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// VisibleBlocks is a free data retrieval call binding the contract method 0xa4f603d5.
//
// Solidity: function visibleBlocks() view returns(uint64)
func (_LotteryBook *LotteryBookSession) VisibleBlocks() (uint64, error) {
	return _LotteryBook.Contract.VisibleBlocks(&_LotteryBook.CallOpts)
}

// VisibleBlocks is a free data retrieval call binding the contract method 0xa4f603d5.
//
// Solidity: function visibleBlocks() view returns(uint64)
func (_LotteryBook *LotteryBookCallerSession) VisibleBlocks() (uint64, error) {
	return _LotteryBook.Contract.VisibleBlocks(&_LotteryBook.CallOpts)
}

// Claim is a paid mutator transaction binding the contract method 0x531809dc.
//
// Solidity: function claim(bytes32 id, bytes4 revealRange, uint8 sig_v, bytes32 sig_r, bytes32 sig_s, uint64 receiverSalt, bytes32[] proof) returns()
func (_LotteryBook *LotteryBookTransactor) Claim(opts *bind.TransactOpts, id [32]byte, revealRange [4]byte, sig_v uint8, sig_r [32]byte, sig_s [32]byte, receiverSalt uint64, proof [][32]byte) (*types.Transaction, error) {
	return _LotteryBook.contract.Transact(opts, "claim", id, revealRange, sig_v, sig_r, sig_s, receiverSalt, proof)
}

// Claim is a paid mutator transaction binding the contract method 0x531809dc.
//
// Solidity: function claim(bytes32 id, bytes4 revealRange, uint8 sig_v, bytes32 sig_r, bytes32 sig_s, uint64 receiverSalt, bytes32[] proof) returns()
func (_LotteryBook *LotteryBookSession) Claim(id [32]byte, revealRange [4]byte, sig_v uint8, sig_r [32]byte, sig_s [32]byte, receiverSalt uint64, proof [][32]byte) (*types.Transaction, error) {
	return _LotteryBook.Contract.Claim(&_LotteryBook.TransactOpts, id, revealRange, sig_v, sig_r, sig_s, receiverSalt, proof)
}

// Claim is a paid mutator transaction binding the contract method 0x531809dc.
//
// Solidity: function claim(bytes32 id, bytes4 revealRange, uint8 sig_v, bytes32 sig_r, bytes32 sig_s, uint64 receiverSalt, bytes32[] proof) returns()
func (_LotteryBook *LotteryBookTransactorSession) Claim(id [32]byte, revealRange [4]byte, sig_v uint8, sig_r [32]byte, sig_s [32]byte, receiverSalt uint64, proof [][32]byte) (*types.Transaction, error) {
	return _LotteryBook.Contract.Claim(&_LotteryBook.TransactOpts, id, revealRange, sig_v, sig_r, sig_s, receiverSalt, proof)
}

// DestroyLottery is a paid mutator transaction binding the contract method 0xadd6cadb.
//
// Solidity: function destroyLottery(bytes32 id) returns()
func (_LotteryBook *LotteryBookTransactor) DestroyLottery(opts *bind.TransactOpts, id [32]byte) (*types.Transaction, error) {
	return _LotteryBook.contract.Transact(opts, "destroyLottery", id)
}

// DestroyLottery is a paid mutator transaction binding the contract method 0xadd6cadb.
//
// Solidity: function destroyLottery(bytes32 id) returns()
func (_LotteryBook *LotteryBookSession) DestroyLottery(id [32]byte) (*types.Transaction, error) {
	return _LotteryBook.Contract.DestroyLottery(&_LotteryBook.TransactOpts, id)
}

// DestroyLottery is a paid mutator transaction binding the contract method 0xadd6cadb.
//
// Solidity: function destroyLottery(bytes32 id) returns()
func (_LotteryBook *LotteryBookTransactorSession) DestroyLottery(id [32]byte) (*types.Transaction, error) {
	return _LotteryBook.Contract.DestroyLottery(&_LotteryBook.TransactOpts, id)
}

// NewLottery is a paid mutator transaction binding the contract method 0xac209f21.
//
// Solidity: function newLottery(bytes32 id, uint64 blockNumber, uint64 salt) payable returns()
func (_LotteryBook *LotteryBookTransactor) NewLottery(opts *bind.TransactOpts, id [32]byte, blockNumber uint64, salt uint64) (*types.Transaction, error) {
	return _LotteryBook.contract.Transact(opts, "newLottery", id, blockNumber, salt)
}

// NewLottery is a paid mutator transaction binding the contract method 0xac209f21.
//
// Solidity: function newLottery(bytes32 id, uint64 blockNumber, uint64 salt) payable returns()
func (_LotteryBook *LotteryBookSession) NewLottery(id [32]byte, blockNumber uint64, salt uint64) (*types.Transaction, error) {
	return _LotteryBook.Contract.NewLottery(&_LotteryBook.TransactOpts, id, blockNumber, salt)
}

// NewLottery is a paid mutator transaction binding the contract method 0xac209f21.
//
// Solidity: function newLottery(bytes32 id, uint64 blockNumber, uint64 salt) payable returns()
func (_LotteryBook *LotteryBookTransactorSession) NewLottery(id [32]byte, blockNumber uint64, salt uint64) (*types.Transaction, error) {
	return _LotteryBook.Contract.NewLottery(&_LotteryBook.TransactOpts, id, blockNumber, salt)
}

// ResetLottery is a paid mutator transaction binding the contract method 0x38f7e296.
//
// Solidity: function resetLottery(bytes32 id, bytes32 newid, uint64 newRevealNumber, uint64 newSalt) payable returns()
func (_LotteryBook *LotteryBookTransactor) ResetLottery(opts *bind.TransactOpts, id [32]byte, newid [32]byte, newRevealNumber uint64, newSalt uint64) (*types.Transaction, error) {
	return _LotteryBook.contract.Transact(opts, "resetLottery", id, newid, newRevealNumber, newSalt)
}

// ResetLottery is a paid mutator transaction binding the contract method 0x38f7e296.
//
// Solidity: function resetLottery(bytes32 id, bytes32 newid, uint64 newRevealNumber, uint64 newSalt) payable returns()
func (_LotteryBook *LotteryBookSession) ResetLottery(id [32]byte, newid [32]byte, newRevealNumber uint64, newSalt uint64) (*types.Transaction, error) {
	return _LotteryBook.Contract.ResetLottery(&_LotteryBook.TransactOpts, id, newid, newRevealNumber, newSalt)
}

// ResetLottery is a paid mutator transaction binding the contract method 0x38f7e296.
//
// Solidity: function resetLottery(bytes32 id, bytes32 newid, uint64 newRevealNumber, uint64 newSalt) payable returns()
func (_LotteryBook *LotteryBookTransactorSession) ResetLottery(id [32]byte, newid [32]byte, newRevealNumber uint64, newSalt uint64) (*types.Transaction, error) {
	return _LotteryBook.Contract.ResetLottery(&_LotteryBook.TransactOpts, id, newid, newRevealNumber, newSalt)
}

// LotteryBookLotteryClaimedIterator is returned from FilterLotteryClaimed and is used to iterate over the raw logs and unpacked data for LotteryClaimed events raised by the LotteryBook contract.
type LotteryBookLotteryClaimedIterator struct {
	Event *LotteryBookLotteryClaimed // Event containing the contract specifics and raw log

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
func (it *LotteryBookLotteryClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LotteryBookLotteryClaimed)
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
		it.Event = new(LotteryBookLotteryClaimed)
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
func (it *LotteryBookLotteryClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LotteryBookLotteryClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LotteryBookLotteryClaimed represents a LotteryClaimed event raised by the LotteryBook contract.
type LotteryBookLotteryClaimed struct {
	Id  [32]byte
	Raw types.Log // Blockchain specific contextual infos
}

// FilterLotteryClaimed is a free log retrieval operation binding the contract event 0x4c02162f394fb7efbecba1d186e234f1fe96b1f5f5b4fe67591b4b3e87c1881f.
//
// Solidity: event lotteryClaimed(bytes32 indexed id)
func (_LotteryBook *LotteryBookFilterer) FilterLotteryClaimed(opts *bind.FilterOpts, id [][32]byte) (*LotteryBookLotteryClaimedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LotteryBook.contract.FilterLogs(opts, "lotteryClaimed", idRule)
	if err != nil {
		return nil, err
	}
	return &LotteryBookLotteryClaimedIterator{contract: _LotteryBook.contract, event: "lotteryClaimed", logs: logs, sub: sub}, nil
}

// WatchLotteryClaimed is a free log subscription operation binding the contract event 0x4c02162f394fb7efbecba1d186e234f1fe96b1f5f5b4fe67591b4b3e87c1881f.
//
// Solidity: event lotteryClaimed(bytes32 indexed id)
func (_LotteryBook *LotteryBookFilterer) WatchLotteryClaimed(opts *bind.WatchOpts, sink chan<- *LotteryBookLotteryClaimed, id [][32]byte) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LotteryBook.contract.WatchLogs(opts, "lotteryClaimed", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LotteryBookLotteryClaimed)
				if err := _LotteryBook.contract.UnpackLog(event, "lotteryClaimed", log); err != nil {
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

// ParseLotteryClaimed is a log parse operation binding the contract event 0x4c02162f394fb7efbecba1d186e234f1fe96b1f5f5b4fe67591b4b3e87c1881f.
//
// Solidity: event lotteryClaimed(bytes32 indexed id)
func (_LotteryBook *LotteryBookFilterer) ParseLotteryClaimed(log types.Log) (*LotteryBookLotteryClaimed, error) {
	event := new(LotteryBookLotteryClaimed)
	if err := _LotteryBook.contract.UnpackLog(event, "lotteryClaimed", log); err != nil {
		return nil, err
	}
	return event, nil
}

// LotteryBookLotteryCreatedIterator is returned from FilterLotteryCreated and is used to iterate over the raw logs and unpacked data for LotteryCreated events raised by the LotteryBook contract.
type LotteryBookLotteryCreatedIterator struct {
	Event *LotteryBookLotteryCreated // Event containing the contract specifics and raw log

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
func (it *LotteryBookLotteryCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LotteryBookLotteryCreated)
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
		it.Event = new(LotteryBookLotteryCreated)
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
func (it *LotteryBookLotteryCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LotteryBookLotteryCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LotteryBookLotteryCreated represents a LotteryCreated event raised by the LotteryBook contract.
type LotteryBookLotteryCreated struct {
	Creator common.Address
	Id      [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLotteryCreated is a free log retrieval operation binding the contract event 0x741e16afb90ee258a4466d76efeb566820839a26f04fe2a8ce4f4733d5dfefcf.
//
// Solidity: event lotteryCreated(address indexed creator, bytes32 id)
func (_LotteryBook *LotteryBookFilterer) FilterLotteryCreated(opts *bind.FilterOpts, creator []common.Address) (*LotteryBookLotteryCreatedIterator, error) {

	var creatorRule []interface{}
	for _, creatorItem := range creator {
		creatorRule = append(creatorRule, creatorItem)
	}

	logs, sub, err := _LotteryBook.contract.FilterLogs(opts, "lotteryCreated", creatorRule)
	if err != nil {
		return nil, err
	}
	return &LotteryBookLotteryCreatedIterator{contract: _LotteryBook.contract, event: "lotteryCreated", logs: logs, sub: sub}, nil
}

// WatchLotteryCreated is a free log subscription operation binding the contract event 0x741e16afb90ee258a4466d76efeb566820839a26f04fe2a8ce4f4733d5dfefcf.
//
// Solidity: event lotteryCreated(address indexed creator, bytes32 id)
func (_LotteryBook *LotteryBookFilterer) WatchLotteryCreated(opts *bind.WatchOpts, sink chan<- *LotteryBookLotteryCreated, creator []common.Address) (event.Subscription, error) {

	var creatorRule []interface{}
	for _, creatorItem := range creator {
		creatorRule = append(creatorRule, creatorItem)
	}

	logs, sub, err := _LotteryBook.contract.WatchLogs(opts, "lotteryCreated", creatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LotteryBookLotteryCreated)
				if err := _LotteryBook.contract.UnpackLog(event, "lotteryCreated", log); err != nil {
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

// ParseLotteryCreated is a log parse operation binding the contract event 0x741e16afb90ee258a4466d76efeb566820839a26f04fe2a8ce4f4733d5dfefcf.
//
// Solidity: event lotteryCreated(address indexed creator, bytes32 id)
func (_LotteryBook *LotteryBookFilterer) ParseLotteryCreated(log types.Log) (*LotteryBookLotteryCreated, error) {
	event := new(LotteryBookLotteryCreated)
	if err := _LotteryBook.contract.UnpackLog(event, "lotteryCreated", log); err != nil {
		return nil, err
	}
	return event, nil
}
