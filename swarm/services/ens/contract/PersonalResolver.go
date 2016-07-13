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

// PersonalResolverABI is the input ABI used to generate the binding from.
const PersonalResolverABI = `[{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"setOwner","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"bytes32[]"}],"name":"deletePrivateRR","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"isPersonalResolver","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":true,"inputs":[{"name":"id","type":"bytes32"}],"name":"getExtended","outputs":[{"name":"data","type":"bytes"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"string"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"name":"setRR","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"bytes32[]"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"name":"setPrivateRR","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"qtype","type":"bytes32"},{"name":"index","type":"uint16"}],"name":"resolve","outputs":[{"name":"rcode","type":"uint16"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"string"}],"name":"deleteRR","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"label","type":"bytes32"}],"name":"findResolver","outputs":[{"name":"rcode","type":"uint16"},{"name":"ttl","type":"uint32"},{"name":"rnode","type":"bytes12"},{"name":"raddress","type":"address"}],"type":"function"},{"inputs":[],"type":"constructor"}]`

// PersonalResolverBin is the compiled bytecode used for deploying new contracts.
const PersonalResolverBin = `60606040525b33600060006101000a81548173ffffffffffffffffffffffffffffffffffffffff02191690830217905550600160016000506000600074010000000000000000000000000000000000000000028152602001908152602001600020600050600201600050819055505b611b498061007c6000396000f3606060405236156100a0576000357c01000000000000000000000000000000000000000000000000000000009004806313af4035146100a25780631b370194146100ba5780633f5665e7146101165780638021061c146101395780638bba944d146101bd5780638da5cb5b1461024057806391c8e7b914610279578063a16fdafa146102f9578063bc06183d14610361578063edc0277c146103c0576100a0565b005b6100b8600480803590602001909190505061042a565b005b610114600480803590602001909190803590602001908201803590602001919190808060200260200160405190810160405280939291908181526020018383602002808284378201915050505050509090919050506104b5565b005b6101236004805050610591565b6040518082815260200191505060405180910390f35b61014f600480803590602001909190505061059f565b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156101af5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61023e6004808035906020019091908035906020019082018035906020019191908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509090919080359060200190919080359060200190919080359060200190919080359060200190919050506105d3565b005b61024d6004805050610669565b604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6102f76004808035906020019091908035906020019082018035906020019191908080602002602001604051908101604052809392919081815260200183836020028082843782019150505050505090909190803590602001909190803590602001909190803590602001909190803590602001909190505061068f565b005b610321600480803590602001909190803590602001909190803590602001909190505061071f565b604051808661ffff1681526020018581526020018463ffffffff1681526020018361ffff1681526020018281526020019550505050505060405180910390f35b6103be6004808035906020019091908035906020019082018035906020019191908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505090909190505061084a565b005b6103df6004808035906020019091908035906020019091905050610932565b604051808561ffff1681526020018463ffffffff1681526020018381526020018273ffffffffffffffffffffffffffffffffffffffff16815260200194505050505060405180910390f35b600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561048657610002565b80600060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908302179055505b50565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561051557610002565b610523848460008651610997565b9150915060008160020160005054141561053c57610002565b60018160020160005054111561057b5761057681600001600050600070010000000000000000000000000000000002600060006000610a7f565b61058a565b610589848460008651610b1c565b5b5b50505050565b60006001905061059c565b90565b6020604051908101604052806000815260200150602060405190810160405280600081526020015090506105ce565b919050565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561063357610002565b6106478861064289610d119090565b610d5b565b9150915061065e8160000160005087878787610a7f565b5b5050505050505050565b600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156106ef57610002565b6106fd888860008a51610997565b915091506107148160000160005087878787610a7f565b5b5050505050505050565b600060006000600060006000600060008861ffff16111561073f5761083d565b600160005060008b815260200190815260200160002060005091506000826002016000505414156107755760039650865061083d565b8160000160005090507f2a000000000000000000000000000000000000000000000000000000000000008914806107cc57508060000160009054906101000a90047001000000000000000000000000000000000289145b15610838578060000160009054906101000a900470010000000000000000000000000000000002955085508060000160109054906101000a900463ffffffff16945084508060000160149054906101000a900461ffff169350835080600101600050549250825061083d565b61083d565b5050939792965093509350565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156108aa57610002565b6108be846108b985610d119090565b610d5b565b915091506000816002016000505414156108d757610002565b6001816002016000505411156109165761091181600001600050600070010000000000000000000000000000000002600060006000610a7f565b61092b565b61092a8461092585610d119090565b611066565b5b5b50505050565b600060006000600060006109468787611580565b925082506001600050600084815260200190815260200160002060005090506000816002016000505414156109805760039450845061098d565b610e109350835030915081505b5092959194509250565b6000600060006000600060018688010392505b8683121515610a4d576109d189898581518110156100025790602001906020020151611580565b9150600160005060008381526020019081526020016000206000509050600081600201600050541415610a3957600181600201600050819055506001600160005060008b81526020019081526020016000206000506002016000828282505401925050819055505b81985088505b8280600190039350506109aa565b88600160005060008b815260200190815260200160002060005080905094509450610a73565b50505094509492505050565b60208150602060ff161115610a9357610002565b838560000160006101000a8154816fffffffffffffffffffffffffffffffff021916908370010000000000000000000000000000000090040217905550828560000160106101000a81548163ffffffff02191690830217905550818560000160146101000a81548161ffff021916908302179055508085600101600050819055505b5050505050565b60006000610b3286866001870160018703610997565b9150915060016000506000610b5c8488600081518110156100025790602001906020020151611580565b815260200190815260200160002060006000820160006000820160006101000a8154906fffffffffffffffffffffffffffffffff02191690556000820160106101000a81549063ffffffff02191690556000820160146101000a81549061ffff02191690556001820160005060009055505060028201600050600090556003820160005080546000825590600052602060002090810190610c7a9190610bfd565b80821115610c76576000818150805460018160011615610100020316600290046000825580601f10610c2f5750610c6c565b601f016020900490600052602060002090810190610c6b9190610c4d565b80821115610c675760008181506000905550600101610c4d565b5090565b5b5050600101610bfd565b5090565b5b50505060018160020160008282825054039250508190555060018160020160005054148015610ce457506000700100000000000000000000000000000000028160000160005060000160009054906101000a900470010000000000000000000000000000000002145b8015610cf05750600183115b15610d0857610d0786866001870160018703610b1c565b5b5b505050505050565b604060405190810160405280600081526020016000815260200150600060208301905060406040519081016040528084518152602001828152602001509150610d55565b50919050565b6000600060406040519081016040528060008152602001600081526020015060406040519081016040528060008152602001600081526020015060006000610ddd604060405190810160405280600181526020017f2e00000000000000000000000000000000000000000000000000000000000000815260200150610d119090565b93505b610deb876115c59090565b151561103557610dfe84886115db909190565b9250610e0b836115c59090565b15610e1557610002565b610e2988610e24856116099090565b611580565b915060016000506000838152602001908152602001600020600050905060008160020160005054141561102b5760018160030160005081818054905001915081815481835581811511610f0d57818360005260206000209182019101610f0c9190610e8f565b80821115610f08576000818150805460018160011615610100020316600290046000825580601f10610ec15750610efe565b601f016020900490600052602060002090810190610efd9190610edf565b80821115610ef95760008181506000905550600101610edf565b5090565b5b5050600101610e8f565b5090565b5b50505050610f1c8361161b9090565b816003016000506001836003016000508054905003815481101561000257906000526020600020900160005b509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10610f9257805160ff1916838001178555610fc3565b82800160010185558215610fc3579182015b82811115610fc2578251826000505591602001919060010190610fa4565b5b509050610fee9190610fd0565b80821115610fea5760008181506000905550600101610fd0565b5090565b5050600181600201600050819055506001600160005060008a81526020019081526020016000206000506002016000828282505401925050819055505b8197508750610de0565b87600160005060008a81526020019081526020016000206000508090509550955061105b565b505050509250929050565b604060405190810160405280600081526020016000815260200150600060006000600060006110db6110d2604060405190810160405280600181526020017f2e00000000000000000000000000000000000000000000000000000000000000815260200150610d119090565b88611692909190565b95506110f1886110ec896116c09090565b610d5b565b9450945083600301600050805490509250600091505b828210156113ab576111de866111d68660030160005085815481101561000257906000526020600020900160005b508054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156111ca5780601f1061119f576101008083540402835291602001916111ca565b820191906000526020600020905b8154815290600101906020018083116111ad57829003601f168201915b5050505050610d119090565b611708909190565b1561139d578360030160005060018403815481101561000257906000526020600020900160005b508460030160005083815481101561000257906000526020600020900160005b509080546001816001161561010002031660029004828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061127357805485556112b0565b828001600101855582156112b057600052602060002091601f016020900482015b828111156112af578254825591600101919060010190611294565b5b5090506112db91906112bd565b808211156112d757600081815060009055506001016112bd565b5090565b505060018460030160005081818054905003915081815481835581811511611394578183600052602060002091820191016113939190611316565b8082111561138f576000818150805460018160011615610100020316600290046000825580601f106113485750611385565b601f0160209004906000526020600020908101906113849190611366565b808211156113805760008181506000905550600101611366565b5090565b5b5050600101611316565b5090565b5b505050506113ab565b5b8180600101925050611107565b6113b6866116099090565b9050600160005060006113c98784611580565b815260200190815260200160002060006000820160006000820160006101000a8154906fffffffffffffffffffffffffffffffff02191690556000820160106101000a81549063ffffffff02191690556000820160146101000a81549061ffff021916905560018201600050600090555050600282016000506000905560038201600050805460008255906000526020600020908101906114e7919061146a565b808211156114e3576000818150805460018160011615610100020316600290046000825580601f1061149c57506114d9565b601f0160209004906000526020600020908101906114d891906114ba565b808211156114d457600081815060009055506001016114ba565b5090565b5b505060010161146a565b5090565b5b5050506001846002016000828282505403925050819055506001846002016000505414801561155157506000700100000000000000000000000000000000028460000160005060000160009054906101000a900470010000000000000000000000000000000002145b80156115655750611563876115c59090565b155b15611575576115748888611066565b5b5b5050505050505050565b6000828260405180838152600c018281526020019250505060405180910390207401000000000000000000000000000000000000000080910402905080505b92915050565b6000600082600001511490506115d6565b919050565b604060405190810160405280600081526020016000815260200150611601838383611724565b505b92915050565b6000815160208301512090505b919050565b60206040519081016040528060008152602001506020604051908101604052806000815260200150600083600001516040518059106116575750595b9080825280602002602001820160405250915060208201905061168381856020015186600001516117da565b81925061168b565b5050919050565b6040604051908101604052806000815260200160008152602001506116b883838361182d565b505b92915050565b6040604051908101604052806000815260200160008152602001506040604051908101604052808360000151815260200183602001518152602001509050611703565b919050565b6000600061171684846118fc565b14905061171e565b92915050565b604060405190810160405280600081526020016000815260200150600061175d85600001518660200151866000015187602001516119c7565b905080836020019090818152602001505084602001518103856000015103836000019090818152602001505084602001518114156117aa57600085600001909081815260200150506117ca565b836000015183600001510185600001818151039150909081815260200150505b8291506117d2565b509392505050565b60005b6020821015156118095782518452602084019350835060208301925082505b60208203915081506117dd565b6001826020036101000a039050801983511681855116818117865250505b50505050565b60406040519081016040528060008152602001600081526020015060006118668560000151866020015186600001518760200151611a83565b90508460200151836020019090818152602001505084602001518103836000019090818152602001505084600001518560200151018114156118b757600085600001909081815260200150506118ec565b836000015183600001510185600001818151039150909081815260200150508360000151810185602001909081815260200150505b8291506118f4565b509392505050565b6000600060006000600060006000600060008a6000015197508a600001518a60000151101561192f578960000151975087505b8a60200151965089602001519550600094505b878510156119a7578651935085519250828414151561198957600185896020030160080260020a03199150818316828516039050600081141515611988578098506119b9565b5b602087019650865060208601955085505b6020850194508450611942565b89600001518b600001510398506119b9565b505050505050505092915050565b60006000600060008786111515611a7057602086111515611a325760018660200360080260020a031980865116878a03890194505b808286511614611a1e576001850394508860018601116119fc57889450611a24565b87850194505b5050829350611a7856611a6f565b85852091508588038701925082505b8683101515611a6e57858320905080821415611a61578583019350611a78565b6001830392508250611a41565b5b5b869350611a78565b505050949350505050565b600060006000600060008887111515611b3357602087111515611aea5760018760200360080260020a031980875116888b038a018a96505b818388511614611adb57600187019650806001880310611abb578b8b0196505b505050839450611b3d56611b32565b868620915087935083506000925082505b86890383111515611b3157868420905080821415611b1b57839450611b3d565b60018401935083505b8280600101935050611afb565b5b5b8888019450611b3d565b5050505094935050505056`

// DeployPersonalResolver deploys a new Ethereum contract, binding an instance of PersonalResolver to it.
func DeployPersonalResolver(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PersonalResolver, error) {
	parsed, err := abi.JSON(strings.NewReader(PersonalResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PersonalResolverBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PersonalResolver{PersonalResolverCaller: PersonalResolverCaller{contract: contract}, PersonalResolverTransactor: PersonalResolverTransactor{contract: contract}}, nil
}

// PersonalResolver is an auto generated Go binding around an Ethereum contract.
type PersonalResolver struct {
	PersonalResolverCaller     // Read-only binding to the contract
	PersonalResolverTransactor // Write-only binding to the contract
}

// PersonalResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PersonalResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PersonalResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PersonalResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PersonalResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PersonalResolverSession struct {
	Contract     *PersonalResolver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PersonalResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PersonalResolverCallerSession struct {
	Contract *PersonalResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// PersonalResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PersonalResolverTransactorSession struct {
	Contract     *PersonalResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// PersonalResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type PersonalResolverRaw struct {
	Contract *PersonalResolver // Generic contract binding to access the raw methods on
}

// PersonalResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PersonalResolverCallerRaw struct {
	Contract *PersonalResolverCaller // Generic read-only contract binding to access the raw methods on
}

// PersonalResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PersonalResolverTransactorRaw struct {
	Contract *PersonalResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPersonalResolver creates a new instance of PersonalResolver, bound to a specific deployed contract.
func NewPersonalResolver(address common.Address, backend bind.ContractBackend) (*PersonalResolver, error) {
	contract, err := bindPersonalResolver(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))
	if err != nil {
		return nil, err
	}
	return &PersonalResolver{PersonalResolverCaller: PersonalResolverCaller{contract: contract}, PersonalResolverTransactor: PersonalResolverTransactor{contract: contract}}, nil
}

// NewPersonalResolverCaller creates a new read-only instance of PersonalResolver, bound to a specific deployed contract.
func NewPersonalResolverCaller(address common.Address, caller bind.ContractCaller) (*PersonalResolverCaller, error) {
	contract, err := bindPersonalResolver(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &PersonalResolverCaller{contract: contract}, nil
}

// NewPersonalResolverTransactor creates a new write-only instance of PersonalResolver, bound to a specific deployed contract.
func NewPersonalResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*PersonalResolverTransactor, error) {
	contract, err := bindPersonalResolver(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &PersonalResolverTransactor{contract: contract}, nil
}

// bindPersonalResolver binds a generic wrapper to an already deployed contract.
func bindPersonalResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PersonalResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PersonalResolver *PersonalResolverRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PersonalResolver.Contract.PersonalResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PersonalResolver *PersonalResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PersonalResolver.Contract.PersonalResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PersonalResolver *PersonalResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PersonalResolver.Contract.PersonalResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PersonalResolver *PersonalResolverCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PersonalResolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PersonalResolver *PersonalResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PersonalResolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PersonalResolver *PersonalResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PersonalResolver.Contract.contract.Transact(opts, method, params...)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_PersonalResolver *PersonalResolverCaller) FindResolver(opts *bind.CallOpts, nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	ret := new(struct {
		Rcode    uint16
		Ttl      uint32
		Rnode    [12]byte
		Raddress common.Address
	})
	out := ret
	err := _PersonalResolver.contract.Call(opts, out, "findResolver", nodeId, label)
	return *ret, err
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_PersonalResolver *PersonalResolverSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _PersonalResolver.Contract.FindResolver(&_PersonalResolver.CallOpts, nodeId, label)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_PersonalResolver *PersonalResolverCallerSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _PersonalResolver.Contract.FindResolver(&_PersonalResolver.CallOpts, nodeId, label)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_PersonalResolver *PersonalResolverCaller) GetExtended(opts *bind.CallOpts, id [32]byte) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _PersonalResolver.contract.Call(opts, out, "getExtended", id)
	return *ret0, err
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_PersonalResolver *PersonalResolverSession) GetExtended(id [32]byte) ([]byte, error) {
	return _PersonalResolver.Contract.GetExtended(&_PersonalResolver.CallOpts, id)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_PersonalResolver *PersonalResolverCallerSession) GetExtended(id [32]byte) ([]byte, error) {
	return _PersonalResolver.Contract.GetExtended(&_PersonalResolver.CallOpts, id)
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_PersonalResolver *PersonalResolverCaller) IsPersonalResolver(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _PersonalResolver.contract.Call(opts, out, "isPersonalResolver")
	return *ret0, err
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_PersonalResolver *PersonalResolverSession) IsPersonalResolver() (bool, error) {
	return _PersonalResolver.Contract.IsPersonalResolver(&_PersonalResolver.CallOpts)
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_PersonalResolver *PersonalResolverCallerSession) IsPersonalResolver() (bool, error) {
	return _PersonalResolver.Contract.IsPersonalResolver(&_PersonalResolver.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_PersonalResolver *PersonalResolverCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _PersonalResolver.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_PersonalResolver *PersonalResolverSession) Owner() (common.Address, error) {
	return _PersonalResolver.Contract.Owner(&_PersonalResolver.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_PersonalResolver *PersonalResolverCallerSession) Owner() (common.Address, error) {
	return _PersonalResolver.Contract.Owner(&_PersonalResolver.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_PersonalResolver *PersonalResolverCaller) Resolve(opts *bind.CallOpts, nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	ret := new(struct {
		Rcode uint16
		Rtype [16]byte
		Ttl   uint32
		Len   uint16
		Data  [32]byte
	})
	out := ret
	err := _PersonalResolver.contract.Call(opts, out, "resolve", nodeId, qtype, index)
	return *ret, err
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_PersonalResolver *PersonalResolverSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _PersonalResolver.Contract.Resolve(&_PersonalResolver.CallOpts, nodeId, qtype, index)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_PersonalResolver *PersonalResolverCallerSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _PersonalResolver.Contract.Resolve(&_PersonalResolver.CallOpts, nodeId, qtype, index)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_PersonalResolver *PersonalResolverTransactor) DeletePrivateRR(opts *bind.TransactOpts, rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "deletePrivateRR", rootNodeId, name)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_PersonalResolver *PersonalResolverSession) DeletePrivateRR(rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeletePrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) DeletePrivateRR(rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeletePrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_PersonalResolver *PersonalResolverTransactor) DeleteRR(opts *bind.TransactOpts, rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "deleteRR", rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_PersonalResolver *PersonalResolverSession) DeleteRR(rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeleteRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) DeleteRR(rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeleteRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(newOwner address) returns()
func (_PersonalResolver *PersonalResolverTransactor) SetOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "setOwner", newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(newOwner address) returns()
func (_PersonalResolver *PersonalResolverSession) SetOwner(newOwner common.Address) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetOwner(&_PersonalResolver.TransactOpts, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(newOwner address) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) SetOwner(newOwner common.Address) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetOwner(&_PersonalResolver.TransactOpts, newOwner)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactor) SetPrivateRR(opts *bind.TransactOpts, rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "setPrivateRR", rootNodeId, name, rtype, ttl, len, data)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverSession) SetPrivateRR(rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetPrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) SetPrivateRR(rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetPrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactor) SetRR(opts *bind.TransactOpts, rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "setRR", rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverSession) SetRR(rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) SetRR(rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}
