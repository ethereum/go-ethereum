// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package release

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ReleaseOracleABI is the input ABI used to generate the binding from.
const ReleaseOracleABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"proposedVersion\",\"outputs\":[{\"name\":\"major\",\"type\":\"uint32\"},{\"name\":\"minor\",\"type\":\"uint32\"},{\"name\":\"patch\",\"type\":\"uint32\"},{\"name\":\"commit\",\"type\":\"bytes20\"},{\"name\":\"pass\",\"type\":\"address[]\"},{\"name\":\"fail\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"signers\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"user\",\"type\":\"address\"}],\"name\":\"demote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"user\",\"type\":\"address\"}],\"name\":\"authVotes\",\"outputs\":[{\"name\":\"promote\",\"type\":\"address[]\"},{\"name\":\"demote\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"currentVersion\",\"outputs\":[{\"name\":\"major\",\"type\":\"uint32\"},{\"name\":\"minor\",\"type\":\"uint32\"},{\"name\":\"patch\",\"type\":\"uint32\"},{\"name\":\"commit\",\"type\":\"bytes20\"},{\"name\":\"time\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"nuke\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"authProposals\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"user\",\"type\":\"address\"}],\"name\":\"promote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"major\",\"type\":\"uint32\"},{\"name\":\"minor\",\"type\":\"uint32\"},{\"name\":\"patch\",\"type\":\"uint32\"},{\"name\":\"commit\",\"type\":\"bytes20\"}],\"name\":\"release\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"signers\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// ReleaseOracleBin is the compiled bytecode used for deploying new contracts.
const ReleaseOracleBin = `0x606060405234156200001057600080fd5b60405162001395380380620013958339810160405280805190910190506000815115156200009a57600160a060020a0333166000908152602081905260409020805460ff1916600190811790915580548082016200006f838262000157565b5060009182526020909120018054600160a060020a03191633600160a060020a03161790556200014f565b5060005b81518110156200014f576001600080848481518110620000ba57fe5b90602001906020020151600160a060020a031681526020810191909152604001600020805460ff191691151591909117905560018054808201620000ff838262000157565b916000526020600020900160008484815181106200011957fe5b906020019060200201518254600160a060020a039182166101009390930a9283029190920219909116179055506001016200009e565b5050620001a7565b8154818355818115116200017e576000838152602090206200017e91810190830162000183565b505050565b620001a491905b80821115620001a057600081556001016200018a565b5090565b90565b6111de80620001b76000396000f3006060604052600436106100985763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166326db7648811461009d57806346f0975a1461017e5780635c3d005d146101e457806364ed31fe146102055780639d888e86146102bd578063bc8fbbf814610318578063bf8ecf9c1461032b578063d0e0813a1461033e578063d67cbec91461035d575b600080fd5b34156100a857600080fd5b6100b0610397565b60405163ffffffff80881682528681166020830152851660408201526bffffffffffffffffffffffff198416606082015260c0608082018181529060a0830190830185818151815260200191508051906020019060200280838360005b8381101561012557808201518382015260200161010d565b50505050905001838103825284818151815260200191508051906020019060200280838360005b8381101561016457808201518382015260200161014c565b505050509050019850505050505050505060405180910390f35b341561018957600080fd5b6101916104bc565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156101d05780820151838201526020016101b8565b505050509050019250505060405180910390f35b34156101ef57600080fd5b610203600160a060020a0360043516610525565b005b341561021057600080fd5b610224600160a060020a0360043516610533565b604051808060200180602001838103835285818151815260200191508051906020019060200280838360005b83811015610268578082015183820152602001610250565b50505050905001838103825284818151815260200191508051906020019060200280838360005b838110156102a757808201518382015260200161028f565b5050505090500194505050505060405180910390f35b34156102c857600080fd5b6102d0610627565b60405163ffffffff95861681529385166020850152919093166040808401919091526bffffffffffffffffffffffff199093166060830152608082015260a001905180910390f35b341561032357600080fd5b6102036106cf565b341561033657600080fd5b6101916106df565b341561034957600080fd5b610203600160a060020a0360043516610745565b341561036857600080fd5b61020363ffffffff600435811690602435811690604435166bffffffffffffffffffffffff1960643516610750565b6000806000806103a5611051565b6103ad611051565b6004546006805463ffffffff808416936401000000008104821693680100000000000000008204909216926c01000000000000000000000000918290049091029190600790829060208082020160405190810160405280929190818152602001828054801561044557602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610427575b50505050509150808054806020026020016040519081016040528092919081815260200182805480156104a157602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610483575b50505050509050955095509550955095509550909192939495565b6104c4611051565b600180548060200260200160405190810160405280929190818152602001828054801561051a57602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116104fc575b505050505090505b90565b610530816000610764565b50565b61053b611051565b610543611051565b600160a060020a03831660009081526002602090815260409182902080549092600184019284929182820290910190519081016040528092919081815260200182805480156105bb57602002820191906000526020600020905b8154600160a060020a0316815260019091019060200180831161059d575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561061757602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116105f9575b5050505050905091509150915091565b6000806000806000806008805490506000141561065357600095508594508493508392508291506106c7565b60088054600019810190811061066557fe5b600091825260209091206004909102018054600182015463ffffffff80831699506401000000008304811698506801000000000000000083041696506c0100000000000000000000000091829004909102945067ffffffffffffffff16925090505b509091929394565b6106dd600080808080610c01565b565b6106e7611051565b600380548060200260200160405190810160405280929190818152602001828054801561051a57602002820191906000526020600020908154600160a060020a031681526001909101906020018083116104fc575050505050905090565b610530816001610764565b61075e848484846001610c01565b50505050565b600160a060020a033316600090815260208190526040812054819060ff161561075e575050600160a060020a0382166000908152600260205260408120905b81548110156107ed578154600160a060020a033316908390839081106107c557fe5b600091825260209091200154600160a060020a031614156107e55761075e565b6001016107a3565b5060005b60018201548110156108405733600160a060020a0316826001018281548110151561081857fe5b600091825260209091200154600160a060020a031614156108385761075e565b6001016107f1565b815415801561085157506001820154155b1561088e5760038054600181016108688382611063565b5060009182526020909120018054600160a060020a031916600160a060020a0386161790555b82156108e65781548290600181016108a68382611063565b5060009182526020909120018054600160a060020a03191633600160a060020a0316179055600154600290835491900490116108e15761075e565b61093a565b8160010180548060010182816108fc9190611063565b5060009182526020909120018054600160a060020a03191633600160a060020a03161790556001546002906001840154919004901161093a5761075e565b8280156109605750600160a060020a03841660009081526020819052604090205460ff16155b156109c457600160a060020a0384166000908152602081905260409020805460ff19166001908117909155805480820161099a8382611063565b5060009182526020909120018054600160a060020a031916600160a060020a038616179055610b09565b821580156109ea5750600160a060020a03841660009081526020819052604090205460ff165b15610b095750600160a060020a0383166000908152602081905260408120805460ff191690555b600154811015610b095783600160a060020a0316600182815481101515610a3457fe5b600091825260209091200154600160a060020a03161415610b0157600180546000198101908110610a6157fe5b60009182526020909120015460018054600160a060020a039092169183908110610a8757fe5b60009182526020909120018054600160a060020a031916600160a060020a03929092169190911790556001805490610ac3906000198301611063565b50600060048181556005805467ffffffffffffffff1916905590600681610aea828261108c565b610af860018301600061108c565b50505050610b09565b600101610a11565b600160a060020a038416600090815260026020526040812090610b2c828261108c565b610b3a60018301600061108c565b5050600090505b60035481101561075e5783600160a060020a0316600382815481101515610b6457fe5b600091825260209091200154600160a060020a03161415610bf957600380546000198101908110610b9157fe5b60009182526020909120015460038054600160a060020a039092169183908110610bb757fe5b60009182526020909120018054600160a060020a031916600160a060020a03929092169190911790556003805490610bf3906000198301611063565b5061075e565b600101610b41565b600160a060020a033316600090815260208190526040812054819060ff16156110485782158015610c325750600654155b15610c3c57611048565b6006541515610cb7576004805463ffffffff191663ffffffff8981169190911767ffffffff00000000191664010000000089831602176bffffffff000000000000000019166801000000000000000091881691909102176bffffffffffffffffffffffff166c01000000000000000000000000808704021790555b828015610d41575060045463ffffffff8881169116141580610cec575060045463ffffffff8781166401000000009092041614155b80610d0e575060045463ffffffff868116680100000000000000009092041614155b80610d4157506004546c0100000000000000000000000090819004026bffffffffffffffffffffffff1990811690851614155b15610d4b57611048565b506006905060005b8154811015610d9d578154600160a060020a03331690839083908110610d7557fe5b600091825260209091200154600160a060020a03161415610d9557611048565b600101610d53565b5060005b6001820154811015610df05733600160a060020a03168260010182815481101515610dc857fe5b600091825260209091200154600160a060020a03161415610de857611048565b600101610da1565b8215610e48578154829060018101610e088382611063565b5060009182526020909120018054600160a060020a03191633600160a060020a031617905560015460029083549190049011610e4357611048565b610e9c565b816001018054806001018281610e5e9190611063565b5060009182526020909120018054600160a060020a03191633600160a060020a031617905560015460029060018401549190049011610e9c57611048565b821561100f576005805467ffffffffffffffff19164267ffffffffffffffff161790556008805460018101610ed183826110aa565b6000928352602090922060048054928102909101805463ffffffff191663ffffffff9384161780825582546401000000009081900485160267ffffffff000000001990911617808255825468010000000000000000908190049094169093026bffffffff0000000000000000199093169290921780835581546c01000000000000000000000000908190048102819004026bffffffffffffffffffffffff90911617825560055460018301805467ffffffffffffffff191667ffffffffffffffff909216919091179055600680549192916002830190610fb490829084906110d6565b5060018281018054610fc992840191906110d6565b5050600060048181556005805467ffffffffffffffff191690559450925060069150829050610ff8828261108c565b61100660018301600061108c565b50505050611048565b600060048181556005805467ffffffffffffffff1916905590600681611035828261108c565b61104360018301600061108c565b505050505b50505050505050565b60206040519081016040526000815290565b81548183558181151161108757600083815260209020611087918101908301611126565b505050565b50805460008255906000526020600020908101906105309190611126565b815481835581811511611087576004028160040283600052602060002091820191016110879190611140565b8280548282559060005260206000209081019282156111165760005260206000209182015b828111156111165782548255916001019190600101906110fb565b5061112292915061118e565b5090565b61052291905b80821115611122576000815560010161112c565b61052291905b8082111561112257600080825560018201805467ffffffffffffffff191690556002820181611175828261108c565b61118360018301600061108c565b505050600401611146565b61052291905b80821115611122578054600160a060020a03191681556001016111945600a165627a7a72305820aa9c89b9c569e44fa08285a00ab47bcdf0a01ebbc54e3cd864450622b5e559a40029`

// DeployReleaseOracle deploys a new Ethereum contract, binding an instance of ReleaseOracle to it.
func DeployReleaseOracle(auth *bind.TransactOpts, backend bind.ContractBackend, signers []common.Address) (common.Address, *types.Transaction, *ReleaseOracle, error) {
	parsed, err := abi.JSON(strings.NewReader(ReleaseOracleABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ReleaseOracleBin), backend, signers)
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
	contract, err := bindReleaseOracle(address, backend, backend)
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

// AuthProposals is a free data retrieval call binding the contract method 0xbf8ecf9c.
//
// Solidity: function authProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCaller) AuthProposals(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _ReleaseOracle.contract.Call(opts, out, "authProposals")
	return *ret0, err
}

// AuthProposals is a free data retrieval call binding the contract method 0xbf8ecf9c.
//
// Solidity: function authProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleSession) AuthProposals() ([]common.Address, error) {
	return _ReleaseOracle.Contract.AuthProposals(&_ReleaseOracle.CallOpts)
}

// AuthProposals is a free data retrieval call binding the contract method 0xbf8ecf9c.
//
// Solidity: function authProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) AuthProposals() ([]common.Address, error) {
	return _ReleaseOracle.Contract.AuthProposals(&_ReleaseOracle.CallOpts)
}

// AuthVotes is a free data retrieval call binding the contract method 0x64ed31fe.
//
// Solidity: function authVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleCaller) AuthVotes(opts *bind.CallOpts, user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	ret := new(struct {
		Promote []common.Address
		Demote  []common.Address
	})
	out := ret
	err := _ReleaseOracle.contract.Call(opts, out, "authVotes", user)
	return *ret, err
}

// AuthVotes is a free data retrieval call binding the contract method 0x64ed31fe.
//
// Solidity: function authVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleSession) AuthVotes(user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	return _ReleaseOracle.Contract.AuthVotes(&_ReleaseOracle.CallOpts, user)
}

// AuthVotes is a free data retrieval call binding the contract method 0x64ed31fe.
//
// Solidity: function authVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) AuthVotes(user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	return _ReleaseOracle.Contract.AuthVotes(&_ReleaseOracle.CallOpts, user)
}

// CurrentVersion is a free data retrieval call binding the contract method 0x9d888e86.
//
// Solidity: function currentVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, time uint256)
func (_ReleaseOracle *ReleaseOracleCaller) CurrentVersion(opts *bind.CallOpts) (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Time   *big.Int
}, error) {
	ret := new(struct {
		Major  uint32
		Minor  uint32
		Patch  uint32
		Commit [20]byte
		Time   *big.Int
	})
	out := ret
	err := _ReleaseOracle.contract.Call(opts, out, "currentVersion")
	return *ret, err
}

// CurrentVersion is a free data retrieval call binding the contract method 0x9d888e86.
//
// Solidity: function currentVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, time uint256)
func (_ReleaseOracle *ReleaseOracleSession) CurrentVersion() (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Time   *big.Int
}, error) {
	return _ReleaseOracle.Contract.CurrentVersion(&_ReleaseOracle.CallOpts)
}

// CurrentVersion is a free data retrieval call binding the contract method 0x9d888e86.
//
// Solidity: function currentVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, time uint256)
func (_ReleaseOracle *ReleaseOracleCallerSession) CurrentVersion() (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Time   *big.Int
}, error) {
	return _ReleaseOracle.Contract.CurrentVersion(&_ReleaseOracle.CallOpts)
}

// ProposedVersion is a free data retrieval call binding the contract method 0x26db7648.
//
// Solidity: function proposedVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, pass address[], fail address[])
func (_ReleaseOracle *ReleaseOracleCaller) ProposedVersion(opts *bind.CallOpts) (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Pass   []common.Address
	Fail   []common.Address
}, error) {
	ret := new(struct {
		Major  uint32
		Minor  uint32
		Patch  uint32
		Commit [20]byte
		Pass   []common.Address
		Fail   []common.Address
	})
	out := ret
	err := _ReleaseOracle.contract.Call(opts, out, "proposedVersion")
	return *ret, err
}

// ProposedVersion is a free data retrieval call binding the contract method 0x26db7648.
//
// Solidity: function proposedVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, pass address[], fail address[])
func (_ReleaseOracle *ReleaseOracleSession) ProposedVersion() (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Pass   []common.Address
	Fail   []common.Address
}, error) {
	return _ReleaseOracle.Contract.ProposedVersion(&_ReleaseOracle.CallOpts)
}

// ProposedVersion is a free data retrieval call binding the contract method 0x26db7648.
//
// Solidity: function proposedVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, pass address[], fail address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) ProposedVersion() (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Pass   []common.Address
	Fail   []common.Address
}, error) {
	return _ReleaseOracle.Contract.ProposedVersion(&_ReleaseOracle.CallOpts)
}

// Signers is a free data retrieval call binding the contract method 0x46f0975a.
//
// Solidity: function signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCaller) Signers(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _ReleaseOracle.contract.Call(opts, out, "signers")
	return *ret0, err
}

// Signers is a free data retrieval call binding the contract method 0x46f0975a.
//
// Solidity: function signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleSession) Signers() ([]common.Address, error) {
	return _ReleaseOracle.Contract.Signers(&_ReleaseOracle.CallOpts)
}

// Signers is a free data retrieval call binding the contract method 0x46f0975a.
//
// Solidity: function signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) Signers() ([]common.Address, error) {
	return _ReleaseOracle.Contract.Signers(&_ReleaseOracle.CallOpts)
}

// Demote is a paid mutator transaction binding the contract method 0x5c3d005d.
//
// Solidity: function demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Demote(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "demote", user)
}

// Demote is a paid mutator transaction binding the contract method 0x5c3d005d.
//
// Solidity: function demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleSession) Demote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Demote(&_ReleaseOracle.TransactOpts, user)
}

// Demote is a paid mutator transaction binding the contract method 0x5c3d005d.
//
// Solidity: function demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Demote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Demote(&_ReleaseOracle.TransactOpts, user)
}

// Nuke is a paid mutator transaction binding the contract method 0xbc8fbbf8.
//
// Solidity: function nuke() returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Nuke(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "nuke")
}

// Nuke is a paid mutator transaction binding the contract method 0xbc8fbbf8.
//
// Solidity: function nuke() returns()
func (_ReleaseOracle *ReleaseOracleSession) Nuke() (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Nuke(&_ReleaseOracle.TransactOpts)
}

// Nuke is a paid mutator transaction binding the contract method 0xbc8fbbf8.
//
// Solidity: function nuke() returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Nuke() (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Nuke(&_ReleaseOracle.TransactOpts)
}

// Promote is a paid mutator transaction binding the contract method 0xd0e0813a.
//
// Solidity: function promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Promote(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "promote", user)
}

// Promote is a paid mutator transaction binding the contract method 0xd0e0813a.
//
// Solidity: function promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleSession) Promote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Promote(&_ReleaseOracle.TransactOpts, user)
}

// Promote is a paid mutator transaction binding the contract method 0xd0e0813a.
//
// Solidity: function promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Promote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Promote(&_ReleaseOracle.TransactOpts, user)
}

// Release is a paid mutator transaction binding the contract method 0xd67cbec9.
//
// Solidity: function release(major uint32, minor uint32, patch uint32, commit bytes20) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Release(opts *bind.TransactOpts, major uint32, minor uint32, patch uint32, commit [20]byte) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "release", major, minor, patch, commit)
}

// Release is a paid mutator transaction binding the contract method 0xd67cbec9.
//
// Solidity: function release(major uint32, minor uint32, patch uint32, commit bytes20) returns()
func (_ReleaseOracle *ReleaseOracleSession) Release(major uint32, minor uint32, patch uint32, commit [20]byte) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Release(&_ReleaseOracle.TransactOpts, major, minor, patch, commit)
}

// Release is a paid mutator transaction binding the contract method 0xd67cbec9.
//
// Solidity: function release(major uint32, minor uint32, patch uint32, commit bytes20) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Release(major uint32, minor uint32, patch uint32, commit [20]byte) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Release(&_ReleaseOracle.TransactOpts, major, minor, patch, commit)
}
