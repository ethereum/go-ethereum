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
const ReleaseOracleBin = `0x606060405234156200001057600080fd5b60405162001393380380620013938339810160405280805190910190506000815115156200009a57600160a060020a0333166000908152602081905260409020805460ff1916600190811790915580548082016200006f838262000157565b5060009182526020909120018054600160a060020a03191633600160a060020a03161790556200014f565b5060005b81518110156200014f576001600080848481518110620000ba57fe5b90602001906020020151600160a060020a031681526020810191909152604001600020805460ff191691151591909117905560018054808201620000ff838262000157565b916000526020600020900160008484815181106200011957fe5b906020019060200201518254600160a060020a039182166101009390930a9283029190920219909116179055506001016200009e565b5050620001a7565b8154818355818115116200017e576000838152602090206200017e91810190830162000183565b505050565b620001a491905b80821115620001a057600081556001016200018a565b5090565b90565b6111dc80620001b76000396000f300606060405236156100965763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166326db7648811461009b57806346f0975a1461017c5780635c3d005d146101e257806364ed31fe146102035780639d888e86146102bb578063bc8fbbf814610316578063bf8ecf9c14610329578063d0e0813a1461033c578063d67cbec91461035b575b600080fd5b34156100a657600080fd5b6100ae610395565b60405163ffffffff80881682528681166020830152851660408201526bffffffffffffffffffffffff198416606082015260c0608082018181529060a0830190830185818151815260200191508051906020019060200280838360005b8381101561012357808201518382015260200161010b565b50505050905001838103825284818151815260200191508051906020019060200280838360005b8381101561016257808201518382015260200161014a565b505050509050019850505050505050505060405180910390f35b341561018757600080fd5b61018f6104ba565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156101ce5780820151838201526020016101b6565b505050509050019250505060405180910390f35b34156101ed57600080fd5b610201600160a060020a0360043516610523565b005b341561020e57600080fd5b610222600160a060020a0360043516610531565b604051808060200180602001838103835285818151815260200191508051906020019060200280838360005b8381101561026657808201518382015260200161024e565b50505050905001838103825284818151815260200191508051906020019060200280838360005b838110156102a557808201518382015260200161028d565b5050505090500194505050505060405180910390f35b34156102c657600080fd5b6102ce610625565b60405163ffffffff95861681529385166020850152919093166040808401919091526bffffffffffffffffffffffff199093166060830152608082015260a001905180910390f35b341561032157600080fd5b6102016106cd565b341561033457600080fd5b61018f6106dd565b341561034757600080fd5b610201600160a060020a0360043516610743565b341561036657600080fd5b61020163ffffffff600435811690602435811690604435166bffffffffffffffffffffffff196064351661074e565b6000806000806103a361104f565b6103ab61104f565b6004546006805463ffffffff808416936401000000008104821693680100000000000000008204909216926c01000000000000000000000000918290049091029190600790829060208082020160405190810160405280929190818152602001828054801561044357602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610425575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561049f57602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610481575b50505050509050955095509550955095509550909192939495565b6104c261104f565b600180548060200260200160405190810160405280929190818152602001828054801561051857602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116104fa575b505050505090505b90565b61052e816000610762565b50565b61053961104f565b61054161104f565b600160a060020a03831660009081526002602090815260409182902080549092600184019284929182820290910190519081016040528092919081815260200182805480156105b957602002820191906000526020600020905b8154600160a060020a0316815260019091019060200180831161059b575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561061557602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116105f7575b5050505050905091509150915091565b6000806000806000806008805490506000141561065157600095508594508493508392508291506106c5565b60088054600019810190811061066357fe5b600091825260209091206004909102018054600182015463ffffffff80831699506401000000008304811698506801000000000000000083041696506c0100000000000000000000000091829004909102945067ffffffffffffffff16925090505b509091929394565b6106db600080808080610bff565b565b6106e561104f565b600380548060200260200160405190810160405280929190818152602001828054801561051857602002820191906000526020600020908154600160a060020a031681526001909101906020018083116104fa575050505050905090565b61052e816001610762565b61075c848484846001610bff565b50505050565b600160a060020a033316600090815260208190526040812054819060ff161561075c575050600160a060020a0382166000908152600260205260408120905b81548110156107eb578154600160a060020a033316908390839081106107c357fe5b600091825260209091200154600160a060020a031614156107e35761075c565b6001016107a1565b5060005b600182015481101561083e5733600160a060020a0316826001018281548110151561081657fe5b600091825260209091200154600160a060020a031614156108365761075c565b6001016107ef565b815415801561084f57506001820154155b1561088c5760038054600181016108668382611061565b5060009182526020909120018054600160a060020a031916600160a060020a0386161790555b82156108e45781548290600181016108a48382611061565b5060009182526020909120018054600160a060020a03191633600160a060020a0316179055600154600290835491900490116108df5761075c565b610938565b8160010180548060010182816108fa9190611061565b5060009182526020909120018054600160a060020a03191633600160a060020a0316179055600154600290600184015491900490116109385761075c565b82801561095e5750600160a060020a03841660009081526020819052604090205460ff16155b156109c257600160a060020a0384166000908152602081905260409020805460ff1916600190811790915580548082016109988382611061565b5060009182526020909120018054600160a060020a031916600160a060020a038616179055610b07565b821580156109e85750600160a060020a03841660009081526020819052604090205460ff165b15610b075750600160a060020a0383166000908152602081905260408120805460ff191690555b600154811015610b075783600160a060020a0316600182815481101515610a3257fe5b600091825260209091200154600160a060020a03161415610aff57600180546000198101908110610a5f57fe5b60009182526020909120015460018054600160a060020a039092169183908110610a8557fe5b60009182526020909120018054600160a060020a031916600160a060020a03929092169190911790556001805490610ac1906000198301611061565b50600060048181556005805467ffffffffffffffff1916905590600681610ae8828261108a565b610af660018301600061108a565b50505050610b07565b600101610a0f565b600160a060020a038416600090815260026020526040812090610b2a828261108a565b610b3860018301600061108a565b5050600090505b60035481101561075c5783600160a060020a0316600382815481101515610b6257fe5b600091825260209091200154600160a060020a03161415610bf757600380546000198101908110610b8f57fe5b60009182526020909120015460038054600160a060020a039092169183908110610bb557fe5b60009182526020909120018054600160a060020a031916600160a060020a03929092169190911790556003805490610bf1906000198301611061565b5061075c565b600101610b3f565b600160a060020a033316600090815260208190526040812054819060ff16156110465782158015610c305750600654155b15610c3a57611046565b6006541515610cb5576004805463ffffffff191663ffffffff8981169190911767ffffffff00000000191664010000000089831602176bffffffff000000000000000019166801000000000000000091881691909102176bffffffffffffffffffffffff166c01000000000000000000000000808704021790555b828015610d3f575060045463ffffffff8881169116141580610cea575060045463ffffffff8781166401000000009092041614155b80610d0c575060045463ffffffff868116680100000000000000009092041614155b80610d3f57506004546c0100000000000000000000000090819004026bffffffffffffffffffffffff1990811690851614155b15610d4957611046565b506006905060005b8154811015610d9b578154600160a060020a03331690839083908110610d7357fe5b600091825260209091200154600160a060020a03161415610d9357611046565b600101610d51565b5060005b6001820154811015610dee5733600160a060020a03168260010182815481101515610dc657fe5b600091825260209091200154600160a060020a03161415610de657611046565b600101610d9f565b8215610e46578154829060018101610e068382611061565b5060009182526020909120018054600160a060020a03191633600160a060020a031617905560015460029083549190049011610e4157611046565b610e9a565b816001018054806001018281610e5c9190611061565b5060009182526020909120018054600160a060020a03191633600160a060020a031617905560015460029060018401549190049011610e9a57611046565b821561100d576005805467ffffffffffffffff19164267ffffffffffffffff161790556008805460018101610ecf83826110a8565b6000928352602090922060048054928102909101805463ffffffff191663ffffffff9384161780825582546401000000009081900485160267ffffffff000000001990911617808255825468010000000000000000908190049094169093026bffffffff0000000000000000199093169290921780835581546c01000000000000000000000000908190048102819004026bffffffffffffffffffffffff90911617825560055460018301805467ffffffffffffffff191667ffffffffffffffff909216919091179055600680549192916002830190610fb290829084906110d4565b5060018281018054610fc792840191906110d4565b5050600060048181556005805467ffffffffffffffff191690559450925060069150829050610ff6828261108a565b61100460018301600061108a565b50505050611046565b600060048181556005805467ffffffffffffffff1916905590600681611033828261108a565b61104160018301600061108a565b505050505b50505050505050565b60206040519081016040526000815290565b81548183558181151161108557600083815260209020611085918101908301611124565b505050565b508054600082559060005260206000209081019061052e9190611124565b81548183558181151161108557600402816004028360005260206000209182019101611085919061113e565b8280548282559060005260206000209081019282156111145760005260206000209182015b828111156111145782548255916001019190600101906110f9565b5061112092915061118c565b5090565b61052091905b80821115611120576000815560010161112a565b61052091905b8082111561112057600080825560018201805467ffffffffffffffff191690556002820181611173828261108a565b61118160018301600061108a565b505050600401611144565b61052091905b80821115611120578054600160a060020a03191681556001016111925600a165627a7a7230582008741e499656a10518e57b8480347f2057c123a4c105b88e0f7b88e2f5a796820029`

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
	ReleaseOracleEventer    // Event listener binding to the contract
}

// ReleaseOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type ReleaseOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReleaseOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ReleaseOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReleaseOracleEventer is an auto generated write-only Go binding around an Ethereum contract.
type ReleaseOracleEventer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
	address  common.Address      // Contract address
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
	contract, err := bindReleaseOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ReleaseOracle{ReleaseOracleCaller: ReleaseOracleCaller{contract: contract}, ReleaseOracleTransactor: ReleaseOracleTransactor{contract: contract}}, nil
}

// NewReleaseOracleCaller creates a new read-only instance of ReleaseOracle, bound to a specific deployed contract.
func NewReleaseOracleCaller(address common.Address, caller bind.ContractCaller) (*ReleaseOracleCaller, error) {
	contract, err := bindReleaseOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ReleaseOracleCaller{contract: contract}, nil
}

// NewReleaseOracleTransactor creates a new write-only instance of ReleaseOracle, bound to a specific deployed contract.
func NewReleaseOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*ReleaseOracleTransactor, error) {
	contract, err := bindReleaseOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ReleaseOracleTransactor{contract: contract}, nil
}

// NewReleaseOracleEventer creates a new listen only instance of ReleaseOracle, bound to a specific deployed contract.
func NewReleaseOracleEventer(address common.Address, eventer bind.ContractEventer) (*ReleaseOracleEventer, error) {
	contract, err := bindReleaseOracle(address, nil, nil, eventer)
	if err != nil {
		return nil, err
	}
	return &ReleaseOracleEventer{contract: contract, address: address}, nil
}

// bindReleaseOracle binds a generic wrapper to an already deployed contract.
func bindReleaseOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, eventer bind.ContractEventer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ReleaseOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, eventer), nil
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
