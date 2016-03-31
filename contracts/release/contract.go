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
const ReleaseOracleABI = `[{"constant":false,"inputs":[],"name":"Nuke","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"}],"name":"Release","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"AuthProposals","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"CurrentVersion","outputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"time","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"ProposedVersion","outputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"pass","type":"address[]"},{"name":"fail","type":"address[]"}],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"Promote","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"major","type":"uint32"},{"name":"minor","type":"uint32"},{"name":"patch","type":"uint32"},{"name":"commit","type":"bytes20"},{"name":"release","type":"bool"}],"name":"updateRelease","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"}],"name":"Demote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"user","type":"address"}],"name":"AuthVotes","outputs":[{"name":"promote","type":"address[]"},{"name":"demote","type":"address[]"}],"type":"function"},{"constant":true,"inputs":[],"name":"Signers","outputs":[{"name":"","type":"address[]"}],"type":"function"},{"constant":false,"inputs":[{"name":"user","type":"address"},{"name":"authorize","type":"bool"}],"name":"updateSigner","outputs":[],"type":"function"},{"inputs":[],"type":"constructor"}]`

// ReleaseOracleBin is the compiled bytecode used for deploying new contracts.
const ReleaseOracleBin = `0x6060604052600160a060020a0333166000908152602081905260409020805460ff1916600190811790915580548082018083558281838015829011606257818360005260206000209182019101606291905b808211156094576000815584016051565b505050919090600052602060002090016000508054600160a060020a0319163317905550611262806100986000396000f35b5090566060604052361561008d5760e060020a60003504630443b1ad811461008f5780630d618178146100a0578063282fe4e5146100bd5780632b225f291461012c5780634c327071146101515780636195db9c14610276578063645dce721461028757806380bbbd4a146102d6578063a29226f2146102e7578063f04c4758146103e1578063f460590b1461044d575b005b61008d61072060008080808061029a565b61008d60043560243560443560643561071a84848484600161029a565b6104d360408051602081810183526000825282516003805480840283018401909552848252929390929183018282801561044257602002820191906000526020600020908154600160a060020a0316815260019190910190602001808311610423575b5050505050905061044a565b61051d600060006000600060006000600860005080549050600014156106895761070a565b610551604080516020818101835260008083528351808301855281815260045460068054875181870281018701909852808852939687968796879691959463ffffffff818116956401000000008304821695604060020a840490921694606060020a93849004909302939092600792918491908301828280156101fe57602002820191906000526020600020905b8154600160a060020a03168152600191909101906020018083116101df575b505050505091508080548060200260200160405190810160405280929190818152602001828054801561025b57602002820191906000526020600020905b8154600160a060020a031681526001919091019060200180831161023c575b50505050509050955095509550955095509550909192939495565b61008d600435610712816001610457565b61008d6004356024356044356064356084355b600160a060020a033316600090815260208190526040812054819060ff16156111f957821580156102cc575060065481145b15610c81576111f9565b61008d600435610712816000610457565b6106046004356040805160208181018352600080835283518083018552818152600160a060020a038616825260028352908490208054855181850281018501909652808652939491939092600184019291849183018282801561037457602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610355575b50505050509150808054806020026020016040519081016040528092919081815260200182805480156103d157602002820191906000526020600020905b8154600160a060020a03168152600191909101906020018083116103b2575b5050505050905091509150915091565b604080516020818101835260008252600180548451818402810184019095528085526104d3949283018282801561044257602002820191906000526020600020905b8154600160a060020a0316815260019190910190602001808311610423575b505050505090505b90565b61008d6004356024355b600160a060020a033316600090815260208190526040812054819060ff161561071a57600160a060020a038416815260026020526040812091505b8154811015610722578154600160a060020a033316908390839081101561000257600091825260209091200154600160a060020a0316141561076d5761071a565b60405180806020018281038252838181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019250505060405180910390f35b6040805163ffffffff9687168152948616602086015294909216838501526060830152608082015290519081900360a00190f35b604051808763ffffffff1681526020018663ffffffff1681526020018563ffffffff16815260200184815260200180602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050019850505050505050505060405180910390f35b6040518080602001806020018381038352858181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f1509050018381038252848181518152602001915080519060200190602002808383829060006004602084601f0104600f02600301f15090500194505050505060405180910390f35b600880546000198101908110156100025760009182526004027ff3f7a9fe364faab93b216da50a3214154f22a0a2b415b23a84c8169e8b636ee30190508054600182015463ffffffff8281169950640100000000830481169850604060020a8304169650606060020a91829004909102945067ffffffffffffffff16925090505b509091929394565b50565b505050505b50505050565b565b5060005b60018201548110156107755733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a031614156107bf5761071a565b600101610492565b8154600014801561078a575060018201546000145b156107e757600380546001810180835582818380158290116107c7578183600052602060002091820191016107c7919061086d565b600101610726565b5050506000928352506020909120018054600160a060020a031916851790555b82156108855781546001810180845583919082818380158290116108ba576000838152602090206108ba91810190830161086d565b5050506000928352506020909120018054600160a060020a031916851790555b600160a060020a038416600090815260026020908152604082208054838255818452918320909291610b6991908101905b80821115610881576000815560010161086d565b5090565b81600101600050805480600101828181548183558181151161097657818360005260206000209182019101610976919061086d565b5050506000928352506020909120018054600160a060020a031916331790556001548254600290910490116108ee5761071a565b8280156109145750600160a060020a03841660009081526020819052604090205460ff16155b156109ad57600160a060020a0384166000908152602081905260409020805460ff191660019081179091558054808201808355828183801582901161081c57600083905261081c9060008051602061124283398151915290810190830161086d565b5050506000928352506020909120018054600160a060020a031916331790556001805490830154600290910490116108ee5761071a565b821580156109d35750600160a060020a03841660009081526020819052604090205460ff165b1561083c5750600160a060020a0383166000908152602081905260408120805460ff191690555b60015481101561083c5783600160a060020a03166001600050828154811015610002576000919091526000805160206112428339815191520154600160a060020a03161415610add576001805460001981019081101561000257600091825260008051602061124283398151915201909054906101000a9004600160a060020a0316600160005082815481101561000257600080516020611242833981519152018054600160a060020a03191690921790915580546000198101808355909190828015829011610ae557818360005260206000209182019101610ae5919061086d565b6001016109fa565b5050600060048181556005805467ffffffffffffffff19169055600680548382558184529194509192508290610b3f907ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f9081019061086d565b5060018201805460008083559182526020909120610b5f9181019061086d565b505050505061083c565b5060018201805460008083559182526020909120610b899181019061086d565b506000925050505b60035481101561071a5783600160a060020a03166003600050828154811015610002576000919091526000805160206112228339815191520154600160a060020a03161415610c79576003805460001981019081101561000257600091825260008051602061122283398151915201909054906101000a9004600160a060020a0316600360005082815481101561000257600080516020611222833981519152018054600160a060020a03191690921790915580546000198101808355909190828015829011610715576107159060008051602061122283398151915290810190830161086d565b600101610b91565b60065460001415610cdf576004805463ffffffff1916881767ffffffff0000000019166401000000008802176bffffffff00000000000000001916604060020a8702176bffffffffffffffffffffffff16606060020a808704021790555b828015610d49575060045463ffffffff8881169116141580610d155750600454640100000000900463ffffffff90811690871614155b80610d32575060045463ffffffff808716604060020a9092041614155b80610d495750600454606060020a90819004028414155b15610d53576111f9565b506006905060005b8154811015610d9c578154600160a060020a033316908390839081101561000257600091825260209091200154600160a060020a03161415610de7576111f9565b5060005b6001820154811015610def5733600160a060020a03168260010160005082815481101561000257600091825260209091200154600160a060020a03161415610e26576111f9565b600101610d5b565b8215610e2e578154600181018084558391908281838015829011610e6357818360005260206000209182019101610e63919061086d565b600101610da0565b816001016000508054806001018281815481835581811511610ee657818360005260206000209182019101610ee6919061086d565b5050506000928352506020909120018054600160a060020a03191633179055600154825460029091049011610e97576111f9565b8215610f1d576005805467ffffffffffffffff19164217905560088054600181018083558281838015829011610f7257600402816004028360005260206000209182019101610f72919061108b565b5050506000928352506020909120018054600160a060020a03191633179055600180549083015460029091049011610e97576111f9565b600060048181556005805467ffffffffffffffff19169055600680548382558184529192918290611202907ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f9081019061086d565b5050509190906000526020600020906004020160005060048054825463ffffffff191663ffffffff9182161780845582546401000000009081900483160267ffffffff000000001991909116178084558254604060020a908190049092169091026bffffffff00000000000000001991909116178083558154606060020a908190048102819004026bffffffffffffffffffffffff9190911617825560055460018301805467ffffffffffffffff191667ffffffffffffffff9092169190911790556006805460028401805482825560008281526020902094959491928392918201918582156110ea5760005260206000209182015b828111156110ea578254825591600101919060010190611068565b505050506001015b8082111561088157600080825560018201805467ffffffffffffffff191690556002820180548282558183526020832083916110ca919081019061086d565b50600182018054600080835591825260209091206110839181019061086d565b506111109291505b80821115610881578054600160a060020a03191681556001016110f2565b505060018181018054918401805480835560008381526020902092938301929091821561115e5760005260206000209182015b8281111561115e578254825591600101919060010190611143565b5061116a9291506110f2565b5050600060048181556005805467ffffffffffffffff19169055600680548382558184529197509195509093508492506111c991507ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f9081019061086d565b50600182018054600080835591825260209091206111e99181019061086d565b50505050506111f9565b50505050505b50505050505050565b50600182018054600080835591825260209091206111f39181019061086d56c2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85bb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6`

// DeployReleaseOracle deploys a new Ethereum contract, binding an instance of ReleaseOracle to it.
func DeployReleaseOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ReleaseOracle, error) {
	parsed, err := abi.JSON(strings.NewReader(ReleaseOracleABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ReleaseOracleBin), backend)
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

// AuthProposals is a free data retrieval call binding the contract method 0x282fe4e5.
//
// Solidity: function AuthProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCaller) AuthProposals(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _ReleaseOracle.contract.Call(opts, out, "AuthProposals")
	return *ret0, err
}

// AuthProposals is a free data retrieval call binding the contract method 0x282fe4e5.
//
// Solidity: function AuthProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleSession) AuthProposals() ([]common.Address, error) {
	return _ReleaseOracle.Contract.AuthProposals(&_ReleaseOracle.CallOpts)
}

// AuthProposals is a free data retrieval call binding the contract method 0x282fe4e5.
//
// Solidity: function AuthProposals() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) AuthProposals() ([]common.Address, error) {
	return _ReleaseOracle.Contract.AuthProposals(&_ReleaseOracle.CallOpts)
}

// AuthVotes is a free data retrieval call binding the contract method 0xa29226f2.
//
// Solidity: function AuthVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleCaller) AuthVotes(opts *bind.CallOpts, user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	ret := new(struct {
		Promote []common.Address
		Demote  []common.Address
	})
	out := ret
	err := _ReleaseOracle.contract.Call(opts, out, "AuthVotes", user)
	return *ret, err
}

// AuthVotes is a free data retrieval call binding the contract method 0xa29226f2.
//
// Solidity: function AuthVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleSession) AuthVotes(user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	return _ReleaseOracle.Contract.AuthVotes(&_ReleaseOracle.CallOpts, user)
}

// AuthVotes is a free data retrieval call binding the contract method 0xa29226f2.
//
// Solidity: function AuthVotes(user address) constant returns(promote address[], demote address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) AuthVotes(user common.Address) (struct {
	Promote []common.Address
	Demote  []common.Address
}, error) {
	return _ReleaseOracle.Contract.AuthVotes(&_ReleaseOracle.CallOpts, user)
}

// CurrentVersion is a free data retrieval call binding the contract method 0x2b225f29.
//
// Solidity: function CurrentVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, time uint256)
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
	err := _ReleaseOracle.contract.Call(opts, out, "CurrentVersion")
	return *ret, err
}

// CurrentVersion is a free data retrieval call binding the contract method 0x2b225f29.
//
// Solidity: function CurrentVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, time uint256)
func (_ReleaseOracle *ReleaseOracleSession) CurrentVersion() (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Time   *big.Int
}, error) {
	return _ReleaseOracle.Contract.CurrentVersion(&_ReleaseOracle.CallOpts)
}

// CurrentVersion is a free data retrieval call binding the contract method 0x2b225f29.
//
// Solidity: function CurrentVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, time uint256)
func (_ReleaseOracle *ReleaseOracleCallerSession) CurrentVersion() (struct {
	Major  uint32
	Minor  uint32
	Patch  uint32
	Commit [20]byte
	Time   *big.Int
}, error) {
	return _ReleaseOracle.Contract.CurrentVersion(&_ReleaseOracle.CallOpts)
}

// ProposedVersion is a free data retrieval call binding the contract method 0x4c327071.
//
// Solidity: function ProposedVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, pass address[], fail address[])
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
	err := _ReleaseOracle.contract.Call(opts, out, "ProposedVersion")
	return *ret, err
}

// ProposedVersion is a free data retrieval call binding the contract method 0x4c327071.
//
// Solidity: function ProposedVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, pass address[], fail address[])
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

// ProposedVersion is a free data retrieval call binding the contract method 0x4c327071.
//
// Solidity: function ProposedVersion() constant returns(major uint32, minor uint32, patch uint32, commit bytes20, pass address[], fail address[])
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

// Signers is a free data retrieval call binding the contract method 0xf04c4758.
//
// Solidity: function Signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCaller) Signers(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _ReleaseOracle.contract.Call(opts, out, "Signers")
	return *ret0, err
}

// Signers is a free data retrieval call binding the contract method 0xf04c4758.
//
// Solidity: function Signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleSession) Signers() ([]common.Address, error) {
	return _ReleaseOracle.Contract.Signers(&_ReleaseOracle.CallOpts)
}

// Signers is a free data retrieval call binding the contract method 0xf04c4758.
//
// Solidity: function Signers() constant returns(address[])
func (_ReleaseOracle *ReleaseOracleCallerSession) Signers() ([]common.Address, error) {
	return _ReleaseOracle.Contract.Signers(&_ReleaseOracle.CallOpts)
}

// Demote is a paid mutator transaction binding the contract method 0x80bbbd4a.
//
// Solidity: function Demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Demote(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "Demote", user)
}

// Demote is a paid mutator transaction binding the contract method 0x80bbbd4a.
//
// Solidity: function Demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleSession) Demote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Demote(&_ReleaseOracle.TransactOpts, user)
}

// Demote is a paid mutator transaction binding the contract method 0x80bbbd4a.
//
// Solidity: function Demote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Demote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Demote(&_ReleaseOracle.TransactOpts, user)
}

// Nuke is a paid mutator transaction binding the contract method 0x0443b1ad.
//
// Solidity: function Nuke() returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Nuke(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "Nuke")
}

// Nuke is a paid mutator transaction binding the contract method 0x0443b1ad.
//
// Solidity: function Nuke() returns()
func (_ReleaseOracle *ReleaseOracleSession) Nuke() (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Nuke(&_ReleaseOracle.TransactOpts)
}

// Nuke is a paid mutator transaction binding the contract method 0x0443b1ad.
//
// Solidity: function Nuke() returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Nuke() (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Nuke(&_ReleaseOracle.TransactOpts)
}

// Promote is a paid mutator transaction binding the contract method 0x6195db9c.
//
// Solidity: function Promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Promote(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "Promote", user)
}

// Promote is a paid mutator transaction binding the contract method 0x6195db9c.
//
// Solidity: function Promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleSession) Promote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Promote(&_ReleaseOracle.TransactOpts, user)
}

// Promote is a paid mutator transaction binding the contract method 0x6195db9c.
//
// Solidity: function Promote(user address) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Promote(user common.Address) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Promote(&_ReleaseOracle.TransactOpts, user)
}

// Release is a paid mutator transaction binding the contract method 0x0d618178.
//
// Solidity: function Release(major uint32, minor uint32, patch uint32, commit bytes20) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) Release(opts *bind.TransactOpts, major uint32, minor uint32, patch uint32, commit [20]byte) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "Release", major, minor, patch, commit)
}

// Release is a paid mutator transaction binding the contract method 0x0d618178.
//
// Solidity: function Release(major uint32, minor uint32, patch uint32, commit bytes20) returns()
func (_ReleaseOracle *ReleaseOracleSession) Release(major uint32, minor uint32, patch uint32, commit [20]byte) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Release(&_ReleaseOracle.TransactOpts, major, minor, patch, commit)
}

// Release is a paid mutator transaction binding the contract method 0x0d618178.
//
// Solidity: function Release(major uint32, minor uint32, patch uint32, commit bytes20) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) Release(major uint32, minor uint32, patch uint32, commit [20]byte) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.Release(&_ReleaseOracle.TransactOpts, major, minor, patch, commit)
}

// UpdateRelease is a paid mutator transaction binding the contract method 0x645dce72.
//
// Solidity: function updateRelease(major uint32, minor uint32, patch uint32, commit bytes20, release bool) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) UpdateRelease(opts *bind.TransactOpts, major uint32, minor uint32, patch uint32, commit [20]byte, release bool) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "updateRelease", major, minor, patch, commit, release)
}

// UpdateRelease is a paid mutator transaction binding the contract method 0x645dce72.
//
// Solidity: function updateRelease(major uint32, minor uint32, patch uint32, commit bytes20, release bool) returns()
func (_ReleaseOracle *ReleaseOracleSession) UpdateRelease(major uint32, minor uint32, patch uint32, commit [20]byte, release bool) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.UpdateRelease(&_ReleaseOracle.TransactOpts, major, minor, patch, commit, release)
}

// UpdateRelease is a paid mutator transaction binding the contract method 0x645dce72.
//
// Solidity: function updateRelease(major uint32, minor uint32, patch uint32, commit bytes20, release bool) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) UpdateRelease(major uint32, minor uint32, patch uint32, commit [20]byte, release bool) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.UpdateRelease(&_ReleaseOracle.TransactOpts, major, minor, patch, commit, release)
}

// UpdateSigner is a paid mutator transaction binding the contract method 0xf460590b.
//
// Solidity: function updateSigner(user address, authorize bool) returns()
func (_ReleaseOracle *ReleaseOracleTransactor) UpdateSigner(opts *bind.TransactOpts, user common.Address, authorize bool) (*types.Transaction, error) {
	return _ReleaseOracle.contract.Transact(opts, "updateSigner", user, authorize)
}

// UpdateSigner is a paid mutator transaction binding the contract method 0xf460590b.
//
// Solidity: function updateSigner(user address, authorize bool) returns()
func (_ReleaseOracle *ReleaseOracleSession) UpdateSigner(user common.Address, authorize bool) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.UpdateSigner(&_ReleaseOracle.TransactOpts, user, authorize)
}

// UpdateSigner is a paid mutator transaction binding the contract method 0xf460590b.
//
// Solidity: function updateSigner(user address, authorize bool) returns()
func (_ReleaseOracle *ReleaseOracleTransactorSession) UpdateSigner(user common.Address, authorize bool) (*types.Transaction, error) {
	return _ReleaseOracle.Contract.UpdateSigner(&_ReleaseOracle.TransactOpts, user, authorize)
}
