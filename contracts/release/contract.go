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
const ReleaseOracleABI = `[{"constant":true,"inputs":[],"name":"proposedVersion","outputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"pass","type":"address[]"},{"name":"fail","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"signers","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"demote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"user","type":"address"}],"name":"authVotes","outputs":[{"name":"promote","type":"address[]"},{"name":"demote","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"currentVersion","outputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"time","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[],"name":"nuke","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"authProposals","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"promote","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"}],"name":"release","outputs":[],"type":"function"},{"inputs":[{"name":"signers","type":"address[]"}],"type":"constructor"}]`

// ReleaseOracleBin is the compiled bytecode used for deploying new contracts.
const ReleaseOracleBin = `0x606060405260405161135338038061135383398101604052805101600081516000141561008457600160a060020a0333168152602081905260408120805460ff19166001908117909155805480820180835582818380158290116100ff576000838152602090206100ff9181019083015b8082111561012f5760008155600101610070565b5060005b815181101561011f5760016000600050600084848151811015610002576020908102909101810151600160a060020a03168252810191909152604001600020805460ff1916909117905560018054808201808355828183801582901161013357600083815260209020610133918101908301610070565b5050506000928352506020909120018054600160a060020a031916331790555b50506111df806101746000396000f35b5090565b50505091909060005260206000209001600084848151811015610002575050506020838102850101518154600160a060020a0319161790555060010161008856606060405236156100775760e060020a600035046326db7648811461007957806346f0975a1461019e5780635c3d005d1461020a57806364ed31fe146102935780639d888e861461038d578063bc8fbbf8146103b2578063bf8ecf9c146103fc578063d0e0813a14610468578063d67cbec914610479575b005b610496604080516020818101835260008083528351808301855281815260045460068054875181870281018701909852808852939687968796879691959463ffffffff818116956401000000008304821695604060020a840490921694606060020a938490049093029390926007929184919083018282801561012657602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610107575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561018357602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610164575b50505050509050955095509550955095509550909192939495565b6040805160208181018352600082526001805484518184028101840190955280855261055894928301828280156101ff57602002820191906000526020600020905b8154600160a060020a03168152600191909101906020018083116101e0575b505050505090505b90565b61007760043561066d8160005b600160a060020a033316600090815260208190526040812054819060ff161561070057600160a060020a038416815260026020526040812091505b8154811015610706578154600160a060020a033316908390839081101561000257600091825260209091200154600160a060020a0316141561075157610700565b6105a26004356040805160208181018352600080835283518083018552818152600160a060020a038616825260028352908490208054855181850281018501909652808652939491939092600184019291849183018282801561032057602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610301575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561037d57602002820191906000526020600020905b8154600160a060020a031681526001919091019060200180831161035e575b5050505050905091509150915091565b61062760006000600060006000600060086000508054905060001415610670576106f1565b6100776106f96000808080805b600160a060020a033316600090815260208190526040812054819060ff16156111b657821580156103f257506006546000145b15610c2e576111b6565b6040805160208181018352600082526003805484518184028101840190955280855261055894928301828280156101ff57602002820191906000526020600020908154600160a060020a03168152600191909101906020018083116101e0575b50505050509050610207565b61007760043561066d816001610217565b6100776004356024356044356064356107008484848460016103bf565b604051808763ffffffff1681526020018663ffffffff1681526020018563ffffffff168152602001846bffffffffffffffffffffffff1916815260200180602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600302600f01f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600302600f01f1509050019850505050505050505060405180910390f35b60405180806020018281038252838181518152602001915080519060200190602002808383829060006004602084601f0104600302600f01f1509050019250505060405180910390f35b6040518080602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600302600f01f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600302600f01f15090500194505050505060405180910390f35b6040805163ffffffff9687168152948616602086015292909416838301526bffffffffffffffffffffffff19166060830152608082019290925290519081900360a00190f35b50565b600880546000198101908110156100025760009182526004027ff3f7a9fe364faab93b216da50a3214154f22a0a2b415b23a84c8169e8b636ee30190508054600182015463ffffffff8281169950640100000000830481169850604060020a8304169650606060020a91829004909102945067ffffffffffffffff16925090505b509091929394565b565b505050505b50505050565b5060005b60018201548110156107595733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a031614156107a357610700565b600101610252565b8154600014801561076e575060018201546000145b156107cb57600380546001810180835582818380158290116107ab578183600052602060002091820191016107ab9190610851565b60010161070a565b5050506000928352506020909120018054600160a060020a031916851790555b821561086957815460018101808455839190828183801582901161089e5760008381526020902061089e918101908301610851565b5050506000928352506020909120018054600160a060020a031916851790555b600160a060020a038416600090815260026020908152604082208054838255818452918320909291610b2f91908101905b808211156108655760008155600101610851565b5090565b816001016000508054806001018281815481835581811511610950578183600052602060002091820191016109509190610851565b5050506000928352506020909120018054600160a060020a031916331790556001548254600290910490116108d257610700565b8280156108f85750600160a060020a03841660009081526020819052604090205460ff16155b1561098757600160a060020a0384166000908152602081905260409020805460ff1916600190811790915580548082018083558281838015829011610800578183600052602060002091820191016108009190610851565b5050506000928352506020909120018054600160a060020a031916331790556001805490830154600290910490116108d257610700565b821580156109ad5750600160a060020a03841660009081526020819052604090205460ff165b156108205750600160a060020a0383166000908152602081905260408120805460ff191690555b6001548110156108205783600160a060020a0316600160005082815481101561000257600091825260209091200154600160a060020a03161415610aa357600180546000198101908110156100025760206000908120929052600180549290910154600160a060020a031691839081101561000257906000526020600020900160006101000a815481600160a060020a030219169083021790555060016000508054809190600190039090815481835581811511610aab57600083815260209020610aab918101908301610851565b6001016109d4565b5050600060048181556005805467ffffffffffffffff19169055600680548382558184529194509192508290610b05907ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f90810190610851565b5060018201805460008083559182526020909120610b2591810190610851565b5050505050610820565b5060018201805460008083559182526020909120610b4f91810190610851565b506000925050505b6003548110156107005783600160a060020a0316600360005082815481101561000257600091825260209091200154600160a060020a03161415610c2657600380546000198101908110156100025760206000908120929052600380549290910154600160a060020a031691839081101561000257906000526020600020900160006101000a815481600160a060020a0302191690830217905550600360005080548091906001900390908154818355818115116106fb576000838152602090206106fb918101908301610851565b600101610b57565b60065460001415610c8c576004805463ffffffff1916881767ffffffff0000000019166401000000008802176bffffffff00000000000000001916604060020a8702176bffffffffffffffffffffffff16606060020a808704021790555b828015610d08575060045463ffffffff8881169116141580610cc1575060045463ffffffff8781166401000000009092041614155b80610cde575060045463ffffffff868116604060020a9092041614155b80610d085750600454606060020a90819004026bffffffffffffffffffffffff1990811690851614155b15610d12576111b6565b506006905060005b8154811015610d5b578154600160a060020a033316908390839081101561000257600091825260209091200154600160a060020a03161415610da6576111b6565b5060005b6001820154811015610dae5733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a03161415610de3576111b6565b600101610d1a565b8215610deb578154600181018084558391908281838015829011610e2057600083815260209020610e20918101908301610851565b600101610d5f565b816001016000508054806001018281815481835581811511610ea357818360005260206000209182019101610ea39190610851565b5050506000928352506020909120018054600160a060020a03191633179055600154825460029091049011610e54576111b6565b8215610eda576005805467ffffffffffffffff19164217905560088054600181018083558281838015829011610f2f57600402816004028360005260206000209182019101610f2f9190611048565b5050506000928352506020909120018054600160a060020a03191633179055600180549083015460029091049011610e54576111b6565b600060048181556005805467ffffffffffffffff191690556006805483825581845291929182906111bf907ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f90810190610851565b5050509190906000526020600020906004020160005060048054825463ffffffff191663ffffffff9182161780845582546401000000009081900483160267ffffffff000000001991909116178084558254604060020a908190049092169091026bffffffff00000000000000001991909116178083558154606060020a908190048102819004026bffffffffffffffffffffffff9190911617825560055460018301805467ffffffffffffffff191667ffffffffffffffff9092169190911790556006805460028401805482825560008281526020902094959491928392918201918582156110a75760005260206000209182015b828111156110a7578254825591600101919060010190611025565b505050506004015b8082111561086557600080825560018201805467ffffffffffffffff191690556002820180548282558183526020832083916110879190810190610851565b506001820180546000808355918252602090912061104091810190610851565b506110cd9291505b80821115610865578054600160a060020a03191681556001016110af565b505060018181018054918401805480835560008381526020902092938301929091821561111b5760005260206000209182015b8281111561111b578254825591600101919060010190611100565b506111279291506110af565b5050600060048181556005805467ffffffffffffffff191690556006805483825581845291975091955090935084925061118691507ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f90810190610851565b50600182018054600080835591825260209091206111a691810190610851565b50505050506111b6565b50505050505b50505050505050565b50600182018054600080835591825260209091206111b09181019061085156`

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
