// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package contract

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ENSABI is the input ABI used to generate the binding from.
const ENSABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"resolver\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"label\",\"type\":\"bytes32\"},{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"setSubnodeOwner\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"ttl\",\"type\":\"uint64\"}],\"name\":\"setTTL\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"ttl\",\"outputs\":[{\"name\":\"\",\"type\":\"uint64\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"setResolver\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"setOwner\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"inputs\":[],\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"label\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NewOwner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"NewResolver\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"ttl\",\"type\":\"uint64\"}],\"name\":\"NewTTL\",\"type\":\"event\"}]"

// ENSBin is the compiled bytecode used for deploying new contracts.
const ENSBin = `0x606060405260008080526020527fad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb580546c0100000000000000000000000033810204600160a060020a031990911617905561044b8061005e6000396000f3606060405236156100615760e060020a60003504630178b8bf811461006657806302571be31461009257806306ab5923146100ba57806314ab9038146100f657806316a25cbd1461012f5780631896f70a146101635780635b0fc9c31461019c575b610002565b34610002576101d5600435600081815260208190526040902060010154600160a060020a03165b919050565b34610002576101d5600435600081815260208190526040902054600160a060020a031661008d565b34610002576101f16004356024356044356000838152602081905260408120548490600160a060020a0390811633919091161461021057610002565b34610002576101f16004356024356000828152602081905260409020548290600160a060020a039081163391909116146102b357610002565b34610002576101f360043560008181526020819052604090206001015467ffffffffffffffff60a060020a9091041661008d565b34610002576101f16004356024356000828152602081905260409020548290600160a060020a0390811633919091161461035657610002565b34610002576101f16004356024356000828152602081905260409020548290600160a060020a039081163391909116146103d257610002565b60408051600160a060020a039092168252519081900360200190f35b005b6040805167ffffffffffffffff9092168252519081900360200190f35b60408051868152602080820187905282519182900383018220600160a060020a03871683529251929450869288927fce0457fe73731f824cc272376169235128c118b49d344817417c6d108d155e8292908290030190a382600060005060008460001916815260200190815260200160002060005060000160006101000a815481600160a060020a030219169083606060020a9081020402179055505050505050565b6040805167ffffffffffffffff84168152905184917f1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa68919081900360200190a26000838152602081905260409020600101805478010000000000000000000000000000000000000000000000008085020460a060020a027fffffffff0000000000000000ffffffffffffffffffffffffffffffffffffffff909116179055505050565b60408051600160a060020a0384168152905184917f335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a0919081900360200190a260008381526020819052604090206001018054606060020a8085020473ffffffffffffffffffffffffffffffffffffffff19909116179055505050565b60408051600160a060020a0384168152905184917fd4735d920b0f87494915f556dd9b54c8f309026070caea5c737245152564d266919081900360200190a260008381526020819052604090208054606060020a8085020473ffffffffffffffffffffffffffffffffffffffff1990911617905550505056`

// DeployENS deploys a new Ethereum contract, binding an instance of ENS to it.
func DeployENS(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ENS, error) {
	parsed, err := abi.JSON(strings.NewReader(ENSABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ENSBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ENS{ENSCaller: ENSCaller{contract: contract}, ENSTransactor: ENSTransactor{contract: contract}}, nil
}

// ENS is an auto generated Go binding around an Ethereum contract.
type ENS struct {
	ENSCaller     // Read-only binding to the contract
	ENSTransactor // Write-only binding to the contract
}

// ENSCaller is an auto generated read-only Go binding around an Ethereum contract.
type ENSCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ENSTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ENSTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ENSSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ENSSession struct {
	Contract     *ENS              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ENSCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ENSCallerSession struct {
	Contract *ENSCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ENSTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ENSTransactorSession struct {
	Contract     *ENSTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ENSRaw is an auto generated low-level Go binding around an Ethereum contract.
type ENSRaw struct {
	Contract *ENS // Generic contract binding to access the raw methods on
}

// ENSCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ENSCallerRaw struct {
	Contract *ENSCaller // Generic read-only contract binding to access the raw methods on
}

// ENSTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ENSTransactorRaw struct {
	Contract *ENSTransactor // Generic write-only contract binding to access the raw methods on
}

// NewENS creates a new instance of ENS, bound to a specific deployed contract.
func NewENS(address common.Address, backend bind.ContractBackend) (*ENS, error) {
	contract, err := bindENS(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ENS{ENSCaller: ENSCaller{contract: contract}, ENSTransactor: ENSTransactor{contract: contract}}, nil
}

// NewENSCaller creates a new read-only instance of ENS, bound to a specific deployed contract.
func NewENSCaller(address common.Address, caller bind.ContractCaller) (*ENSCaller, error) {
	contract, err := bindENS(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &ENSCaller{contract: contract}, nil
}

// NewENSTransactor creates a new write-only instance of ENS, bound to a specific deployed contract.
func NewENSTransactor(address common.Address, transactor bind.ContractTransactor) (*ENSTransactor, error) {
	contract, err := bindENS(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ENSTransactor{contract: contract}, nil
}

// bindENS binds a generic wrapper to an already deployed contract.
func bindENS(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ENSABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ENS *ENSRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ENS.Contract.ENSCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ENS *ENSRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ENS.Contract.ENSTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ENS *ENSRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ENS.Contract.ENSTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ENS *ENSCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ENS.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ENS *ENSTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ENS.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ENS *ENSTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ENS.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x02571be3.
//
// Solidity: function owner(node bytes32) constant returns(address)
func (_ENS *ENSCaller) Owner(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _ENS.contract.Call(opts, out, "owner", node)
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x02571be3.
//
// Solidity: function owner(node bytes32) constant returns(address)
func (_ENS *ENSSession) Owner(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Owner(&_ENS.CallOpts, node)
}

// Owner is a free data retrieval call binding the contract method 0x02571be3.
//
// Solidity: function owner(node bytes32) constant returns(address)
func (_ENS *ENSCallerSession) Owner(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Owner(&_ENS.CallOpts, node)
}

// Resolver is a free data retrieval call binding the contract method 0x0178b8bf.
//
// Solidity: function resolver(node bytes32) constant returns(address)
func (_ENS *ENSCaller) Resolver(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _ENS.contract.Call(opts, out, "resolver", node)
	return *ret0, err
}

// Resolver is a free data retrieval call binding the contract method 0x0178b8bf.
//
// Solidity: function resolver(node bytes32) constant returns(address)
func (_ENS *ENSSession) Resolver(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Resolver(&_ENS.CallOpts, node)
}

// Resolver is a free data retrieval call binding the contract method 0x0178b8bf.
//
// Solidity: function resolver(node bytes32) constant returns(address)
func (_ENS *ENSCallerSession) Resolver(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Resolver(&_ENS.CallOpts, node)
}

// Ttl is a free data retrieval call binding the contract method 0x16a25cbd.
//
// Solidity: function ttl(node bytes32) constant returns(uint64)
func (_ENS *ENSCaller) Ttl(opts *bind.CallOpts, node [32]byte) (uint64, error) {
	var (
		ret0 = new(uint64)
	)
	out := ret0
	err := _ENS.contract.Call(opts, out, "ttl", node)
	return *ret0, err
}

// Ttl is a free data retrieval call binding the contract method 0x16a25cbd.
//
// Solidity: function ttl(node bytes32) constant returns(uint64)
func (_ENS *ENSSession) Ttl(node [32]byte) (uint64, error) {
	return _ENS.Contract.Ttl(&_ENS.CallOpts, node)
}

// Ttl is a free data retrieval call binding the contract method 0x16a25cbd.
//
// Solidity: function ttl(node bytes32) constant returns(uint64)
func (_ENS *ENSCallerSession) Ttl(node [32]byte) (uint64, error) {
	return _ENS.Contract.Ttl(&_ENS.CallOpts, node)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(node bytes32, owner address) returns()
func (_ENS *ENSTransactor) SetOwner(opts *bind.TransactOpts, node [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setOwner", node, owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(node bytes32, owner address) returns()
func (_ENS *ENSSession) SetOwner(node [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetOwner(&_ENS.TransactOpts, node, owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(node bytes32, owner address) returns()
func (_ENS *ENSTransactorSession) SetOwner(node [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetOwner(&_ENS.TransactOpts, node, owner)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(node bytes32, resolver address) returns()
func (_ENS *ENSTransactor) SetResolver(opts *bind.TransactOpts, node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setResolver", node, resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(node bytes32, resolver address) returns()
func (_ENS *ENSSession) SetResolver(node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetResolver(&_ENS.TransactOpts, node, resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(node bytes32, resolver address) returns()
func (_ENS *ENSTransactorSession) SetResolver(node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetResolver(&_ENS.TransactOpts, node, resolver)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0x06ab5923.
//
// Solidity: function setSubnodeOwner(node bytes32, label bytes32, owner address) returns()
func (_ENS *ENSTransactor) SetSubnodeOwner(opts *bind.TransactOpts, node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setSubnodeOwner", node, label, owner)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0x06ab5923.
//
// Solidity: function setSubnodeOwner(node bytes32, label bytes32, owner address) returns()
func (_ENS *ENSSession) SetSubnodeOwner(node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetSubnodeOwner(&_ENS.TransactOpts, node, label, owner)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0x06ab5923.
//
// Solidity: function setSubnodeOwner(node bytes32, label bytes32, owner address) returns()
func (_ENS *ENSTransactorSession) SetSubnodeOwner(node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetSubnodeOwner(&_ENS.TransactOpts, node, label, owner)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(node bytes32, ttl uint64) returns()
func (_ENS *ENSTransactor) SetTTL(opts *bind.TransactOpts, node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setTTL", node, ttl)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(node bytes32, ttl uint64) returns()
func (_ENS *ENSSession) SetTTL(node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _ENS.Contract.SetTTL(&_ENS.TransactOpts, node, ttl)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(node bytes32, ttl uint64) returns()
func (_ENS *ENSTransactorSession) SetTTL(node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _ENS.Contract.SetTTL(&_ENS.TransactOpts, node, ttl)
}

// FIFSRegistrarABI is the input ABI used to generate the binding from.
const FIFSRegistrarABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"subnode\",\"type\":\"bytes32\"},{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"register\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"inputs\":[{\"name\":\"ensAddr\",\"type\":\"address\"},{\"name\":\"node\",\"type\":\"bytes32\"}],\"type\":\"constructor\"}]"

// FIFSRegistrarBin is the compiled bytecode used for deploying new contracts.
const FIFSRegistrarBin = `0x60606040818152806106aa833960a0905251608051600080546c0100000000000000000000000080850204600160a060020a031990911617905560405182906103aa806100a58339018082600160a060020a03168152602001915050604051809103906000f0801560025760018054600160a060020a0319166c01000000000000000000000000928302929092049190911790556002555061025b8061044f6000396000f360606040526040516020806103aa833950608060405251600080546c0100000000000000000000000080840204600160a060020a0319909116179055506103608061004a6000396000f3606060405236156100565760e060020a600035046301ffc9a781146100635780632dff6941146100d75780633b3b57de1461010057806341b9dc2b1461013c578063c3d014d6146101da578063d5fa2b0014610267575b34610002576102f4610002565b34610002576102f660043560007f3b3b57de00000000000000000000000000000000000000000000000000000000600160e060020a0319831614806100d157507fd8389dc500000000000000000000000000000000000000000000000000000000600160e060020a03198316145b92915050565b346100025760043560009081526002602052604090205460408051918252519081900360200190f35b3461000257600435600090815260016020526040902054600160a060020a031660408051600160a060020a039092168252519081900360200190f35b34610002576102f660043560243560007f6164647200000000000000000000000000000000000000000000000000000000821480156101915750600083815260016020526040902054600160a060020a031615155b806101d357507f6861736800000000000000000000000000000000000000000000000000000000821480156101d3575060008381526002602052604090205415155b9392505050565b34610002576102f460043560243560008054604080516020908101849052815160e060020a6302571be30281526004810187905291518694600160a060020a033381169516936302571be393602480830194919391928390030190829087803b156100025760325a03f11561000257505060405151600160a060020a031691909114905061030a57610002565b34610002576102f460043560243560008054604080516020908101849052815160e060020a6302571be30281526004810187905291518694600160a060020a033381169516936302571be393602480830194919391928390030190829087803b156100025760325a03f11561000257505060405151600160a060020a031691909114905061031d57610002565b005b604080519115158252519081900360200190f35b5060009182526002602052604090912055565b600083815260016020526040902080546c010000000000000000000000008085020473ffffffffffffffffffffffffffffffffffffffff1990911617905550505056606060405260e060020a6000350463d22057a9811461001e575b610002565b34610002576100f1600435602435600254604080519182526020808301859052815192839003820183206000805494830181905283517f02571be300000000000000000000000000000000000000000000000000000000815260048101839052935191949093600160a060020a03909116926302571be39260248084019382900301818787803b156100025760325a03f11561000257505060405151915050600160a060020a038116158015906100e7575033600160a060020a031681600160a060020a031614155b156100f357610002565b005b60008054600254604080517f06ab5923000000000000000000000000000000000000000000000000000000008152600481019290925260248201889052600160a060020a03308116604484015290519216926306ab59239260648084019382900301818387803b156100025760325a03f11561000257505060008054600154604080517f1896f70a00000000000000000000000000000000000000000000000000000000815260048101889052600160a060020a0392831660248201529051919092169350631896f70a9260448084019391929182900301818387803b156100025760325a03f11561000257505060008054604080517f5b0fc9c300000000000000000000000000000000000000000000000000000000815260048101879052600160a060020a0388811660248301529151919092169350635b0fc9c39260448084019391929182900301818387803b156100025760325a03f115610002575050505050505056`

// DeployFIFSRegistrar deploys a new Ethereum contract, binding an instance of FIFSRegistrar to it.
func DeployFIFSRegistrar(auth *bind.TransactOpts, backend bind.ContractBackend, ensAddr common.Address, node [32]byte) (common.Address, *types.Transaction, *FIFSRegistrar, error) {
	parsed, err := abi.JSON(strings.NewReader(FIFSRegistrarABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(FIFSRegistrarBin), backend, ensAddr, node)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &FIFSRegistrar{FIFSRegistrarCaller: FIFSRegistrarCaller{contract: contract}, FIFSRegistrarTransactor: FIFSRegistrarTransactor{contract: contract}}, nil
}

// FIFSRegistrar is an auto generated Go binding around an Ethereum contract.
type FIFSRegistrar struct {
	FIFSRegistrarCaller     // Read-only binding to the contract
	FIFSRegistrarTransactor // Write-only binding to the contract
}

// FIFSRegistrarCaller is an auto generated read-only Go binding around an Ethereum contract.
type FIFSRegistrarCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FIFSRegistrarTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FIFSRegistrarTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FIFSRegistrarSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FIFSRegistrarSession struct {
	Contract     *FIFSRegistrar    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FIFSRegistrarCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FIFSRegistrarCallerSession struct {
	Contract *FIFSRegistrarCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// FIFSRegistrarTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FIFSRegistrarTransactorSession struct {
	Contract     *FIFSRegistrarTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// FIFSRegistrarRaw is an auto generated low-level Go binding around an Ethereum contract.
type FIFSRegistrarRaw struct {
	Contract *FIFSRegistrar // Generic contract binding to access the raw methods on
}

// FIFSRegistrarCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FIFSRegistrarCallerRaw struct {
	Contract *FIFSRegistrarCaller // Generic read-only contract binding to access the raw methods on
}

// FIFSRegistrarTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FIFSRegistrarTransactorRaw struct {
	Contract *FIFSRegistrarTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFIFSRegistrar creates a new instance of FIFSRegistrar, bound to a specific deployed contract.
func NewFIFSRegistrar(address common.Address, backend bind.ContractBackend) (*FIFSRegistrar, error) {
	contract, err := bindFIFSRegistrar(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FIFSRegistrar{FIFSRegistrarCaller: FIFSRegistrarCaller{contract: contract}, FIFSRegistrarTransactor: FIFSRegistrarTransactor{contract: contract}}, nil
}

// NewFIFSRegistrarCaller creates a new read-only instance of FIFSRegistrar, bound to a specific deployed contract.
func NewFIFSRegistrarCaller(address common.Address, caller bind.ContractCaller) (*FIFSRegistrarCaller, error) {
	contract, err := bindFIFSRegistrar(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &FIFSRegistrarCaller{contract: contract}, nil
}

// NewFIFSRegistrarTransactor creates a new write-only instance of FIFSRegistrar, bound to a specific deployed contract.
func NewFIFSRegistrarTransactor(address common.Address, transactor bind.ContractTransactor) (*FIFSRegistrarTransactor, error) {
	contract, err := bindFIFSRegistrar(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &FIFSRegistrarTransactor{contract: contract}, nil
}

// bindFIFSRegistrar binds a generic wrapper to an already deployed contract.
func bindFIFSRegistrar(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(FIFSRegistrarABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FIFSRegistrar *FIFSRegistrarRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _FIFSRegistrar.Contract.FIFSRegistrarCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FIFSRegistrar *FIFSRegistrarRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FIFSRegistrar.Contract.FIFSRegistrarTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FIFSRegistrar *FIFSRegistrarRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FIFSRegistrar.Contract.FIFSRegistrarTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FIFSRegistrar *FIFSRegistrarCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _FIFSRegistrar.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FIFSRegistrar *FIFSRegistrarTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FIFSRegistrar.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FIFSRegistrar *FIFSRegistrarTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FIFSRegistrar.Contract.contract.Transact(opts, method, params...)
}

// Register is a paid mutator transaction binding the contract method 0xd22057a9.
//
// Solidity: function register(subnode bytes32, owner address) returns()
func (_FIFSRegistrar *FIFSRegistrarTransactor) Register(opts *bind.TransactOpts, subnode [32]byte, owner common.Address) (*types.Transaction, error) {
	return _FIFSRegistrar.contract.Transact(opts, "register", subnode, owner)
}

// Register is a paid mutator transaction binding the contract method 0xd22057a9.
//
// Solidity: function register(subnode bytes32, owner address) returns()
func (_FIFSRegistrar *FIFSRegistrarSession) Register(subnode [32]byte, owner common.Address) (*types.Transaction, error) {
	return _FIFSRegistrar.Contract.Register(&_FIFSRegistrar.TransactOpts, subnode, owner)
}

// Register is a paid mutator transaction binding the contract method 0xd22057a9.
//
// Solidity: function register(subnode bytes32, owner address) returns()
func (_FIFSRegistrar *FIFSRegistrarTransactorSession) Register(subnode [32]byte, owner common.Address) (*types.Transaction, error) {
	return _FIFSRegistrar.Contract.Register(&_FIFSRegistrar.TransactOpts, subnode, owner)
}

// PublicResolverABI is the input ABI used to generate the binding from.
const PublicResolverABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"content\",\"outputs\":[{\"name\":\"ret\",\"type\":\"bytes32\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"addr\",\"outputs\":[{\"name\":\"ret\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"kind\",\"type\":\"bytes32\"}],\"name\":\"has\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"setContent\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"setAddr\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"inputs\":[{\"name\":\"ensAddr\",\"type\":\"address\"}],\"type\":\"constructor\"},{\"payable\":false,\"type\":\"fallback\"}]"

// PublicResolverBin is the compiled bytecode used for deploying new contracts.
const PublicResolverBin = `0x60606040526040516020806103aa833950608060405251600080546c0100000000000000000000000080840204600160a060020a0319909116179055506103608061004a6000396000f3606060405236156100565760e060020a600035046301ffc9a781146100635780632dff6941146100d75780633b3b57de1461010057806341b9dc2b1461013c578063c3d014d6146101da578063d5fa2b0014610267575b34610002576102f4610002565b34610002576102f660043560007f3b3b57de00000000000000000000000000000000000000000000000000000000600160e060020a0319831614806100d157507fd8389dc500000000000000000000000000000000000000000000000000000000600160e060020a03198316145b92915050565b346100025760043560009081526002602052604090205460408051918252519081900360200190f35b3461000257600435600090815260016020526040902054600160a060020a031660408051600160a060020a039092168252519081900360200190f35b34610002576102f660043560243560007f6164647200000000000000000000000000000000000000000000000000000000821480156101915750600083815260016020526040902054600160a060020a031615155b806101d357507f6861736800000000000000000000000000000000000000000000000000000000821480156101d3575060008381526002602052604090205415155b9392505050565b34610002576102f460043560243560008054604080516020908101849052815160e060020a6302571be30281526004810187905291518694600160a060020a033381169516936302571be393602480830194919391928390030190829087803b156100025760325a03f11561000257505060405151600160a060020a031691909114905061030a57610002565b34610002576102f460043560243560008054604080516020908101849052815160e060020a6302571be30281526004810187905291518694600160a060020a033381169516936302571be393602480830194919391928390030190829087803b156100025760325a03f11561000257505060405151600160a060020a031691909114905061031d57610002565b005b604080519115158252519081900360200190f35b5060009182526002602052604090912055565b600083815260016020526040902080546c010000000000000000000000008085020473ffffffffffffffffffffffffffffffffffffffff1990911617905550505056`

// DeployPublicResolver deploys a new Ethereum contract, binding an instance of PublicResolver to it.
func DeployPublicResolver(auth *bind.TransactOpts, backend bind.ContractBackend, ensAddr common.Address) (common.Address, *types.Transaction, *PublicResolver, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PublicResolverBin), backend, ensAddr)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}}, nil
}

// PublicResolver is an auto generated Go binding around an Ethereum contract.
type PublicResolver struct {
	PublicResolverCaller     // Read-only binding to the contract
	PublicResolverTransactor // Write-only binding to the contract
}

// PublicResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PublicResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PublicResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PublicResolverSession struct {
	Contract     *PublicResolver   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PublicResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PublicResolverCallerSession struct {
	Contract *PublicResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// PublicResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PublicResolverTransactorSession struct {
	Contract     *PublicResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// PublicResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type PublicResolverRaw struct {
	Contract *PublicResolver // Generic contract binding to access the raw methods on
}

// PublicResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PublicResolverCallerRaw struct {
	Contract *PublicResolverCaller // Generic read-only contract binding to access the raw methods on
}

// PublicResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PublicResolverTransactorRaw struct {
	Contract *PublicResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPublicResolver creates a new instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolver(address common.Address, backend bind.ContractBackend) (*PublicResolver, error) {
	contract, err := bindPublicResolver(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}}, nil
}

// NewPublicResolverCaller creates a new read-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverCaller(address common.Address, caller bind.ContractCaller) (*PublicResolverCaller, error) {
	contract, err := bindPublicResolver(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &PublicResolverCaller{contract: contract}, nil
}

// NewPublicResolverTransactor creates a new write-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*PublicResolverTransactor, error) {
	contract, err := bindPublicResolver(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &PublicResolverTransactor{contract: contract}, nil
}

// bindPublicResolver binds a generic wrapper to an already deployed contract.
func bindPublicResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PublicResolver *PublicResolverRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PublicResolver.Contract.PublicResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PublicResolver *PublicResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PublicResolver.Contract.PublicResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PublicResolver *PublicResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PublicResolver.Contract.PublicResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PublicResolver *PublicResolverCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PublicResolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PublicResolver *PublicResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PublicResolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PublicResolver *PublicResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PublicResolver.Contract.contract.Transact(opts, method, params...)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_PublicResolver *PublicResolverCaller) Addr(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "addr", node)
	return *ret0, err
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_PublicResolver *PublicResolverSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_PublicResolver *PublicResolverCallerSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_PublicResolver *PublicResolverCaller) Content(opts *bind.CallOpts, node [32]byte) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "content", node)
	return *ret0, err
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_PublicResolver *PublicResolverSession) Content(node [32]byte) ([32]byte, error) {
	return _PublicResolver.Contract.Content(&_PublicResolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_PublicResolver *PublicResolverCallerSession) Content(node [32]byte) ([32]byte, error) {
	return _PublicResolver.Contract.Content(&_PublicResolver.CallOpts, node)
}

// Has is a free data retrieval call binding the contract method 0x41b9dc2b.
//
// Solidity: function has(node bytes32, kind bytes32) constant returns(bool)
func (_PublicResolver *PublicResolverCaller) Has(opts *bind.CallOpts, node [32]byte, kind [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "has", node, kind)
	return *ret0, err
}

// Has is a free data retrieval call binding the contract method 0x41b9dc2b.
//
// Solidity: function has(node bytes32, kind bytes32) constant returns(bool)
func (_PublicResolver *PublicResolverSession) Has(node [32]byte, kind [32]byte) (bool, error) {
	return _PublicResolver.Contract.Has(&_PublicResolver.CallOpts, node, kind)
}

// Has is a free data retrieval call binding the contract method 0x41b9dc2b.
//
// Solidity: function has(node bytes32, kind bytes32) constant returns(bool)
func (_PublicResolver *PublicResolverCallerSession) Has(node [32]byte, kind [32]byte) (bool, error) {
	return _PublicResolver.Contract.Has(&_PublicResolver.CallOpts, node, kind)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_PublicResolver *PublicResolverCaller) SupportsInterface(opts *bind.CallOpts, interfaceID [4]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "supportsInterface", interfaceID)
	return *ret0, err
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_PublicResolver *PublicResolverSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_PublicResolver *PublicResolverCallerSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(node bytes32, addr address) returns()
func (_PublicResolver *PublicResolverTransactor) SetAddr(opts *bind.TransactOpts, node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setAddr", node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(node bytes32, addr address) returns()
func (_PublicResolver *PublicResolverSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(node bytes32, addr address) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(node bytes32, hash bytes32) returns()
func (_PublicResolver *PublicResolverTransactor) SetContent(opts *bind.TransactOpts, node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setContent", node, hash)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(node bytes32, hash bytes32) returns()
func (_PublicResolver *PublicResolverSession) SetContent(node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContent(&_PublicResolver.TransactOpts, node, hash)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(node bytes32, hash bytes32) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetContent(node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContent(&_PublicResolver.TransactOpts, node, hash)
}

// ResolverABI is the input ABI used to generate the binding from.
const ResolverABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"content\",\"outputs\":[{\"name\":\"ret\",\"type\":\"bytes32\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"addr\",\"outputs\":[{\"name\":\"ret\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"qtype\",\"type\":\"uint16\"},{\"name\":\"qclass\",\"type\":\"uint16\"},{\"name\":\"index\",\"type\":\"uint32\"}],\"name\":\"dnsrr\",\"outputs\":[{\"name\":\"rtype\",\"type\":\"uint16\"},{\"name\":\"rclass\",\"type\":\"uint16\"},{\"name\":\"data\",\"type\":\"bytes\"}],\"payable\":false,\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"a\",\"type\":\"address\"}],\"name\":\"AddrChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"ContentChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"qtype\",\"type\":\"uint16\"},{\"indexed\":false,\"name\":\"qclass\",\"type\":\"uint16\"},{\"indexed\":false,\"name\":\"index\",\"type\":\"uint32\"}],\"name\":\"DnsrrChanged\",\"type\":\"event\"}]"

// ResolverBin is the compiled bytecode used for deploying new contracts.
const ResolverBin = `0x`

// DeployResolver deploys a new Ethereum contract, binding an instance of Resolver to it.
func DeployResolver(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Resolver, error) {
	parsed, err := abi.JSON(strings.NewReader(ResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ResolverBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Resolver{ResolverCaller: ResolverCaller{contract: contract}, ResolverTransactor: ResolverTransactor{contract: contract}}, nil
}

// Resolver is an auto generated Go binding around an Ethereum contract.
type Resolver struct {
	ResolverCaller     // Read-only binding to the contract
	ResolverTransactor // Write-only binding to the contract
}

// ResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type ResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ResolverSession struct {
	Contract     *Resolver         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ResolverCallerSession struct {
	Contract *ResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ResolverTransactorSession struct {
	Contract     *ResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type ResolverRaw struct {
	Contract *Resolver // Generic contract binding to access the raw methods on
}

// ResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ResolverCallerRaw struct {
	Contract *ResolverCaller // Generic read-only contract binding to access the raw methods on
}

// ResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ResolverTransactorRaw struct {
	Contract *ResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewResolver creates a new instance of Resolver, bound to a specific deployed contract.
func NewResolver(address common.Address, backend bind.ContractBackend) (*Resolver, error) {
	contract, err := bindResolver(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Resolver{ResolverCaller: ResolverCaller{contract: contract}, ResolverTransactor: ResolverTransactor{contract: contract}}, nil
}

// NewResolverCaller creates a new read-only instance of Resolver, bound to a specific deployed contract.
func NewResolverCaller(address common.Address, caller bind.ContractCaller) (*ResolverCaller, error) {
	contract, err := bindResolver(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &ResolverCaller{contract: contract}, nil
}

// NewResolverTransactor creates a new write-only instance of Resolver, bound to a specific deployed contract.
func NewResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*ResolverTransactor, error) {
	contract, err := bindResolver(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &ResolverTransactor{contract: contract}, nil
}

// bindResolver binds a generic wrapper to an already deployed contract.
func bindResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Resolver *ResolverRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Resolver.Contract.ResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Resolver *ResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Resolver.Contract.ResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Resolver *ResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Resolver.Contract.ResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Resolver *ResolverCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Resolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Resolver *ResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Resolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Resolver *ResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Resolver.Contract.contract.Transact(opts, method, params...)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_Resolver *ResolverCaller) Addr(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Resolver.contract.Call(opts, out, "addr", node)
	return *ret0, err
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_Resolver *ResolverSession) Addr(node [32]byte) (common.Address, error) {
	return _Resolver.Contract.Addr(&_Resolver.CallOpts, node)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_Resolver *ResolverCallerSession) Addr(node [32]byte) (common.Address, error) {
	return _Resolver.Contract.Addr(&_Resolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_Resolver *ResolverCaller) Content(opts *bind.CallOpts, node [32]byte) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Resolver.contract.Call(opts, out, "content", node)
	return *ret0, err
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_Resolver *ResolverSession) Content(node [32]byte) ([32]byte, error) {
	return _Resolver.Contract.Content(&_Resolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_Resolver *ResolverCallerSession) Content(node [32]byte) ([32]byte, error) {
	return _Resolver.Contract.Content(&_Resolver.CallOpts, node)
}

// Dnsrr is a free data retrieval call binding the contract method 0xfcca1f4e.
//
// Solidity: function dnsrr(node bytes32, qtype uint16, qclass uint16, index uint32) constant returns(rtype uint16, rclass uint16, data bytes)
func (_Resolver *ResolverCaller) Dnsrr(opts *bind.CallOpts, node [32]byte, qtype uint16, qclass uint16, index uint32) (struct {
	Rtype  uint16
	Rclass uint16
	Data   []byte
}, error) {
	ret := new(struct {
		Rtype  uint16
		Rclass uint16
		Data   []byte
	})
	out := ret
	err := _Resolver.contract.Call(opts, out, "dnsrr", node, qtype, qclass, index)
	return *ret, err
}

// Dnsrr is a free data retrieval call binding the contract method 0xfcca1f4e.
//
// Solidity: function dnsrr(node bytes32, qtype uint16, qclass uint16, index uint32) constant returns(rtype uint16, rclass uint16, data bytes)
func (_Resolver *ResolverSession) Dnsrr(node [32]byte, qtype uint16, qclass uint16, index uint32) (struct {
	Rtype  uint16
	Rclass uint16
	Data   []byte
}, error) {
	return _Resolver.Contract.Dnsrr(&_Resolver.CallOpts, node, qtype, qclass, index)
}

// Dnsrr is a free data retrieval call binding the contract method 0xfcca1f4e.
//
// Solidity: function dnsrr(node bytes32, qtype uint16, qclass uint16, index uint32) constant returns(rtype uint16, rclass uint16, data bytes)
func (_Resolver *ResolverCallerSession) Dnsrr(node [32]byte, qtype uint16, qclass uint16, index uint32) (struct {
	Rtype  uint16
	Rclass uint16
	Data   []byte
}, error) {
	return _Resolver.Contract.Dnsrr(&_Resolver.CallOpts, node, qtype, qclass, index)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_Resolver *ResolverCaller) SupportsInterface(opts *bind.CallOpts, interfaceID [4]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Resolver.contract.Call(opts, out, "supportsInterface", interfaceID)
	return *ret0, err
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_Resolver *ResolverSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _Resolver.Contract.SupportsInterface(&_Resolver.CallOpts, interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_Resolver *ResolverCallerSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _Resolver.Contract.SupportsInterface(&_Resolver.CallOpts, interfaceID)
}
