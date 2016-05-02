// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

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
const ReleaseOracleABI = `[{"constant":true,"inputs":[],"name":"currentVersion","outputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"time","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"proposedVersion","outputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"pass","type":"address[]"},{"name":"fail","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"signers","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"authProposals","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[{"name":"user","type":"address"}],"name":"authVotes","outputs":[{"name":"promote","type":"address[]"},{"name":"demote","type":"address[]"}],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"promote","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"demote","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"}],"name":"release","outputs":[],"type":"function"},{"constant":false,"inputs":[],"name":"nuke","outputs":[],"type":"function"},{"inputs":[{"name":"signers","type":"address[]"}],"type":"constructor"}]`

// ReleaseOracleBin is the compiled bytecode used for deploying new contracts.
const ReleaseOracleBin = `0x60606040526040516114033803806114038339810160405280510160008151600014156100ce57600160a060020a0333168152602081905260408120805460ff191660019081179091558054808201808355828183801582901161015e5781836000526020600020918201910161015e91905b8082111561019957600081558401610072565b505050919090600052602060002090016000848481518110156100025790602001906020020151909190916101000a815481600160a060020a0302191690830217905550506001015b815181101561018957600160006000506000848481518110156100025790602001906020020151600160a060020a0316815260200190815260200160002060006101000a81548160ff0219169083021790555060016000508054806001018281815481835581811511610085576000839052610085906000805160206113e3833981519152908101908301610072565b5050506000929092526000805160206113e3833981519152018054600160a060020a03191633179055505b50506112458061019e6000396000f35b50905600606060405236156100775760e060020a600035046326db7648811461007957806346f0975a1461019e5780635c3d005d1461020a57806364ed31fe146102935780639d888e861461038d578063bc8fbbf8146103b2578063bf8ecf9c146103fb578063d0e0813a14610467578063d67cbec914610478575b005b610495604080516020818101835260008083528351808301855281815260045460068054875181870281018701909852808852939687968796879691959463ffffffff818116956401000000008304821695604060020a840490921694606060020a938490049093029390926007929184919083018282801561012657602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610107575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561018357602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610164575b50505050509050955095509550955095509550909192939495565b6040805160208181018352600082526001805484518184028101840190955280855261054894928301828280156101ff57602002820191906000526020600020905b8154600160a060020a03168152600191909101906020018083116101e0575b505050505090505b90565b6100776004356106d78160005b600160a060020a033316600090815260208190526040812054819060ff16156106df57600160a060020a038416815260026020526040812091505b81548110156106e7578154600160a060020a033316908390839081101561000257600091825260209091200154600160a060020a03161415610732576106df565b6105926004356040805160208181018352600080835283518083018552818152600160a060020a038616825260028352908490208054855181850281018501909652808652939491939092600184019291849183018282801561032057602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610301575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561037d57602002820191906000526020600020905b8154600160a060020a031681526001919091019060200180831161035e575b5050505050905091509150915091565b6106176000600060006000600060006008600050805490506000141561064e576106cf565b6100776106e56000808080805b600160a060020a033316600090815260208190526040812054819060ff161561121c57821580156103f1575060065481145b15610ca55761121c565b6040805160208181018352600082526003805484518184028101840190955280855261054894928301828280156101ff57602002820191906000526020600020908154600160a060020a03168152600191909101906020018083116101e0575b50505050509050610207565b6100776004356106d7816001610217565b6100776004356024356044356064356106df8484848460016103bf565b604051808763ffffffff1681526020018663ffffffff1681526020018563ffffffff16815260200184815260200180602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019850505050505050505060405180910390f35b60405180806020018281038252838181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019250505060405180910390f35b6040518080602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f15090500194505050505060405180910390f35b6040805163ffffffff9687168152948616602086015292909416838301526060830152608082019290925290519081900360a00190f35b600880546000198101908110156100025760009182526004027ff3f7a9fe364faab93b216da50a3214154f22a0a2b415b23a84c8169e8b636ee30190508054600182015463ffffffff8281169950640100000000830481169850604060020a8304169650606060020a91829004909102945067ffffffffffffffff16925090505b509091929394565b50565b505050505b50505050565b565b5060005b600182015481101561073a5733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a03161415610784576106df565b600101610252565b8154600014801561074f575060018201546000145b156107ac576003805460018101808355828183801582901161078c5781836000526020600020918201910161078c9190610834565b6001016106eb565b5050506000928352506020909120018054600160a060020a031916851790555b821561084c578154600181018084558391908281838015829011610881578183600052602060002091820191016108819190610834565b5050506000928352506020909120018054600160a060020a031916851790555b600160a060020a038416600090815260026020908152604082208054838255818452918320909291610b5c91908101905b808211156108485760008155600101610834565b5090565b816001016000508054806001018281815481835581811511610933578183600052602060002091820191016109339190610834565b5050506000928352506020909120018054600160a060020a031916331790556001548254600290910490116108b5576106df565b8280156108db5750600160a060020a03841660009081526020819052604090205460ff16155b1561096a57600160a060020a0384166000908152602081905260409020805460ff19166001908117909155805480820180835582818380158290116107e3578183600052602060002091820191016107e39190610834565b5050506000928352506020909120018054600160a060020a031916331790556001805490830154600290910490116108b5576106df565b821580156109905750600160a060020a03841660009081526020819052604090205460ff165b156108035750600160a060020a0383166000908152602081905260408120805460ff191690555b6001548110156108035783600160a060020a03166001600050828154811015610002576000919091527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf60154600160a060020a03161415610ad057600180546000198101908110156100025760009182527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf601909054906101000a9004600160a060020a03166001600050828154811015610002577fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6018054600160a060020a03191690921790915580546000198101808355909190828015829011610ad857818360005260206000209182019101610ad89190610834565b6001016109b7565b5050600060048181556005805467ffffffffffffffff19169055600680548382558184529194509192508290610b32907ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f90810190610834565b5060018201805460008083559182526020909120610b5291810190610834565b5050505050610803565b5060018201805460008083559182526020909120610b7c91810190610834565b506000925050505b6003548110156106df5783600160a060020a03166003600050828154811015610002576000919091527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b0154600160a060020a03161415610c9d57600380546000198101908110156100025760009182527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b01909054906101000a9004600160a060020a03166003600050828154811015610002577fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b018054600160a060020a031916909217909155805460001981018083559091908280158290116106da578183600052602060002091820191016106da9190610834565b600101610b84565b60065460001415610d03576004805463ffffffff1916881767ffffffff0000000019166401000000008802176bffffffff00000000000000001916604060020a8702176bffffffffffffffffffffffff16606060020a808704021790555b828015610d6d575060045463ffffffff8881169116141580610d395750600454640100000000900463ffffffff90811690871614155b80610d56575060045463ffffffff868116604060020a9092041614155b80610d6d5750600454606060020a90819004028414155b15610d775761121c565b506006905060005b8154811015610dc0578154600160a060020a033316908390839081101561000257600091825260209091200154600160a060020a03161415610e0b5761121c565b5060005b6001820154811015610e135733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a03161415610e485761121c565b600101610d7f565b8215610e50578154600181018084558391908281838015829011610e8557600083815260209020610e85918101908301610834565b600101610dc4565b816001016000508054806001018281815481835581811511610f0857818360005260206000209182019101610f089190610834565b5050506000928352506020909120018054600160a060020a03191633179055600154825460029091049011610eb95761121c565b8215610f3f576005805467ffffffffffffffff19164217905560088054600181018083558281838015829011610f9457600402816004028360005260206000209182019101610f9491906110ae565b5050506000928352506020909120018054600160a060020a03191633179055600180549083015460029091049011610eb95761121c565b600060048181556005805467ffffffffffffffff19169055600680548382558184529192918290611225907ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f90810190610834565b5050509190906000526020600020906004020160005060048054825463ffffffff191663ffffffff9182161780845582546401000000009081900483160267ffffffff000000001991909116178084558254604060020a908190049092169091026bffffffff00000000000000001991909116178083558154606060020a908190048102819004026bffffffffffffffffffffffff9190911617825560055460018301805467ffffffffffffffff191667ffffffffffffffff9290921691909117905560068054600284018054828255600082815260209020949594919283929182019185821561110d5760005260206000209182015b8281111561110d57825482559160010191906001019061108b565b505050506004015b8082111561084857600080825560018201805467ffffffffffffffff191690556002820180548282558183526020832083916110ed9190810190610834565b50600182018054600080835591825260209091206110a691810190610834565b506111339291505b80821115610848578054600160a060020a0319168155600101611115565b50506001818101805491840180548083556000838152602090209293830192909182156111815760005260206000209182015b82811115611181578254825591600101919060010190611166565b5061118d929150611115565b5050600060048181556005805467ffffffffffffffff19169055600680548382558184529197509195509093508492506111ec91507ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f90810190610834565b506001820180546000808355918252602090912061120c91810190610834565b505050505061121c565b50505050505b50505050505050565b50600182018054600080835591825260209091206112169181019061083456b10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6`

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
